import { Button } from "@vrooli/react-component-library/Button/2";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

export type CapabilityGateProps = {
  scope: string;
  withheld: string;
  unblock: string;
  expiresAt: string;
  viewerIsOwner: boolean;
  onAnswer?: (grant: boolean) => void;
};

export function CapabilityGate({ scope, withheld, unblock, expiresAt, viewerIsOwner, onAnswer }: CapabilityGateProps) {
  const { t } = useTranslation();
  return (
    <section data-testid="capability-gate" role="alert" className="flex flex-col gap-2 rounded-lg border p-4">
      <h3 className="font-semibold">{t(strings.console.capabilityGate.title)}</h3>
      <p data-testid="capability-gate-withheld"><strong>{t(strings.console.capabilityGate.withheld)}</strong> {withheld} ({scope})</p>
      <p data-testid="capability-gate-unblock"><strong>{t(strings.console.capabilityGate.unblock)}</strong> {unblock}</p>
      <p data-testid="capability-gate-expiry" role="status"><strong>{t(strings.console.capabilityGate.expires)}</strong> {expiresAt}</p>
      {viewerIsOwner ? (
        <div className="flex gap-2">
          <Button type="button" data-testid="capability-gate-grant" onClick={() => onAnswer?.(true)}>{t(strings.console.capabilityGate.grantOnce)}</Button>
          <Button type="button" variant="secondary" data-testid="capability-gate-deny" onClick={() => onAnswer?.(false)}>{t(strings.console.capabilityGate.notNow)}</Button>
        </div>
      ) : (
        <p data-testid="capability-gate-permission-denied" className="text-sm text-app-muted-foreground">{t(strings.console.capabilityGate.ownerOnly)}</p>
      )}
    </section>
  );
}
