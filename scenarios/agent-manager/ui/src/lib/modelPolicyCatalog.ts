import type {
  ModelPolicyCatalog,
  ModelPolicyRunnerInventory,
} from "@vrooli/proto-types/agent-manager/v1/api/service_pb";
import { ModelSelectionType } from "@vrooli/proto-types/agent-manager/v1/domain/types_pb";
import type { RunnerType } from "../types";

export function catalogInventoryForRunner(
  catalog: ModelPolicyCatalog | undefined,
  runnerType: RunnerType,
): ModelPolicyRunnerInventory | undefined {
  return catalog?.runners.find((runner) => runner.runnerType === runnerType);
}

export interface PolicyOption {
  ref: string;
  label: string;
  primaryModel?: string;
}

// Only show policies whose first candidate targets the selected runner. This
// keeps the runner selector meaningful while the policy still owns later
// cross-runner fallback candidates.
export function policyOptionsForRunner(
  catalog: ModelPolicyCatalog | undefined,
  runnerType: RunnerType,
): PolicyOption[] {
  const result: PolicyOption[] = [];
  for (const policy of catalog?.policies ?? []) {
    if (policy.candidates[0]?.runnerType !== runnerType) continue;
    const candidate = policy.candidates.find(
      (entry) =>
        entry.runnerType === runnerType &&
        entry.selectionType === ModelSelectionType.MODEL &&
        entry.model.trim() !== "",
    );
    result.push({
      ref: policy.name,
      label: policy.intent ? policy.intent.charAt(0).toUpperCase() + policy.intent.slice(1) : policy.name,
      primaryModel: candidate?.model || undefined,
    });
  }
  return result.sort((a, b) => a.label.localeCompare(b.label));
}
