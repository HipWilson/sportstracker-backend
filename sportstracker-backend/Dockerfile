FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod ./
RUN go mod download || true

COPY . .
RUN go mod tidy && go build -o sportstracker ./cmd/server

FROM alpine:latest
WORKDIR /app

COPY --from=builder /app/sportstracker .
COPY --from=builder /app/docs ./docs

RUN mkdir -p uploads

EXPOSE 8080
CMD ["./sportstracker"]