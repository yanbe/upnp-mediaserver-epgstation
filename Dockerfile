FROM golang:1.18-alpine

LABEL Name="UPNP ContentDirectory Server"
LABEL Version="0.0.1"

ENV ROOT=/go/src/app
WORKDIR ${ROOT}

RUN apk add --update --no-cache git

COPY go.mod ./
RUN go mod download

CMD ["go", "run", "main.go"]
