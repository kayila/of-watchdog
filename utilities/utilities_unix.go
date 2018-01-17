// +build !windows

package utilities

import (
	"encoding/json"
	"io/ioutil"
	"os"
	"path/filepath"
	"syscall"
)

func pipePath() string {
	return filepath.Join(os.TempDir(), pipeName())
}

func createPipe(path string) {
	oldmask := syscall.Umask(000)
	syscall.Mkfifo(path, 0666)
	syscall.Umask(oldmask)
}
func CreatePipeIfNotExists(path string) {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		createPipe(path)
	}
}

func ReadFromPipe(path string, c chan<- HTTPControl) {
	pipe, err := os.Open(path)
	if err == nil {
		s, err := ioutil.ReadAll(pipe)
		if err == nil {
			control := HTTPControl{}
			err = json.Unmarshal(s, &control)
			if err == nil {
				c <- control
			}
		}
	}
}
