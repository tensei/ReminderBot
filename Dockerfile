FROM golang:latest

WORKDIR /go/src/reminderbot
COPY . .

RUN go get -v ./...
RUN go install -v ./...

CMD [ "reminderbot" ]