FROM golang:1.27-alpine AS builder

WORKDIR /build
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -ldflags="-s -w" -o kaze ./cmd/kaze

FROM alpine:3.24

RUN apk add --no-cache tzdata

WORKDIR /app
COPY --from=builder /build/kaze .
COPY --from=builder /build/web ./web

EXPOSE 4533

ENTRYPOINT ["./kaze", "-config", "/app/config/config.yml"]
