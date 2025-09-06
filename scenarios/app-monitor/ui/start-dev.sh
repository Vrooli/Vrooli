#!/bin/bash
# Development startup script for App Monitor UI

echo "🚀 Starting App Monitor Development Environment..."
echo ""

# Set default ports if not already set
export VITE_PORT=${VITE_PORT:-5173}
export UI_PORT=${UI_PORT:-8085}
export API_PORT=${API_PORT:-8090}

# Pass environment variables to Vite
export VITE_API_PORT=$API_PORT
export VITE_UI_PORT=$UI_PORT

echo "📍 Port Configuration:"
echo "  • Vite Dev Server: http://localhost:$VITE_PORT (React UI)"
echo "  • Express Server:  http://localhost:$UI_PORT (WebSocket & API proxy)"
echo "  • Go API Server:   http://localhost:$API_PORT (Backend API)"
echo ""

# Check if node_modules exists
if [ ! -d "node_modules" ]; then
    echo "📦 Installing dependencies..."
    npm install
fi

echo "✨ Starting servers..."
echo ""
echo "➡️  Access the UI at: http://localhost:$VITE_PORT"
echo ""

# Start both servers
npm run dev