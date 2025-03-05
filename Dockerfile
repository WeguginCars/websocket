FROM golang:1.23.2 AS builder

WORKDIR /app

COPY . .

RUN go mod download

COPY .env .

RUN CGO_ENABLED=0 GOOS=linux go build -C ./cmd -a -installsuffix cgo -o ./../myapp .

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/myapp .
COPY --from=builder /app/casbin/model.conf ./casbin/
COPY --from=builder /app/casbin/policy.csv ./casbin/
COPY --from=builder /app/app.log ./
COPY --from=builder /app/.env .

EXPOSE 4444

CMD ["./myapp"]