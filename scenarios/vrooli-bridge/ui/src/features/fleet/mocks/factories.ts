/**
 * Test factory for the registry Node proto message. Lives with the fleet
 * feature (not in the cross-domain test-utils) so deleting the feature folder
 * takes its doubles along. `makeNode()` returns a TRUSTED online node by
 * default; tests override the fields the case is about.
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  NodeSchema,
  NodeStatus,
  type Node,
} from "@vrooli/proto-types/vrooli-bridge/v1/registry/registry_pb";
import {
  NodeQueueSchema,
  type NodeQueue,
} from "@vrooli/proto-types/vrooli-bridge/v1/queue/queue_pb";
import {
  GetOnboardingResponseSchema,
  OnboardingOpSchema,
  OnboardingState,
  OnboardingStepEventSchema,
  OnboardingStepStatus,
  type GetOnboardingResponse,
  type OnboardingOp,
  type OnboardingStepEvent,
} from "@vrooli/proto-types/vrooli-bridge/v1/onboard/onboard_pb";

export const makeNode = (overrides: MessageInitShape<typeof NodeSchema> = {}): Node =>
  create(NodeSchema, {
    id: "node-1",
    name: "mac-mini-office",
    os: "darwin",
    arch: "arm64",
    revision: "abc1234567def",
    status: NodeStatus.ONLINE,
    online: true,
    ...overrides,
  });

/**
 * A node's live scheduler snapshot. Defaults to idle (no running/queued jobs);
 * tests override `running` / `queued` to exercise the live job-status row.
 */
export const makeNodeQueue = (
  overrides: MessageInitShape<typeof NodeQueueSchema> = {},
): NodeQueue =>
  create(NodeQueueSchema, {
    nodeId: "node-1",
    concurrencyLimit: 2,
    running: 0,
    queued: 0,
    ...overrides,
  });

/**
 * A durable onboarding op. Defaults to a mid-flight BOOTSTRAPPING op; tests
 * override `state` / `failureReason` / `nodeId` to exercise terminal banners.
 */
export const makeOnboardingOp = (
  overrides: MessageInitShape<typeof OnboardingOpSchema> = {},
): OnboardingOp =>
  create(OnboardingOpSchema, {
    id: "op-1",
    host: "node-01.example.com",
    port: 22,
    user: "root",
    nodeName: "mac-mini-office",
    targetRevision: "@cp",
    state: OnboardingState.BOOTSTRAPPING,
    ...overrides,
  });

/** One append-only step event; defaults to a completed `clone`. */
export const makeStepEvent = (
  overrides: MessageInitShape<typeof OnboardingStepEventSchema> = {},
): OnboardingStepEvent =>
  create(OnboardingStepEventSchema, {
    opId: "op-1",
    sequence: 1n,
    stepId: "clone",
    status: OnboardingStepStatus.OK,
    ...overrides,
  });

/** A GetOnboarding response (op + event history) for the progress view. */
export const makeGetOnboardingResponse = (
  op: MessageInitShape<typeof OnboardingOpSchema> = {},
  events: OnboardingStepEvent[] = [],
): GetOnboardingResponse =>
  create(GetOnboardingResponseSchema, { op: makeOnboardingOp(op), events });
