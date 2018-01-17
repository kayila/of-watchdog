// +build windows

package utilities

import (
	"bufio"
	"encoding/json"
	"fmt"
	"gopkg.in/natefinch/npipe.v2"
	"io/ioutil"
)

func pipePath() string {
	// Using Sprintf here because go1.9.1/1.9.2 has an error with
	// `path.filepath.Join` which strips the leading \\. from the file path
	return fmt.Sprintf("\\\\.\\pipe\\%s", pipeName())
}

func CreatePipeIfNotExists(path string) {
	// We are using the `npipe.Listen` function, so no need to pre-create the
	// the pipe. Do nothing here.
}

func ReadFromPipe(path string, c chan<- HTTPControl) {
	ln, err := npipe.Listen(path)
	if err != nil {
		panic(fmt.Sprintf("Couldn't create pipe: %s", err))
	}

	// Accept the first connection to the pipe
	conn, err := ln.Accept()
	if err != nil {
		panic(fmt.Sprintf(
			"Error occured while trying to accept pipe connection: %s",
			err,
		))
	}

	// Process it
	reader := bufio.NewReader(conn)
	s, err := ioutil.ReadAll(reader)
	if err != nil {
		panic(fmt.Sprintf("Error converting connection to buffer: %s", err))
	}
	control := HTTPControl{}
	err = json.Unmarshal(s, &control)
	if err != nil {
		panic(fmt.Sprintf("Error decoding json: %s", err))
	}
	c <- control
}
