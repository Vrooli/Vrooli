import { createScenarioServer, proxyWebSocketUpgrade } from '@vrooli/api-base/server';

const app = createScenarioServer({
  uiPort: process.env.UI_PORT || 36221,
  apiPort: process.env.API_PORT || 15000,
  distDir: './dist',
  serviceName: 'app-issue-tracker',
  version: '2.0.0',
  verbose: true
});

const PORT = process.env.UI_PORT || 36221;
const API_PORT = process.env.API_PORT || 15000;

const server = app.listen(PORT, () => {
  const displayHost = process.env.UI_HOST || '127.0.0.1';
  console.log(`App Issue Tracker UI available at http://${displayHost}:${PORT}`);
  console.log(`Proxying API requests to http://127.0.0.1:${API_PORT}`);
});

// Handle WebSocket upgrade requests
server.on('upgrade', (req, socket, head) => {
  // WebSocket endpoint is at /api/v1/ws on the API server
  if (req.url?.startsWith('/api')) {
    proxyWebSocketUpgrade(req, socket, head, {
      apiPort: API_PORT,
      verbose: true
    });
  } else {
    socket.destroy();
  }
});
