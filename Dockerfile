# syntax=docker/dockerfile:1

# -- Stage 1: Build thin Go layer --
FROM golang:1.26.0-alpine AS build

RUN apk add --no-cache git

WORKDIR /src

# Cache dependencies
COPY agent/go.mod agent/go.sum ./
RUN go mod download

# Copy source and build
COPY agent/*.go .
RUN go build -o /sme-assistant .

# -- Stage 2: Production image --
FROM alpine:3.23

# CLI tools for agent's bash/exec tool + Python for skills
RUN apk add --no-cache \
      ca-certificates tzdata \
      jq curl grep sed gawk \
      coreutils findutils \
      python3 py3-pip \
      git github-cli \
    && pip3 install --no-cache-dir --break-system-packages openpyxl

# Workspace directory (volume mount point)
RUN mkdir -p /workspace
WORKDIR /workspace

# Copy binary
COPY --from=build /sme-assistant /usr/local/bin/sme-assistant

# Copy picoclaw config
COPY agent/config/config.json /root/.picoclaw/config.json

# Stage workspace templates for runtime copy (volume mount hides baked-in files)
COPY agent/workspace/ /etc/picoclaw/workspace-templates/

# Entrypoint: inject env vars into config, seed workspace templates, run server
COPY <<'EOF' /usr/local/bin/entrypoint.sh
#!/bin/sh
set -e

CONFIG=/root/.picoclaw/config.json

# Inject OPENAI_API_KEY into picoclaw config
if [ -n "$OPENAI_API_KEY" ]; then
  tmp=$(jq --arg key "$OPENAI_API_KEY" '.providers.openai.api_key = $key' "$CONFIG") \
    && printf '%s\n' "$tmp" > "$CONFIG"
fi

# Map GH_TOKEN for GitHub CLI
if [ -n "$GH_TOKEN" ]; then
  export GITHUB_TOKEN="$GH_TOKEN"
fi

# Copy workspace templates if not already present (don't overwrite user data)
for f in /etc/picoclaw/workspace-templates/*; do
  base="$(basename "$f")"
  if [ ! -e "/workspace/$base" ]; then
    cp -r "$f" "/workspace/$base"
  fi
done

exec sme-assistant "$@"
EOF
RUN chmod +x /usr/local/bin/entrypoint.sh

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
  CMD curl -sf http://localhost:8080/api/health || exit 1

ENTRYPOINT ["/usr/local/bin/entrypoint.sh"]
