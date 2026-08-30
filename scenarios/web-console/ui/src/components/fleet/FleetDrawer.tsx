import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { CircleAlert, Plus, Radio, RefreshCw } from "lucide-react";
import { FullPageDrawer } from "@vrooli/react-component-library/FullPageDrawer/1";
import { Button } from "../ui/button";
import { strings } from "../../consts/strings";
import { useFleet, useFleetMutations } from "../../hooks/useFleet";
import type { IssuedCode, JoinRequest, Machine } from "../../api/machines";
import MachineCard, { JoinRequestCard } from "./MachineCard";
import AddMachine from "../machines/AddMachine";
import ReviewRequest from "../machines/ReviewRequest";
import GrantPicker from "../machines/GrantPicker";
import { useDevices, useDeviceMutations } from "../../hooks/useDevices";
import DeviceCard from "./DeviceCard";
import FleetRail from "./FleetRail";

/**
 * The machines surface.
 *
 * Four screens, one object. A machine is linked once for the whole
 * installation — not once per app that wants to reach it — so this lives in
 * the app the person already has open rather than behind a detour through the
 * control plane's own interface.
 */

type View =
  | { kind: "list" }
  | { kind: "add" }
  | { kind: "review"; request: JoinRequest }
  | { kind: "grant-new"; request: JoinRequest }
  | { kind: "grant-existing"; machine: Machine };

interface FleetDrawerProps {
  open: boolean;
  onClose: () => void;
  onStartSession?: (machine: Machine) => void;
}

export default function FleetDrawer({ open, onClose, onStartSession }: FleetDrawerProps) {
  const { t } = useTranslation();
  const [view, setView] = useState<View>({ kind: "list" });
  const [code, setCode] = useState<IssuedCode | null>(null);
  const [failure, setFailure] = useState("");

  const fleetQuery = useFleet(open);
  const devicesQuery = useDevices(open);
  const { disconnect, giveControl, rename } = useDeviceMutations();
  const { issueCode, decide, setGrant } = useFleetMutations();
  const fleet = fleetQuery.data;

  // Reopening should start at the list. A surface that resumes mid-approval
  // would ask the operator to confirm words they are no longer looking at.
  useEffect(() => {
    if (!open) {
      setView({ kind: "list" });
      setCode(null);
      setFailure("");
    }
  }, [open]);

  const goList = useCallback(() => { setView({ kind: "list" }); setFailure(""); }, []);

  const reportFailure = useCallback((error: unknown) => {
    setFailure(error instanceof Error ? error.message : String(error));
  }, []);

  const handleIssueCode = useCallback(() => {
    setFailure("");
    issueCode.mutate("", { onSuccess: setCode, onError: reportFailure });
  }, [issueCode, reportFailure]);

  const handleDeny = useCallback((request: JoinRequest) => {
    setFailure("");
    decide.mutate(
      { requestId: request.id, approve: false, confirmationWords: [], preset: "" },
      { onSuccess: goList, onError: reportFailure },
    );
  }, [decide, goList, reportFailure]);

  const handleLink = useCallback((request: JoinRequest, preset: string) => {
    setFailure("");
    decide.mutate(
      { requestId: request.id, approve: true, confirmationWords: request.confirmationWords, preset },
      { onSuccess: goList, onError: reportFailure },
    );
  }, [decide, goList, reportFailure]);

  const handleSaveGrant = useCallback((machine: Machine, preset: string) => {
    setFailure("");
    setGrant.mutate(
      { machineId: machine.target.id, preset },
      { onSuccess: goList, onError: reportFailure },
    );
  }, [goList, reportFailure, setGrant]);

  // A request the operator is reviewing can be answered elsewhere, or expire.
  // Following the live list back to the machines view is more honest than
  // leaving a stale approval on screen.
  useEffect(() => {
    if (!fleet) return;
    const pendingID = view.kind === "review" ? view.request.id : view.kind === "grant-new" ? view.request.id : "";
    if (pendingID && !fleet.joinRequests.some((request) => request.id === pendingID)) {
      setView({ kind: "list" });
    }
  }, [fleet, view]);

  const controlPlaneLine = useMemo(() => {
    if (!fleet) return "";
    return fleet.controlPlane.reachable
      ? t(strings.machines.controlPlaneHealthy, { endpoint: fleet.controlPlane.endpoint })
      : t(strings.machines.controlPlaneUnreachable);
  }, [fleet, t]);

  const presets = fleet?.presets ?? [];
  const remoteMachines = fleet?.machines.filter((machine) => machine.target.kind !== "local") ?? [];
  const reachableMachines = remoteMachines.filter((machine) => machine.target.available).length;

  return (
    <FullPageDrawer
      avoidKeyboard
      open={open}
      onClose={onClose}
      closeLabel={t(strings.machines.closeAriaLabel)}
      title={t(strings.fleet.title)}
      testId="machines-drawer"
    >
      <div className="flex h-full min-h-0 flex-col">
        {view.kind === "list" && (
          <div className="min-h-0 flex-1 overflow-y-auto py-5">
            <p className="px-5 pb-5 text-sm text-wc-text-muted">{t(strings.fleet.subtitle)}</p>
            <FleetRail testId="fleet-rail-devices" eyebrow={t(strings.fleet.devices)} description={t(strings.fleet.devicesDescription)} count={devicesQuery.data?.length ?? 0}>
              {(devicesQuery.data ?? []).map((device) => (
                <DeviceCard
                  key={`${device.deviceId}-${device.firstSeenAt}`}
                  device={device}
                  onGiveControl={(item) => {
                    const session = item.sessions.find((attachment) => !attachment.holdsLease);
                    if (session) giveControl.mutate({ deviceId: item.deviceId, sessionId: session.sessionId });
                  }}
                  onDropOld={(item) => { disconnect.mutate({ deviceId: item.deviceId }); }}
                  onRename={(item) => {
                    if (!item.isSelf) return;
                    const label = window.prompt(t(strings.fleet.rename), item.deviceLabel);
                    if (label?.trim()) rename.mutate(label);
                  }}
                />
              ))}
            </FleetRail>
            <div className="mt-8">
              <FleetRail testId="fleet-rail-machines" eyebrow={t(strings.fleet.machines)} description={t(strings.fleet.machinesDescription)} count={(fleet?.machines.length ?? 0) + (fleet?.joinRequests.length ?? 0)}>
                <div className="w-[268px] shrink-0 space-y-3 rounded-xl border border-wc-default bg-wc-surface-base/40 p-4">
                  <div data-testid="machines-count-summary" className="text-xs text-wc-text-muted">{t(strings.machines.countSummary, { linked: remoteMachines.length, reachable: reachableMachines })}</div>
                  <div className="flex gap-2">
                    <Button size="sm" data-testid="machines-add" onClick={() => { setView({ kind: "add" }); }}><Plus className="me-1.5 h-4 w-4" aria-hidden />{t(strings.machines.addMachine)}</Button>
                    <Button variant="outline" size="sm" data-testid="machines-refresh" onClick={() => { void fleetQuery.refetch(); }} disabled={fleetQuery.isFetching}><RefreshCw className={fleetQuery.isFetching ? "h-4 w-4 animate-spin" : "h-4 w-4"} aria-hidden /><span className="sr-only">{t(strings.machines.refresh)}</span></Button>
                  </div>
                </div>
                {fleetQuery.isLoading ? (
                  <div data-testid="machines-loading" className="w-[268px] shrink-0 rounded-xl border border-wc-default bg-wc-surface-input/50 p-4 text-sm text-wc-text-muted">{t(strings.machines.loading)}</div>
                ) : (
                  <>
                    {(fleet?.joinRequests ?? []).map((request) => (
                      <JoinRequestCard key={request.id} request={request} onReview={() => { setView({ kind: "review", request }); }} />
                    ))}
                    {(fleet?.machines ?? []).map((machine) => (
                      <MachineCard key={machine.target.id} machine={machine} onStartSession={onStartSession} onManage={(item) => { setView({ kind: "grant-existing", machine: item }); }} />
                    ))}
                    {remoteMachines.length === 0 && (
                      <div data-testid="machines-empty" className="w-[268px] shrink-0 rounded-xl border border-dashed border-wc-default bg-wc-surface-base/40 p-5 text-center">
                        <p className="text-sm font-medium text-wc-text-primary">{fleet?.status === "unenrolled" ? t(strings.machines.unenrolledTitle) : t(strings.machines.emptyTitle)}</p>
                        <p className="mt-1 text-xs leading-5 text-wc-text-faint">{fleet?.message || t(strings.machines.emptyBody)}</p>
                        {fleet?.recoveryAction && <p className="mt-2 text-xs text-amber-200/80">{fleet.recoveryAction}</p>}
                      </div>
                    )}
                  </>
                )}
              </FleetRail>
            </div>
          </div>
        )}

        {view.kind === "add" && (
          <AddMachine
            requests={fleet?.joinRequests ?? []}
            code={code}
            issuing={issueCode.isPending}
            onIssueCode={handleIssueCode}
            onReview={(request) => { setView({ kind: "review", request }); }}
            onBack={goList}
            controlPlaneConsoleUrl={fleet?.controlPlane.consoleUrl ?? ""}
          />
        )}

        {view.kind === "review" && (
          <ReviewRequest
            request={view.request}
            denying={decide.isPending}
            onBack={() => { setView({ kind: "add" }); }}
            onDeny={() => { handleDeny(view.request); }}
            onContinue={() => { setView({ kind: "grant-new", request: view.request }); }}
          />
        )}

        {view.kind === "grant-new" && (
          <GrantPicker
            name={view.request.name}
            presets={presets}
            mode="link"
            busy={decide.isPending}
            onBack={() => { setView({ kind: "review", request: view.request }); }}
            onConfirm={(preset) => { handleLink(view.request, preset); }}
          />
        )}

        {view.kind === "grant-existing" && (
          <GrantPicker
            name={view.machine.target.label}
            presets={presets}
            initialPreset={view.machine.grant.preset}
            mode="manage"
            busy={setGrant.isPending}
            onBack={goList}
            onConfirm={(preset) => { handleSaveGrant(view.machine, preset); }}
          />
        )}

        {failure && (
          <div
            data-testid="machines-failure"
            role="alert"
            className="mx-auto mb-3 flex w-full max-w-4xl gap-3 rounded-xl border border-rose-400/25 bg-rose-400/10 p-3 text-sm text-rose-100"
          >
            <CircleAlert className="mt-0.5 h-4 w-4 shrink-0 text-rose-300" aria-hidden />
            <div className="min-w-0">
              <div className="font-medium">{t(strings.machines.actionFailed)}</div>
              <div className="mt-0.5 break-words text-xs text-rose-200/80">{failure}</div>
            </div>
          </div>
        )}

        {/* The footer is the architectural argument in one sentence: a machine
            is linked once, for the whole installation. */}
        <footer className="shrink-0 border-t border-wc-default bg-wc-surface-raised/95 px-5 py-3 text-xs text-wc-text-faint">
          <div className="mx-auto flex w-full max-w-4xl flex-wrap items-center justify-between gap-2">
          <span data-testid="machines-footer">{t(strings.machines.footer)}</span>
          {controlPlaneLine && (
            <span
              data-testid="machines-control-plane"
              className={`inline-flex items-center gap-1.5 ${fleet?.controlPlane.reachable ? "" : "text-amber-200/80"}`}
            >
              <Radio className="h-3 w-3" aria-hidden />
              {controlPlaneLine}
            </span>
          )}
          </div>
        </footer>
      </div>
    </FullPageDrawer>
  );
}
