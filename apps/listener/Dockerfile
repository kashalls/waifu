# Build from the workspace root:
#   docker build -f apps/listener/Dockerfile -t waifu-listener .
FROM golang:1.24-alpine AS builder
WORKDIR /workspace
# Each Dockerfile builds one app; disable workspace mode so Go does not
# try to load the other app's go.mod. The replace directive in go.mod
# handles the local pkg/event dependency.
ENV GOWORK=off

# Cache dependency downloads before copying source.
COPY pkg/event/go.mod ./pkg/event/
COPY apps/listener/go.mod apps/listener/go.sum ./apps/listener/
RUN cd apps/listener && go mod download

# Copy source and build.
COPY pkg/event/ ./pkg/event/
COPY apps/listener/ ./apps/listener/
RUN cd apps/listener && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /listener .

FROM alpine:3.23
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=builder /listener .

USER nobody
ENTRYPOINT ["./listener"]
