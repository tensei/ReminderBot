# build stage
FROM golang:alpine AS build-env
RUN apk --no-cache add build-base git bzr mercurial gcc
ADD . /src
RUN cd /src && go mod download && go build -o reminderbot

# final stage
FROM alpine
WORKDIR /app
RUN apk add --no-cache tzdata
COPY --from=build-env /src/reminderbot /app/
COPY --from=build-env /src/config.json /app/
ENTRYPOINT ./reminderbot