#!/bin/bash
# Development startup script for App Monitor UI

echo "🚀 Starting App Monitor Development Environment..."
echo ""

# Check required environment variables
if [ -z "$VITE_PORT" ] || [ -z "$API_PORT" ]; then
    echo "❌ Error: VITE_PORT and API_PORT environment variables are required"
    echo ""
    echo "Example:"
    echo "  export VITE_PORT=21800"
    echo "  export API_PORT=21600" 
    echo "  ./start-dev.sh"
    exit 1
fi

# Pass environment variables to Vite
export VITE_API_PORT=$API_PORT

echo "📍 Port Configuration:"
echo "  • Vite Dev Server: http://localhost:$VITE_PORT (React UI with hot reload)"
echo "  • Go API Server:   http://localhost:$API_PORT (Backend API & WebSocket)"
echo ""
echo "🔧 Development Features:"
echo "  • Hot reload for UI changes"
echo "  • TypeScript compilation"
echo "  • Proxied API calls to Go backend"
echo "  • WebSocket connection to Go backend"
echo ""

# Check if node_modules exists
if [ ! -d "node_modules" ]; then
    echo "📦 Installing dependencies..."
    npm install
fi

echo "✨ Starting Vite dev server..."
echo ""
echo "➡️  Access the UI at: http://localhost:$VITE_PORT"
echo ""

# Start simplified development server (just Vite)
npm run dev