import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { CircleAlert, Radio } from "lucide-react";
import { FullPageDrawer } from "@vrooli/react-component-library/FullPageDrawer/1";
import { strings } from "../../consts/strings";
import { useFleet, useFleetMutations } from "../../hooks/useFleet";
import type { IssuedCode, JoinRequest, Machine } from "../../api/machines";
import MachineList from "./MachineList";
import AddMachine from "./AddMachine";
import ReviewRequest from "./ReviewRequest";
import GrantPicker from "./GrantPicker";

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

interface MachinesDrawerProps {
  open: boolean;
  onClose: () => void;
}

export default function MachinesDrawer({ open, onClose }: MachinesDrawerProps) {
  const { t } = useTranslation();
  const [view, setView] = useState<View>({ kind: "list" });
  const [code, setCode] = useState<IssuedCode | null>(null);
  const [failure, setFailure] = useState("");

  const fleetQuery = useFleet(open);
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

  return (
    <FullPageDrawer
      avoidKeyboard
      open={open}
      onClose={onClose}
      closeLabel={t(strings.machines.closeAriaLabel)}
      title={t(strings.machines.title)}
      testId="machines-drawer"
    >
      <div className="flex h-full min-h-0 flex-col">
        {view.kind === "list" && (
          <MachineList
            fleet={fleet}
            loading={fleetQuery.isLoading}
            refreshing={fleetQuery.isFetching && !fleetQuery.isLoading}
            onRefresh={() => { void fleetQuery.refetch(); }}
            onAdd={() => { setView({ kind: "add" }); }}
            onManage={(machine) => { setView({ kind: "grant-existing", machine }); }}
            onReview={(request) => { setView({ kind: "review", request }); }}
          />
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
