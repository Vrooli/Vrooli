const express = require('express');
const path = require('path');
const fs = require('fs');

const app = express();
const uiPort = Number(process.env.UI_PORT || process.env.PORT || 3000);

const distDir = path.join(__dirname, 'dist');
const distIndex = path.join(distDir, 'index.html');
const publicDir = fs.existsSync(distIndex) ? distDir : __dirname;

if (publicDir === __dirname) {
  console.warn('⚠️  UI bundle not found. Serving files from ui/. Run `npm run build` for optimized assets.');
}

app.get('/health', (_req, res) => {
  res.json({
    status: 'healthy',
    scenario: 'code-smell',
    timestamp: new Date().toISOString()
  });
});

app.use(express.static(publicDir, { extensions: ['html'] }));

app.get('*', (_req, res) => {
  res.sendFile(path.join(publicDir, 'index.html'));
});

app.listen(uiPort, '0.0.0.0', () => {
  console.log('=====================================');
  console.log('✅ Code Smell UI server ready');
  console.log(`🌐 UI:     http://localhost:${uiPort}`);
  console.log(`💚 Health: http://localhost:${uiPort}/health`);
  console.log('=====================================');
});
