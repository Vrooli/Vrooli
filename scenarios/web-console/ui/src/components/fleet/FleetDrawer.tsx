import type { InstallOutcome } from "../../api/capabilities";
import { useTranslation } from "react-i18next";
import { FullPageDrawer } from "@vrooli/react-component-library/FullPageDrawer/1";
import { strings } from "../../consts/strings";
import type { Machine } from "../../api/machines";
import FleetPanel from "./FleetPanel";

/**
 * The machines surface, as a drawer.
 *
 * A machine is linked once for the whole installation — not once per app that
 * wants to reach it — so this lives in the app the person already has open
 * rather than behind a detour through the control plane's own interface.
 *
 * This is the launcher's "manage machines" entrance; the same body also renders
 * inside Settings → Devices. Both share {@link FleetPanel} so linking,
 * approving, and configuring a machine have one implementation.
 */

interface FleetDrawerProps {
  open: boolean;
  onClose: () => void;
  onStartSession?: (machine: Machine) => void;
  onInstallCapability?: (capabilityID: string, target: Machine["target"]) => Promise<InstallOutcome>;
}

export default function FleetDrawer({ open, onClose, onStartSession, onInstallCapability }: FleetDrawerProps) {
  const { t } = useTranslation();
  return (
    <FullPageDrawer
      avoidKeyboard
      open={open}
      onClose={onClose}
      closeLabel={t(strings.machines.closeAriaLabel)}
      title={t(strings.fleet.title)}
      testId="machines-drawer"
    >
      <FleetPanel open={open} onStartSession={onStartSession} onInstallCapability={onInstallCapability} />
    </FullPageDrawer>
  );
}
