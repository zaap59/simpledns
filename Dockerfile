# Stage 1: Builder
FROM golang:1.24-alpine AS build
ARG VERSION
ENV VERSION=${VERSION:-edge}

WORKDIR /go/src/app

COPY . .

RUN go mod download && \
    go build -o simpledns -ldflags "-X main.version=${VERSION}" .

# Stage 2: Runtime
FROM scratch

COPY --from=build /go/src/app/simpledns /simpledns

EXPOSE 53 53/udp

WORKDIR /etc/simpledns/zones
WORKDIR /etc/simpledns

COPY --from=build /go/src/app/config.yaml /etc/simpledns/config.yaml

ENTRYPOINT ["/simpledns", "-config-file", "/etc/simpledns/config.yaml"]
