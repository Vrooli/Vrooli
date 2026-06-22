import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { startScenarioServer } from '@vrooli/api-base/server';

const __dirname = path.dirname(fileURLToPath(import.meta.url));

startScenarioServer({
  uiPort: process.env.UI_PORT,
  apiPort: process.env.API_PORT,
  apiHost: (process.env.API_HOST || '127.0.0.1').trim() || '127.0.0.1',
  distDir: path.resolve(__dirname, 'dist'),
  serviceName: 'system-monitor',
  version: process.env.npm_package_version,
  corsOrigins: '*',
});
