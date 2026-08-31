import { useTranslation } from "react-i18next";
import { Button } from "../ui/button";
import type { JoinRequest, Machine } from "../../api/machines";
import { strings } from "../../consts/strings";
import { humanAge } from "../machines/age";
import { GrantLine } from "../machines/grant";
import { machineDrawState, reachabilityDetail, statusPill } from "../machines/MachineList";
import { machineTestID } from "../machines/testids";
import { DeviceSilhouette } from "../terminal/device/DeviceSilhouette";
import FleetCard from "./FleetCard";
import MachineSilhouette from "./MachineSilhouette";

export function MachineCard({ machine, onManage, onStartSession, onInstallCapability, onReview }: { machine: Machine; onManage?: (machine: Machine) => void; onStartSession?: (machine: Machine) => void; onInstallCapability?: (capabilityID: string, target: Machine["target"]) => void; onReview?: () => void }) {
  const { t } = useTranslation();
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
          <div className="mt-2 flex flex-wrap gap-2">
            {missingCapabilities.map((fact) => (
              <Button key={fact.key} size="sm" variant="outline" onClick={() => { onInstallCapability(fact.key.slice("capability:".length), machine.target); }}>
                Install {fact.label || fact.key.slice("capability:".length)}
              </Button>
            ))}
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
