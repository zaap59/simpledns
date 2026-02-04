# Stage 1: Builder
FROM golang:1.24-alpine AS build

WORKDIR /app

COPY . .

RUN go mod download && \
    go build -o simpledns .

# Stage 2: Runtime
FROM scratch

COPY --from=build /app/simpledns /simpledns

EXPOSE 53 53/udp

WORKDIR /etc/simpledns/zones
WORKDIR /etc/simpledns

ENTRYPOINT ["/simpledns"]
