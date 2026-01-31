#!/bin/bash

echo "🧹 Stopping and removing go_app container, volumes, and network..."
docker compose rm -sfv go_app

echo "🔄 Rebuilding go_app..."
cd "$(dirname "$0")" || exit 1

docker compose build --no-cache go_app && \
docker compose up -d go_app

echo "✅ go_app rebuilt and restarted!"
