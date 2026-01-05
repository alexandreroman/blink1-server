FROM golang:1.25-bookworm AS builder
WORKDIR /app
COPY go.mod ./
RUN go mod download
COPY main.go .
RUN go build -o blink1-server .

FROM ubuntu:jammy
RUN apt-get update && apt-get install -y ca-certificates && update-ca-certificates && apt-get clean
RUN addgroup --gid 1000 user && adduser --uid 1000 --gid 1000 --no-create-home user

WORKDIR /app
COPY --from=builder /app/blink1-server .
COPY vendor vendor
RUN chown -R user:user /app

USER user
EXPOSE 8080
CMD ["./blink1-server"]
