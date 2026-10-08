import type { Capability } from "@vrooli/proto-types/content-desk/v1/capabilities/capabilities_pb";
import { strings } from "../../consts/strings";

export const DRAFT_TERMINAL_STATUSES: readonly string[] = ["published", "abandoned"];

export type NextActionKey = (typeof strings.board.nextAction)[keyof typeof strings.board.nextAction];
export type DimensionLabelKey = (typeof strings.board.dimensions)[keyof typeof strings.board.dimensions];

const NEXT_ACTION_KEYS: Readonly<Record<string, NextActionKey>> = {
  requested: strings.board.nextAction.requested,
  drafting: strings.board.nextAction.drafting,
  drafted: strings.board.nextAction.drafted,
  checking: strings.board.nextAction.checking,
  blocked: strings.board.nextAction.blocked,
  reviewed: strings.board.nextAction.reviewed,
  approved: strings.board.nextAction.approved,
};

export function isTerminalDraft(status: string): boolean {
  return DRAFT_TERMINAL_STATUSES.includes(status);
}

export function draftNextActionKey(status: string): NextActionKey {
  return NEXT_ACTION_KEYS[status] ?? strings.board.nextAction.drift;
}

export interface CapabilityDimension {
  key: string;
  label: DimensionLabelKey;
  healthy: boolean;
  value: string;
}

export interface CapabilityGap {
  id: string;
  name: string;
  owner: string;
  nextAction: string;
  dimensions: CapabilityDimension[];
}

function dimension(
  key: string,
  label: DimensionLabelKey,
  value: string,
  healthyValues: readonly string[],
): CapabilityDimension {
  return { key, label, healthy: healthyValues.includes(value), value };
}

export function capabilityGaps(capabilities: readonly Capability[]): CapabilityGap[] {
  return capabilities.flatMap((capability) => {
    const dimensions = [
      dimension("definition", strings.board.dimensions.definition, capability.definitionStatus, ["documented"]),
      dimension("implementation", strings.board.dimensions.implementation, capability.implementationStatus, ["implemented"]),
      dimension("operational", strings.board.dimensions.operational, capability.operationalReadiness, ["qualified-in-environment"]),
      dimension("quality", strings.board.dimensions.quality, capability.outputQuality, ["accepted-by-review"]),
      dimension("distribution", strings.board.dimensions.distribution, capability.distributionConnectivity, [
        "connected",
        "not-applicable",
      ]),
    ].filter((entry) => !entry.healthy);
    if (dimensions.length === 0) return [];
    return [
      {
        id: capability.id,
        name: capability.name,
        owner: capability.owner,
        nextAction: capability.nextAction,
        dimensions,
      },
    ];
  });
}
