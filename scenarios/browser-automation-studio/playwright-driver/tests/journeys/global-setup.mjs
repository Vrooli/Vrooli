// Resolves an isolated non-live BAS instance and starts only the local fixture site.
// Never starts, stops or restarts BAS: an absent or unhealthy target fails the run.
import { execFileSync } from 'node:child_process';
import { randomUUID } from 'node:crypto';
import { mkdir, writeFile } from 'node:fs/promises';
import { dirname, join, resolve } from 'node:path';
import { fileURLToPath } from 'node:url';
import { startJourneySite } from '../../../fixtures/journey-site/server.mjs';
import { isActiveJourneyEngagement } from './engagement.mjs';
import { resolveJourneyTarget } from './target.mjs';

const SCENARIO = 'browser-automation-studio';
const instance = (process.env.BAS_JOURNEY_INSTANCE ?? 'shadow').trim();
const engagement = process.env.BAS_JOURNEY_ENGAGEMENT ?? 'bas-goal';

const run = (command, args) =>
  execFileSync(command, args, { encoding: 'utf8', timeout: 30000 }).trim();

async function healthy(url, isReady) {
  const response = await fetch(url, { signal: AbortSignal.timeout(5000) }).catch(() => undefined);
  return response?.ok === true && isReady(await response.json());
}

export default async function globalSetup() {
  const hasExplicitTarget = [
    'BAS_JOURNEY_API_URL',
    'BAS_JOURNEY_UI_URL',
    'BAS_JOURNEY_DRIVER_URL',
  ].some((key) => Boolean(process.env[key]?.trim()));
  if (!hasExplicitTarget) {
    const { engagements = [] } = JSON.parse(
      run('git-control-tower', ['baseline', 'status', '--json'])
    );
    if (
      !engagements.some((item) => isActiveJourneyEngagement(item, SCENARIO, engagement, instance))
    ) {
      throw new Error(
        `shadow unavailable: engagement ${engagement} (${instance}) is not active for ${SCENARIO}`
      );
    }
  }
  const target = resolveJourneyTarget(process.env, (selectedInstance) =>
    Object.fromEntries(
      run('vrooli', ['scenario', 'port', SCENARIO, '--instance', selectedInstance])
        .split('\n')
        .filter(Boolean)
        .map((line) => line.split('=', 2))
    )
  );
  const { api, ui, driver } = target;
  const [apiReady, uiReady, driverReady] = await Promise.all([
    healthy(`${api}/health`, (body) => body.status === 'healthy' && body.readiness === true),
    fetch(ui, { signal: AbortSignal.timeout(5000) })
      .then((response) => response.ok)
      .catch(() => false),
    healthy(`${driver}/health`, (body) => body.status === 'ok' && body.ready === true),
  ]);
  if (!apiReady || !uiReady || !driverReady) {
    throw new Error(
      `${target.source} target unavailable: ${instance} health api=${apiReady} ui=${uiReady} driver=${driverReady}`
    );
  }
  const site = await startJourneySite();
  globalThis.__BAS_JOURNEY_SITE__ = site;
  const runId = randomUUID();
  const driverRoot = resolve(dirname(fileURLToPath(import.meta.url)), '../..');
  const evidenceDir = resolve(
    process.env.BAS_JOURNEY_EVIDENCE_DIR ?? join(driverRoot, 'test-results', 'mb3-journeys')
  );
  await mkdir(evidenceDir, { recursive: true });
  const evidenceFile = join(evidenceDir, `journey-${runId}.jsonl`);
  await writeFile(
    evidenceFile,
    `${JSON.stringify({
      schema_version: 1,
      run_id: runId,
      recorded_at: new Date().toISOString(),
      journey_id: 'MB3',
      event: 'journey-suite.started',
      kind: 'receipt',
      outcome: 'ok',
      facts: { instance, target_source: target.source, fixture_ready: true },
    })}\n`,
    { encoding: 'utf8', flag: 'wx', mode: 0o600 }
  );
  // Jest workers inherit this environment; support.ts is its only reader.
  Object.assign(process.env, {
    BAS_API_URL: api,
    BAS_UI_URL: ui,
    BAS_DRIVER_URL: driver,
    BAS_FIXTURE_ORIGIN: site.origin,
    BAS_JOURNEY_RUN_ID: runId,
    BAS_JOURNEY_EVIDENCE_FILE: evidenceFile,
  });
}
