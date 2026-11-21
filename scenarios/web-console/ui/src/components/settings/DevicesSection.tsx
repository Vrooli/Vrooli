import type { InstallOutcome } from "../../api/capabilities";
import type { Machine } from "../../api/machines";
import FleetPanel from "../fleet/FleetPanel";

interface DevicesSectionProps {
  /** True while the settings modal (and this tab) is on screen. */
  open: boolean;
  onRequestClose: () => void;
  /** Start a session on a machine — closes settings and opens the launcher. */
  onStartSessionForMachine?: (machine: Machine) => void;
  onInstallCapability?: (capabilityID: string, target: Machine["target"]) => Promise<InstallOutcome>;
}

/**
 * Devices settings tab. Machines used to be reached from a dedicated nav icon;
 * folding them here keeps the top bar to what is used mid-session (new session,
 * settings, pending approvals) while the fleet view stays one click deep. The
 * body is {@link FleetPanel}, shared with the launcher's drawer entrance.
 */
export default function DevicesSection({
  open,
  onRequestClose,
  onStartSessionForMachine,
  onInstallCapability,
}: DevicesSectionProps) {
  return (
    <div className="h-full min-h-0">
      <FleetPanel
        open={open}
        onStartSession={onStartSessionForMachine
          ? (machine) => { onRequestClose(); onStartSessionForMachine(machine); }
          : undefined}
        onInstallCapability={onInstallCapability}
      />
    </div>
  );
}
