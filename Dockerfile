FROM alpine
COPY script/ca-certificates.crt /etc/ssl/certs/
COPY dist/traefik /
EXPOSE 80

RUN ln -sf /proc/1/fd/1 /tmp/logpipe

ENTRYPOINT ["/traefik"]
