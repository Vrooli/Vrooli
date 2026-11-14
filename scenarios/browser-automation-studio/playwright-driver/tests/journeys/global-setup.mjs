// Resolves the goal's shadow BAS and starts the fixture site for the journey suite.
// Never starts, stops or restarts any BAS instance: an absent or unhealthy shadow fails the run.
import { execFileSync } from 'node:child_process';
import { startJourneySite } from '../../../fixtures/journey-site/server.mjs';

const SCENARIO = 'browser-automation-studio';
const instance = process.env.BAS_JOURNEY_INSTANCE ?? 'shadow';
const engagement = process.env.BAS_JOURNEY_ENGAGEMENT ?? 'bas-goal';

const run = (command, args) => execFileSync(command, args, { encoding: 'utf8', timeout: 30000 }).trim();

async function healthy(url, isReady) {
  const response = await fetch(url, { signal: AbortSignal.timeout(5000) }).catch(() => undefined);
  return response?.ok === true && isReady(await response.json());
}

export default async function globalSetup() {
  const { engagements = [] } = JSON.parse(run('git-control-tower', ['baseline', 'status', '--json']));
  if (!engagements.some((item) => item.Scenario === SCENARIO && item.Slug === engagement && item.Mode === instance)) {
    throw new Error(`shadow unavailable: engagement ${engagement} (${instance}) is not active for ${SCENARIO}`);
  }
  const ports = Object.fromEntries(run('vrooli', ['scenario', 'port', SCENARIO, '--instance', instance])
    .split('\n').filter(Boolean).map((line) => line.split('=', 2)));
  const api = `http://127.0.0.1:${ports.API_PORT}`;
  const driver = `http://127.0.0.1:${ports.PLAYWRIGHT_DRIVER_PORT}`;
  const [apiReady, driverReady] = await Promise.all([
    healthy(`${api}/health`, (body) => body.status === 'healthy' && body.readiness === true),
    healthy(`${driver}/health`, (body) => body.status === 'ok' && body.ready === true),
  ]);
  if (!ports.API_PORT || !ports.PLAYWRIGHT_DRIVER_PORT || !apiReady || !driverReady) {
    throw new Error(`shadow unavailable: ${instance} health api=${apiReady} driver=${driverReady}`);
  }
  const site = await startJourneySite();
  globalThis.__BAS_JOURNEY_SITE__ = site;
  // Jest workers inherit this environment; support.ts is its only reader.
  Object.assign(process.env, { BAS_API_URL: api, BAS_DRIVER_URL: driver, BAS_FIXTURE_ORIGIN: site.origin });
}
