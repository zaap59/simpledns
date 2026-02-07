# Stage 1: Builder
FROM golang:1.24-alpine AS build
ARG VERSION
ENV VERSION=${VERSION:-edge}

WORKDIR /app

RUN apk -U add --no-cache build-base

COPY . .

RUN go mod download && \
    CGO_ENABLED=1 go build -o simpledns -ldflags "-X main.version=${VERSION}" .

# Stage 2: Runtime
FROM gcr.io/distroless/base-debian13

COPY --from=build /app/simpledns /simpledns

EXPOSE 53 53/udp

WORKDIR /etc/simpledns/zones
WORKDIR /etc/simpledns

COPY --from=build /app/config.yaml /etc/simpledns/config.yaml

ENTRYPOINT ["/simpledns", "-config-file", "/etc/simpledns/config.yaml"]
