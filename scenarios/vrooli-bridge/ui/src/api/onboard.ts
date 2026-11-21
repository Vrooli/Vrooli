import { createClient } from "@connectrpc/connect";
import type { MessageInitShape } from "@bufbuild/protobuf";
import {
  OnboardService,
  OnboardingState,
  OnboardingStepStatus,
  SourceMode,
  StartOnboardingRequestSchema,
  type OnboardingOp,
  type OnboardingStepEvent,
  type StartOnboardingRequest,
  type GetOnboardingResponse,
  type ListOnboardingsResponse,
} from "@vrooli/proto-types/vrooli-bridge/v1/onboard/onboard_pb";

import { transport } from "./client";

/**
 * Plain-object init shape for a StartOnboarding request — what the typed client
 * (and the mutation hook) accept. It is the un-branded structural form of
 * StartOnboardingRequest (no `$typeName`), so callers pass a literal object.
 */
export type StartOnboardingInput = MessageInitShape<typeof StartOnboardingRequestSchema>;

/**
 * Typed client for the OnboardService — one-shot node onboarding
 * (onboard domain). StartOnboarding drives a raw SSH host from bare OS to a
 * paired, ONLINE fleet agent as a durable, server-owned op; GetOnboarding
 * returns the op plus its full persisted step-event history so the fleet
 * surface can render live progress by re-reading it. The owner's SSH password
 * is carried once in the StartOnboarding request body and never stored client-
 * side (see OnboardNodeForm) — it never touches the durable record either.
 */
export const onboardClient = createClient(OnboardService, transport);

export { OnboardingState, OnboardingStepStatus, SourceMode };
export type {
  OnboardingOp,
  OnboardingStepEvent,
  StartOnboardingRequest,
  GetOnboardingResponse,
  ListOnboardingsResponse,
};
