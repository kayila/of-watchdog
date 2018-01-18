FROM golang:1.9.1
RUN mkdir -p /go/src/github.com/openfaas-incubator/of-watchdog
WORKDIR /go/src/github.com/openfaas-incubator/of-watchdog

ENV DEP_VERSION v0.3.2

RUN wget https://github.com/golang/dep/releases/download/${DEP_VERSION}/dep-linux-amd64 -O /bin/dep && chmod +x /bin/dep

COPY Gopkg.toml .
COPY main.go    .
COPY config     config
COPY functions  functions
COPY utilities  utilities

# Download dependencies
RUN dep ensure

# Run a gofmt and exclude all vendored code.
RUN test -z "$(gofmt -l $(find . -type f -name '*.go' -not -path "./vendor/*"))"

RUN go test -v ./...

# Stripping via -ldflags "-s -w" 
RUN CGO_ENABLED=0 GOOS=linux go build -a -ldflags "-s -w" -installsuffix cgo -o of-watchdog . \
    && CGO_ENABLED=0 GOOS=darwin go build -a -ldflags "-s -w" -installsuffix cgo -o of-watchdog-darwin . \
    && GOARM=6 GOARCH=arm CGO_ENABLED=0 GOOS=linux go build -a -ldflags "-s -w" -installsuffix cgo -o of-watchdog-armhf . \
    && GOARCH=arm64 CGO_ENABLED=0 GOOS=linux go build -a -ldflags "-s -w" -installsuffix cgo -o of-watchdog-arm64 . \
    && GOOS=windows CGO_ENABLED=0 go build -a -ldflags "-s -w" -installsuffix cgo -o of-watchdog.exe .
