FROM golang:alpine as builder

ENV GO111MODULE=on

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN go build github.com/PaBah/gofermart/cmd/gophermart

FROM alpine:latest

WORKDIR /root/

COPY --from=builder /app/gophermart .

EXPOSE 8081

ENTRYPOINT ["/root/gophermart", "-d", "host=postgres_db user=postgres password=postgres dbname=postgres", "-r", "http://accrual_service:8080"]