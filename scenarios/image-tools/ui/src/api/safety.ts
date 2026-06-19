import { createClient } from "@connectrpc/connect";
import {
  ConsentWeight,
  DeploymentTier,
  SafetyService,
  type OpWeight,
  type SafetyPolicy,
} from "@vrooli/proto-types/image-tools/v1/safety/safety_pb";

import { transport } from "./client";

/**
 * Typed Connect client for the Responsible-Use policy. `GetPolicy` is a pure
 * read of the resolved deployment-tier policy (tier, enforced controls, the
 * per-op consent-weight table, and a human summary). The submit edge enforces
 * the gate server-side; this client only *reports* the policy so the UI can
 * show what's enforced and ask for consent before a high-weight op.
 */
export const safetyClient = createClient(SafetyService, transport);

/** Read the resolved Responsible-Use policy for the running deployment tier. */
export const getPolicy = (): Promise<SafetyPolicy> => safetyClient.getPolicy({});

/**
 * The set of operations whose consent weight is HIGH per `policy.opWeights`.
 * These are the identity-altering ops the public-tier gate blocks unless
 * `consentAffirmed` is set. Falls back to nothing when the table is empty so a
 * caller never over-gates on a misconfigured policy.
 */
export const highConsentOps = (policy: SafetyPolicy | null | undefined): ReadonlySet<string> =>
  new Set(
    (policy?.opWeights ?? [])
      .filter((w) => w.weight === ConsentWeight.HIGH)
      .map((w) => w.operation),
  );

/**
 * Whether `operation` needs an affirmed-consent checkbox before submit: the
 * policy requires consent (public tier) AND the op is high-weight. On the local
 * tier `requireConsent` is false, so this is always false and the checkbox
 * stays out of the way.
 */
export const needsConsent = (policy: SafetyPolicy | null | undefined, operation: string): boolean =>
  !!policy?.requireConsent && highConsentOps(policy).has(operation);

export { ConsentWeight, DeploymentTier };
export type { OpWeight, SafetyPolicy };
