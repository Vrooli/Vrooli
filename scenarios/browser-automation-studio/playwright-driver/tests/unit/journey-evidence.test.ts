import { mkdtempSync, readFileSync, rmSync } from 'node:fs';
import { tmpdir } from 'node:os';
import { join } from 'node:path';
import { measureJourney, recordJourneyReceipt } from '../journeys/support';

describe('BAS UI journey evidence', () => {
  const originalFile = process.env.BAS_JOURNEY_EVIDENCE_FILE;
  const originalRunId = process.env.BAS_JOURNEY_RUN_ID;
  let directory: string;
  let evidenceFile: string;

  beforeEach(() => {
    directory = mkdtempSync(join(tmpdir(), 'bas-journey-evidence-'));
    evidenceFile = join(directory, 'journey.jsonl');
    process.env.BAS_JOURNEY_EVIDENCE_FILE = evidenceFile;
    process.env.BAS_JOURNEY_RUN_ID = 'test-run';
  });

  afterEach(() => {
    if (originalFile === undefined) delete process.env.BAS_JOURNEY_EVIDENCE_FILE;
    else process.env.BAS_JOURNEY_EVIDENCE_FILE = originalFile;
    if (originalRunId === undefined) delete process.env.BAS_JOURNEY_RUN_ID;
    else process.env.BAS_JOURNEY_RUN_ID = originalRunId;
    rmSync(directory, { recursive: true, force: true });
  });

  it('writes timing samples and durable receipts as versioned JSON lines', async () => {
    const sessionId = await measureJourney(
      'J26',
      'setup.canvas_ready',
      async () => 'session-26',
      (id) => ({ session_id: id })
    );
    recordJourneyReceipt('J26', 'recording.closed', {
      session_id: sessionId,
      status: 'closed',
      profile_persisted: true,
    });

    const [timing, receipt] = readFileSync(evidenceFile, 'utf8')
      .trim()
      .split('\n')
      .map((line) => JSON.parse(line));
    expect(timing).toMatchObject({
      schema_version: 1,
      run_id: 'test-run',
      journey_id: 'J26',
      event: 'setup.canvas_ready',
      kind: 'timing',
      outcome: 'ok',
      facts: { session_id: 'session-26' },
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
    });
  });

  it('persists failed milestones before rethrowing the journey error', async () => {
    await expect(
      measureJourney('J28', 'cancel.to_terminal', async () => {
        throw new Error('expected test failure');
      })
    ).rejects.toThrow('expected test failure');
    const [record] = readFileSync(evidenceFile, 'utf8')
      .trim()
      .split('\n')
      .map((line) => JSON.parse(line));
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
