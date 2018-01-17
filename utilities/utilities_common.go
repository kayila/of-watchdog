package utilities

import (
	"github.com/satori/go.uuid"
)

type HTTPControl struct {
	Status  int               `json:status`
	Headers map[string]string `json:headers`
}

func pipeName() string {
	u := uuid.NewV4()
	return u.String()
}

func CreatePipe() string {
	path := pipePath()
	CreatePipeIfNotExists(path)
	return path
}
