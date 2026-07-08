#!/bin/sh
set -e

# docker-entrypoint.sh — thefeed-server entrypoint
# Sets up data directory and passes flags to the server.
# Config comes from env vars (read automatically by the server).

DATA_DIR="${THEFEED_DATA_DIR:-/data}"
mkdir -p "$DATA_DIR"

cd "$DATA_DIR"

# Create stub channel files if missing
[ -f channels.txt ] || cat > channels.txt << 'EOF'
# Telegram channel usernames (one per line)
# Lines starting with # are comments
EOF

[ -f x_accounts.txt ] || cat > x_accounts.txt << 'EOF'
# X usernames (one per line, without @)
EOF

# Build flag list
FLAGS="--data-dir $DATA_DIR"

# Listen: Fly.io maps external 53 → internal port 5300.
# Use :53 (default) when the process can bind it directly.
# Override with THEFEED_LISTEN env var.
LISTEN="${THEFEED_LISTEN:-:5300}"
FLAGS="$FLAGS --listen $LISTEN"

# --no-telegram is required if Telegram env vars are not set.
# The server binary does NOT read an env var for this flag,
# so we pass it explicitly.
if [ "${THEFEED_NO_TELEGRAM}" = "1" ] || [ -z "$TELEGRAM_API_ID" ]; then
    FLAGS="$FLAGS --no-telegram"
fi

echo "[entrypoint] starting: thefeed-server --listen $LISTEN"
echo "[entrypoint] domain: ${THEFEED_DOMAIN:-not set}"

exec thefeed-server $FLAGS
