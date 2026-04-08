# Build from the workspace root:
#   docker build -f apps/ingestor/Dockerfile -t waifu-ingestor .
FROM golang:1.24-alpine AS builder
WORKDIR /workspace
# Each Dockerfile builds one app; disable workspace mode so Go does not
# try to load the other app's go.mod. The replace directive in go.mod
# handles the local pkg/event dependency.
ENV GOWORK=off

# Cache dependency downloads before copying source.
COPY pkg/event/go.mod ./pkg/event/
COPY apps/ingestor/go.mod apps/ingestor/go.sum ./apps/ingestor/
RUN cd apps/ingestor && go mod download

# Copy source and build.
COPY pkg/event/ ./pkg/event/
COPY apps/ingestor/ ./apps/ingestor/
RUN cd apps/ingestor && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /ingestor .

FROM alpine:3.23
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=builder /ingestor .

USER nobody
ENTRYPOINT ["./ingestor"]
