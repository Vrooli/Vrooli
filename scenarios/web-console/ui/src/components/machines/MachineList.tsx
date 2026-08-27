import { useTranslation } from "react-i18next";
import { CheckCircle2, Laptop, Loader2, Plus, RefreshCw, Server, WifiOff } from "lucide-react";
import { Button } from "../ui/button";
import { strings } from "../../consts/strings";
import { machineTestID } from "./testids";
import type { Fleet, JoinRequest, Machine } from "../../api/machines";
import { humanAge } from "./age";
import { GrantLine } from "./grant";

/**
 * Screen 01 — the machines panel.
 *
 * Machines are an object this app manages, the way it already manages
 * sessions. Every row answers the two questions that decide whether a machine
 * is usable: does it answer, and what is it allowed to do. Permission is shown
 * as a sentence, because nobody should have to read `*:read` to know what a
 * machine can do.
 */

interface MachineListProps {
  fleet?: Fleet;
  loading: boolean;
  refreshing: boolean;
  onRefresh: () => void;
  onAdd: () => void;
  onManage: (machine: Machine) => void;
  onReview: (request: JoinRequest) => void;
}

/** The narrow shape these helpers need from i18next, so a row helper does not
 *  inherit the full generic translator type. */
type Translate = (key: string, options?: Record<string, unknown>) => string;

function reachabilityDetail(machine: Machine, t: Translate): string {
  if (machine.target.kind === "local") return t(strings.machines.thisComputerDetail);
  if (machine.heartbeatAgeSeconds <= 0 && !machine.target.available) return t(strings.machines.neverResponded);
  const age = humanAge(machine.heartbeatAgeSeconds);
  return machine.target.available
    ? t(strings.machines.respondedAgo, { age })
    : t(strings.machines.lastResponded, { age });
}

function statusPill(machine: Machine, t: Translate) {
  if (machine.target.kind === "local") {
    return { label: t(strings.machines.statusLocal), tone: "border-wc-default bg-wc-surface-input text-wc-text-secondary", icon: <Laptop className="h-3.5 w-3.5" aria-hidden /> };
  }
  if (machine.target.available) {
    return { label: t(strings.machines.statusReachable), tone: "border-emerald-400/30 bg-emerald-400/10 text-emerald-300", icon: <CheckCircle2 className="h-3.5 w-3.5" aria-hidden /> };
  }
  if (machine.heartbeatAgeSeconds <= 0) {
    return { label: t(strings.machines.statusNeverResponded), tone: "border-slate-400/25 bg-slate-400/10 text-slate-300", icon: <WifiOff className="h-3.5 w-3.5" aria-hidden /> };
  }
  return { label: t(strings.machines.statusNotResponding), tone: "border-amber-400/30 bg-amber-400/10 text-amber-200", icon: <WifiOff className="h-3.5 w-3.5" aria-hidden /> };
}

function MachineRow({ machine, onManage }: { machine: Machine; onManage: (machine: Machine) => void }) {
  const { t } = useTranslation();
  const translate = t as Translate;
  const isLocal = machine.target.kind === "local";
  const pill = statusPill(machine, translate);
  const platform = [machine.target.os, machine.target.arch].filter(Boolean).join(" · ");
  const label = isLocal ? t(strings.machines.thisComputer) : machine.target.label;

  return (
    <li
      data-testid={`machines-row-${machineTestID(machine.target.id)}`}
      className="flex flex-wrap items-center gap-x-4 gap-y-3 rounded-xl border border-wc-default bg-wc-surface-input px-4 py-3.5 transition hover:border-wc-accent/50"
    >
      <span
        className={`flex h-10 w-10 shrink-0 items-center justify-center rounded-lg ${isLocal ? "bg-wc-accent/15 text-wc-accent" : "bg-wc-surface-base text-wc-text-secondary"}`}
      >
        {isLocal ? <Laptop className="h-5 w-5" aria-hidden /> : <Server className="h-5 w-5" aria-hidden />}
      </span>

      <span className="min-w-0 flex-1 basis-48">
        <span className="block truncate text-sm font-medium text-wc-text-primary">{label}</span>
        <span className="mt-0.5 block truncate text-xs text-wc-text-faint">
          {[platform, reachabilityDetail(machine, translate)].filter(Boolean).join(" · ")}
        </span>
        {/* The recovery action rides with the row, so an unreachable machine
            always offers something the operator can do from inside the app. */}
        {!machine.target.available && machine.target.recovery_action && (
          <span className="mt-1 block text-xs leading-5 text-amber-200/80">{machine.target.recovery_action}</span>
        )}
      </span>

      <span className={`inline-flex shrink-0 items-center gap-1.5 rounded-full border px-2.5 py-1 text-[11px] font-medium ${pill.tone}`}>
        {pill.icon}
        {pill.label}
      </span>

      <span className="min-w-0 basis-full sm:basis-52">
        <GrantLine grant={machine.grant} />
      </span>

      {machine.manageable ? (
        <Button
          variant="outline"
          size="sm"
          data-testid={`machines-manage-${machineTestID(machine.target.id)}`}
          onClick={() => { onManage(machine); }}
        >
          {machine.target.available ? t(strings.machines.manage) : t(strings.machines.reconnect)}
        </Button>
      ) : (
        <span className="w-[4.5rem]" aria-hidden />
      )}
    </li>
  );
}

function JoinRequestBanner({ request, onReview }: { request: JoinRequest; onReview: (request: JoinRequest) => void }) {
  const { t } = useTranslation();
  const platform = [request.os, request.arch, request.endpoint].filter(Boolean).join(" · ");
  return (
    <li
      data-testid={`machines-join-request-${machineTestID(request.id)}`}
      className="flex flex-wrap items-center gap-x-4 gap-y-3 rounded-xl border border-wc-accent/40 bg-wc-accent/10 px-4 py-3.5"
    >
      <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-lg bg-wc-accent/20 text-wc-accent">
        <Server className="h-5 w-5" aria-hidden />
      </span>
      <span className="min-w-0 flex-1">
        <span className="block truncate text-sm font-medium text-wc-text-primary">
          {t(strings.machines.reviewTitle, { name: request.name })}
        </span>
        <span className="mt-0.5 block truncate text-xs text-wc-text-faint">
          {[platform, t(strings.machines.askedToJoin, { age: humanAge(request.requestedAgeSeconds) })]
            .filter(Boolean)
            .join(" · ")}
        </span>
      </span>
      <Button size="sm" data-testid={`machines-review-${machineTestID(request.id)}`} onClick={() => { onReview(request); }}>
        {t(strings.machines.review)}
      </Button>
    </li>
  );
}

export default function MachineList({ fleet, loading, refreshing, onRefresh, onAdd, onManage, onReview }: MachineListProps) {
  const { t } = useTranslation();
  const machines = fleet?.machines ?? [];
  const remote = machines.filter((machine) => machine.target.kind !== "local");
  const reachable = remote.filter((machine) => machine.target.available).length;
  const requests = fleet?.joinRequests ?? [];

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="mx-auto flex w-full max-w-4xl flex-wrap items-center justify-between gap-3 px-5 pt-5">
        <h2 data-testid="machines-count-summary" className="text-sm text-wc-text-muted">
          {t(strings.machines.countSummary, { linked: remote.length, reachable })}
        </h2>
        <div className="flex items-center gap-2">
          <button
            type="button"
            data-testid="machines-refresh"
            onClick={onRefresh}
            disabled={refreshing}
            aria-label={t(strings.machines.refresh)}
            className="inline-flex min-h-11 items-center gap-1.5 rounded-lg px-3 text-xs font-medium text-wc-text-secondary transition hover:bg-wc-surface-input hover:text-wc-text-primary disabled:opacity-60"
          >
            <RefreshCw className={refreshing ? "h-4 w-4 animate-spin" : "h-4 w-4"} aria-hidden />
            <span className="hidden sm:inline">{refreshing ? t(strings.machines.refreshing) : t(strings.machines.refresh)}</span>
          </button>
          <Button size="sm" data-testid="machines-add" onClick={onAdd}>
            <Plus className="me-1.5 h-4 w-4" aria-hidden />
            {t(strings.machines.addMachine)}
          </Button>
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto">
        <div className="mx-auto w-full max-w-4xl space-y-2 px-5 pb-5 pt-4">
        {loading && machines.length === 0 ? (
          <div data-testid="machines-loading" className="flex items-center gap-3 rounded-xl border border-wc-default bg-wc-surface-input/50 p-4 text-sm text-wc-text-muted">
            <Loader2 className="h-4 w-4 animate-spin text-wc-accent" aria-hidden />
            {t(strings.machines.loading)}
          </div>
        ) : (
          <ul className="space-y-2">
            {requests.map((request) => (
              <JoinRequestBanner key={request.id} request={request} onReview={onReview} />
            ))}
            {machines.map((machine) => (
              <MachineRow key={machine.target.id} machine={machine} onManage={onManage} />
            ))}
          </ul>
        )}

        {!loading && remote.length === 0 && (
          <div data-testid="machines-empty" className="rounded-xl border border-dashed border-wc-default bg-wc-surface-base/40 p-5 text-center">
            <p className="text-sm font-medium text-wc-text-primary">
              {fleet?.status === "unenrolled" ? t(strings.machines.unenrolledTitle) : t(strings.machines.emptyTitle)}
            </p>
            <p className="mx-auto mt-1 max-w-md text-xs leading-5 text-wc-text-faint">
              {fleet?.message || t(strings.machines.emptyBody)}
            </p>
            {fleet?.recoveryAction && <p className="mt-2 text-xs text-amber-200/80">{fleet.recoveryAction}</p>}
          </div>
        )}
        </div>
      </div>
    </div>
  );
}
