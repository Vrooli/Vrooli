/** Scenario-managed UI server. Public HTML uses the published presentation owner. */
import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';
import { pathToFileURL } from 'node:url';
import { createScenarioServer } from '@vrooli/api-base/server';
import { createPresentationDocuments } from './server/presentation-documents.mjs';

export function createLandingServer({ uiPort, apiPort, distDir = './dist', fetchImplementation = fetch }) {
  const validPort = value => /^\d+$/.test(String(value)) && Number(value) > 0 && Number(value) <= 65535;
  if (!validPort(uiPort) || !validPort(apiPort)) throw new Error('UI_PORT and API_PORT must be ports between 1 and 65535');
  const indexPath = resolve(distDir, 'index.html');
  return createScenarioServer({
    uiPort, apiPort, distDir, serviceName: 'landing-page-business-suite', version: '1.0.0', corsOrigins: '*',
    setupRoutes: app => {
      // Handle HTML before express.static: its root index uses sendFile, which
      // bypasses send-interception and previously skipped root-page SEO entirely.
      app.use(createPresentationDocuments({
        apiBase: `http://127.0.0.1:${apiPort}`, fetchImplementation,
        readIndex: () => readFile(indexPath, 'utf8'),
      }));
    },
  });
}

if (process.argv[1] && import.meta.url === pathToFileURL(resolve(process.argv[1])).href) {
  createLandingServer({ uiPort: process.env.UI_PORT, apiPort: process.env.API_PORT }).listen(process.env.UI_PORT);
}
