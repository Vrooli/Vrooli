// Settings dialog with Model Registry, Model Pricing, Maintenance, and Investigation tabs

import { useCallback, useRef, useState } from "react";
import { Button } from "../../ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../../ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../../ui/tabs";
import { PurgeConfirmDialog, type PurgePreview } from "../PurgeConfirmDialog";
import { InvestigationTab } from "./InvestigationTab";
import type { InvestigationTabHandle } from "./InvestigationTab";
import { OrchestrationTab } from "./OrchestrationTab";
import type { OrchestrationTabHandle } from "./OrchestrationTab";
import { MaintenanceTab } from "./MaintenanceTab";
import { ModelPricingTab } from "./ModelPricingTab";
import { ModelPolicyTab } from "./ModelPolicyTab";
import { useInvestigationSettings, useMaintenance, useModelPolicyCatalog } from "../../../hooks/useApi";
import { useOrchestrationSettings } from "../../../hooks/useOrchestrationSettings";
import { PurgeTarget } from "@vrooli/proto-types/agent-manager/v1/api/service_pb";

const TAB_DESCRIPTIONS: Record<string, string> = {
  models: "Inspect the active Git-managed model policy catalog and activation state",
  pricing: "View and manage model pricing with overrides",
  investigation: "Configure investigation and apply-fix agent behavior",
  orchestration: "Configure run lifecycle, safety, health detection, and termination behavior",
  maintenance: "Purge data and manage service controls",
};

interface SettingsDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onPurgeComplete: () => void;
}

export function SettingsDialog({
  open,
  onOpenChange,
  onPurgeComplete,
}: SettingsDialogProps) {
  const [activeTab, setActiveTab] = useState("models");

  // API hooks
  const modelPolicy = useModelPolicyCatalog({ enabled: open });
  const maintenance = useMaintenance();
  const investigationSettings = useInvestigationSettings();

  // Investigation ref + dirty state for unified footer
  const investigationRef = useRef<InvestigationTabHandle>(null);
  const [investigationDirty, setInvestigationDirty] = useState(false);

  // Orchestration ref + dirty state for unified footer
  const orchestrationSettings = useOrchestrationSettings();
  const orchestrationRef = useRef<OrchestrationTabHandle>(null);
  const [orchestrationDirty, setOrchestrationDirty] = useState(false);

  // Purge state
  const [purgePattern, setPurgePattern] = useState("^test-.*");
  const [purgeError, setPurgeError] = useState<string | null>(null);
  const [purgeLoading, setPurgeLoading] = useState(false);
  const [purgeConfirmOpen, setPurgeConfirmOpen] = useState(false);
  const [purgePreview, setPurgePreview] = useState<PurgePreview | null>(null);
  const [purgeTargets, setPurgeTargets] = useState<PurgeTarget[]>([]);
  const [purgeActionLabel, setPurgeActionLabel] = useState("");

  const handlePurgePreview = useCallback(
    async (targets: PurgeTarget[], label: string) => {
      setPurgeError(null);
      setPurgeLoading(true);
      try {
        const counts = await maintenance.previewPurge(purgePattern, targets);
        setPurgePreview({
          profiles: counts.profiles ?? 0,
          tasks: counts.tasks ?? 0,
          runs: counts.runs ?? 0,
        });
        setPurgeTargets(targets);
        setPurgeActionLabel(label);
        setPurgeConfirmOpen(true);
      } catch (err) {
        setPurgeError((err as Error).message);
      } finally {
        setPurgeLoading(false);
      }
    },
    [maintenance, purgePattern]
  );

  const handlePurgeExecute = useCallback(async () => {
    setPurgeError(null);
    setPurgeLoading(true);
    try {
      await maintenance.executePurge(purgePattern, purgeTargets);
      setPurgeConfirmOpen(false);
      onOpenChange(false);
      onPurgeComplete();
    } catch (err) {
      setPurgeError((err as Error).message);
    } finally {
      setPurgeLoading(false);
    }
  }, [maintenance, purgePattern, purgeTargets, onOpenChange, onPurgeComplete]);

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange} fullScreenMobile>
        <DialogContent fullScreenMobile className="sm:max-w-[90vw] xl:max-w-7xl">
          <DialogHeader onClose={() => onOpenChange(false)} className="p-4 sm:p-6">
            <DialogTitle>Settings</DialogTitle>
            <DialogDescription>{TAB_DESCRIPTIONS[activeTab]}</DialogDescription>
          </DialogHeader>

          <Tabs value={activeTab} onValueChange={setActiveTab} className="flex flex-col flex-1 min-h-0">
            {/* Tab bar — sticky, scrollable on mobile */}
            <div className="px-4 sm:px-6 pb-2 pt-1 shrink-0 border-b border-border">
              <TabsList className="flex w-full overflow-x-auto no-scrollbar sm:grid sm:grid-cols-5">
                <TabsTrigger value="models" className="shrink-0">Model Policy</TabsTrigger>
                <TabsTrigger value="pricing" className="shrink-0">Model Pricing</TabsTrigger>
                <TabsTrigger value="investigation" className="shrink-0">Investigation</TabsTrigger>
                <TabsTrigger value="orchestration" className="shrink-0">Orchestration</TabsTrigger>
                <TabsTrigger value="maintenance" className="shrink-0">Maintenance</TabsTrigger>
              </TabsList>
            </div>

            {/* Scrollable tab content */}
            <div className="flex-1 min-h-0 overflow-y-auto px-4 sm:px-6 py-4">
              <TabsContent value="models" className="mt-0">
                <ModelPolicyTab data={modelPolicy.data} loading={modelPolicy.loading} error={modelPolicy.error} />
              </TabsContent>
              <TabsContent value="pricing" className="mt-0">
                <ModelPricingTab />
              </TabsContent>
              <TabsContent value="investigation" className="mt-0">
                <InvestigationTab
                  ref={investigationRef}
                  settings={investigationSettings.data}
                  loading={investigationSettings.loading}
                  error={investigationSettings.error}
                  onSave={investigationSettings.updateSettings}
                  onReset={investigationSettings.resetSettings}
                  onDirtyChange={setInvestigationDirty}
                />
              </TabsContent>
              <TabsContent value="orchestration" className="mt-0">
                <OrchestrationTab
                  ref={orchestrationRef}
                  settings={orchestrationSettings.data}
                  loading={orchestrationSettings.loading}
                  error={orchestrationSettings.error}
                  onSave={orchestrationSettings.updateSettings}
                  onReset={orchestrationSettings.resetSettings}
                  onDirtyChange={setOrchestrationDirty}
                />
              </TabsContent>
              <TabsContent value="maintenance" className="mt-0">
                <MaintenanceTab
                  purgePattern={purgePattern}
                  onPurgePatternChange={setPurgePattern}
                  loading={purgeLoading}
                  error={purgeError}
                  onPurgePreview={handlePurgePreview}
                />
              </TabsContent>
            </div>
          </Tabs>

          <DialogFooter className="p-4 sm:p-6">
            {activeTab === "orchestration" && orchestrationDirty && (
              <>
                <Button
                  variant="outline"
                  onClick={() => orchestrationRef.current?.reset()}
                  disabled={orchestrationRef.current?.saving}
                >
                  Reset to Defaults
                </Button>
                <Button
                  onClick={() => orchestrationRef.current?.save()}
                  disabled={orchestrationRef.current?.saving}
                >
                  {orchestrationRef.current?.saving ? "Saving..." : "Save"}
                </Button>
              </>
            )}
            {activeTab === "investigation" && investigationDirty && (
              <>
                <Button
                  variant="outline"
                  onClick={() => investigationRef.current?.reset()}
                  disabled={investigationRef.current?.saving}
                >
                  Reset to Defaults
                </Button>
                <Button
                  onClick={() => investigationRef.current?.save()}
                  disabled={investigationRef.current?.saving}
                >
                  {investigationRef.current?.saving ? "Saving..." : "Save"}
                </Button>
              </>
            )}
            <Button variant="outline" onClick={() => onOpenChange(false)}>
              Close
            </Button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <PurgeConfirmDialog
        open={purgeConfirmOpen}
        onOpenChange={setPurgeConfirmOpen}
        actionLabel={purgeActionLabel}
        pattern={purgePattern}
        preview={purgePreview}
        loading={purgeLoading}
        error={purgeError}
        onConfirm={handlePurgeExecute}
      />
    </>
  );
}
