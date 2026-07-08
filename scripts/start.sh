#!/bin/bash
set -e

# ==============================================================
# start.sh — thefeed-server deployment script
#
# Guides you through deploying to Fly.io or Railway.
# Usage:
#   bash scripts/start.sh --domain t.example.com --key passphrase
#   bash scripts/start.sh --platform flyio --domain t.example.com --key passphrase
#   bash scripts/start.sh --help
# ==============================================================

red='\033[0;31m'; green='\033[0;32m'; yellow='\033[0;33m'; blue='\033[0;34m'; plain='\033[0m'

VERSION="1.0.0"
DOMAIN="${THEFEED_DOMAIN:-}"
KEY="${THEFEED_KEY:-}"
PLATFORM=""

# --- Parse CLI args ---
while [[ $# -gt 0 ]]; do
    case "$1" in
        --domain)     DOMAIN="$2"; shift 2 ;;
        --key)        KEY="$2"; shift 2 ;;
        --platform)   PLATFORM="$2"; shift 2 ;;
        --with-telegram)
            NO_TELEGRAM="0"; echo "WARNING: Telegram mode requires API credentials at runtime."; shift ;;
        --help|-h)
            echo "Usage: bash $0 [OPTIONS]"
            echo ""
            echo "  --domain DOMAIN       Feed DNS domain (required)"
            echo "  --key PASSPHRASE      Encryption passphrase (required)"
            echo "  --platform PLATFORM   Target: flyio | railway | auto"
            echo "  --with-telegram       Enable Telegram login"
            echo ""
            echo "Env vars: THEFEED_DOMAIN, THEFEED_KEY"
            echo ""
            echo "Quick:"
            echo "  export THEFEED_DOMAIN=t.example.com THEFEED_KEY=mykey"
            echo "  bash $0"
            exit 0 ;;
        *) echo -e "${red}Unknown: $1${plain}"; exit 1 ;;
    esac
done

if [ -z "$DOMAIN" ] || [ -z "$KEY" ]; then
    echo -e "${red}Error: --domain and --key are required${plain}"
    echo "Usage: bash $0 --domain t.example.com --key mykey"
    exit 1
fi

# --- Select platform ---
if [ -z "$PLATFORM" ]; then
    echo ""
    echo -e "${green}Where do you want to deploy thefeed-server?${plain}"
    echo "  1) Fly.io  ${yellow}(recommended — UDP 53, static IPv4, free tier)${plain}"
    echo "  2) Railway  ${yellow}(Docker host, UDP 53 uncertain, free tier with limits)${plain}"
    echo ""
    read -rp "Choose [1/2]: " choice
    case "$choice" in
        2) PLATFORM="railway" ;;
        *) PLATFORM="flyio" ;;
    esac
fi

echo ""
echo -e "${blue}Target:     ${PLATFORM}${plain}"
echo -e "${blue}Domain:     ${DOMAIN}${plain}"
echo ""

# ==============================================================
case "$PLATFORM" in
    flyio)
        # Delegate to deploy-fly.sh
        THEFEED_DOMAIN="$DOMAIN" THEFEED_KEY="$KEY" bash "$(dirname "$0")/deploy-fly.sh"
        ;;
    railway)
        echo -e "${green}======================================================${plain}"
        echo -e "${green}  Railway Deployment Instructions${plain}"
        echo -e "${green}======================================================${plain}"
        echo ""
        echo -e "${yellow}Step 1: Push your code to a GitHub repo${plain}"
        echo "  git init && git add . && git commit -m 'initial'"
        echo "  gh repo create thefeed-server --public --push"
        echo "  # or: create a repo manually and: git remote add origin ... && git push -u origin main"
        echo ""
        echo -e "${yellow}Step 2: Create Railway project${plain}"
        echo "  - Go to https://railway.app/new"
        echo "  - Select 'Deploy from GitHub repo'"
        echo "  - Choose your thefeed-server repo"
        echo ""
        echo -e "${yellow}Step 3: Configure Railway${plain}"
        echo "  - Railway auto-detects Dockerfile.free (set via railway.json)"
        echo "  - Add environment variables:"
        echo "    ${green}THEFEED_DOMAIN=${DOMAIN}${plain}"
        echo "    ${green}THEFEED_KEY=${KEY}${plain}"
        echo "    THEFEED_NO_TELEGRAM=1"
        echo "    THEFEED_LISTEN=:5300"
        echo "  - Add a volume: ${green}/data${plain} (persistent disk)"
        echo ""
        echo -e "${yellow}Step 4: Expose UDP port 53${plain}"
        echo "  - In Railway, add a public networking UDP port"
        echo "    Port: 53 → Forward: 5300"
        echo "  ${red}Note: Railway UDP 53 support is unverified.${plain}"
        echo "  If DNS queries fail, use Fly.io instead."
        echo ""
        echo -e "${yellow}Step 5: DNS setup${plain}"
        echo "  Railway provides a *.railway.app domain but NOT a static IP."
        echo "  You need a static IP for NS delegation."
        echo "  ${yellow}Option A:${plain} Use Railway's TCP proxy + DNS-over-TLS (not supported by thefeed)"
        echo "  ${yellow}Option B:${plain} Use Fly.io (has static IPv4) and skip Railway"
        echo ""
        echo -e "${green}For a fully working setup, use Fly.io instead:${plain}"
        echo "  bash $0 --platform flyio --domain ${DOMAIN} --key ${KEY}"
        echo ""
        echo -e "${blue}If you still want to try Railway:${plain}"
        echo "  After deploy, run: railway shell"
        echo "  Inside the shell: thefeed-server --print-config --data-dir /data"
        echo ""
        ;;
    *)
        echo -e "${red}Unknown platform: ${PLATFORM}${plain}"
        echo "Supported: flyio, railway"
        exit 1
        ;;
esac
