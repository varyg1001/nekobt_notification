FROM golang:1.27-alpine AS builder

WORKDIR /app

COPY go.mod .
COPY go.sum .

RUN go mod download

COPY . .

RUN go build -o bin .

ENTRYPOINT ["/app/bin"]
