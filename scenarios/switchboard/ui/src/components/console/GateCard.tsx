import { useMutation, useQueryClient } from "@tanstack/react-query";
import { KeyRound, Timer } from "lucide-react";
import { useEffect, useState } from "react";

import { Button } from "@vrooli/react-component-library/Button/2";

import { consoleApi, type Gate } from "../../api/console";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { msUntil, relativeTime } from "../../lib/time";
import { useSession } from "../../features/session/SessionProvider";

const GATE_STATUS_KEY = {
  granted: strings.console.capabilityGate.status.granted,
  denied: strings.console.capabilityGate.status.denied,
  expired: strings.console.capabilityGate.status.expired,
} as const;

interface GateCardProps {
  gate: Gate;
  /** Whether the viewer is the owner the gate was routed to. */
  viewerIsOwner?: boolean;
  compact?: boolean;
  onAnswered?: (gate: Gate) => void;
  testId?: string;
}

/**
 * A capability gate: the in-thread prompt an agent raises when it needs a
 * scope it was not granted. Routed to the owner only, and it expires unanswered,
 * so the remaining time is the most prominent secondary fact.
 */
export function GateCard({ gate, viewerIsOwner = true, compact, onAnswered, testId = "capability-gate" }: GateCardProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const { withSession } = useSession();
  const [now, setNow] = useState(() => Date.now());
  useEffect(() => {
    const id = window.setInterval(() => setNow(Date.now()), 30_000);
    return () => window.clearInterval(id);
  }, []);
  const remaining = msUntil(gate.expires_at, now);
  const expired = gate.status !== "pending" || remaining <= 0;
  const urgent = !expired && remaining < 5 * 60_000;

  const answer = useMutation({
    mutationFn: (grant: boolean) => withSession(() => consoleApi.answerGate(gate.id, grant)),
    onSuccess: async (updated) => {
      onAnswered?.(updated);
      await queryClient.invalidateQueries({ queryKey: ["console"] });
    },
  });

  return (
    <article
      data-testid={testId}
      role="alert"
      aria-live="polite"
      data-gate-status={gate.status}
      className={[
        "flex flex-col gap-3 rounded-panel border bg-app-surface p-4",
        urgent ? "border-app-warning/60" : "border-app-border",
        compact ? "" : "shadow-subtle",
      ].join(" ")}
    >
      <div className="flex items-start gap-3">
        <span aria-hidden="true" className="grid h-8 w-8 shrink-0 place-items-center rounded-control bg-app-warning/15 text-app-warning">
          <KeyRound className="h-4 w-4" />
        </span>
        <div className="min-w-0 flex-1">
          <h4 className="text-sm font-semibold text-app-foreground">{t(strings.console.capabilityGate.title)}</h4>
          <p data-testid={`${testId}-withheld`} className="mt-1 text-sm text-app-foreground">
            <span className="text-app-muted-foreground">{t(strings.console.capabilityGate.withheld)}</span> {gate.withheld}{" "}
            <code className="rounded-sm bg-app-surface-muted px-1 py-0.5 font-mono text-xs">{gate.scope}</code>
          </p>
          {gate.unblock ? (
            <p data-testid={`${testId}-unblock`} className="mt-1 text-sm text-app-muted-foreground">
              <span>{t(strings.console.capabilityGate.unblock)}</span> {gate.unblock}
            </p>
          ) : null}
        </div>
        <p
          data-testid={`${testId}-expiry`}
          role="status"
          className={["flex shrink-0 items-center gap-1 text-xs font-medium", urgent ? "text-app-warning" : "text-app-muted-foreground"].join(" ")}
        >
          <Timer aria-hidden="true" className="h-3.5 w-3.5" />
          {expired
            ? t(GATE_STATUS_KEY[gate.status === "pending" ? "expired" : gate.status])
            : t(strings.console.capabilityGate.expiresIn, { when: relativeTime(gate.expires_at, now) })}
        </p>
      </div>
      {gate.status === "pending" && !expired ? (
        viewerIsOwner ? (
          <div className="flex flex-wrap gap-2">
            <Button type="button" size="sm" data-testid={`${testId}-grant`} pending={answer.isPending} onClick={() => answer.mutate(true)}>
              {t(strings.console.capabilityGate.grantOnce)}
            </Button>
            <Button type="button" size="sm" variant="secondary" data-testid={`${testId}-deny`} disabled={answer.isPending} onClick={() => answer.mutate(false)}>
              {t(strings.console.capabilityGate.notNow)}
            </Button>
            {answer.isError ? (
              <p role="alert" className="self-center text-xs text-app-danger">
                {answer.error instanceof Error ? answer.error.message : t(strings.errors.unknown)}
              </p>
            ) : null}
          </div>
        ) : (
          <p data-testid={`${testId}-permission-denied`} className="text-xs text-app-muted-foreground">
            {t(strings.console.capabilityGate.ownerOnly)}
          </p>
        )
      ) : null}
    </article>
  );
}
