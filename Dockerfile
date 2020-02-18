FROM alpine:3.10
RUN apk add --update --no-cache ca-certificates git libc6-compat

WORKDIR /
COPY ./bin/manager /manager

ENTRYPOINT ["/manager"]
