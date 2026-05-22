#!/usr/bin/env sh
set -eu
cd "$(dirname "$0")/.."
export YFCDN_ADMIN_PASSWORD="${YFCDN_ADMIN_PASSWORD:-admin123}"
go run ./cmd/control -addr :8080 -data data/yfcdn.json -static web
