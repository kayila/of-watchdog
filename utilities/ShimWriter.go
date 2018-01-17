package utilities

import (
	"net/http"
)

// shimWriter is a struct implementing the http.ResponseWriter interface  This
// is designed to sit in between a 'nested' http.ResponseWriter and any calls
// to it, intercepting the calls and checking for an responsecode/header
// control information being passed through a side pipe. This is done so that
// programs can connect their stdout directly up to this writer, while still
// being able to send custom HTTP response codes and headers.
//
// This cannot be created directly, please use NewShim.
type shimWriter struct {
	Nested      http.ResponseWriter
	controlChan chan HTTPControl
	dirty       bool
}

// Creates and returns a new shimWriter object containing the passed
// http.ResponseWriter and connecting to the control pipe given.
func NewShim(nest http.ResponseWriter, control string) http.ResponseWriter {
	ch := make(chan HTTPControl)
	go ReadFromPipe(control, ch)
	return shimWriter{nest, ch, false}
}

func (s shimWriter) Header() http.Header {
	return s.Nested.Header()
}

func (s shimWriter) WriteHeader(rc int) {
	s.dirty = true
	s.Nested.WriteHeader(rc)
}

func (s shimWriter) Write(bytes []byte) (int, error) {

	if s.dirty == false {
		select {
		case controlMessage := <-s.controlChan:
			// Write headers from the control message
			headers := s.Header()
			for key, value := range controlMessage.Headers {
				headers.Set(key, value)
			}
			if controlMessage.Status != 0 {
				s.WriteHeader(controlMessage.Status)
			} else {
				s.WriteHeader(200)
			}
		default:
			// If nothing was sent yet, explicitly send a 200 header
			s.WriteHeader(200)
		}
		s.dirty = true
	}
	return s.Nested.Write(bytes)
}
