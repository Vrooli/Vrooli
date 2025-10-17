const http = require('http');

const port = process.env.API_PORT || 36250;

function createRequestHandler() {
  return (req, res) => {
    if (req.url === '/health') {
      res.writeHead(200, { 'Content-Type': 'application/json' });
      res.end(JSON.stringify({ status: 'healthy', service: 'simple-test' }));
    } else {
      res.writeHead(200, { 'Content-Type': 'text/plain' });
      res.end('Simple Test App is running!');
    }
  };
}

function startServer(customPort) {
  const serverPort = customPort || port;
  const server = http.createServer(createRequestHandler());

  server.listen(serverPort, () => {
    console.log(`Simple test server running on port ${serverPort}`);
  });

  return server;
}

// Only start server if run directly
if (require.main === module) {
  startServer();
}

module.exports = { createRequestHandler, startServer };