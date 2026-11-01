import { useCallback, useEffect, useMemo, useState } from "react";
import type { GetRoleModelChoicesResponse, RoleModelChoice } from "@vrooli/proto-types/web-console/v1/modelselection/model_selection_pb";
import { modelSelectionClient } from "../../api/modelSelection";

export const VROOLI_CREDITS_SOURCE = "vrooli-credits";

export function roleChoiceKey(choice: RoleModelChoice): string {
  return JSON.stringify([choice.source, choice.model]);
}

/** A metered choice bills per use, so it cannot be chosen from a list past its freshness budget. */
export function choiceBlocked(choice: RoleModelChoice, stale: boolean): boolean {
  return stale && choice.metered;
}

/** The options a model offers, largest first. None may exceed its window. */
export function limitOptions(choice: RoleModelChoice | undefined): number[] {
  if (!choice) return [];
  const window = Number(choice.contextWindowTokens);
  const limits = choice.contextLimitTokens.map(Number).filter((tokens) => tokens > 0 && (window === 0 || tokens <= window));
  return [...new Set(limits)].sort((a, b) => b - a);
}

/** The last explicit model choice is a device preference, not catalog state. */
const choiceStorageKey = (targetId: string, agentId: string) => `wc-launcher-model-choice:${targetId}:${agentId}`;

function readStoredChoice(key: string): string | null {
  try { return localStorage.getItem(key); } catch { return null; }
}

function writeStoredChoice(key: string, value: string | null) {
  try {
    if (value === null) localStorage.removeItem(key);
    else localStorage.setItem(key, value);
  } catch {
    // A blocked storage write only costs the preference, never the launch.
  }
}

/** The selected harness's model list and the operator's choice from it, per launch. */
export function useModelPolicy(open: boolean, agentId: string | null, targetId: string, launchCommand: string) {
  const [policy, setPolicy] = useState<GetRoleModelChoicesResponse | null>(null);
  const [loading, setLoading] = useState(false);
  const [failed, setFailed] = useState(false);
  const [explicitKey, setExplicitKey] = useState<string | null>(null);
  const [revision, setRevision] = useState(0);
  const storageKey = agentId ? choiceStorageKey(targetId, agentId) : "";

  useEffect(() => {
    setPolicy(null);
    setExplicitKey(null);
    setFailed(false);
    if (!open || !agentId) { setLoading(false); return; }
    let cancelled = false;
    setLoading(true);
    modelSelectionClient.getRoleModelChoices({ agentId, targetId, launchCommand })
      .then((response) => {
        if (cancelled) return;
        setPolicy(response);
        const stored = storageKey ? readStoredChoice(storageKey) : null;
        if (stored && response.choices.some((choice) => roleChoiceKey(choice) === stored)) setExplicitKey(stored);
      })
      .catch(() => { if (!cancelled) setFailed(true); })
      .finally(() => { if (!cancelled) setLoading(false); });
    return () => { cancelled = true; };
  }, [agentId, launchCommand, open, revision, storageKey, targetId]);

  const choose = useCallback((key: string | null) => {
    setExplicitKey(key);
    if (storageKey) writeStoredChoice(storageKey, key);
  }, [storageKey]);

  const choices = useMemo(() => policy?.choices ?? [], [policy]);
  const stale = Boolean(policy && (policy.catalogStale || policy.policyStatus === "stale"));
  // Ready and stale both offer the list: a stale list warns rather than hides (LD-2).
  const selectable = Boolean(policy && (policy.policyStatus === "ready" || policy.policyStatus === "stale") && choices.length > 0 && policy.modelFlag);
  const defaultChoice = choices.find((choice) => choice.isDefault) ?? choices[0];
  const explicitChoice = explicitKey === null ? undefined : choices.find((choice) => roleChoiceKey(choice) === explicitKey);
  const selectedChoice = explicitChoice ?? defaultChoice;
  const sources = useMemo(() => [...new Set(choices.map((choice) => choice.source))], [choices]);
  const sourceChoices = choices.filter((choice) => choice.source === selectedChoice?.source);

  return {
    policy, loading, failed, choices, sources, sourceChoices, stale, selectable,
    defaultChoice, explicitChoice, selectedChoice,
    choose,
    usesVrooliCredits: selectedChoice?.source === VROOLI_CREDITS_SOURCE,
    /** The command runs as written, so no setting here can reach it. */
    literal: Boolean(policy?.commandLiteral),
    /** Reads the list again, after a refresh. */
    reload: () => { setRevision((value) => value + 1); },
  };
}

export type ModelPolicyController = ReturnType<typeof useModelPolicy>;

/** A name for a model a usage mode reports, from the list when the list knows it. */
export function modelLabel(model: ModelPolicyController, catalogId: string | undefined): string | undefined {
  if (!catalogId) return undefined;
  return model.choices.find((choice) => choice.model === catalogId)?.displayName ?? catalogId;
}
