import path from 'node:path';
import { fileURLToPath } from 'node:url';
import { startScenarioServer } from '@vrooli/api-base/server';

// Both ports come from the scenario lifecycle. createScenarioServer already
// rejects either one when it is missing or unparseable, so these guards change
// no behaviour: the process still refuses to start. What they change is the
// message — the library reports "Invalid UI_PORT configuration", which names
// the symptom, and these name the cause and the fix.
const uiPort = process.env.UI_PORT;
if (!uiPort) {
  throw new Error(
    'UI_PORT is not set. The scenario lifecycle supplies it — start this scenario with `vrooli scenario start system-monitor` rather than running server.js directly.',
  );
}

const apiPort = process.env.API_PORT;
if (!apiPort) {
  throw new Error(
    'API_PORT is not set. The scenario lifecycle supplies it — start this scenario with `vrooli scenario start system-monitor` rather than running server.js directly.',
  );
}

const __dirname = path.dirname(fileURLToPath(import.meta.url));

startScenarioServer({
  uiPort,
  apiPort,
  apiHost: (process.env.API_HOST || '127.0.0.1').trim() || '127.0.0.1',
  distDir: path.resolve(__dirname, 'dist'),
  serviceName: 'system-monitor',
  version: process.env.npm_package_version || '0.0.0',
  corsOrigins: '*',
});
