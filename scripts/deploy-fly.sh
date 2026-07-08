#!/bin/bash
set -e

# ==============================================================
# deploy-fly.sh — Automated deployment of thefeed-server to Fly.io
# ==============================================================
# Usage:
#   export THEFEED_DOMAIN=t.example.com
#   export THEFEED_KEY="your-secret-passphrase"
#   bash scripts/deploy-fly.sh
#
# Prerequisites: flyctl (https://fly.io/docs/hands-on/install-flyctl/)
# ==============================================================

red='\033[0;31m'
green='\033[0;32m'
yellow='\033[0;33m'
blue='\033[0;34m'
plain='\033[0m'

echo -e "${green}======================================================${plain}"
echo -e "${green}  thefeed-server — Fly.io Deployment${plain}"
echo -e "${green}======================================================${plain}"
echo ""

# --- Check prerequisites ---
if ! command -v fly &>/dev/null; then
    echo -e "${red}Error: 'fly' CLI not found. Install it first:${plain}"
    echo "  curl -L https://fly.io/install.sh | sh"
    exit 1
fi

# --- Gather config ---
DOMAIN="${THEFEED_DOMAIN:-}"
KEY="${THEFEED_KEY:-}"

if [ -z "$DOMAIN" ]; then
    read -rp "Enter your feed domain (e.g., t.example.com): " DOMAIN
fi
if [ -z "$KEY" ]; then
    read -rp "Enter encryption passphrase: " KEY
fi

# --- Login to Fly.io (if needed) ---
echo -e "${blue}Checking Fly.io auth...${plain}"
fly auth whois &>/dev/null || fly auth login

# --- Create app ---
APP_NAME="thefeed-$(echo $DOMAIN | tr '.' '-' | tr '[:upper:]' '[:lower:]')"
echo -e "${green}Creating app: ${APP_NAME}${plain}"
fly apps create "$APP_NAME" --machines 2>/dev/null || echo -e "${yellow}App already exists, using it${plain}"

# --- Set secrets ---
echo -e "${green}Setting secrets...${plain}"
fly secrets set \
    THEFEED_DOMAIN="$DOMAIN" \
    THEFEED_KEY="$KEY" \
    THEFEED_NO_TELEGRAM="1" \
    THEFEED_ALLOW_MANAGE="0" \
    THEFEED_LISTEN=":5300" \
    THEFEED_MSG_LIMIT="15" \
    THEFEED_FETCH_INTERVAL="10" \
    THEFEED_PADDING="32" \
    --app "$APP_NAME"

# --- Create volume for persistent storage ---
echo -e "${green}Creating volume...${plain}"
fly volumes create thefeed_data --region ams --size 1 --app "$APP_NAME" 2>/dev/null || \
    echo -e "${yellow}Volume already exists${plain}"

# --- Deploy ---
echo -e "${green}Deploying...${plain}"
fly deploy --app "$APP_NAME" --dockerfile Dockerfile.free --remote-only

# --- Show IP and config ---
echo ""
echo -e "${green}======================================================${plain}"
echo -e "${green}  Deployment Complete!${plain}"
echo -e "${green}======================================================${plain}"
echo ""
echo -e "${blue}Your server IP:${plain}"
fly ips list --app "$APP_NAME"
echo ""
echo -e "${blue}DNS records to create:${plain}"
IP=$(fly ips list --app "$APP_NAME" 2>/dev/null | grep -vi 'type' | grep 'v4' | awk '{print $2}')
echo ""
echo -e "${yellow}  1) A record:  ns.example.com  →  ${IP}${plain}"
echo -e "${yellow}  2) NS record: ${DOMAIN}  →  ns.example.com${plain}"
echo ""
echo -e "${blue}Generate config URI (run this on any machine with the binary):${plain}"
echo "  ./files/thefeed-server-linux-amd64 --print-config \\"
echo "    --data-dir /tmp --domain ${DOMAIN} --key ${KEY}"
echo ""
echo -e "${blue}Or directly on Fly.io:${plain}"
echo "  fly ssh console -C \"thefeed-server --print-config --data-dir /data\" --app ${APP_NAME}"
echo ""
echo -e "${blue}View logs:${plain}"
echo "  fly logs --app ${APP_NAME}"
echo ""
