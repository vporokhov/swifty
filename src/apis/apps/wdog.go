package swyapi

type SwdFunctionDesc struct {
	PodToken	string		`json:"podtoken"`
	Timeout		uint64		`json:"timeout"`
	Build		bool		`json:"build"`
}

type SwdFunctionRun struct {
	PodToken	string			`json:"podtoken"`
	Args		map[string]string	`json:"args"`
}

type SwdFunctionRunResult struct {
	Return		string		`json:"return"`
	Code		int		`json:"code"`
	Stdout		string		`json:"stdout"`
	Stderr		string		`json:"stderr"`
	Time		uint		`json:"time"` /* usec */
	CTime		uint		`json:"ctime"` /* usec too */
}
