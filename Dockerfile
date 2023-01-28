FROM golang:1.19-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
COPY . .

ENV CGO_ENABLED=0
RUN go test ./...
RUN go build

FROM scratch
COPY --from=builder /app/init /init
ENTRYPOINT [ "/init" ]
