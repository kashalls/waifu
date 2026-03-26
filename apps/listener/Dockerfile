# Build from the workspace root:
#   docker build -f apps/listener/Dockerfile -t waifu-listener .
FROM golang:1.24-alpine AS builder
WORKDIR /workspace

# Copy workspace manifest so Go can resolve all modules.
COPY go.work go.work.sum ./

# Cache dependency downloads.
COPY pkg/event/go.mod ./pkg/event/
COPY apps/listener/go.mod apps/listener/go.sum ./apps/listener/
RUN go mod download -modfile apps/listener/go.mod

# Copy source and build.
COPY pkg/event/ ./pkg/event/
COPY apps/listener/ ./apps/listener/
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /listener ./apps/listener

FROM alpine:3.21
RUN apk --no-cache add ca-certificates tzdata
WORKDIR /app

COPY --from=builder /listener .

USER nobody
ENTRYPOINT ["./listener"]
