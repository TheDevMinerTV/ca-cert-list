FROM golang:1.21 AS builder
WORKDIR /build

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /build/app


FROM ghcr.io/thedevminertv/gostatic:1.2.5

COPY --from=builder /build/app /usr/local/bin/build-html
COPY --chown=app:app ./entrypoint.sh /
RUN chmod +x /entrypoint.sh /usr/local/bin/build-html
COPY ./public /static
