FROM golang:1.24.1 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN --mount=type=cache,target="./root/.cache/go-build" go build -o /app/subpub /app/cmd/

FROM ubuntu:24.04

WORKDIR /app

COPY --from=builder /app/subpub .
COPY --from=builder /app/configs /app/configs
ENTRYPOINT [ "./subpub" ]