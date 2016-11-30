package orm

type NotFoundError struct {
	msg string
}

func (p NotFoundError)Error() string {
	return p.msg
}

func NewNotFoundError() error {
	return NotFoundError{msg:"data is not found"}
}