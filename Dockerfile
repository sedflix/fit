FROM golang:latest as builder

LABEL maintainer="Siddharth <sedflix@gmail.com>"

WORKDIR /app
COPY src/ .
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o fit .


FROM alpine:latest

RUN apk --no-cache add ca-certificates tzdata
ENV TZ=Asia/Kolkata
ENV GIN_MODE=release

WORKDIR /root/
COPY src/web web

COPY --from=builder /app/fit .
EXPOSE 8080

CMD ["./fit"]