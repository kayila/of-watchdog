package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/openfaas-incubator/of-watchdog/config"
	"github.com/openfaas-incubator/of-watchdog/functions"
)

func main() {
	watchdogConfig, configErr := config.New(os.Environ())
	if configErr != nil {
		fmt.Fprintf(os.Stderr, configErr.Error())
		os.Exit(-1)
	}

	if len(watchdogConfig.FunctionProcess) == 0 {
		fmt.Fprintf(os.Stderr, "Provide a \"function_process\" or \"fprocess\" environmental variable for your function.\n")
		os.Exit(-1)
	}

	s := &http.Server{
		Addr:           fmt.Sprintf(":%d", watchdogConfig.TCPPort),
		ReadTimeout:    watchdogConfig.HTTPReadTimeout,
		WriteTimeout:   watchdogConfig.HTTPWriteTimeout,
		MaxHeaderBytes: 1 << 20, // Max header of 1MB
	}

	var requestHandler http.HandlerFunc

	switch watchdogConfig.OperationalMode {
	case config.ModeStreaming:
		log.Println("OperationalMode: Streaming")
		requestHandler = makeForkRequestHandler(watchdogConfig)
		break
	case config.ModeSerializing:
		log.Println("OperationalMode: Serializing")
		requestHandler = makeSerializingForkRequestHandler(watchdogConfig)
		break
	case config.ModeAfterBurn:
		log.Println("OperationalMode: AfterBurn")
		requestHandler = makeAfterBurnRequestHandler(watchdogConfig)
		break
	}

	if err := lock(); err != nil {
		log.Panic(err.Error())
	}

	http.HandleFunc("/", requestHandler)
	log.Fatal(s.ListenAndServe())
}

func lock() error {
	lockFile := filepath.Join(os.TempDir(), ".lock")
	log.Printf("Writing lock file at: %s", lockFile)
	return ioutil.WriteFile(lockFile, nil, 0600)

}

func makeAfterBurnRequestHandler(watchdogConfig config.WatchdogConfig) func(http.ResponseWriter, *http.Request) {

	commandName, arguments := watchdogConfig.Process()
	functionInvoker := functions.AfterBurnFunctionRunner{
		Process:     commandName,
		ProcessArgs: arguments,
	}

	fmt.Printf("Forking - %s %s\n", commandName, arguments)
	functionInvoker.Start()

	return func(w http.ResponseWriter, r *http.Request) {

		req := functions.FunctionRequest{
			Process:      commandName,
			ProcessArgs:  arguments,
			InputReader:  r.Body,
			OutputWriter: w,
		}

		functionInvoker.Mutex.Lock()

		err := functionInvoker.Run(req, r.ContentLength, r, w)

		if err != nil {
			w.WriteHeader(500)
			w.Write([]byte(err.Error()))
		}

		functionInvoker.Mutex.Unlock()
	}
}

func makeSerializingForkRequestHandler(watchdogConfig config.WatchdogConfig) func(http.ResponseWriter, *http.Request) {
	functionInvoker := functions.SerializingForkFunctionRunner{
		HardTimeout: watchdogConfig.HardTimeout,
	}

	return func(w http.ResponseWriter, r *http.Request) {

		var environment []string

		if watchdogConfig.InjectCGIHeaders {
			environment = getEnvironment(r)
		}

		commandName, arguments := watchdogConfig.Process()
		req := functions.FunctionRequest{
			Process:       commandName,
			ProcessArgs:   arguments,
			InputReader:   r.Body,
			ContentLength: &r.ContentLength,
			OutputWriter:  w,
			Environment:   environment,
		}

		w.Header().Set("Content-Type", watchdogConfig.ContentType)
		err := functionInvoker.Run(req, w)
		if err != nil {
			log.Println(err)
		}
	}
}

func makeForkRequestHandler(watchdogConfig config.WatchdogConfig) func(http.ResponseWriter, *http.Request) {
	functionInvoker := functions.ForkFunctionRunner{
		HardTimeout: watchdogConfig.HardTimeout,
	}

	return func(w http.ResponseWriter, r *http.Request) {

		var environment []string

		if watchdogConfig.InjectCGIHeaders {
			environment = getEnvironment(r)
		}

		commandName, arguments := watchdogConfig.Process()
		req := functions.FunctionRequest{
			Process:      commandName,
			ProcessArgs:  arguments,
			InputReader:  r.Body,
			OutputWriter: w,
			Environment:  environment,
		}

		w.Header().Set("Content-Type", watchdogConfig.ContentType)
		err := functionInvoker.Run(req)
		if err != nil {
			log.Println(err.Error())

			// Probably cannot write to client if we already have written a header
			// w.WriteHeader(500)
			// w.Write([]byte(err.Error()))
		}
	}
}

func getEnvironment(r *http.Request) []string {
	var envs []string

	envs = os.Environ()
	for k, v := range r.Header {
		kv := fmt.Sprintf("Http_%s=%s", strings.Replace(k, "-", "_", -1), v[0])
		envs = append(envs, kv)
	}
	envs = append(envs, fmt.Sprintf("Http_Method=%s", r.Method))

	if len(r.URL.RawQuery) > 0 {
		envs = append(envs, fmt.Sprintf("Http_Query=%s", r.URL.RawQuery))
	}

	if len(r.URL.Path) > 0 {
		envs = append(envs, fmt.Sprintf("Http_Path=%s", r.URL.Path))
	}

	return envs
}
