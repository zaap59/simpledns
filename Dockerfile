# Stage 1: Builder
FROM golang:1.24-alpine AS build
ARG VERSION
ENV VERSION=${VERSION:-edge}

WORKDIR /app

COPY . .

RUN go mod download && \
    go build -o simpledns -ldflags "-X main.version=${VERSION}" .

# Stage 2: Runtime
FROM scratch

COPY --from=build /app/simpledns /simpledns

EXPOSE 53 53/udp

WORKDIR /etc/simpledns/zones
WORKDIR /etc/simpledns

ENTRYPOINT ["/simpledns"]
