const http = require('http');

const port = process.env.API_PORT || 36250;

const server = http.createServer((req, res) => {
  if (req.url === '/health') {
    res.writeHead(200, { 'Content-Type': 'application/json' });
    res.end(JSON.stringify({ status: 'healthy', service: 'simple-test' }));
  } else {
    res.writeHead(200, { 'Content-Type': 'text/plain' });
    res.end('Simple Test App is running!');
  }
});

server.listen(port, () => {
  console.log(`Simple test server running on port ${port}`);
});