# App Monitor UI - React + TypeScript + Vite

## Quick Start

### Install Dependencies
```bash
npm install
```

### Development Mode
Run both the Vite dev server and the Express backend server concurrently:
```bash
npm run dev
```

This will:
- Start Vite dev server on port 5173 with hot module replacement
- Start Express/WebSocket server on port 8085
- Enable React DevTools and source maps for debugging

### Individual Services
```bash
# Run only Vite dev server (frontend)
npm run dev:vite

# Run only Express server (backend)
npm run dev:server
```

### Production Build
```bash
# Build the React app
npm run build

# Serve the production build
NODE_ENV=production npm start
```

## Debugging Features

### 1. Source Maps
- Enabled in development for easy debugging
- Set breakpoints directly in TypeScript code
- Full stack traces with original file locations

### 2. React DevTools
- Install React Developer Tools browser extension
- Inspect component props, state, and hooks
- Profile performance and track re-renders

### 3. Network Tab
- Monitor API calls in browser DevTools
- Check WebSocket connections under WS tab
- All API requests logged to console in development

### 4. TypeScript Type Checking
```bash
# Run type checking without building
npm run typecheck

# Run ESLint
npm run lint
```

### 5. Debug Mode
For verbose logging and forced refresh:
```bash
npm run debug
```

## Project Structure

```
ui/
├── src/
│   ├── components/       # React components
│   │   ├── Layout.tsx    # Main layout with sidebar
│   │   ├── AppCard.tsx   # App display card
│   │   ├── AppModal.tsx  # App details modal
│   │   └── views/        # Page views
│   │       ├── AppsView.tsx
│   │       ├── MetricsView.tsx
│   │       ├── LogsView.tsx
│   │       ├── ResourcesView.tsx
│   │       └── TerminalView.tsx
│   ├── hooks/            # Custom React hooks
│   │   └── useWebSocket.ts
│   ├── services/         # API services
│   │   └── api.ts
│   ├── types/            # TypeScript type definitions
│   │   └── index.ts
│   ├── App.tsx          # Main app component with routing
│   ├── App.css          # Global styles
│   └── main.tsx         # Entry point
├── server.js            # Express server with WebSocket
├── vite.config.ts       # Vite configuration
├── tsconfig.json        # TypeScript configuration
└── package.json         # Dependencies and scripts
```

## Key Features

- **Hot Module Replacement**: Instant updates without page refresh
- **TypeScript**: Full type safety and IntelliSense
- **React Router**: Client-side routing with browser history
- **WebSocket Hook**: Real-time updates with automatic reconnection
- **API Service Layer**: Centralized error handling and logging
- **Responsive Design**: Mobile-friendly Matrix theme

## Environment Variables

Create `.env` file for custom configuration:
```env
UI_PORT=8085        # Express server port
API_PORT=8090       # API server port
VITE_PORT=5173      # Vite dev server port
```

## Troubleshooting

### Port Already in Use
```bash
# Kill process on specific port
lsof -i :5173 && kill -9 $(lsof -t -i :5173)
```

### Clean Install
```bash
npm run clean
npm install
```

### Check TypeScript Errors
```bash
npm run typecheck
```

### WebSocket Connection Issues
- Check browser console for connection errors
- Verify server is running on correct port
- Check for CORS or firewall issues

## Browser Compatibility

- Chrome/Edge: Full support with DevTools
- Firefox: Full support with React DevTools
- Safari: Full support (may need to enable Developer menu)