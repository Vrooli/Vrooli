import { useTranslation } from "react-i18next";
import { AlertTriangle, CheckCircle2, MoreHorizontal } from "lucide-react";
import type { StatusTone } from "@vrooli/react-component-library/StatusBadge/1";
import { IconButton } from "@vrooli/react-component-library/IconButton/3";
import { Button } from "../ui/button";
import type { JoinRequest, Machine } from "../../api/machines";
import { strings } from "../../consts/strings";
import { humanAge } from "../machines/age";
import { grantSentence } from "../machines/grant";
import { machineDrawState, machineIssues, reachabilityDetail, statusBadge } from "../machines/MachineList";
import { machineTestID } from "../machines/testids";
import { DeviceSilhouette } from "../terminal/device/DeviceSilhouette";
import FleetCard from "./FleetCard";
import MachineSilhouette from "./MachineSilhouette";

/**
 * A machine, as it appears on the shelf.
 *
 * What this card used to do was answer every question about a machine at once:
 * the reachability sentence twice, the full drift list one line at a time, one
 * full-width install button per missing capability, and then three equal-weight
 * actions. The card grew with how broken the machine was, and the third action
 * did not fit in 268px, so `Configure` was cut off before the shelf had
 * scrolled anywhere.
 *
 * It now answers two: can I use it, and does it need me. Everything else is one
 * tap away in the machine's own detail sheet, which is also where `Manage` and
 * `Configure` went — they were never two features, only two tabs.
 *
 * The action row is one primary whose verb follows reachability, one secondary
 * that opens the sheet, and an overflow. Nothing here is a pill: these controls
 * sit in a row inside a card whose corners are `--radius-control`, and a
 * stadium beside them reads as belonging to a different app.
 */

export function MachineCard({
  machine,
  onOpen,
  onStartSession,
  onOverflow,
}: {
  machine: Machine;
  /** Opens the detail sheet, optionally on a named tab. */
  onOpen?: (machine: Machine, tab?: "overview" | "permissions" | "configuration" | "activity") => void;
  onStartSession?: (machine: Machine) => void;
  onOverflow?: (machine: Machine) => void;
}) {
  const { t } = useTranslation();
  const translate = t as (key: string, options?: Record<string, unknown>) => string;
  const isLocal = machine.target.kind === "local";
  const badge = statusBadge(machine, translate);
  const title = isLocal ? t(strings.machines.thisComputer) : machine.target.label;
  const platform = [machine.target.os, machine.target.arch].filter(Boolean).join(" · ");
  const issues = machineIssues(machine);

  return (
    <div data-testid={`machines-row-${machineTestID(machine.target.id)}`} className="shrink-0">
      <FleetCard
        testId={`fleet-card-machine-${machine.target.id}`}
        title={title}
        meta={[platform, reachabilityDetail(machine, translate)].filter(Boolean).join(" · ")}
        status={badge.label}
        statusTone={badge.tone as StatusTone}
        silhouette={
          isLocal ? (
            // The local machine is the computer this console is running on — a
            // screen someone is looking at, not a headless box.
            <DeviceSilhouette archetype="laptop" keyboardShare={0} kbOpen={false} screenLit />
          ) : (
            <MachineSilhouette state={machineDrawState(machine)} />
          )
        }
        state={
          issues.count > 0 ? (
            <button
              type="button"
              data-testid={`machines-issues-${machineTestID(machine.target.id)}`}
              onClick={() => { onOpen?.(machine, "configuration"); }}
              className="flex w-full items-center justify-between gap-2 rounded-lg border border-amber-400/30 bg-amber-400/10 px-2.5 py-2 text-start transition hover:border-amber-400/60"
            >
              <span className="flex min-w-0 items-center gap-2">
                <AlertTriangle className="h-3.5 w-3.5 shrink-0 text-amber-300" aria-hidden />
                <span className="truncate text-xs text-amber-200">
                  {t("machines.needsAttention", { count: issues.count })}
                </span>
              </span>
              <span aria-hidden className="shrink-0 text-xs text-wc-text-faint">›</span>
            </button>
          ) : (
            <div className="flex items-center gap-2 rounded-lg border border-wc-default px-2.5 py-2">
              <CheckCircle2 className="h-3.5 w-3.5 shrink-0 text-emerald-400/70" aria-hidden />
              <span className="truncate text-xs text-wc-text-faint">{t(strings.machines.nothingToFix)}</span>
            </div>
          )
        }
        actions={
          <>
            {machine.target.available && onStartSession ? (
              <Button
                size="sm"
                className="flex-1"
                data-testid={`machines-start-session-${machineTestID(machine.target.id)}`}
                onClick={() => { onStartSession(machine); }}
              >
                {t(strings.fleet.startSession)}
              </Button>
            ) : (
              <Button
                size="sm"
                variant="outline"
                className="flex-1"
                data-testid={`machines-reconnect-${machineTestID(machine.target.id)}`}
                disabled={!machine.manageable}
                onClick={() => { onOpen?.(machine, "configuration"); }}
              >
                {t(strings.machines.reconnect)}
              </Button>
            )}
            {onOpen && (
              <Button
                size="sm"
                variant="outline"
                data-testid={`machines-details-${machineTestID(machine.target.id)}`}
                onClick={() => { onOpen(machine); }}
              >
                {t(strings.machines.details)}
              </Button>
            )}
            {onOverflow && (
              <IconButton
                size="sm"
                shape="rounded"
                surface="soft"
                aria-label={t(strings.machines.moreActions, { name: title })}
                data-testid={`machines-overflow-${machineTestID(machine.target.id)}`}
                onClick={() => { onOverflow(machine); }}
              >
                <MoreHorizontal aria-hidden />
              </IconButton>
            )}
          </>
        }
      >
        {grantSentence(machine.grant, translate)}
      </FleetCard>
    </div>
  );
}

export function JoinRequestCard({ request, onReview }: { request: JoinRequest; onReview: () => void }) {
  const { t } = useTranslation();
  const platform = [request.os, request.arch, request.endpoint].filter(Boolean).join(" · ");
  return (
    <div data-testid={`machines-join-request-${machineTestID(request.id)}`} className="shrink-0">
      <FleetCard
        testId={`fleet-card-join-request-${machineTestID(request.id)}`}
        title={t(strings.machines.reviewTitle, { name: request.name })}
        meta={[platform, t(strings.machines.askedToJoin, { age: humanAge(request.requestedAgeSeconds) })].filter(Boolean).join(" · ")}
        status={t(strings.machines.review)}
        statusTone="info"
        silhouette={<MachineSilhouette state="unenrolled" />}
        state={
          <div className="flex items-center gap-2 rounded-lg border border-wc-accent/30 bg-wc-accent/10 px-2.5 py-2">
            <span className="truncate text-xs text-wc-accent">{t(strings.machines.reviewDerived)}</span>
          </div>
        }
        actions={
          <Button
            size="sm"
            className="flex-1"
            data-testid={`machines-review-${machineTestID(request.id)}`}
            onClick={onReview}
          >
            {t(strings.machines.review)}
          </Button>
        }
      >
        {t(strings.machines.askedToJoin, { age: humanAge(request.requestedAgeSeconds) })}
      </FleetCard>
    </div>
  );
}

export default MachineCard;
