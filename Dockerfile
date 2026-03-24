# syntax=docker/dockerfile:1

FROM golang:1.25-alpine AS build

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY cmd ./cmd
COPY internal ./internal
COPY web ./web

RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /out/rbooth ./cmd/server

FROM alpine:3.21

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata \
  && mkdir -p /app/data /mnt/storage/media/rbooth

COPY --from=build /out/rbooth /app/rbooth
COPY web /app/web

ENV PORT=8325
ENV DATA_DIR=/app/data
ENV MEDIA_DIR=/mnt/storage/media/rbooth

EXPOSE 8325

CMD ["/app/rbooth"]
