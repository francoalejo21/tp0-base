FROM python:3.9.7-slim
COPY server /
RUN python -m unittest tests/test_common.py
ENTRYPOINT ["/bin/sh"]

FROM golang:1.17 AS builder
LABEL intermediateStageToBeDeleted=true

RUN mkdir -p /build
WORKDIR /build/
COPY . .
# CGO_ENABLED must be disabled to run go binary in Alpine
RUN CGO_ENABLED=0 GOOS=linux go build -mod vendor -o bin/server github.com/7574-sistemas-distribuidos/docker-compose-init/server

FROM busybox:latest
COPY --from=builder /build/bin/server /server
COPY ./server/config.ini /config.ini
ENTRYPOINT ["/bin/sh"]
