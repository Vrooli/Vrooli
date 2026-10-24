import {
  OwnershipState,
  IngressSource,
} from "@vrooli/proto-types/tunnel-manager/v1/config/config_pb";

import { strings } from "../../consts/strings";
import type { BadgeTone } from "../../components/ui/StatusBadge";

type StateKey = (typeof strings.drift.state)[keyof typeof strings.drift.state];
type SourceKey = (typeof strings.drift.source)[keyof typeof strings.drift.source];

// Static enum→key maps. The `strings/no-unused-keys` eslint rule is a literal
// static scan, so every leaf needs a literal `strings.drift.state.*` /
// `strings.drift.source.*` reference — these records provide exactly that, and
// keep the helpers below typed to the strings-subtree union (never `string`).
const STATE_LABEL: Record<OwnershipState, StateKey> = {
  [OwnershipState.UNSPECIFIED]: strings.drift.state.unknown,
  [OwnershipState.MANAGED]: strings.drift.state.managed,
  [OwnershipState.MISSING]: strings.drift.state.missing,
  [OwnershipState.EXTERNAL_OK]: strings.drift.state.externalOk,
  [OwnershipState.ORPHANED]: strings.drift.state.orphaned,
  [OwnershipState.IGNORED]: strings.drift.state.ignored,
  [OwnershipState.UNMANAGED]: strings.drift.state.unmanaged,
};

const STATE_TONE: Record<OwnershipState, BadgeTone> = {
  [OwnershipState.UNSPECIFIED]: "neutral",
  [OwnershipState.MANAGED]: "success",
  [OwnershipState.MISSING]: "info",
  [OwnershipState.EXTERNAL_OK]: "success",
  [OwnershipState.ORPHANED]: "warning",
  [OwnershipState.IGNORED]: "neutral",
  [OwnershipState.UNMANAGED]: "danger",
};

const SOURCE_LABEL: Record<IngressSource, SourceKey> = {
  [IngressSource.UNSPECIFIED]: strings.drift.source.unknown,
  [IngressSource.SCENARIO]: strings.drift.source.scenario,
  [IngressSource.EXTERNAL]: strings.drift.source.external,
};

/** Map an ownership state to its translation key. */
export function ownershipStateLabel(state: OwnershipState): StateKey {
  return STATE_LABEL[state];
}

/** Map an ownership state to a badge tone. */
export function ownershipStateTone(state: OwnershipState): BadgeTone {
  return STATE_TONE[state];
}

/** Map an ingress source to its translation key. */
export function ingressSourceLabel(source: IngressSource): SourceKey {
  return SOURCE_LABEL[source];
}

/**
 * A live URL is one whose ingress is actually serving on Cloudflare
 * (managed or tracked-external). MISSING means desired-but-not-yet-live.
 */
export function isStateLive(state: OwnershipState): boolean {
  return state === OwnershipState.MANAGED || state === OwnershipState.EXTERNAL_OK;
}
