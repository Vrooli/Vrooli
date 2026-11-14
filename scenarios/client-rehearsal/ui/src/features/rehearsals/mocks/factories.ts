import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  RehearsalSchema,
  TraceEventSchema,
  type Rehearsal,
  type TraceEvent,
} from "@vrooli/proto-types/client-rehearsal/v1/rehearsals/rehearsals_pb";

export type { Rehearsal, TraceEvent };

export const makeTraceEvent = (overrides: MessageInitShape<typeof TraceEventSchema> = {}): TraceEvent =>
  create(TraceEventSchema, {
    sequence: 1,
    operation: "crm.upsertContact",
    requestSummary: "Contact payload with synthetic credentials",
    responseSummary: "Synthetic CRM accepted contact",
    status: "succeeded",
    ...overrides,
  });

export const makeRehearsal = (overrides: MessageInitShape<typeof RehearsalSchema> = {}): Rehearsal =>
  create(RehearsalSchema, {
    id: "rehearsal-1",
    title: "Lead intake rehearsal",
    sourceFormat: "n8n-json",
    sourceSha256: "workflow-hash",
    credentialReplacements: 2,
    simulatorProfile: "synthetic-default",
    state: 1,
    trace: [makeTraceEvent()],
    faultSummary: "A seeded CRM timeout was recovered.",
    repairSummary: "Retry completed after the simulated fault.",
    handoffMarkdown: "# Handoff\n\nSynthetic CRM contact was upserted and the sheet row was appended.",
    reviewNote: "Reviewed synthetic run evidence in the rehearsal workspace.",
    ...overrides,
  });
