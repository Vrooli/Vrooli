// Settings dialog with Model Registry and Maintenance tabs

import { useCallback, useState } from "react";
import { Button } from "../../ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "../../ui/dialog";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "../../ui/tabs";
import { PurgeConfirmDialog, type PurgePreview } from "../PurgeConfirmDialog";
import { MaintenanceTab } from "./MaintenanceTab";
import { ModelRegistryTab } from "./ModelRegistryTab";
import { useModelRegistryEditor } from "../../../hooks/useModelRegistryEditor";
import { useMaintenance, useModelRegistry, useRunners } from "../../../hooks/useApi";
import { PurgeTarget } from "@vrooli/proto-types/agent-manager/v1/api/service_pb";

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
  const modelRegistry = useModelRegistry();
  const runners = useRunners();
  const maintenance = useMaintenance();

  // Model registry editor
  const editor = useModelRegistryEditor({
    data: modelRegistry.data,
    isActive: open,
    updateRegistry: modelRegistry.updateRegistry,
  });

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
      <Dialog open={open} onOpenChange={onOpenChange} contentClassName="max-w-5xl">
        <DialogContent>
          <DialogHeader>
            <DialogTitle>Settings</DialogTitle>
            <DialogDescription>Manage model registry configuration and maintenance actions.</DialogDescription>
          </DialogHeader>
          <DialogBody className="space-y-6">
            <Tabs value={activeTab} onValueChange={setActiveTab}>
              <TabsList className="grid w-full grid-cols-2">
                <TabsTrigger value="models">Model Registry</TabsTrigger>
                <TabsTrigger value="maintenance">Maintenance</TabsTrigger>
              </TabsList>
              <TabsContent value="models" className="mt-4">
                <ModelRegistryTab
                  draft={editor.draft}
                  loading={modelRegistry.loading}
                  loadError={modelRegistry.error}
                  error={editor.error}
                  newRunnerKey={editor.newRunnerKey}
                  onNewRunnerKeyChange={editor.setNewRunnerKey}
                  knownRunners={runners.data ?? undefined}
                  onAddRunner={editor.addRunner}
                  onRemoveRunner={editor.removeRunner}
                  onAddFallbackRunner={editor.addFallbackRunner}
                  onUpdateFallbackRunner={editor.updateFallbackRunner}
                  onRemoveFallbackRunner={editor.removeFallbackRunner}
                  onAddModel={editor.addModel}
                  onRemoveModel={editor.removeModel}
                  onUpdateModel={editor.updateModel}
                  onUpdatePreset={editor.updatePreset}
                />
              </TabsContent>
              <TabsContent value="maintenance" className="mt-4">
                <MaintenanceTab
                  purgePattern={purgePattern}
                  onPurgePatternChange={setPurgePattern}
                  loading={purgeLoading}
                  error={purgeError}
                  onPurgePreview={handlePurgePreview}
                />
              </TabsContent>
            </Tabs>
          </DialogBody>
          <DialogFooter>
            {activeTab === "models" ? (
              <>
                <Button variant="outline" onClick={editor.reset} disabled={!editor.draft}>
                  Reset
                </Button>
                <Button
                  onClick={editor.save}
                  disabled={!editor.draft || editor.saving}
                >
                  {editor.saving ? "Saving..." : "Save"}
                </Button>
                <Button variant="outline" onClick={() => onOpenChange(false)}>
                  Close
                </Button>
              </>
            ) : (
              <Button variant="outline" onClick={() => onOpenChange(false)}>
                Close
              </Button>
            )}
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
