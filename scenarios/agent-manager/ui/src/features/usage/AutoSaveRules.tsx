import { Button } from "@vrooli/react-component-library/Button/2";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { buildApiUrl } from "@vrooli/api-base";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { fetchWithDeadline, resolveAgentManagerApiBase } from "../../lib/api";
import { formatAt } from "./format";

/** A standing rule: when a runner's window is projected to run out, Save usage starts by itself. */
export interface AutoSaveRule {
  runner: string;
  since: string;
  by: string;
  starts: number;
  last_start?: string;
}

const RULES_PATH = "/usage/modes/auto-save";
const rulesKey = ["usage", "auto-save"] as const;

function rulesUrl(): string {
  return buildApiUrl(RULES_PATH, { baseUrl: resolveAgentManagerApiBase(true) });
}

async function readRules(): Promise<AutoSaveRule[]> {
  const response = await fetchWithDeadline(rulesUrl(), { headers: { Accept: "application/json" }, cache: "no-store" });
  if (!response.ok) throw new Error(`Auto-save rules: ${response.status}`);
  const body = (await response.json()) as { rules?: AutoSaveRule[] };
  return body.rules ?? [];
}

export function useAutoSaveRules() {
  return useQuery({ queryKey: rulesKey, queryFn: readRules, retry: false, refetchInterval: 60_000 });
}

function useTurnOffAutoSave() {
  const queryClient = useQueryClient();
  return useMutation({
    mutationFn: async (runner: string) => {
      const response = await fetchWithDeadline(rulesUrl(), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ runner, enabled: false }),
      });
      if (!response.ok) throw new Error((await response.text()).trim() || `Could not turn the rule off (${response.status})`);
    },
    onSuccess: () => queryClient.invalidateQueries({ queryKey: rulesKey }),
  });
}

const runnerName = (runner: string) => (runner === "claude" ? "Claude" : "Codex");

/** One row per active rule, so a rule the operator set once stays visible and one tap from off. It shows nothing while no rule is on. */
export function AutoSaveRules() {
  const rules = useAutoSaveRules();
  const turnOff = useTurnOffAutoSave();
  if (!rules.data || rules.data.length === 0) return null;
  return (
    <section aria-label="Auto-save rules" data-testid="auto-save-rules" className="space-y-2 rounded-lg border border-primary/50 bg-card p-4">
      {rules.data.map((rule) => (
        <div key={rule.runner} className="flex flex-wrap items-start justify-between gap-3" data-testid={`auto-save-rule-${rule.runner}`}>
          <div className="min-w-0">
            <div className="flex flex-wrap items-center gap-2"><StatusBadge tone="info">Rule on</StatusBadge><h3 className="min-w-0 break-words text-base font-semibold">Save usage starts by itself for {runnerName(rule.runner)}</h3></div>
            <p className="text-sm text-muted-foreground">
              When a {runnerName(rule.runner)} window is projected to run out, new work switches to cheaper models until the reset and you get a notice instead of a question. Set {formatAt(rule.since)}.{" "}
              {rule.starts > 0 ? `Started ${rule.starts} time${rule.starts === 1 ? "" : "s"}${rule.last_start ? `, last ${formatAt(rule.last_start)}` : ""}.` : "It has not started a mode yet."}
            </p>
          </div>
          <Button variant="warning" disabled={turnOff.isPending} onClick={() => turnOff.mutate(rule.runner)} data-testid={`auto-save-off-${rule.runner}`}>Turn rule off</Button>
        </div>
      ))}
      {turnOff.error ? <p role="alert" className="text-sm text-destructive" data-testid="auto-save-error">{turnOff.error.message}</p> : null}
    </section>
  );
}
