/*
 * © 2018 SwiftyCloud OÜ. All rights reserved.
 * Info: info@swifty.cloud
 */

package swyapi

import (
	"swifty/common/xrest"
)

const (
	GateGenErr	uint = xrest.GenErr
	GateBadRequest	uint = xrest.BadRequest
	GateBadResp	uint = xrest.BadResp

	GateDbError	uint = 4	// Error requesting database (except NotFound)
	GateDuplicate	uint = 5	// ID duplication
	GateNotFound	uint = 6	// No resource found
	GateFsError	uint = 7	// Error accessing file(s)
	GateNotAvail	uint = 8	// Operation not available on selected object
	GateLimitHit	uint = 9	// Resource limitation
)

type ProjectList struct {
}

type ProjectDel struct {
	Project		string			`json:"project"`
}

type FunctionStats struct {
	Called		uint64			`json:"called"`
	Timeouts	uint64			`json:"timeouts"`
	Errors		uint64			`json:"errors"`
	LastCall	string			`json:"lastcall,omitempty"`
	Time		uint64			`json:"time"`
	GBS		float64			`json:"gbs"`
	BytesOut	uint64			`json:"bytesout"`
	Till		string			`json:"till,omitempty"`
	From		string			`json:"from,omitempty"`
}

type FunctionStatsResp struct {
	Stats		[]FunctionStats		`json:"stats"`
}

type TenantStatsFn struct {
	Called		uint64			`json:"called"`
	GBS		float64			`json:"gbs"`
	BytesOut	uint64			`json:"bytesout"`
	Till		string			`json:"till,omitempty"`
	From		string			`json:"from,omitempty"`
}

type TenantStatsMware struct {
	Count		int			`json:"count"`
	DU		*uint64			`json:"disk_usage,omitempty"` /* in ... KB */
}

type S3NsStats struct {
	CntObjects		int64		`json:"cnt-objects"`
	CntBytes		int64		`json:"cnt-bytes"`
	OutBytes		int64		`json:"out-bytes"`
	OutBytesWeb		int64		`json:"out-bytes-web"`
}

type TenantStatsResp struct {
	Stats		[]TenantStatsFn			`json:"stats,omitempty"`
	Mware		map[string]*TenantStatsMware	`json:"mware,omitempty"`
	S3		*S3NsStats			`json:"s3,omitempty"`
}

type FunctionInfo struct {
	Name		string			`json:"name,omitempty"`
	Project		string			`json:"project,omitempty"`
	Labels		[]string		`json:"labels,omitempty"`
	State		string			`json:"state"`
	Version		string			`json:"version"`
	RdyVersions	[]string		`json:"rversions,omitempty"`
	Code		*FunctionCode		`json:"code,omitempty"`
	URL		string			`json:"url,omitempty"`
	Stats		[]FunctionStats		`json:"stats,omitempty"`
	Size		*FunctionSize		`json:"size,omitempty"`
	AuthCtx		string			`json:"authctx,omitempty"`
	UserData	string			`json:"userdata,omitempty"`
	Id		string			`json:"id"`
}

type FunctionMdat struct {
	Cookie		string			`json:"cookie"`
	PodToken	string			`json:"pod_token"`
	RL		[]uint			`json:"rl"`
	BR		[]uint			`json:"br"`
	IPs		[]string		`json:"ips,omitempty"`
	Hosts		[]string		`json:"hosts,omitempty"`
	Dep		string			`json:"depname,omitempty"`
}

//type RunCmd struct {
//	Exe		string			`json:"exe"`
//	Args		[]interface{}		`json:"args,omitempty"`
//}

type FunctionCode struct {
	Lang		string			`json:"lang"`
	Env		[]string		`json:"env,omitempty"`
}

type FunctionSources struct {
	Repo		string			`json:"repo,omitempty"`
	Code		string			`json:"code,omitempty"`
	URL		string			`json:"url,omitempty"`
	Sync		bool			`json:"sync"`
}

type FunctionSize struct {
	Memory		uint			`json:"memory"`
	Timeout		uint			`json:"timeout"` /* msec */
	Rate		uint			`json:"rate,omitempty"`
	Burst		uint			`json:"burst,omitempty"`
}

type FunctionWait struct {
	Timeout		uint			`json:"timeout"`
	Version		string			`json:"version,omitempty"`
}

type FunctionEventCron struct {
	Tab		string			`json:"tab"`
	Args		map[string]string	`json:"args"`
}

type FunctionEventS3 struct {
	Bucket		string			`json:"bucket"`
	Ops		string			`json:"ops,omitempty"`
	Pattern		string			`json:"pattern,omitempty"`
}

type FunctionEventWebsock struct {
	MwName		string			`json:"name"`
	MType		*int			`json:"mtype,omitempty"`
}

type FunctionEvent struct {
	Id		string			`json:"id,omitempty"`
	Name		string			`json:"name"`
	Source		string			`json:"source"`
	Cron		*FunctionEventCron	`json:"cron,omitempty"`
	S3		*FunctionEventS3	`json:"s3,omitempty"`
	URL		string			`json:"url,omitempty"`
	WS		*FunctionEventWebsock	`json:"websocket,omitempty" yaml:"websocket,omitempty"`
}

type MwareAdd struct {
	Name		string			`json:"name"`
	Project		string			`json:"project,omitempty"`
	Type		string			`json:"type"`
	UserData	string			`json:"userdata,omitempty"`
	AuthCtx		string			`json:"authctx,omitempty"`
}

type MwareTypeInfo struct {
	Envs		[]string		`json:"envs"`
}

type MwareInfo struct {
	Id		string			`json:"id"`
	Labels		[]string		`json:"labels,omitempty"`
	Name		string			`json:"name"`
	Project		string			`json:"project,omitempty"`
	Type		string			`json:"type"`
	UserData	string			`json:"userdata,omitempty"`
	DU		*uint64			`json:"disk_usage,omitempty"` /* in ... KB */
	URL		*string			`json:"url,omitempty"`
}

func (i *MwareInfo)SetDU(bytes uint64) {
	kb := bytes >> 10
	i.DU = &kb
}

type S3Access struct {
	Bucket		string			`json:"bucket"`
	Lifetime	uint32			`json:"lifetime"` /* seconds */
	Access		[]string		`json:"access"`
}

type S3Creds struct {
	Endpoint	string			`json:"endpoint"`
	Key		string			`json:"key"`
	Secret		string			`json:"secret"`
	Expires		uint32			`json:"expires"` /* in seconds */
	AccID		string			`json:"accid"`
}

type FunctionAdd struct {
	Name		string			`json:"name"`
	Project		string			`json:"project,omitempty"`
	Sources		*FunctionSources	`json:"sources,omitempty"`
	Code		FunctionCode		`json:"code"`
	Size		FunctionSize		`json:"size"`
	Mware		[]string		`json:"mware,omitempty"`
	S3Buckets	[]string		`json:"s3buckets,omitempty"`
	Accounts	[]string		`json:"accounts,omitempty"`
	UserData	string			`json:"userdata,omitempty"`
	AuthCtx		string			`json:"authctx,omitempty"`

	Events		[]FunctionEvent		`json:"-" yaml:"events"` /* Deploy only */
}

type FunctionUpdate struct {
	UserData	*string			`json:"userdata,omitempty"`
	State		string			`json:"state,omitempty"`
}

type ProjectItem struct {
	Project		string			`json:"project"`
}

type LogEntry struct {
	Event		string			`json:"event"`
	Ts		string			`json:"ts"`
	Text		string			`json:"text"`
}

type DeployInclude struct {
	DeploySource				`yaml:",inline"`
}

type DeployDescription struct {
	Include		[]*DeployInclude	`yaml:"include"`
	Functions	[]*FunctionAdd		`yaml:"functions"`
	Mwares		[]*MwareAdd		`yaml:"mwares"`
	Routers		[]*RouterAdd		`yaml:"routers"`

	Labels		[]string		`yaml:"labels,omitempty"` // Trusted repos only
}

type DeploySource struct {
	Descr		string			`json:"desc,omitempty" yaml:"desc,omitempty"`
	Repo		string			`json:"repo,omitempty" yaml:"repo,omitempty"`
	URL		string			`json:"url,omitempty" yaml:"url,omitempty"`
}

type DeployStart struct {
	Name		string			`json:"name"`
	Project		string			`json:"project,omitempty"`
	From		DeploySource		`json:"from"`
	Params		map[string]string	`json:"parameters"`
}

type DeployItemInfo struct {
	Type		string			`json:"type"`
	Name		string			`json:"name"`
	State		string			`json:"state,omitempty"`
	Id		string			`json:"id,omitempty"`
}

type DeployInfo struct {
	Id		string			`json:"id,omitempty"`
	Name		string			`json:"name"`
	Project		string			`json:"project"`
	Labels		[]string		`json:"labels,omitempty"`
	State		string			`json:"state"`
	Items		[]*DeployItemInfo	`json:"items,omitempty"`
}

type AuthAdd struct {
	Name		string			`json:"name"`
	Project		string			`json:"project"`
	Type		string			`json:"type"`
}

type RepoAdd struct {
	Type		string			`json:"type"`
	URL		string			`json:"url"`
	AccID		string			`json:"account_id,omitempty"`
	UserData	string			`json:"userdata,omitempty"`
	Pull		string			`json:"pulling,omitempty"`
}

type RepoUpdate struct {
	Pull		*string			`json:"pulling,omitempty"`
}

type RepoInfo struct {
	Id		string			`json:"id"`
	Type		string			`json:"type"`
	URL		string			`json:"url"`
	State		string			`json:"state"`
	Commit		string			`json:"commit"`
	UserData	string			`json:"userdata,omitempty"`
	AccID		string			`json:"account_id,omitempty"`
	Pull		string			`json:"pulling,omitempty"`
	Desc		bool			`json:"desc"`
	DU_Kb		uint64			`json:"disk_usage"`
}

func (ri *RepoInfo)SetDU(bytes uint64) {
	ri.DU_Kb = bytes>>10
}

type RepoEntry struct {
	Name		string			`json:"name" yaml:"name"`
	Path		string			`json:"path" yaml:"path"`
	Description	string			`json:"desc" yaml:"desc"`
	Lang		string			`json:"lang,omitempty" yaml:"lang,omitempty"`
}

type RepoDesc struct {
	Description	string			`json:"desc" yaml:"desc"`
	Entries		[]*RepoEntry		`json:"files" yaml:"files"`
}

type RepoFile struct {
	Label		string		`json:"label"`
	Path		string		`json:"path"`
	Type		string		`json:"type"`
	Lang		*string		`json:"lang,omitempty"`
	Children	*[]*RepoFile	`json:"children,omitempty"`
}

type RouterEntry struct {
	Method		string		`json:"method"`
	Path		string		`json:"path"`
	Call		string		`json:"call"`
	AuthCtx		string		`json:"authctx,omitempty"`
	Key		string		`json:"key,omitempty"`
}

type RouterAdd struct {
	Name		string		`json:"name"`
	Project		string		`json:"project"`
	Table		[]*RouterEntry	`json:"table"`
}

type RouterInfo struct {
	Id		string		`json:"id"`
	Name		string		`json:"name"`
	Project		string		`json:"project"`
	Labels		[]string	`json:"labels,omitempty"`
	TLen		int		`json:"table_len"`
	URL		string		`json:"url"`
}

type PkgAdd struct {
	Name		string		`json:"name"`
}

type PkgInfo struct {
	Id		string		`json:"id"`	// name
}

type PkgLangStat struct {
	DU_Kb		uint64			`json:"disk_usage"` /* in ... KB */
}

func (ps *PkgLangStat)SetDU(bytes uint64) {
	ps.DU_Kb = bytes>>10
}

type PkgStat struct {
	DU_Kb		uint64			`json:"disk_usage"` /* in ... KB */
	Lang		map[string]*PkgLangStat	`json:"lang"`
}

func (ps *PkgStat)SetDU(bytes uint64) {
	ps.DU_Kb = bytes>>10
}

/*
 * This type is not seen by wdog itself, instead, it's described
 * by each wdog runner by smth like "Request"
 */
type FunctionRun struct {
	Event		string			`json:"event"`
	Args		map[string]string	`json:"args"`
	ContentType	string			`json:"content,omitempty"`
	Body		string			`json:"body,omitempty"`
	Claims		map[string]interface{}	`json:"claims,omitempty"` // JWT
	Method		*string			`json:"method,omitempty"`
	Path		*string			`json:"path,omitempty"`
	Key		string			`json:"key,omitempty"`
	Src		*FunctionSources	`json:"src,omitempty"`
}
