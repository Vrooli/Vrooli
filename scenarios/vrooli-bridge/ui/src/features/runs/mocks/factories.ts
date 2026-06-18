/**
 * Test factories for the runs domain. Live with the feature (not the
 * cross-domain test-utils) so deleting the feature folder takes its doubles
 * along. `makeRun()` returns a TERMINAL passed run by default; tests override
 * the fields the case is about. `makeRunEvent()` builds a single channel event.
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import {
  RunSchema,
  RunStatus,
  type Run,
} from "@vrooli/proto-types/vrooli-bridge/v1/runs/runs_pb";
import {
  RunEventSchema,
  RunEventKind,
  type RunEvent,
} from "@vrooli/proto-types/vrooli-bridge/v1/channel/channel_pb";

export const makeRun = (overrides: MessageInitShape<typeof RunSchema> = {}): Run =>
  create(RunSchema, {
    id: "run-1",
    nodeId: "node-1",
    scenario: "web-console",
    verb: "scenario test",
    args: ["web-console"],
    status: RunStatus.PASSED,
    exitCode: 0,
    timeoutSeconds: 900n,
    createdAt: timestampFromDate(new Date("2026-06-18T10:00:00Z")),
    startedAt: timestampFromDate(new Date("2026-06-18T10:00:05Z")),
    finishedAt: timestampFromDate(new Date("2026-06-18T10:02:05Z")),
    ...overrides,
  });

export const makeRunEvent = (
  overrides: MessageInitShape<typeof RunEventSchema> = {},
): RunEvent =>
  create(RunEventSchema, {
    runId: "run-1",
    kind: RunEventKind.LOG,
    sequence: 1n,
    logChunk: "compiling…",
    emittedAt: timestampFromDate(new Date("2026-06-18T10:00:10Z")),
    ...overrides,
  });
