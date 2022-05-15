FROM golang:1.18-alpine

LABEL Name="UPnP MediaServer cooperate with EPGStation"
LABEL Version="0.0.1"

RUN apk add --update --no-cache git

ENV ROOT=/go/src/app
WORKDIR ${ROOT}

COPY go.mod ./
RUN go mod download github.com/google/uuid@v1.3.0
RUN go mod tidy

CMD ["go", "run", "main.go"]
