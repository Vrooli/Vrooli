import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "../ui/button";
import type { JoinRequest, Machine } from "../../api/machines";
import type { InstallOutcome } from "../../api/capabilities";
import { strings } from "../../consts/strings";
import { humanAge } from "../machines/age";
import { GrantLine } from "../machines/grant";
import { machineDrawState, reachabilityDetail, statusPill } from "../machines/MachineList";
import { machineTestID } from "../machines/testids";
import { DeviceSilhouette } from "../terminal/device/DeviceSilhouette";
import FleetCard from "./FleetCard";
import MachineSilhouette from "./MachineSilhouette";

export function MachineCard({ machine, onManage, onConfigure, onStartSession, onInstallCapability, onReview }: { machine: Machine; onManage?: (machine: Machine) => void; onConfigure?: (machine: Machine) => void; onStartSession?: (machine: Machine) => void; onInstallCapability?: (capabilityID: string, target: Machine["target"]) => Promise<InstallOutcome>; onReview?: () => void }) {
  const { t } = useTranslation();
  const [installing, setInstalling] = useState<string[]>([]);
  const [installOutcomes, setInstallOutcomes] = useState<Record<string, InstallOutcome>>({});
  const isLocal = machine.target.kind === "local";
  const pill = statusPill(machine, t as (key: string, options?: Record<string, unknown>) => string);
  const title = isLocal ? t(strings.machines.thisComputer) : machine.target.label;
  const platform = [machine.target.os, machine.target.arch].filter(Boolean).join(" · ");
  const missingCapabilities = (machine.target.readiness ?? []).filter((fact) => fact.key.startsWith("capability:") && fact.state === "missing");
  return (
    <div data-testid={`machines-row-${machineTestID(machine.target.id)}`} className="shrink-0">
      <FleetCard
      testId={`fleet-card-machine-${machine.target.id}`}
      title={title}
      meta={[platform, reachabilityDetail(machine, t as (key: string, options?: Record<string, unknown>) => string), isLocal ? t(strings.fleet.alsoDevice) : ""].filter(Boolean).join(" · ")}
      state={pill.label}
      stateTone={isLocal ? "accent" : machine.target.available ? "accent" : "warning"}
      silhouette={isLocal
        // The local machine is the computer this console is running on — it is
        // a screen the operator is looking at, not a headless box, so it is
        // drawn by the device artwork. That is also what visually ties the
        // devices and machines sections of the drawer together.
        ? <DeviceSilhouette archetype="laptop" keyboardShare={0} kbOpen={false} screenLit />
        : <MachineSilhouette state={machineDrawState(machine)} />}
      actions={(machine.manageable && onManage) || onStartSession ? <>
        {onStartSession && <Button size="sm" data-testid={`machines-start-session-${machineTestID(machine.target.id)}`} className="min-h-11" onClick={() => { onStartSession(machine); }} disabled={!machine.target.available}>{t(strings.fleet.startSession)}</Button>}
        {machine.manageable && onManage && <Button size="sm" data-testid={`machines-manage-${machineTestID(machine.target.id)}`} className="min-h-11" onClick={() => { onManage(machine); }}>{machine.target.available ? t(strings.machines.manage) : t(strings.machines.reconnect)}</Button>}
        {machine.manageable && onConfigure && <Button size="sm" variant="outline" data-testid={`machines-configure-${machineTestID(machine.target.id)}`} className="min-h-11" onClick={() => { onConfigure(machine); }} disabled={!machine.target.available}>Configure</Button>}
      </> : undefined}
    >
        <GrantLine grant={machine.grant} />
        {(machine.drift ?? []).length > 0 && (
          <div data-testid={`machines-drift-${machineTestID(machine.target.id)}`} className="mt-2 text-xs leading-5 text-amber-200/90">
            <span className="font-medium">Configuration drift</span>
            {(machine.drift ?? []).map((item) => <div key={`${item.kind}:${item.name}`}>{item.name}: {item.reason}</div>)}
          </div>
        )}
        {!machine.target.available && machine.target.recovery_action && <span className="mt-1 block text-xs leading-5 text-amber-200/80">{machine.target.recovery_action}</span>}
        {onInstallCapability && missingCapabilities.length > 0 && (
          <div className="mt-2 flex flex-col gap-2">
            {missingCapabilities.map((fact) => {
              const capabilityID = fact.key.slice("capability:".length);
              const outcome = installOutcomes[capabilityID];
              return (
                <div key={fact.key} className="flex flex-wrap items-center gap-2">
                  <Button size="sm" variant="outline" disabled={installing.includes(capabilityID)} onClick={() => {
                    setInstalling((current) => [...current, capabilityID]);
                    // The install reports the machine's verdict, not the
                    // installer's exit code, and this card renders all three
                    // answers rather than assuming the good one.
                    void Promise.resolve(onInstallCapability(capabilityID, machine.target))
                      .catch((error: unknown): InstallOutcome => {
                        console.error("capability install failed", fact.key, error);
                        return { status: "failed" };
                      })
                      .then((result) => { setInstallOutcomes((current) => ({ ...current, [capabilityID]: result })); })
                      .finally(() => { setInstalling((current) => current.filter((id) => id !== capabilityID)); });
                  }}>
                    {installing.includes(capabilityID) ? "Installing…" : `Install ${fact.label || capabilityID}`}
                  </Button>
                  {outcome && (
                    <span
                      data-testid={`machines-install-outcome-${machineTestID(machine.target.id)}-${capabilityID}`}
                      title={outcome.message}
                      className={outcome.status === "installed" ? "text-xs text-emerald-300" : outcome.status === "failed" ? "text-xs text-red-300" : "text-xs text-amber-200"}
                    >
                      {outcome.status === "installed" ? "Installed" : outcome.status === "failed" ? "Install failed" : "Not confirmed"}
                    </span>
                  )}
                </div>
              );
            })}
          </div>
        )}
        {onReview && <Button size="sm" className="mt-2 min-h-11" onClick={onReview}>{t(strings.machines.review)}</Button>}
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
        state={t(strings.machines.review)}
        stateTone="accent"
        silhouette={<MachineSilhouette state="unenrolled" />}
        actions={<Button size="sm" className="min-h-11" data-testid={`machines-review-${machineTestID(request.id)}`} onClick={onReview}>{t(strings.machines.review)}</Button>}
      >
        {t(strings.machines.reviewDerived)}
      </FleetCard>
    </div>
  );
}

export default MachineCard;
