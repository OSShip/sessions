FROM golang:1.26-alpine AS builder
WORKDIR /app
COPY utils/ /app/utils/
COPY services/sessions/ /app/services/sessions/
WORKDIR /app/services/sessions
RUN go mod download && CGO_ENABLED=0 go build -o /sessions .

FROM alpine:3.20
RUN apk add --no-cache ca-certificates wget
COPY --from=builder /sessions /sessions
COPY jitsi.pk /jitsi.pk
EXPOSE 8084
CMD ["/sessions"]
