/**
 * Shared support for the preservation journeys. global-setup.mjs resolves the
 * goal's shadow BAS and starts the fixture site; this module is the only
 * reader of that environment and the only BAS/driver/fixture client.
 */
import { randomUUID } from 'node:crypto';
import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import {
  create, fromJson, toJson,
  type DescMessage, type DescMethodUnary, type JsonValue, type MessageInitShape, type MessageShape,
} from '@bufbuild/protobuf';
import { ActionDefinitionSchema, type ActionDefinition } from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import { KeyboardParamsSchema, ShortcutParamsSchema, DragDropParamsSchema } from '@vrooli/proto-types/browser-automation-studio/v1/actions/action_pb';
import type { StepOutcome } from '@vrooli/proto-types/browser-automation-studio/v1/execution/driver_pb';
import { ExecutionMode } from '@vrooli/proto-types/browser-automation-studio/v1/workflows/definition_pb';
import { WorkflowsService } from '@vrooli/proto-types/browser-automation-studio/v1/api/service_pb';
import { createTypedInstruction } from '../helpers/instruction-factory';
import type { CloseSessionRequest, SessionSpec, StartSessionRequest, StartSessionResponse } from '../../src/types';

export const TIMEOUT_MS = { request: 30000, wait: 10000 } as const;
export const VIEWPORT = { width: 640, height: 480 } as const;

type Box = { left: number; top: number; width: number; height: number };
const layout = JSON.parse(readFileSync(join(__dirname, '../../../fixtures/journey-site/layout.json'), 'utf8')) as Record<'counter' | 'input' | 'paste' | 'compose', Box>;
/** Viewport centre of a fixture control, for pointer input. */
export const target = (name: keyof typeof layout): { x: number; y: number } => {
  const box = layout[name];
  return { x: box.left + box.width / 2, y: box.top + box.height / 2 };
};

function required(name: 'BAS_API_URL' | 'BAS_DRIVER_URL' | 'BAS_FIXTURE_ORIGIN'): string {
  const value = process.env[name];
  if (!value) throw new Error(`${name} is unset; run journeys with jest.journeys.config.js`);
  return value;
}
export const env = { api: required('BAS_API_URL'), driver: required('BAS_DRIVER_URL'), fixture: required('BAS_FIXTURE_ORIGIN') };

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
  method: DescMethodUnary<I, O>, request: MessageInitShape<I>,
): Promise<MessageShape<O>> {
  const body = await send(`${env.api}/${method.parent.typeName}/${method.name}`, toJson(method.input, create(method.input, request)));
  return fromJson(method.output, body as JsonValue, { ignoreUnknownFields: true });
}

/** Call a BAS REST route whose body is a generated message. */
export async function rest<O extends DescMessage>(schema: O, path: string, body?: unknown): Promise<MessageShape<O>> {
  return fromJson(schema, await send(`${env.api}/api/v1${path}`, body) as JsonValue, { ignoreUnknownFields: true });
}

/** Call a BAS REST route whose body has no generated message. */
export const apiRoute = <T>(path: string, body?: unknown): Promise<T> => send(`${env.api}/api/v1${path}`, body) as Promise<T>;

/** Call a driver route; driver bodies are the TypeScript types in src/types. */
export const driver = <T>(path: string, body?: unknown, expectedStatus?: number): Promise<T> =>
  send(`${env.driver}${path}`, body, expectedStatus) as Promise<T>;

/** A typed action, built by the shared instruction factory. */
export const action = (type: string, params: Record<string, unknown>): ActionDefinition =>
  type === 'shortcut'
    ? create(ActionDefinitionSchema, { type: 21, params: { case: 'shortcut', value: create(ShortcutParamsSchema, { shortcut: String(params.shortcut), selector: params.selector as string | undefined }) } })
    : type === 'keyboard'
      ? create(ActionDefinitionSchema, { type: 9, params: { case: 'keyboard', value: create(KeyboardParamsSchema, { key: params.key as string | undefined, keys: (params.keys as string[] | undefined) ?? [] }) } })
      : type === 'drag-drop'
        ? create(ActionDefinitionSchema, { type: 22, params: { case: 'dragDrop', value: create(DragDropParamsSchema, { sourceSelector: String(params.source), targetSelector: String(params.target), steps: 3 }) } })
        : createTypedInstruction(type, params).action ?? create(ActionDefinitionSchema);

export type Lease = { sessionId: string; leaseId: string; executionId: string; sequence: number };
export const ownership = (lease: Lease): CloseSessionRequest => ({ execution_id: lease.executionId, lease_id: lease.leaseId });

export async function openLease(options: { storageState?: SessionSpec['storage_state']; reuseMode?: SessionSpec['reuse_mode']; labels?: SessionSpec['labels'] } = {}): Promise<Lease> {
  const request: StartSessionRequest = {
    execution_id: randomUUID(), workflow_id: randomUUID(),
    session_options: {
      viewport: VIEWPORT,
      reuse_mode: options.reuseMode ?? 'fresh',
      frame_scale: 'css',
      base_url: env.fixture,
      storage_state: options.storageState,
      labels: options.labels,
    },
  };
  const started = await driver<StartSessionResponse>('/session/start', request);
  if (!started.session_id || !started.lease_id) throw new Error(`Driver returned no lease: ${JSON.stringify(started)}`);
  return { sessionId: started.session_id, leaseId: started.lease_id, executionId: request.execution_id, sequence: 0 };
}

export async function runAction(lease: Lease, definition: ActionDefinition): Promise<StepOutcome> {
  const sequence = ++lease.sequence;
  const body = await driver<JsonValue>(`/session/${lease.sessionId}/run`, {
    ...ownership(lease), operation_sequence: sequence, invocation_id: `${lease.executionId}:${sequence}`, attempt: 1,
    instruction: { index: sequence - 1, node_id: `journey-${sequence}`, action: toJson(ActionDefinitionSchema, definition) },
  });
  // Driver responses include common JsonValue fields as ordinary JSON values;
  // its REST envelope isn't protobuf JSON's tagged JsonValue representation.
  const outcome = body as StepOutcome;
  if (!outcome.success) throw new Error(`Driver rejected ${definition.params.case}: ${JSON.stringify(body)}`);
  return outcome;
}

export const closeLease = (lease: Lease): Promise<unknown> => driver(`/session/${lease.sessionId}/close`, ownership(lease));
export const releaseLease = (lease: Lease): Promise<unknown> => driver(`/session/${lease.sessionId}/release`, ownership(lease));

/** Run `body` with open leases and close every one afterwards, newest first. */
export async function withLeases<T>(body: (open: typeof openLease) => Promise<T>): Promise<T> {
  const leases: Lease[] = [];
  try {
    return await body(async (options) => { const lease = await openLease(options); leases.push(lease); return lease; });
  } finally {
    for (const lease of leases.reverse()) await closeLease(lease).catch(() => undefined);
  }
}

/** A one-node observer workflow for ExecuteAdhocWorkflow. */
export function adhoc(name: string, nodeAction: ActionDefinition, waitForCompletion: boolean, resilience?: { maxAttempts: number; delayMs: number }):
  MessageInitShape<typeof WorkflowsService.method.executeAdhocWorkflow.input> {
  return {
    metadata: { name: `${name} ${randomUUID()}` },
    flowDefinition: {
      metadata: { name, executionMode: ExecutionMode.OBSERVER },
      settings: { headless: true, timeoutMs: 20000 },
      nodes: [{ id: randomUUID(), action: nodeAction, executionSettings: { timeoutMs: 20000, resilience } }],
    },
    waitForCompletion,
    parameters: { headless: true, viewportWidth: VIEWPORT.width, viewportHeight: VIEWPORT.height },
  };
}

export type FixtureEffect = { sequence: number; context: string; held?: boolean };
export type FixtureState = { effects: FixtureEffect[]; released: Array<{ sequence: number }>; inputs: Array<{ type: string; value: string; context: string }>; scrolls: Array<{ y: number }>; fingerprints: Array<Record<string, unknown>>; retryAttempts: number };

export async function fixtureState(): Promise<FixtureState> {
  return (await fetch(new URL('/journey-state', env.fixture))).json() as Promise<FixtureState>;
}

/** Long-poll the fixture until more than `after` records of `kind` exist. */
export async function fixtureAfter(kind: 'effect' | 'release' | 'input' | 'scroll' | 'fingerprint', after: number): Promise<FixtureState> {
  const url = new URL('/journey-wait', env.fixture);
  url.search = new URLSearchParams({ kind, after: String(after), timeout_ms: String(TIMEOUT_MS.wait) }).toString();
  const response = await fetch(url, { signal: AbortSignal.timeout(TIMEOUT_MS.wait + 1000) });
  if (!response.ok) throw new Error(`Fixture saw no ${kind} after ${after} (${response.status})`);
  return response.json() as Promise<FixtureState>;
}

/** Wait until the fixture observes a matching input, without sleeping. */
export async function fixtureInput(match: (input: FixtureState['inputs'][number]) => boolean): Promise<FixtureState['inputs'][number]> {
  let state = await fixtureState();
  for (;;) {
    const found = state.inputs.find(match);
    if (found) return found;
    state = await fixtureAfter('input', state.inputs.length).catch((error: Error) => {
      throw new Error(`${error.message}; observed ${JSON.stringify(state.inputs.slice(-8))}`);
    });
  }
}
