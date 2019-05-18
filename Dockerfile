FROM golang:1-alpine

RUN mkdir -p /etc

WORKDIR /go/src/app

RUN apk add --update --no-cache git

COPY . .

RUN go get -d -v ./...
RUN go install -v ./...

CMD app -port  2525 /etc/smtptoxmpp.conf
