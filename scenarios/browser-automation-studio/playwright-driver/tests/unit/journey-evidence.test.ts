import { mkdtempSync, readFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import {
  measureJourney,
  recordJourneyReceipt,
  recordJourneyRuntimeIdentity,
} from '../journeys/support';

type EvidenceRecord = {
  journey_id: string;
  event: string;
  kind: 'timing' | 'receipt';
  outcome: 'ok' | 'failed';
  elapsed_ms?: number;
  facts?: Record<string, unknown>;
  error_type?: string;
  runtime_identity?: {
    browser_version: string;
    driver_version: string;
    driver_browser_version?: string;
  };
};

const readEvidence = (path: string): EvidenceRecord[] =>
  readFileSync(path, 'utf8')
    .trim()
    .split('\n')
    .map((line) => JSON.parse(line) as EvidenceRecord);

describe('BAS UI journey evidence', () => {
  const originalFile = process.env.BAS_JOURNEY_EVIDENCE_FILE;
  const originalRunId = process.env.BAS_JOURNEY_RUN_ID;
  const originalDriverURL = process.env.BAS_DRIVER_URL;
  let directory: string;
  let evidenceFile: string;
  let fetchSpy: jest.SpyInstance | undefined;

  beforeEach(() => {
    directory = mkdtempSync(join(tmpdir(), 'bas-journey-evidence-'));
    evidenceFile = join(directory, 'journey.jsonl');
    process.env.BAS_JOURNEY_EVIDENCE_FILE = evidenceFile;
    process.env.BAS_JOURNEY_RUN_ID = 'test-run';
    process.env.BAS_DRIVER_URL = 'http://127.0.0.1:43210';
  });

  afterEach(() => {
    if (originalFile === undefined) delete process.env.BAS_JOURNEY_EVIDENCE_FILE;
    else process.env.BAS_JOURNEY_EVIDENCE_FILE = originalFile;
    if (originalRunId === undefined) delete process.env.BAS_JOURNEY_RUN_ID;
    else process.env.BAS_JOURNEY_RUN_ID = originalRunId;
    if (originalDriverURL === undefined) delete process.env.BAS_DRIVER_URL;
    else process.env.BAS_DRIVER_URL = originalDriverURL;
    fetchSpy?.mockRestore();
    rmSync(directory, { recursive: true, force: true });
  });

  it('writes timing samples and durable receipts as versioned JSON lines', async () => {
    fetchSpy = jest.spyOn(globalThis, 'fetch').mockResolvedValue(
      new Response(
        JSON.stringify({
          status: 'ok',
          ready: true,
          timestamp: '2026-11-21T00:00:00.000Z',
          sessions: 0,
          driver_version: 'driver-test-17',
          browser: { healthy: true, version: 'Chromium 140.0.0.0' },
        }),
        { status: 200, headers: { 'Content-Type': 'application/json' } }
      )
    );
    await recordJourneyRuntimeIdentity('J26', 'Chromium 140.0.0.0');
    const sessionId = await measureJourney(
      'J26',
      'setup.canvas_ready',
      async () => {
        await Promise.resolve();
        return 'session-26';
      },
      (id) => ({ session_id: id })
    );
    recordJourneyReceipt('J26', 'recording.closed', {
      session_id: sessionId,
      status: 'closed',
      profile_persisted: true,
    });

    const [identity, timing, receipt] = readEvidence(evidenceFile);
    expect(identity).toMatchObject({
      journey_id: 'J26',
      event: 'runtime.identity',
      runtime_identity: {
        browser_version: 'Chromium 140.0.0.0',
        driver_version: 'driver-test-17',
        driver_browser_version: 'Chromium 140.0.0.0',
      },
    });
    expect(timing).toMatchObject({
      schema_version: 1,
      run_id: 'test-run',
      journey_id: 'J26',
      event: 'setup.canvas_ready',
      kind: 'timing',
      outcome: 'ok',
      facts: { session_id: 'session-26' },
      runtime_identity: identity.runtime_identity,
    });
    expect(timing.elapsed_ms).toEqual(expect.any(Number));
    expect(receipt).toMatchObject({
      schema_version: 1,
      run_id: 'test-run',
      journey_id: 'J26',
      event: 'recording.closed',
      kind: 'receipt',
      outcome: 'ok',
      facts: { session_id: 'session-26', status: 'closed', profile_persisted: true },
      runtime_identity: identity.runtime_identity,
    });
  });

  it('persists failed milestones before rethrowing the journey error', async () => {
    await expect(
      measureJourney('J28', 'cancel.to_terminal', async () => {
        await Promise.resolve();
        throw new Error('expected test failure');
      })
    ).rejects.toThrow('expected test failure');
    const [record] = readEvidence(evidenceFile);
    expect(record).toMatchObject({
      journey_id: 'J28',
      event: 'cancel.to_terminal',
      kind: 'timing',
      outcome: 'failed',
      error_type: 'Error',
    });
    expect(record.elapsed_ms).toEqual(expect.any(Number));
  });
});
