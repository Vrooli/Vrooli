const http = require('http');

const server = http.createServer((req, res) => {
  res.writeHead(200, { 'Content-Type': 'text/html' });
  res.end(`<!DOCTYPE html><html><body><div id="app">Server rendered UI</div><script>console.log('ui ready');</script></body></html>`);
});

server.listen(0, () => {
  console.log('server ready');
});
