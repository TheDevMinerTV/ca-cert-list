FROM ghcr.io/thedevminertv/gostatic:1.2.5
CMD ["-compress-level", "2"]

RUN apk add --no-cache openssl coreutils

COPY --chown=app:app ./entrypoint.sh /entrypoint.sh
COPY --chown=app:app ./generate.sh /generate.sh
COPY ./public /static
