// DOC: docs/reference/cross-platform-effort/machine-linking-ux-2026-08-26.html#screen-06
//
// Incident history, alerts, infrastructure and investigations are computed from
// this computer's own collectors. A remote machine reports live vitals over the
// bridge and nothing else, so the panels that depend on local history must say
// they are not describing the machine in view — an empty "no incidents" panel
// under another machine's name reads as a finding about that machine.
//
// It renders *after* the vitals, and quietly: it explains an absence, and an
// explanation that outweighs the readings it accompanies is in the wrong place.
import { Info } from 'lucide-react';

interface LocalOnlyPanelsProps {
  machineName: string;
}

export const LocalOnlyPanels = ({ machineName }: LocalOnlyPanelsProps) => (
  <p className="machine-local-only" data-testid="local-only-panels">
    <Info size={14} aria-hidden="true" />
    <span>
      Incident history, alerts, infrastructure and investigations are produced by this
      computer&rsquo;s own collectors, so they are not shown for {machineName}. The vitals above come
      from {machineName} itself.
    </span>
  </p>
);
