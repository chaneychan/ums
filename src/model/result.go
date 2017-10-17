package model

type Result struct {
	IsOk     bool
	ErrorMsg string
	User     User
}

func (r *Result) ToString() (str string) {
	str = "errorMsg:" + r.ErrorMsg
	return str
}
