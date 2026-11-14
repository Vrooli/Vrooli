import { z } from 'zod'
import { RunEventType } from '@vrooli/proto-types/agent-manager/v1/domain/types_pb'
import { parseOrThrow } from './safeParse'

const payload = z.record(z.string(), z.unknown()).nullish()
const discriminator = z.union([z.string(), z.number().int()]).nullish()
const sequence = z.union([z.number(), z.string().regex(/^\d+$/)])
  .refine((value) => Number.isSafeInteger(Number(value)) && Number(value) >= 0, 'Expected a safe nonnegative sequence')

// PM's Connect JsonResponse validates the Value envelope only. Keep the
// open AM event JSON here; protobuf decoding would discard future JSON fields.
const WireEventSchema = z.looseObject({
  id: z.string().min(1),
  run_id: z.string().min(1).optional(),
  runId: z.string().min(1).optional(),
  sequence: sequence.nullish(),
  event_type: discriminator,
  eventType: discriminator,
  timestamp: z.string(),
  message: payload, message_deleted: payload, messageDeleted: payload,
  tool_call: payload, toolCall: payload, tool_result: payload, toolResult: payload,
  status: payload, metric: payload, log: payload, error: payload,
  artifact: payload, progress: payload, cost: payload,
  rate_limit: payload, rateLimit: payload, compaction: payload,
  goal_status_changed: payload, goalStatusChanged: payload, data: payload,
}).refine((event) => !!(event.run_id ?? event.runId), { path: ['runId'], message: 'Run identity is required' })

const WireEventsSchema = z.union([
  z.array(WireEventSchema),
  z.looseObject({ events: z.array(WireEventSchema).nullish() }),
])

export interface RunEvent {
  id: string
  runId: string
  sequence: number
  // An open discriminator: future events remain visible in GenericEvent.
  eventType: string
  timestamp: string
  data: Record<string, unknown>
  payloadType?: string
  raw?: Record<string, unknown>
}

function eventType(value: string | number | null | undefined): string {
  if (value == null || value === '') return 'unspecified'
  const name = typeof value === 'number' ? (RunEventType[value] ?? String(value)) : value
  return name.replace(/^RUN_EVENT_TYPE_/, '').toLowerCase()
}

function normalizePayload(data: Record<string, unknown>): Record<string, unknown> {
  // Only alias fields consumed by existing presenters. Nested objects and all
  // unknown keys stay untouched, including tool inputs with case-sensitive keys.
  const result = { ...data }
  for (const [camel, snake] of [
    ['toolName', 'tool_name'], ['newStatus', 'new_status'],
    ['oldStatus', 'old_status'], ['fromStatus', 'from_status'],
  ] as const) {
    if (!(snake in result) && camel in result) result[snake] = result[camel]
  }
  return result
}

export function decodeRunEvents(input: unknown): RunEvent[] {
  const page = parseOrThrow(WireEventsSchema, input, 'run events')
  const events = Array.isArray(page) ? page : (page.events ?? [])
  return events.map((raw) => {
    const type = eventType(raw.event_type ?? raw.eventType)
    const camelType = type.replace(/_([a-z])/g, (_, letter: string) => letter.toUpperCase())
    const knownType = Object.values(RunEventType).includes(type.toUpperCase())
    const matching = knownType ? (raw[type] ?? raw[camelType]) : undefined
    const typedPayload = payload.safeParse(matching)
    const variants = [
      [type, typedPayload.success ? typedPayload.data : undefined],
      ['progress', raw.progress], ['cost', raw.cost],
      ['rate_limit', raw.rate_limit ?? raw.rateLimit], ['data', raw.data],
    ] as const
    const selected = variants.find(([, data]) => data != null)
    const data = selected?.[1]
    return {
      id: raw.id, runId: raw.run_id ?? raw.runId ?? '', sequence: Number(raw.sequence ?? 0),
      eventType: type, timestamp: raw.timestamp,
      // Unspecified/new events retain every received field, never a fake log,
      // success, or completion. Data already dropped by AM cannot be recovered.
      data: data ? normalizePayload(data) : raw,
      payloadType: selected?.[0],
      raw,
    }
  })
}
