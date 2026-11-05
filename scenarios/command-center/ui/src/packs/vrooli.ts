import { meridianArc } from "../scenes/meridianArc";
import { funnelCascade } from "../scenes/funnelCascade";
import type { ConnectorPack } from "./types";

// Trusted in-tree pack for Milestone 1. Milestone 2 can replace this module
// with a sandboxed data-only pack without changing the engine contract.
export const vrooliPack: ConnectorPack = {
  id: "vrooli",
  sourceDescriptors: ["vrooli-core", "swarm-manager", "landing-page-business-suite", "offer-desk", "deployment-manager", "prompt-manager", "source-ledger"],
  readouts: { ladder: "NextRungReadout", posture: "PostureReadout" },
  compositions: { "meridian-arc": meridianArc, "funnel-cascade": funnelCascade },
};
