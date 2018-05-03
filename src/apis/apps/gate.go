package swyapi

type GateErr struct {
	Code		uint			`json:"code"`
	Message		string			`json:"message"`
}

type FunctionList struct {
	Project		string			`json:"project"`
}

type FunctionListInfo struct {
	Project		string			`json:"project"`
	Periods		int			`json:"periods"`
}

type ProjectList struct {
}

type ProjectDel struct {
	Project		string			`json:"project"`
}

type TenantStatsReq struct {
	Periods		int			`json:"periods"`
}

type FunctionStatsReq struct {
	Project		string			`json:"project"`
	FuncName	string			`json:"name"`
	Periods		int			`json:"periods"`
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

type TenantStats struct {
	Called		uint64			`json:"called"`
	GBS		float64			`json:"gbs"`
	BytesOut	uint64			`json:"bytesout"`
	Till		string			`json:"till,omitempty"`
	From		string			`json:"from,omitempty"`
}

type TenantStatsResp struct {
	Stats		[]TenantStats		`json:"stats"`
}

type FunctionWait struct {
	Project		string			`json:"project"`
	FuncName	string			`json:"name"`
	Timeout		uint32			`json:"timeout"` /* msec */
	Version		string			`json:"version,omitempty"`
}

type FunctionInfo struct {
	Name		string			`json:"name,omitempty"`
	Mware		[]string		`json:"mware"`
	S3Buckets	[]string		`json:"s3buckets"`
	State		string			`json:"state"`
	Version		string			`json:"version"`
	RdyVersions	[]string		`json:"rversions"`
	Code		FunctionCode		`json:"code"`
	URL		string			`json:"url"`
	Stats		[]FunctionStats		`json:"stats"`
	Size		FunctionSize		`json:"size"`
	AuthCtx		string			`json:"authctx,omitempty"`
	UserData	string			`json:"userdata"`
	Id		string			`json:"id"`
}

type RunCmd struct {
	Exe		string			`json:"exe"`
	Args		[]string		`json:"args,omitempty"`
}

type FunctionCode struct {
	Lang		string			`json:"lang"`
	Env		[]string		`json:"env,omitempty"`
}

type FunctionSources struct {
	Type		string			`json:"type"`
	Repo		string			`json:"repo,omitempty"`
	Code		string			`json:"code,omitempty"`
}

type FunctionSize struct {
	Memory		uint64			`json:"memory"`
	Timeout		uint64			`json:"timeout"` /* msec */
	Rate		uint			`json:"rate,omitempty"`
	Burst		uint			`json:"burst,omitempty"`
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

type FunctionEvent struct {
	Id		string			`json:"id,omitempty"`
	Name		string			`json:"name"`
	Source		string			`json:"source"`
	Cron		*FunctionEventCron	`json:"cron,omitempty"`
	S3		*FunctionEventS3	`json:"s3,omitempty"`
}

type MwareAdd struct {
	Project		string			`json:"project"`
	ID		string			`json:"id"`
	Type		string			`json:"type"`
	UserData	string			`json:"userdata,omitempty"`
}

type MwareRemove struct {
	Project		string			`json:"project"`
	ID		string			`json:"id"`
}

type MwareID struct {
	Project		string			`json:"project"`
	ID		string			`json:"id"`
}

type MwareInfo struct {
	ID		string			`json:"id,omitempty"`
	Type		string			`json:"type"`
	UserData	string			`json:"userdata,omitempty"`
	DU		*uint64			`json:"disk_usage,omitempty"` /* in ... KB */
}

func (i *MwareInfo)SetDU(bytes uint64) {
	kb := bytes >> 10
	i.DU = &kb
}

type MwareS3Access struct {
	Project		string			`json:"project"`
	Bucket		string			`json:"bucket"`
	Lifetime	uint32			`json:"lifetime"` /* seconds */
	Access		[]string		`json:"access"`
}

type MwareS3Creds struct {
	Endpoint	string			`json:"endpoint"`
	Key		string			`json:"key"`
	Secret		string			`json:"secret"`
	Expires		uint32			`json:"expires"` /* in seconds */
}

type MwareItem struct {
	ID		string			`json:"id"`
	Type		string			`json:"type"`
}

type MwareList struct {
	Project		string			`json:"project"`
	Type		string			`json:"type"`
}

type FunctionAdd struct {
	Project		string			`json:"project"`
	FuncName	string			`json:"name"`
	Sources		FunctionSources		`json:"sources"`
	Code		FunctionCode		`json:"code"`
	Size		FunctionSize		`json:"size"`
	Mware		[]string		`json:"mware,omitempty"`
	S3Buckets	[]string		`json:"s3buckets,omitempty"`
	UserData	string			`json:"userdata,omitempty"`
	AuthCtx		string			`json:"authctx,omitempty"`
}

type FunctionState struct {
	Project		string			`json:"project"`
	FuncName	string			`json:"name"`
	State		string			`json:"state"`
}

type FunctionUpdate struct {
	Project		string			`json:"project"`
	FuncName	string			`json:"name"`
	Code		string			`json:"code"`
	Size		*FunctionSize		`json:"size,omitempty"`
	Mware		*[]string		`json:"mware,omitempty"`
	S3Buckets	*[]string		`json:"s3buckets,omitempty"`
	AuthCtx		*string			`json:"authctx,omitempty"`
	UserData	string			`json:"userdata"`
}

type FunctionRemove struct {
	Project		string			`json:"project"`
	FuncName	string			`json:"name"`
}

type FunctionRun struct {
	Project		string			`json:"project"`
	FuncName	string			`json:"name"`
	Args		map[string]string	`json:"args,omitempty"`
}

type FunctionRunResult struct {
	Code		int			`json:"code"`
	Return		string			`json:"return"`
	Stdout		string			`json:"stdout"`
	Stderr		string			`json:"stderr"`
}

type FunctionID struct {
	Project		string			`json:"project"`
	FuncName	string			`json:"name"`
}

type FunctionInfoReq struct {
	Project		string			`json:"project"`
	FuncName	string			`json:"name"`
	Periods		int			`json:"periods,omitempty"`
}

type FunctionXID struct {
	Project		string			`json:"project"`
	FuncName	string			`json:"name"`
	Version		string			`json:"version"`
}

type FunctionItem struct {
	FuncName	string			`json:"name"`
	State		string			`json:"state"`
	Timeout		uint64			`json:"timeout"`
	LastCall	string			`json:"lastcall,omitempty"`
}

type ProjectItem struct {
	Project		string			`json:"project"`
}

type FunctionLogEntry struct {
	Event		string			`json:"event"`
	Ts		string			`json:"ts"`
	Text		string			`json:"text"`
}

type DeployId struct {
	Project		string			`json:"project"`
	Name		string			`json:"name"`
}

type DeployItem struct {
	Function	*FunctionAdd		`json:"function,omitempty"`
	Mware		*MwareAdd		`json:"mware,omitempty"`
}

type DeployStart struct {
	Project		string			`json:"project"`
	Name		string			`json:"name"`
	Items		[]DeployItem		`json:"items"`
}

type DeployItemInfo struct {
	Type		string			`json:"type"`
	Name		string			`json:"name"`
	State		string			`json:"state"`
}

type DeployInfo struct {
	State		string			`json:"state"`
	Items		[]*DeployItemInfo	`json:"items"`
}
