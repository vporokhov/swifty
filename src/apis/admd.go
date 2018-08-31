package swyapi

type UserInfo struct {
	ID	string		`json:"id"`
	UId	string		`json:"uid"`
	Name	string		`json:"name,omitempty"`
	Enabled	bool		`json:"enabled,omitempty"`
	Created	string		`json:"created,omitempty"`
	Roles	[]string	`json:"roles,omitempty"`
}

type ModUser struct {
	Enabled	*bool		`json:"enabled,omitempty"`
}

type AddUser struct {
	UId	string		`json:"uid"`
	Pass	string		`json:"pass"`
	Name	string		`json:"name"`
	PlanId	string		`json:"planid"`
}

type PlanLimits struct {
	Id	string			`json:"id,omitempty"`
	Name	string			`json:"name"`
	Fn	*FunctionLimits		`json:"function,omitempty"`
}