/**
 * Shared support for the preservation journeys. global-setup.mjs resolves the
 * goal's shadow BAS and starts the fixture site; this module is the only
 * reader of that environment and the only BAS/driver/fixture client.
 */
import { randomUUID } from 'node:crypto';
import { appendFileSync, readFileSync } from 'node:fs';
import { expect, type Page } from '@playwright/test';
import { join } from 'node:path';
import {
  create,
  fromJson,
  toJson,
  type DescMessage,
  type DescMethodUnary,
  type JsonValue,
  type MessageInitShape,
  type MessageShape,
} from '@bufbuild/protobuf';
import {
  ActionDefinitionSchema,
  type ActionDefinition,
} from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import {
  KeyboardParamsSchema,
  ShortcutParamsSchema,
  DragDropParamsSchema,
} from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import type { StepOutcome } from '@vrooli/proto-types/browser-automation-studio/v1/execution/driver_pb';
import { ExecutionMode } from '@vrooli/proto-types/browser-automation-studio/v1/workflows/definition_pb';
import { WorkflowsService } from '@vrooli/proto-types/browser-automation-studio/v1/api/service_pb';
import { createTypedInstruction } from '../helpers/instruction-factory';
import type {
  CloseSessionRequest,
  HealthResponse,
  SessionSpec,
  StartSessionRequest,
  StartSessionResponse,
} from '../../src/types';

export const TIMEOUT_MS = { request: 30000, wait: 10000 } as const;
export const VIEWPORT = { width: 640, height: 480 } as const;

export type JourneyEvidenceFacts = Record<string, string | number | boolean | null | string[]>;
type JourneyEvidenceRecord = {
  schema_version: 1;
  run_id: string;
  recorded_at: string;
  journey_id: string;
  event: string;
  kind: 'timing' | 'receipt';
  outcome: 'ok' | 'failed';
  elapsed_ms?: number;
  facts?: JourneyEvidenceFacts;
  error_type?: string;
  runtime_identity?: JourneyRuntimeIdentity;
};
export type JourneyRuntimeIdentity = {
  browser_version: string;
  driver_version: string;
  driver_browser_version?: string;
};

export type ResponsivenessMetric =
  | 'setup_to_ready_ms'
  | 'click_to_paint_ms'
  | 'tab_switch_to_first_frame_ms'
  | 'event_fanout_ms'
  | 'event_bytes';
type JourneyID = 'J26' | 'J27' | 'J28' | 'J29' | 'J30';
const responsivenessBudgets = JSON.parse(
  readFileSync(join(__dirname, 'responsiveness-budgets.json'), 'utf8')
) as Record<JourneyID, Record<ResponsivenessMetric, number>>;

export function journeyResponsivenessBudget(
  journeyId: JourneyID,
  metric: ResponsivenessMetric
): number {
  const budget = responsivenessBudgets[journeyId]?.[metric];
  if (!Number.isFinite(budget) || budget <= 0)
    throw new Error(`Missing positive ${metric} budget for ${journeyId}`);
  return budget;
}

const runtimeIdentityByJourney = new Map<string, JourneyRuntimeIdentity>();

const fallbackEvidenceRunId = randomUUID();

function writeJourneyEvidence(
  record: Omit<JourneyEvidenceRecord, 'schema_version' | 'run_id' | 'recorded_at'>
): void {
  const evidenceFile = process.env.BAS_JOURNEY_EVIDENCE_FILE;
  if (!evidenceFile)
    throw new Error(
      'BAS_JOURNEY_EVIDENCE_FILE is unset; journey global setup must own evidence output'
    );
  const complete: JourneyEvidenceRecord = {
    schema_version: 1,
    run_id: process.env.BAS_JOURNEY_RUN_ID ?? fallbackEvidenceRunId,
    recorded_at: new Date().toISOString(),
    ...record,
    runtime_identity: record.runtime_identity ?? runtimeIdentityByJourney.get(record.journey_id),
  };
  appendFileSync(evidenceFile, `${JSON.stringify(complete)}\n`, { encoding: 'utf8', mode: 0o600 });
}

/** Record the exact browser launched by the UI journey and BAS driver's reported release. */
export async function recordJourneyRuntimeIdentity(
  journeyId: string,
  browserVersion: string
): Promise<void> {
  if (!browserVersion.trim()) throw new Error('Journey browser did not report a version');
  const health = await driver<HealthResponse>('/health');
  if (!health.driver_version?.trim())
    throw new Error('BAS driver health did not report driver_version');
  const identity: JourneyRuntimeIdentity = {
    browser_version: browserVersion,
    driver_version: health.driver_version,
    driver_browser_version: health.browser?.version,
  };
  runtimeIdentityByJourney.set(journeyId, identity);
  writeJourneyEvidence({
    journey_id: journeyId,
    event: 'runtime.identity',
    kind: 'receipt',
    outcome: 'ok',
    runtime_identity: identity,
  });
}

/** Time one BAS UI journey milestone with a monotonic clock and retain its result identity. */
export async function measureJourney<T>(
  journeyId: string,
  event: string,
  operation: () => Promise<T>,
  facts?: (result: T) => JourneyEvidenceFacts,
  options?: { budgetMetrics?: ResponsivenessMetric[] }
): Promise<T> {
  const started = process.hrtime.bigint();
  let result: T;
  try {
    result = await operation();
  } catch (error) {
    writeJourneyEvidence({
      journey_id: journeyId,
      event,
      kind: 'timing',
      outcome: 'failed',
      elapsed_ms: Number(process.hrtime.bigint() - started) / 1_000_000,
      error_type: error instanceof Error ? error.name : 'unknown',
    });
    throw error;
  }
  const elapsedMs = Number(process.hrtime.bigint() - started) / 1_000_000;
  const budgetFacts: JourneyEvidenceFacts = {};
  let budgetExceeded = false;
  for (const metric of options?.budgetMetrics ?? []) {
    const budgetMs = journeyResponsivenessBudget(journeyId as JourneyID, metric);
    const met = elapsedMs <= budgetMs;
    budgetFacts[`${metric}_budget_ms`] = budgetMs;
    budgetFacts[`${metric}_budget_met`] = met;
    budgetExceeded ||= !met;
  }
  writeJourneyEvidence({
    journey_id: journeyId,
    event,
    kind: 'timing',
    outcome: budgetExceeded ? 'failed' : 'ok',
    elapsed_ms: elapsedMs,
    facts: { ...facts?.(result), ...budgetFacts },
    error_type: budgetExceeded ? 'ResponsivenessBudgetExceeded' : undefined,
  });
  if (budgetExceeded)
    throw new Error(`${event} exceeded its responsiveness budget (${elapsedMs.toFixed(1)} ms)`);
  return result;
}

/** Record a measurement whose clock is shared with an adjacent user action. */
export function recordJourneyMeasurement(
  journeyId: JourneyID,
  event: string,
  elapsedMs: number,
  metric: ResponsivenessMetric,
  facts: JourneyEvidenceFacts = {}
): void {
  const budget = journeyResponsivenessBudget(journeyId, metric);
  const met = elapsedMs <= budget;
  writeJourneyEvidence({
    journey_id: journeyId,
    event,
    kind: 'timing',
    outcome: met ? 'ok' : 'failed',
    elapsed_ms: elapsedMs,
    facts: { ...facts, [`${metric}_budget_ms`]: budget, [`${metric}_budget_met`]: met },
    error_type: met ? undefined : 'ResponsivenessBudgetExceeded',
  });
  if (!met)
    throw new Error(`${event} exceeded its responsiveness budget (${elapsedMs.toFixed(1)} ms)`);
}

export function recordNotApplicableResponsivenessMetric(
  journeyId: JourneyID,
  event: string,
  metric: ResponsivenessMetric,
  reason: string
): void {
  recordJourneyReceipt(journeyId, event, {
    responsiveness_metric: metric,
    sampled: false,
    reason,
    budget: journeyResponsivenessBudget(journeyId, metric),
  });
}

export type JourneySocketSnapshot = {
  received_frames: number;
  received_bytes: number;
  sent_frames: number;
  sent_bytes: number;
  last_received_at_ns: bigint | null;
};
type MutableSocketStats = { sessionId: string; snapshot: JourneySocketSnapshot };

/** Capture counts and payload sizes only; never persist browser event contents. */
export function trackJourneySocketTraffic(page: Page): {
  snapshot(sessionId?: string): JourneySocketSnapshot;
} {
  const sockets: MutableSocketStats[] = [];
  page.on('websocket', (socket) => {
    const pathname = new URL(socket.url()).pathname;
    const match = pathname.match(/\/ws\/sessions\/([^/]+)$/);
    const stats: MutableSocketStats = {
      sessionId: match ? decodeURIComponent(match[1] ?? '') : '',
      snapshot: {
        received_frames: 0,
        received_bytes: 0,
        sent_frames: 0,
        sent_bytes: 0,
        last_received_at_ns: null,
      },
    };
    sockets.push(stats);
    socket.on('framereceived', (payload) => {
      stats.snapshot.received_frames += 1;
      stats.snapshot.received_bytes += Buffer.byteLength(payload);
      stats.snapshot.last_received_at_ns = process.hrtime.bigint();
    });
    socket.on('framesent', (payload) => {
      stats.snapshot.sent_frames += 1;
      stats.snapshot.sent_bytes += Buffer.byteLength(payload);
    });
  });
  return {
    snapshot(sessionId) {
      const selected = sockets.filter((socket) => !sessionId || socket.sessionId === sessionId);
      return selected.reduce<JourneySocketSnapshot>(
        (total, socket) => ({
          received_frames: total.received_frames + socket.snapshot.received_frames,
          received_bytes: total.received_bytes + socket.snapshot.received_bytes,
          sent_frames: total.sent_frames + socket.snapshot.sent_frames,
          sent_bytes: total.sent_bytes + socket.snapshot.sent_bytes,
          last_received_at_ns:
            socket.snapshot.last_received_at_ns !== null &&
            (total.last_received_at_ns === null ||
              socket.snapshot.last_received_at_ns > total.last_received_at_ns)
              ? socket.snapshot.last_received_at_ns
              : total.last_received_at_ns,
        }),
        {
          received_frames: 0,
          received_bytes: 0,
          sent_frames: 0,
          sent_bytes: 0,
          last_received_at_ns: null,
        }
      );
    },
  };
}

/** A low-cost sampled canvas fingerprint used to detect a newly painted frame. */
export async function canvasFrameSignature(page: Page): Promise<string> {
  return page
    .locator('canvas:visible')
    .first()
    .evaluate((element) => {
      const canvas = element as HTMLCanvasElement;
      const context = canvas.getContext('2d');
      if (!context || canvas.width <= 0 || canvas.height <= 0) return '';
      const points = [
        [0.15, 0.15],
        [0.5, 0.15],
        [0.85, 0.15],
        [0.15, 0.5],
        [0.5, 0.5],
        [0.85, 0.5],
        [0.15, 0.85],
        [0.5, 0.85],
        [0.85, 0.85],
      ];
      return (
        `${canvas.width}x${canvas.height}:` +
        points
          .map(([x, y]) => {
            const pixel = context.getImageData(
              Math.floor(canvas.width * x),
              Math.floor(canvas.height * y),
              1,
              1
            ).data;
            return `${pixel[0]},${pixel[1]},${pixel[2]},${pixel[3]}`;
          })
          .join(';')
      );
    });
}

export async function waitForCanvasFrameChange(
  page: Page,
  before: string,
  timeoutMs: number
): Promise<string> {
  const deadline = Date.now() + timeoutMs;
  while (Date.now() < deadline) {
    const current = await canvasFrameSignature(page);
    if (current && current !== before) return current;
    await new Promise((resolve) => setTimeout(resolve, 50));
  }
  throw new Error(`No visible canvas frame change within ${timeoutMs} ms`);
}

export function elapsedSinceMonotonic(startedAt: bigint, completedAt: bigint): number {
  return Number(completedAt - startedAt) / 1_000_000;
}

/** Persist an exact browser/execution/profile receipt without page content or credentials. */
export function recordJourneyReceipt(
  journeyId: string,
  event: string,
  facts: JourneyEvidenceFacts
): void {
  writeJourneyEvidence({ journey_id: journeyId, event, kind: 'receipt', outcome: 'ok', facts });
}

type JourneyCleanup = { label: string; run: () => Promise<unknown> };

/** Own test-created resources and report every teardown failure after all attempts. */
export class JourneyResourceScope {
  private readonly cleanups: JourneyCleanup[] = [];

  add(label: string, cleanup: () => Promise<unknown>): void {
    this.cleanups.push({ label, run: cleanup });
  }

  async run<T>(body: (resources: this) => Promise<T>): Promise<T> {
    let value!: T;
    let bodyFailed = false;
    let bodyError: unknown;
    try {
      value = await body(this);
    } catch (error) {
      bodyFailed = true;
      bodyError = error;
    }

    const failures: Error[] = [];
    for (const cleanup of this.cleanups.splice(0).reverse()) {
      try {
        await cleanup.run();
      } catch (error) {
        failures.push(new Error(`Could not ${cleanup.label}`, { cause: error }));
      }
    }

    if (bodyFailed && failures.length > 0) {
      throw new AggregateError(
        [bodyError, ...failures],
        'Journey body and resource cleanup failed'
      );
    }
    if (bodyFailed) throw bodyError;
    if (failures.length > 0)
      throw new AggregateError(
        failures,
        `Journey resource cleanup failed: ${failures.map(({ message }) => message).join('; ')}`
      );
    return value;
  }
}

/** Run a journey body and always attempt every registered resource cleanup. */
export const withJourneyResources = <T>(
  body: (resources: JourneyResourceScope) => Promise<T>
): Promise<T> => new JourneyResourceScope().run(body);

type Box = { left: number; top: number; width: number; height: number };
const layout = JSON.parse(
  readFileSync(join(__dirname, '../../../fixtures/journey-site/layout.json'), 'utf8')
) as Record<'counter' | 'input' | 'paste' | 'compose' | 'challenge', Box>;
/** Viewport centre of a fixture control, for pointer input. */
export const target = (name: keyof typeof layout): { x: number; y: number } => {
  const box = layout[name];
  return { x: box.left + box.width / 2, y: box.top + box.height / 2 };
};

function required(
  name: 'BAS_API_URL' | 'BAS_UI_URL' | 'BAS_DRIVER_URL' | 'BAS_FIXTURE_ORIGIN'
): string {
  const value = process.env[name];
  if (!value) throw new Error(`${name} is unset; run journeys with jest.journeys.config.js`);
  return value;
}
export const env = {
  get api() {
    return required('BAS_API_URL');
  },
  get ui() {
    return required('BAS_UI_URL');
  },
  get driver() {
    return required('BAS_DRIVER_URL');
  },
  get fixture() {
    return required('BAS_FIXTURE_ORIGIN');
  },
};

async function send(url: string, body?: unknown, expectedStatus?: number): Promise<unknown> {
  const response = await fetch(url, {
    method: body === undefined ? 'GET' : 'POST',
    headers: { 'Content-Type': 'application/json', 'Connect-Protocol-Version': '1' },
    body: body === undefined ? undefined : JSON.stringify(body),
    signal: AbortSignal.timeout(TIMEOUT_MS.request),
  });
  const text = await response.text();
  if (expectedStatus === undefined ? !response.ok : response.status !== expectedStatus) {
    throw new Error(`${url} returned ${response.status}: ${text.slice(0, 500)}`);
  }
  return text ? JSON.parse(text) : {};
}

/** Call a BAS Connect method by its generated descriptor. */
export async function rpc<I extends DescMessage, O extends DescMessage>(
  method: DescMethodUnary<I, O>,
  request: MessageInitShape<I>
): Promise<MessageShape<O>> {
  const body = await send(
    `${env.api}/${method.parent.typeName}/${method.name}`,
    toJson(method.input, create(method.input, request))
  );
  return fromJson(method.output, body as JsonValue, { ignoreUnknownFields: true });
}

/** Call a BAS REST route whose body is a generated message. */
export async function rest<O extends DescMessage>(
  schema: O,
  path: string,
  body?: unknown
): Promise<MessageShape<O>> {
  return fromJson(schema, (await send(`${env.api}/api/v1${path}`, body)) as JsonValue, {
    ignoreUnknownFields: true,
  });
}

/** Call a BAS REST route whose body has no generated message. */
export const apiRoute = <T>(path: string, body?: unknown): Promise<T> =>
  send(`${env.api}/api/v1${path}`, body) as Promise<T>;

/** Call a driver route; driver bodies are the TypeScript types in src/types. */
export const driver = <T>(path: string, body?: unknown, expectedStatus?: number): Promise<T> =>
  send(`${env.driver}${path}`, body, expectedStatus) as Promise<T>;

/** A typed action, built by the shared instruction factory. */
export const action = (type: string, params: Record<string, unknown>): ActionDefinition =>
  type === 'shortcut'
    ? create(ActionDefinitionSchema, {
        type: 21,
        params: {
          case: 'shortcut',
          value: create(ShortcutParamsSchema, {
            shortcut: String(params.shortcut),
            selector: params.selector as string | undefined,
          }),
        },
      })
    : type === 'keyboard'
      ? create(ActionDefinitionSchema, {
          type: 9,
          params: {
            case: 'keyboard',
            value: create(KeyboardParamsSchema, {
              key: params.key as string | undefined,
              keys: (params.keys as string[] | undefined) ?? [],
            }),
          },
        })
      : type === 'drag-drop'
        ? create(ActionDefinitionSchema, {
            type: 22,
            params: {
              case: 'dragDrop',
              value: create(DragDropParamsSchema, {
                sourceSelector: String(params.source),
                targetSelector: String(params.target),
                steps: 3,
              }),
            },
          })
        : (createTypedInstruction(type, params).action ?? create(ActionDefinitionSchema));

/** Create a workflow through BAS's project and workflow editor UI, then return its route ID. */
export async function createWorkflowThroughEditor(
  page: Page,
  projectId: string,
  definitionJson: string
): Promise<string> {
  await page.goto(`${env.ui}/projects/${projectId}`, {
    waitUntil: 'domcontentloaded',
    timeout: 30000,
  });
  await page.getByTestId('app-ready').waitFor({ state: 'attached', timeout: 30000 });
  await page.getByTestId('workflows-card-view-toggle').click();
  await page.getByTestId('new-workflow-button').click();
  await page.getByTestId('workflow-creation-visual-builder').click();
  await page.waitForURL((url) => url.pathname.startsWith(`/projects/${projectId}/workflows/`), {
    timeout: 30000,
  });

  await page.getByTestId('workflow-builder-canvas').waitFor({ state: 'visible', timeout: 30000 });
  await page.getByTestId('workflow-builder-code-mode-button').click();
  const editor = page.locator('[data-testid="workflow-builder-code-view"] .monaco-editor');
  await editor.waitFor({ state: 'visible', timeout: 30000 });
  await editor.click();
  await page.keyboard.press('Control+A');
  await page.keyboard.insertText(definitionJson);
  await page.getByTestId('workflow-builder-code-apply-button').click();
  await expect(page.getByTestId('workflow-builder-code-validation')).toContainText('Validated', {
    timeout: 30000,
  });

  const workflowId = new URL(page.url()).pathname.split('/').at(-1) ?? '';
  if (!workflowId) throw new Error('BAS workflow editor route did not include a workflow ID');
  const parsedDefinition = JSON.parse(definitionJson) as { nodes?: unknown[] };
  const expectedNodeCount = parsedDefinition.nodes?.length ?? 0;
  await expect
    .poll(
      async () => {
        const { workflow } = await rpc(WorkflowsService.method.getWorkflow, { workflowId });
        return workflow?.flowDefinition?.nodes?.length ?? -1;
      },
      { timeout: 30000 }
    )
    .toBe(expectedNodeCount);
  return workflowId;
}

export type Lease = {
  sessionId: string;
  leaseId: string;
  executionId: string;
  sequence: number;
  settled?: boolean;
};
export const ownership = (lease: Lease): CloseSessionRequest => ({
  execution_id: lease.executionId,
  lease_id: lease.leaseId,
});

export async function openLease(
  options: {
    storageState?: SessionSpec['storage_state'];
    reuseMode?: SessionSpec['reuse_mode'];
    labels?: SessionSpec['labels'];
    viewport?: SessionSpec['viewport'];
    sessionProfileVersion?: SessionSpec['session_profile_version'];
    locale?: SessionSpec['locale'];
    timezone?: SessionSpec['timezone'];
    userAgent?: SessionSpec['user_agent'];
    browserProfile?: SessionSpec['browser_profile'];
  } = {}
): Promise<Lease> {
  const request: StartSessionRequest = {
    execution_id: randomUUID(),
    workflow_id: randomUUID(),
    session_options: {
      viewport: options.viewport ?? VIEWPORT,
      reuse_mode: options.reuseMode ?? 'fresh',
      frame_scale: 'css',
      base_url: env.fixture,
      storage_state: options.storageState,
      labels: options.labels,
      session_profile_version: options.sessionProfileVersion,
      locale: options.locale,
      timezone: options.timezone,
      user_agent: options.userAgent,
      browser_profile: options.browserProfile,
    },
  };
  const started = await driver<StartSessionResponse>('/session/start', request);
  if (!started.session_id || !started.lease_id)
    throw new Error(`Driver returned no lease: ${JSON.stringify(started)}`);
  return {
    sessionId: started.session_id,
    leaseId: started.lease_id,
    executionId: request.execution_id,
    sequence: 0,
  };
}

export async function runAction(lease: Lease, definition: ActionDefinition): Promise<StepOutcome> {
  const sequence = ++lease.sequence;
  const body = await driver<JsonValue>(`/session/${lease.sessionId}/run`, {
    ...ownership(lease),
    operation_sequence: sequence,
    invocation_id: `${lease.executionId}:${sequence}`,
    attempt: 1,
    instruction: {
      index: sequence - 1,
      node_id: `journey-${sequence}`,
      action: toJson(ActionDefinitionSchema, definition),
    },
  });
  // Driver responses include common JsonValue fields as ordinary JSON values;
  // its REST envelope isn't protobuf JSON's tagged JsonValue representation.
  const outcome = body as unknown as StepOutcome;
  if (!outcome.success)
    throw new Error(`Driver rejected ${definition.params.case}: ${JSON.stringify(body)}`);
  return outcome;
}

export const closeLease = async (lease: Lease): Promise<unknown> => {
  const result = await driver(`/session/${lease.sessionId}/close`, ownership(lease));
  lease.settled = true;
  return result;
};
export const releaseLease = async (lease: Lease): Promise<unknown> => {
  const result = await driver(`/session/${lease.sessionId}/release`, ownership(lease));
  lease.settled = true;
  return result;
};

export function registerLeaseCleanup(resources: JourneyResourceScope, lease: Lease): void {
  resources.add('close driver lease', async () => {
    if (!lease.settled) await closeLease(lease);
  });
}

export function registerRecordingCleanup(resources: JourneyResourceScope, lease: Lease): void {
  resources.add('stop and acknowledge recording', () => stopAndAcknowledgeRecording(lease));
}

/** Stop capture and acknowledge the resulting durable timeline before closing its lease. */
export async function stopAndAcknowledgeRecording(lease: Lease): Promise<void> {
  const record = `/session/${lease.sessionId}/record`;
  await driver(`${record}/stop`, ownership(lease));
  const { entries } = await driver<{ entries: Array<{ id?: string }> }>(`${record}/actions`);
  const entry_ids = entries.flatMap(({ id }) => (id ? [id] : []));
  if (entry_ids.length > 0)
    await driver(`${record}/actions/ack`, { ...ownership(lease), entry_ids });
}

/** Run `body` with open leases and close every one afterwards, newest first. */
export async function withLeases<T>(
  body: (open: typeof openLease, resources: JourneyResourceScope) => Promise<T>
): Promise<T> {
  return withJourneyResources(async (resources) =>
    body(async (options) => {
      const lease = await openLease(options);
      registerLeaseCleanup(resources, lease);
      return lease;
    }, resources)
  );
}

/** A one-node observer workflow for ExecuteAdhocWorkflow. */
export function adhoc(
  name: string,
  nodeAction: ActionDefinition,
  waitForCompletion: boolean,
  resilience?: { maxAttempts: number; delayMs: number }
): MessageInitShape<typeof WorkflowsService.method.executeAdhocWorkflow.input> {
  return {
    metadata: { name: `${name} ${randomUUID()}` },
    flowDefinition: {
      metadata: { name, executionMode: ExecutionMode.OBSERVER },
      settings: { headless: true, timeoutMs: 20000 },
      nodes: [
        {
          id: randomUUID(),
          action: nodeAction,
          executionSettings: { timeoutMs: 20000, resilience },
        },
      ],
    },
    waitForCompletion,
    parameters: { headless: true, viewportWidth: VIEWPORT.width, viewportHeight: VIEWPORT.height },
  };
}

export type FixtureEffect = { sequence: number; context: string; held?: boolean };
export type FixtureState = {
  effects: FixtureEffect[];
  released: Array<{ sequence: number }>;
  inputs: Array<{ type: string; value: string; context: string }>;
  scrolls: Array<{ y: number }>;
  fingerprints: Array<Record<string, unknown>>;
  challenges: Array<{ context: string; completedBy: string }>;
  challengeVisits: Array<{ context: string }>;
  profileObservations: Array<{ context: string; localStorage: string | null }>;
  authObservations: Array<{ context: string; hasSyntheticAuth: boolean }>;
  retryAttempts: number;
};

export async function fixtureState(): Promise<FixtureState> {
  return (await fetch(new URL('/journey-state', env.fixture))).json() as Promise<FixtureState>;
}

/** Long-poll the fixture until more than `after` records of `kind` exist. */
export async function fixtureAfter(
  kind:
    | 'effect'
    | 'release'
    | 'input'
    | 'scroll'
    | 'fingerprint'
    | 'challenge'
    | 'challenge-visit'
    | 'profile-observation'
    | 'auth-observation',
  after: number
): Promise<FixtureState> {
  const url = new URL('/journey-wait', env.fixture);
  url.search = new URLSearchParams({
    kind,
    after: String(after),
    timeout_ms: String(TIMEOUT_MS.wait),
  }).toString();
  const response = await fetch(url, { signal: AbortSignal.timeout(TIMEOUT_MS.wait + 1000) });
  if (!response.ok) throw new Error(`Fixture saw no ${kind} after ${after} (${response.status})`);
  return response.json() as Promise<FixtureState>;
}

/** Wait until the fixture observes a matching input, without sleeping. */
export async function fixtureInput(
  match: (input: FixtureState['inputs'][number]) => boolean
): Promise<FixtureState['inputs'][number]> {
  let state = await fixtureState();
  for (;;) {
    const found = state.inputs.find(match);
    if (found) return found;
    state = await fixtureAfter('input', state.inputs.length).catch((error: Error) => {
      throw new Error(`${error.message}; observed ${JSON.stringify(state.inputs.slice(-8))}`);
    });
  }
}
