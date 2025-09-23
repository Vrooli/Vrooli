#!/bin/bash
set -e

cd "$(dirname "$0")"

echo "Installing App Debugger CLI..."

go build -o app-debugger .

chmod +x app-debugger

ln -sf "$(pwd)/app-debugger" ~/.local/bin/app-debugger

echo "App Debugger CLI installed successfully!"