import { Code, ConnectError } from "@connectrpc/connect";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import type { GetRoleModelChoicesResponse, RoleModelChoice } from "@vrooli/proto-types/web-console/v1/modelselection/model_selection_pb";
import { modelSelectionClient } from "../../api/modelSelection";

export const VROOLI_CREDITS_SOURCE = "vrooli-credits";

export function roleChoiceKey(choice: RoleModelChoice): string {
  return JSON.stringify([choice.source, choice.model]);
}

/** A metered choice bills per use, so it cannot be chosen from a list past its freshness budget. */
export function choiceBlocked(choice: RoleModelChoice, stale: boolean): boolean {
  return stale && choice.metered && !choice.policyCurrent;
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
export function useModelPolicy(open: boolean, agentId: string | null, targetId: string, launchCommand: string, businessAccountId = "") {
  const [revision, setRevision] = useState(0);
  // `open` alone cannot distinguish two sessions with the same selection.
  // Advance during the first render of each open so the prior resolved policy
  // is unusable before effects start the new request.
  const wasOpen = useRef(false);
  const openSession = useRef(0);
  const basePolicy = useRef<{ identity: string; sourceConfigurationSupported: boolean; literal: boolean }>();
  if (open && !wasOpen.current) openSession.current += 1;
  wasOpen.current = open;
  const baseIdentity = JSON.stringify([open, openSession.current, agentId, targetId, launchCommand]);
  const requestIdentity = JSON.stringify([baseIdentity, businessAccountId, revision]);
  const [resolved, setResolved] = useState<{
    identity: string;
    policy: GetRoleModelChoicesResponse | null;
    explicitKey: string | null;
    failed: boolean;
  } | null>(null);
  const storageKey = agentId ? choiceStorageKey(targetId, agentId) : "";
  // State from a previous request must become unusable in the render that
  // changes harness, target, command, drawer state, or refresh revision.
  const current = resolved?.identity === requestIdentity ? resolved : undefined;
  const policy = current?.policy ?? null;
  const explicitKey = current?.explicitKey ?? null;
  const failed = current?.failed ?? false;
  const loading = open && Boolean(agentId) && !current;

  useEffect(() => {
    if (!open || !agentId) return;
    let cancelled = false;
    const read = async () => {
      for (let attempt = 0; ; attempt += 1) {
        try {
          return await modelSelectionClient.getRoleModelChoices({ agentId, targetId, launchCommand, businessAccountId });
        } catch (error) {
          const failure = ConnectError.from(error);
          const transient = failure.code === Code.Unavailable || (failure.code === Code.Unknown && /failed to fetch|network.*(changed|error)|networkerror/i.test(failure.message));
          if (cancelled || attempt >= 2 || !transient) throw error;
          await new Promise((resolve) => setTimeout(resolve, 250 * (attempt + 1)));
          if (cancelled) throw error;
        }
      }
    };
    void read()
      .then((response) => {
        if (cancelled) return;
        basePolicy.current = { identity: baseIdentity, sourceConfigurationSupported: response.sourceConfigurationSupported, literal: response.commandLiteral };
        const stored = storageKey ? readStoredChoice(storageKey) : null;
        const explicitKey = stored && response.choices.some((choice) => roleChoiceKey(choice) === stored) ? stored : null;
        setResolved({ identity: requestIdentity, policy: response, explicitKey, failed: false });
      })
      .catch(() => { if (!cancelled) setResolved({ identity: requestIdentity, policy: null, explicitKey: null, failed: true }); });
    return () => { cancelled = true; };
  }, [agentId, baseIdentity, businessAccountId, launchCommand, open, requestIdentity, storageKey, targetId]);

  const choose = useCallback((key: string | null) => {
    setResolved((previous) => previous?.identity === requestIdentity
      ? { ...previous, explicitKey: key }
      : previous);
    if (storageKey) writeStoredChoice(storageKey, key);
  }, [requestIdentity, storageKey]);

  const choices = useMemo(() => policy?.choices ?? [], [policy]);
  const stale = Boolean(policy && (policy.catalogStale || policy.policyStatus === "stale"));
  // Ready and stale both offer the list: a stale list warns rather than hides (LD-2).
  const selectable = Boolean(policy && (policy.policyStatus === "ready" || policy.policyStatus === "stale") && choices.length > 0 && policy.modelFlag);
  const defaultChoice = choices.find((choice) => choice.isDefault && choice.source !== VROOLI_CREDITS_SOURCE)
    ?? choices.find((choice) => choice.source !== VROOLI_CREDITS_SOURCE);
  const explicitChoice = explicitKey === null ? undefined : choices.find((choice) => roleChoiceKey(choice) === explicitKey);
  const selectedChoice = explicitChoice ?? defaultChoice;
  const sources = useMemo(() => [...new Set(choices.map((choice) => choice.source))], [choices]);
  const sourceChoices = choices.filter((choice) => choice.source === selectedChoice?.source);

  return {
    identity: requestIdentity,
    policy, loading, failed, choices, sources, sourceChoices, stale, selectable,
    defaultChoice, explicitChoice, selectedChoice,
    choose,
    usesVrooliCredits: open && !policy?.commandLiteral && selectedChoice?.source === VROOLI_CREDITS_SOURCE,
    /** The command runs as written, so no setting here can reach it. */
    literal: Boolean(policy?.commandLiteral),
    billingProbeReady: open && targetId === "local" && basePolicy.current?.identity === baseIdentity
      && basePolicy.current.sourceConfigurationSupported && !basePolicy.current.literal,
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
