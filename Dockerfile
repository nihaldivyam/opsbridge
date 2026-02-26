# ==========================================
# Stage 1: Builder
# ==========================================
FROM golang:1.24-alpine AS builder

# Install git and SSL ca certificates
RUN apk update && apk add --no-cache git ca-certificates tzdata && update-ca-certificates

# Create a dedicated non-root user (numeric IDs are best for Kubernetes security policies)
ENV USER=appuser
ENV UID=10001 
RUN adduser -D -g "" -h "/nonexistent" -s "/sbin/nologin" -H -u "${UID}" "${USER}"

WORKDIR /build

# Leverage Docker cache for dependencies
COPY go.mod go.sum ./
RUN go mod download

# Copy the rest of the application
COPY . .

ARG TARGETARCH

# Build a strictly static binary
# -trimpath removes your local machine's file paths from the compiled binary
# -ldflags="-w -s" strips debugging information
# -extldflags "-static" ensures absolutely no dynamic linking
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build -trimpath -ldflags="-w -s -extldflags '-static'" -o opsbridge ./cmd/bot

# ==========================================
# Stage 2: Final Production Image (Scratch)
# ==========================================
FROM scratch

# 1. Import the CA certificates so HTTPS calls to Gitea/Mattermost work
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/

# 2. Import the non-root user and group files
COPY --from=builder /etc/passwd /etc/passwd
COPY --from=builder /etc/group /etc/group

# 3. Import the compiled static binary
COPY --from=builder /build/opsbridge /opsbridge

# 4. Enforce the non-root numeric user for Kubernetes PodSecurityStandards
USER 10001:10001

# 5. Optional: Expose port if using the Slash Command HTTP listener
EXPOSE 8080

# Run the binary directly (no shell wrap)
ENTRYPOINT ["/opsbridge"]