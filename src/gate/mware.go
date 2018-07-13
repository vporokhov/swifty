package main

import (
	"gopkg.in/mgo.v2/bson"
	"strings"
	"fmt"
	"context"
	"errors"

	"../apis/apps"
	"../common"
	"../common/crypto"
)

type MwareDesc struct {
	// These objects are kept in Mongo, which requires the below
	// field to be present...
	ObjID		bson.ObjectId	`bson:"_id,omitempty"`

	SwoId				`bson:",inline"`
	Labels		[]string	`bson:"labels"`
	Cookie		string		`bson:"cookie"`
	MwareType	string		`bson:"mwaretype"`	// Middleware type
	Client		string		`bson:"client,omitempty"`		// Middleware client (db user)
	Secret		string		`bson:"secret"`		// Client secret (e.g. password)
	Namespace	string		`bson:"namespace,omitempty"`	// Client namespace (e.g. dbname, mq domain)
	State		int		`bson:"state"`		// Mware state
	UserData	string		`bson:"userdata,omitempty"`
}

var mwStates = map[int]string {
	swy.DBMwareStatePrp:	"preparing",
	swy.DBMwareStateRdy:	"ready",
	swy.DBMwareStateTrm:	"terminating",
	swy.DBMwareStateStl:	"stalled",
	swy.DBMwareStateNo:	"dead",
}

func (mw *MwareDesc)ToState(ctx context.Context, st, from int) error {
	q := bson.M{"_id": mw.ObjID}
	if from != -1 {
		q["state"] = from
	}

	err := dbUpdateSet(ctx, q, bson.M{"state": st}, &MwareDesc{})
	if err == nil {
		mw.State = st
	}

	return err
}

type MwareOps struct {
	Init	func(ctx context.Context, conf *YAMLConfMw, mwd *MwareDesc) (error)
	Fini	func(ctx context.Context, conf *YAMLConfMw, mwd *MwareDesc) (error)
	Event	func(ctx context.Context, conf *YAMLConfMw, source *FnEventDesc, mwd *MwareDesc, on bool) (error)
	GenSec	func(ctx context.Context, conf *YAMLConfMw, fid *SwoId, id string)([][2]string, error)
	GetEnv	func(conf *YAMLConfMw, mwd *MwareDesc) ([][2]string)
	Info	func(ctx context.Context, conf *YAMLConfMw, mwd *MwareDesc, ifo *swyapi.MwareInfo) (error)
	Devel	bool
	LiteOK	bool
}

func mkEnvId(name, mwType, envName, value string) [2]string {
	return [2]string{"MWARE_" + strings.ToUpper(mwType) + strings.ToUpper(name) + "_" + envName, value}
}

func mkEnv(mwd *MwareDesc, envName, value string) [2]string {
	return mkEnvId(mwd.Name, mwd.MwareType, envName, value)
}

func mwGenUserPassEnvs(mwd *MwareDesc, mwaddr string) ([][2]string) {
	return [][2]string{
		mkEnv(mwd, "ADDR", mwaddr),
		mkEnv(mwd, "USER", mwd.Client),
		mkEnv(mwd, "PASS", mwd.Secret),
	}
}

func mwareGetCookie(ctx context.Context, id SwoId, name string) (string, error) {
	var mw MwareDesc

	id.Name = name
	err := dbFind(ctx, id.dbReq(), &mw)
	if err != nil {
		return "", fmt.Errorf("No such mware: %s", id.Str())
	}
	if mw.State != swy.DBMwareStateRdy {
		return "", errors.New("Mware not ready")
	}

	return mw.Cookie, nil
}

func mwareGenerateUserPassClient(ctx context.Context, mwd *MwareDesc) (error) {
	var err error

	mwd.Client, err = swy.GenRandId(32)
	if err != nil {
		ctxlog(ctx).Errorf("Can't generate clnt for %s: %s", mwd.SwoId.Str(), err.Error())
		return err
	}

	mwd.Secret, err = swy.GenRandId(64)
	if err != nil {
		ctxlog(ctx).Errorf("Can't generate secret for %s: %s", mwd.SwoId.Str(), err.Error())
		return err
	}

	return nil
}

var mwareHandlers = map[string]*MwareOps {
	"maria":	&MwareMariaDB,
	"postgres":	&MwarePostgres,
	"rabbit":	&MwareRabbitMQ,
	"mongo":	&MwareMongo,
	"s3":		&MwareS3,
	"authjwt":	&MwareAuthJWT,
}

func mwareGenerateSecret(ctx context.Context, fid *SwoId, typ, id string) ([][2]string, error) {
	handler, ok := mwareHandlers[typ]
	if !ok {
		return nil, fmt.Errorf("No handler for %s mware", typ)
	}

	if handler.GenSec == nil {
		return nil, fmt.Errorf("No secrets generator for %s", typ)
	}

	return handler.GenSec(ctx, &conf.Mware, fid, id)
}

func mwareRemoveId(ctx context.Context, conf *YAMLConfMw, id *SwoId) *swyapi.GateErr {
	var item MwareDesc

	err := dbFind(ctx, id.dbReq(), &item)
	if err != nil {
		return GateErrD(err)
	}

	return item.Remove(ctx, conf)
}

func (item *MwareDesc)Remove(ctx context.Context, conf *YAMLConfMw) *swyapi.GateErr {
	handler, ok := mwareHandlers[item.MwareType]
	if !ok {
		return GateErrC(swy.GateGenErr) /* Shouldn't happen */
	}

	err := item.ToState(ctx, swy.DBMwareStateTrm, item.State)
	if err != nil {
		ctxlog(ctx).Errorf("Can't terminate mware %s", item.SwoId.Str())
		return GateErrM(swy.GateGenErr, "Cannot terminate mware")
	}

	err = handler.Fini(ctx, conf, item)
	if err != nil {
		ctxlog(ctx).Errorf("Failed cleanup for mware %s: %s", item.SwoId.Str(), err.Error())
		goto stalled
	}

	err = swk8sMwSecretRemove(ctx, item.Cookie)
	if err != nil {
		ctxlog(ctx).Errorf("Failed secret cleanup for mware %s: %s", item.SwoId.Str(), err.Error())
		goto stalled
	}

	err = dbRemoveId(ctx, &MwareDesc{}, item.ObjID)
	if err != nil {
		ctxlog(ctx).Errorf("Can't remove mware %s: %s", item.SwoId.Str(), err.Error())
		goto stalled
	}
	gateMwares.WithLabelValues(item.MwareType).Dec()

	return nil

stalled:
	item.ToState(ctx, swy.DBMwareStateStl, -1)
	return GateErrE(swy.GateGenErr, err)
}

func (item *MwareDesc)toFnInfo(ctx context.Context) *swyapi.MwareInfo {
	return &swyapi.MwareInfo {
		ID: item.ObjID.Hex(),
		Name: item.SwoId.Name,
		Type: item.MwareType,
	}
}

func (item *MwareDesc)toInfo(ctx context.Context, conf *YAMLConfMw, details bool) (*swyapi.MwareInfo, *swyapi.GateErr) {
	resp := &swyapi.MwareInfo{
		ID:		item.ObjID.Hex(),
		Name:		item.SwoId.Name,
		Project:	item.SwoId.Project,
		Type:		item.MwareType,
		Labels:		item.Labels,
	}

	if details {
		resp.UserData = item.UserData

		handler, ok := mwareHandlers[item.MwareType]
		if !ok {
			return nil, GateErrC(swy.GateGenErr) /* Shouldn't happen */
		}

		if handler.Info != nil {
			err := handler.Info(ctx, conf, item, resp)
			if err != nil {
				return nil, GateErrE(swy.GateGenErr, err)
			}
		}
	}

	return resp, nil
}

func getMwareDesc(id *SwoId, params *swyapi.MwareAdd) *MwareDesc {
	ret := &MwareDesc {
		SwoId:		*id,
		MwareType:	params.Type,
		State:		swy.DBMwareStatePrp,
		UserData:	params.UserData,
	}

	ret.Cookie = ret.SwoId.Cookie()
	return ret
}

func (mwd *MwareDesc)Setup(ctx context.Context, conf *YAMLConfMw) (string, *swyapi.GateErr) {
	var handler *MwareOps
	var ok bool
	var err, erc error

	ctxlog(ctx).Debugf("set up wmare %s:%s", mwd.SwoId.Str(), mwd.MwareType)

	mwd.ObjID = bson.NewObjectId()
	err = dbInsert(ctx, mwd)
	if err != nil {
		ctxlog(ctx).Errorf("Can't add mware %s: %s", mwd.SwoId.Str(), err.Error())
		return "", GateErrD(err)
	}

	gateMwares.WithLabelValues(mwd.MwareType).Inc()

	handler, ok = mwareHandlers[mwd.MwareType]
	if !ok {
		err = fmt.Errorf("Bad mware type %s", mwd.MwareType)
		goto outdb
	}

	if handler.Devel && !SwyModeDevel {
		err = fmt.Errorf("Bad mware type %s", mwd.MwareType)
		goto outdb
	}

	if isLite() && !handler.LiteOK {
		err = fmt.Errorf("Bad mware type %s", mwd.MwareType)
		goto outdb
	}

	err = handler.Init(ctx, conf, mwd)
	if err != nil {
		err = fmt.Errorf("mware init error: %s", err.Error())
		goto outdb
	}

	err = swk8sMwSecretAdd(ctx, mwd.Cookie, handler.GetEnv(conf, mwd))
	if err != nil {
		goto outh
	}

	mwd.Secret, err = swycrypt.EncryptString(gateSecPas, mwd.Secret)
	if err != nil {
		ctxlog(ctx).Errorf("Mw secret encrypt error: %s", err.Error())
		err = errors.New("Encrypt error")
		goto outs
	}

	mwd.State = swy.DBMwareStateRdy
	err = dbUpdateId(ctx, mwd.ObjID, bson.M {
				"client":	mwd.Client,
				"secret":	mwd.Secret,
				"namespace":	mwd.Namespace,
				"state":	mwd.State }, &MwareDesc{})
	if err != nil {
		ctxlog(ctx).Errorf("Can't update added %s: %s", mwd.SwoId.Str(), err.Error())
		err = errors.New("DB error")
		goto outs
	}

	return mwd.ObjID.Hex(), nil

outs:
	erc = swk8sMwSecretRemove(ctx, mwd.Cookie)
	if erc != nil {
		goto stalled
	}
outh:
	erc = handler.Fini(ctx, conf, mwd)
	if erc != nil {
		goto stalled
	}
outdb:
	erc = dbRemoveId(ctx, &MwareDesc{}, mwd.ObjID)
	if erc != nil {
		goto stalled
	}
	gateMwares.WithLabelValues(mwd.MwareType).Dec()
out:
	ctxlog(ctx).Errorf("mwareSetup: %s", err.Error())
	return "", GateErrE(swy.GateGenErr, err)

stalled:
	mwd.ToState(ctx, swy.DBMwareStateStl, -1)
	goto out
}
