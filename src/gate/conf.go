/*
 * © 2018 SwiftyCloud OÜ. All rights reserved.
 * Info: info@swifty.cloud
 */

package main

import (
	"os"
	"fmt"
	"time"
	"errors"
	"strconv"
	"encoding/hex"
	"swifty/common"
	"swifty/common/http"
	"swifty/common/xrest/sysctl"
)

type YAMLConfWdog struct {
	Volume		string			`yaml:"volume"`
	Port		int			`yaml:"port"`
	ImgPref		string			`yaml:"img-prefix"`
	Namespace	string			`yaml:"k8s-namespace"`
	Proxy		int			`yaml:"proxy"`


	p_port		string
}

func setupMwareAddr(conf *YAMLConf) error {
	var err error

	mc := &conf.Mware

	if mc.Maria != nil {
		mc.Maria.c = xh.ParseXCreds(mc.Maria.Creds)
		mc.Maria.c.Resolve()
		mc.Maria.c.Pass, err = gateSecrets.Get(mc.Maria.c.Pass)
		if err != nil {
			return errors.New("mware.maria secret not found")
		}
	}

	if mc.Rabbit != nil {
		mc.Rabbit.c = xh.ParseXCreds(mc.Rabbit.Creds)
		mc.Rabbit.c.Resolve()
		mc.Rabbit.c.Pass, err = gateSecrets.Get(mc.Rabbit.c.Pass)
		if err != nil {
			return errors.New("mware.rabbit secret not found")
		}
	}

	if mc.Mongo != nil {
		mc.Mongo.c = xh.ParseXCreds(mc.Mongo.Creds)
		mc.Mongo.c.Resolve()
		mc.Mongo.c.Pass, err = gateSecrets.Get(mc.Mongo.c.Pass)
		if err != nil {
			return errors.New("mware.mongo secret not found")
		}
	}

	if mc.Postgres != nil {
		mc.Postgres.c = xh.ParseXCreds(mc.Postgres.Creds)
		mc.Postgres.c.Resolve()
		mc.Postgres.c.Pass, err = gateSecrets.Get(mc.Postgres.c.Pass)
		if err != nil  {
			return errors.New("mware.postgres secret not found")
		}
	}

	if mc.S3 != nil {
		mc.S3.c = xh.ParseXCreds(mc.S3.Creds)
		mc.S3.c.Resolve()
		mc.S3.c.Pass, err = gateSecrets.Get(mc.S3.c.Pass)
		if err != nil {
			return errors.New("mware.s3 secret not found")
		}

		mc.S3.cn = xh.ParseXCreds(mc.S3.Notify)
		mc.S3.cn.Resolve()
		mc.S3.cn.Pass, err = gateSecrets.Get(mc.S3.cn.Pass)
		if err != nil {
			return errors.New("mware.s3.notify secret not found")
		}
	}

	return nil
}

func (cw *YAMLConfWdog)Validate() error {
	if cw.Volume == "" {
		return errors.New("'wdog.volume' not set")
	}
	fi, err := os.Stat(functionsDir())
	if err != nil || !fi.IsDir() {
		return errors.New("'wdog.volume'/functions should be dir")
	}
	fi, err = os.Stat(packagesDir())
	if err != nil || !fi.IsDir() {
		return errors.New("'wdog.volume'/packages should be dir")
	}

	if cw.Port == 0 {
		return errors.New("'wdog.port' not set")
	}
	cw.p_port = strconv.Itoa(cw.Proxy)
	if cw.ImgPref == "" {
		cw.ImgPref = "swifty"
		fmt.Printf("'wdog.img-prefix' not set, using default\n")
	}
	sysctl.AddStringSysctl("wdog_image_prefix", &cw.ImgPref)
	if cw.Namespace == "" {
		fmt.Printf("'wdog.k8s-namespace' not set, will use default\n")
	}
	return nil
}

type YAMLConfDaemon struct {
	Addr		string			`yaml:"address"`
	CallGate	string			`yaml:"callgate"`
	ApiGate		string			`yaml:"apigate"`
	LogLevel	string			`yaml:"loglevel"`
	Prometheus	string			`yaml:"prometheus"`
	HTTPS		*xhttp.YAMLConfHTTPS	`yaml:"https,omitempty"`
}

func (cd *YAMLConfDaemon)Validate() error {
	if cd.Addr == "" {
		return errors.New("'daemon.address' not set, want HOST:PORT value")
	}
	if cd.Prometheus == "" {
		return errors.New("'daemon.prometheus' not set, want HOST:PORT value")
	}
	if cd.CallGate == "" {
		fmt.Printf("'daemon.callgate' not set, gate is callgate\n")
	}
	sysctl.AddStringSysctl("gate_call", &cd.CallGate)
	if cd.ApiGate == "" {
		fmt.Printf("'daemon.apigate' not set, gate is apigate\n")
	}
	sysctl.AddStringSysctl("gate_api", &cd.ApiGate)
	if cd.LogLevel == "" {
		fmt.Printf("'daemon.loglevel' not set, using \"warn\" one\n")
	}
	if cd.HTTPS == nil {
		fmt.Printf("'daemon.https' not set, will try to work over plain http\n")
	}
	return nil
}

type YAMLConfAdmd struct {
	Addr		string			`yaml:"address"`
}

func (ac *YAMLConfAdmd)Validate() error {
	if ac.Addr == "" {
		return errors.New("'admd.address' not set, want HOST:PORT value")
	}
	sysctl.AddStringSysctl("admd_addr", &ac.Addr)
	return nil
}

type YAMLConfRabbit struct {
	Creds		string			`yaml:"creds"`
	AdminPort	string			`yaml:"admport"`
	c		*xh.XCreds
}

type YAMLConfMaria struct {
	Creds		string			`yaml:"creds"`
	QDB		string			`yaml:"quotdb"`
	c		*xh.XCreds
}

type YAMLConfMongo struct {
	Creds		string			`yaml:"creds"`
	c		*xh.XCreds
}

type YAMLConfPostgres struct {
	Creds		string			`yaml:"creds"`
	AdminPort	string			`yaml:"admport"`
	c		*xh.XCreds
}

type YAMLConfS3 struct {
	Creds		string			`yaml:"creds"`
	API		string			`yaml:"api"`
	Notify		string			`yaml:"notify"`
	HiddenKeyTmo	int			`yaml:"hidden-key-timeout"`
	c		*xh.XCreds
	cn		*xh.XCreds
}

type YAMLConfWS struct {
	API		string			`yaml:"api"`
}

type YAMLConfMw struct {
	SecKey		string			`yaml:"mwseckey"`
	Rabbit		*YAMLConfRabbit		`yaml:"rabbit,omitempty"`
	Maria		*YAMLConfMaria		`yaml:"maria,omitempty"`
	Mongo		*YAMLConfMongo		`yaml:"mongo,omitempty"`
	Postgres	*YAMLConfPostgres	`yaml:"postgres,omitempty"`
	S3		*YAMLConfS3		`yaml:"s3,omitempty"`
	WS		*YAMLConfWS		`yaml:"websocket,omitempty"`
}

func (cm *YAMLConfMw)Validate() error {
	if cm.SecKey == "" {
		return errors.New("'middleware.mwseckey' not set")
	}

	v, err := gateSecrets.Get(cm.SecKey)
	if err != nil {
		return errors.New("'middleware.mwseckey' secret not found")
	}

	gateSecPas, err = hex.DecodeString(v)
	if err != nil || len(gateSecPas) < 16 {
		return errors.New("'middleware.mwseckey' format error")
	}

	if cm.S3 != nil {
		if cm.S3.HiddenKeyTmo == 0 {
			cm.S3.HiddenKeyTmo = 120
			fmt.Printf("'middleware.s3.hidden-key-timeout' not set, using default 120sec\n")
		}
		sysctl.AddIntSysctl("s3_hidden_key_timeout_sec", &cm.S3.HiddenKeyTmo)
		sysctl.AddStringSysctl("gate_s3api", &cm.S3.API)
	}

	if cm.WS != nil {
		if cm.WS.API == "" {
			fmt.Printf("'middleware.websocket.api' not set, gate is wsgate\n")
		}
		sysctl.AddStringSysctl("gate_ws", &cm.WS.API)
	}

	return nil
}

type YAMLConfRange struct {
	Min		int			`yaml:"min"`
	Max		int			`yaml:"max"`
	Def		int			`yaml:"def"`
}

type YAMLConfRt struct {
	Timeout		YAMLConfRange		`yaml:"timeout"`
	Memory		YAMLConfRange		`yaml:"memory"`
	MaxReplicas	int			`yaml:"max-replicas"`
}

func (cr *YAMLConfRt)Validate() error {
	if cr.MaxReplicas == 0 {
		cr.MaxReplicas = 32
		fmt.Printf("'runtime.max-replicas' not set, using default 32\n")
	}
	sysctl.AddIntSysctl("fn_replicas_limit", &cr.MaxReplicas)
	if cr.Timeout.Max == 0 {
		cr.Timeout.Max = 60
		fmt.Printf("'runtime.timeout.max' not set, using default 1min\n")
	}
	sysctl.AddIntSysctl("fn_timeout_max_sec", &cr.Timeout.Max)
	if cr.Timeout.Def == 0 {
		cr.Timeout.Def = 1
		fmt.Printf("'runtime.timeout.def' not set, using default 1sec\n")
	}
	sysctl.AddIntSysctl("fn_timeout_def_sec", &cr.Timeout.Def)
	if cr.Memory.Min == 0 {
		cr.Memory.Min = 64
		fmt.Printf("'runtime.memory.min' not set, using default 64m\n")
	}
	sysctl.AddIntSysctl("fn_memory_min_mb", &cr.Memory.Min)
	if cr.Memory.Max == 0 {
		cr.Memory.Max = 1024
		fmt.Printf("'runtime.memory.max' not set, using default 1g\n")
	}
	sysctl.AddIntSysctl("fn_memory_max_mb", &cr.Memory.Max)
	if cr.Memory.Def == 0 {
		cr.Memory.Def = 128
		fmt.Printf("'runtime.memory.def' not set, using default 128m\n")
	}
	sysctl.AddIntSysctl("fn_memory_def_mb", &cr.Memory.Def)
	return nil
}

type YAMLConfDemoRepo struct {
	URL		string			`yaml:"url"`
	AAASDep		string			`yaml:"aaas-dep"`
	EmptySources	string			`yaml:"empty-sources"`
}

func (dr *YAMLConfDemoRepo)Validate() error {
	if dr.URL == "" {
		return errors.New("'demo-repo.url' not set")
	}

	if dr.AAASDep == "" {
		fmt.Printf("'demo-repo.aaas-dep' not set, using default\n")
		dr.AAASDep = "swy-aaas.yaml"
	}
	sysctl.AddStringSysctl("aaas_dep_file", &dr.AAASDep)
	if dr.EmptySources == "" {
		fmt.Printf("'demo-repo.empty-sources' not set, using default\n")
		dr.EmptySources = "functions/empty"
	}
	sysctl.AddStringSysctl("empty_sources_path", &dr.EmptySources)

	return nil
}

type YAMLConf struct {
	Home		string			`yaml:"home"`
	DB		string			`yaml:"db"`
	Daemon		YAMLConfDaemon		`yaml:"daemon"`
	Admd		YAMLConfAdmd		`yaml:"admd"`
	Mware		YAMLConfMw		`yaml:"middleware"`
	Runtime		YAMLConfRt		`yaml:"runtime"`
	Wdog		YAMLConfWdog		`yaml:"wdog"`
	RepoSyncDelay	int			`yaml:"repo-sync-delay"`
	RepoSyncPeriod	int			`yaml:"repo-sync-period"`
	RunRate		int			`yaml:"tryrun-rate"`
	DemoRepo	YAMLConfDemoRepo	`yaml:"demo-repo"`
}

func (c *YAMLConf)Validate() error {
	err := c.Daemon.Validate()
	if err != nil {
		return err
	}
	err = c.Admd.Validate()
	if err != nil {
		return err
	}
	err = c.Mware.Validate()
	if err != nil {
		return err
	}
	err = c.Runtime.Validate()
	if err != nil {
		return err
	}
	err = c.Wdog.Validate()
	if err != nil {
		return err
	}
	err = c.DemoRepo.Validate()
	if err != nil {
		return err
	}
	if c.Home == "" {
		return errors.New("'home' not set")
	}
	if c.RepoSyncDelay == 0 {
		fmt.Printf("'repo-sync-delay' not set, pulls will be unlimited\n")
		if !ModeDevel {
			return errors.New("'repo-sync-delay' not set")
		}
	}
	repoSyncDelay = time.Duration(c.RepoSyncDelay) * time.Second
	sysctl.AddTimeSysctl("repo_sync_delay", &repoSyncDelay)
	if c.RepoSyncPeriod == 0 {
		fmt.Printf("'repo-sync-period' not set, using default 30min\n")
		c.RepoSyncPeriod = 30
	}
	repoSyncPeriod = time.Duration(c.RepoSyncPeriod) * time.Minute
	sysctl.AddTimeSysctl("repo_sync_period", &repoSyncPeriod)
	if c.RunRate == 0 {
		fmt.Printf("'tryrun-rate' not set, using default 1/s\n")
		c.RunRate = 1
	}

	sysctl.AddIntSysctl("fn_tryrun_rate", &c.RunRate)
	return nil
}

var conf YAMLConf
