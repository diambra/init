FROM golang:1.20-alpine3.18 AS builder
WORKDIR /app
COPY go.mod go.sum ./
COPY . .

ENV CGO_ENABLED=0
RUN go test ./...
RUN go build

FROM alpine:3.18
RUN apk add --no-cache git git-lfs
COPY --from=builder /app/init /init
ENTRYPOINT [ "/init" ]
