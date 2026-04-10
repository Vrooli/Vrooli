import { CheckCircle2, Loader2, Settings2, XCircle } from "lucide-react";
import { Button } from "../ui/button";
import { DetailSection } from "../detail/DetailSection";
import { selectors } from "../../consts/selectors";

export interface ScenarioSettingsSectionProps {
  localGreenfield: boolean | null;
  onGreenfieldToggle: () => void;
  updatePending: boolean;
  updateError: boolean;
}

export function ScenarioSettingsSection({
  localGreenfield,
  onGreenfieldToggle,
  updatePending,
  updateError,
}: ScenarioSettingsSectionProps) {
  return (
    <DetailSection title="Scenario Settings" icon={Settings2} data-testid={selectors.scenarioDetails.metadataSection}>
      {updatePending && (
        <Loader2 className="mb-2 h-4 w-4 animate-spin text-cyan-400" />
      )}

      <div className="space-y-4">
        <div className="flex items-center justify-between rounded-lg bg-slate-700/30 p-4">
          <div className="space-y-1">
            <div className="flex items-center gap-2">
              <span className="font-medium text-slate-200">Greenfield Mode</span>
              {localGreenfield ? (
                <CheckCircle2 className="h-4 w-4 text-cyan-400" />
              ) : (
                <XCircle className="h-4 w-4 text-slate-500" />
              )}
            </div>
            <p className="text-sm text-slate-400">
              Treat this scenario as a new project without existing code base
            </p>
          </div>
          <Button
            variant={localGreenfield ? "default" : "outline"}
            size="sm"
            onClick={onGreenfieldToggle}
            disabled={updatePending}
            data-testid={selectors.scenarioDetails.greenfieldToggle}
          >
            {localGreenfield ? "Enabled" : "Disabled"}
          </Button>
        </div>
      </div>

      {updateError && (
        <div className="mt-4 rounded-lg border border-red-500/30 bg-red-500/10 p-3 text-sm text-red-400">
          Failed to update settings. Please try again.
        </div>
      )}
    </DetailSection>
  );
}
