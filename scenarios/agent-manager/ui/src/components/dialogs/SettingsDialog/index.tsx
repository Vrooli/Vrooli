// Settings dialog with Role Policy, Permissions, Model Pricing, Investigation, Orchestration, Notifications, and Maintenance tabs

import { useCallback, useRef, useState } from "react";
import { AppNotificationSettings } from "@vrooli/react-component-library/AppNotificationSettings/1";
import { Button } from "@vrooli/react-component-library/Button/2";
import { OverlayActionBar } from "@vrooli/react-component-library/OverlayActionBar/1";
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
import { RolePolicyTab } from "./RolePolicyTab";
import { PermissionPolicyTab } from "./PermissionPolicyTab";
import { useInvestigationSettings, useMaintenance, usePermissionPolicy, useRolePolicyCatalog } from "../../../hooks/useApi";
import { useOrchestrationSettings } from "../../../hooks/useOrchestrationSettings";
import { overlayActions } from "../../../lib/overlayActions";
import { PurgeTarget } from "@vrooli/proto-types/agent-manager/v1/api/service_pb";

const TAB_DESCRIPTIONS: Record<string, string> = {
  roles: "Inspect the active Git-managed role policy catalog and activation state",
  permissions: "Inspect and reconcile global portable coding-agent permissions",
  pricing: "View and manage model pricing with overrides",
  investigation: "Configure investigation and apply-fix agent behavior",
  orchestration: "Configure run lifecycle, safety, health detection, and termination behavior",
  notifications: "Choose which alerts reach this phone or browser, and how urgent they must be",
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
  const [activeTab, setActiveTab] = useState("roles");

  // API hooks
  const rolePolicy = useRolePolicyCatalog({ enabled: open });
  const permissionPolicy = usePermissionPolicy({ enabled: open });
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
              <TabsList className="flex w-full overflow-x-auto no-scrollbar sm:grid sm:grid-cols-7">
                <TabsTrigger value="roles" className="shrink-0">Role Policy</TabsTrigger>
                <TabsTrigger value="permissions" className="shrink-0">Permissions</TabsTrigger>
                <TabsTrigger value="pricing" className="shrink-0">Model Pricing</TabsTrigger>
                <TabsTrigger value="investigation" className="shrink-0">Investigation</TabsTrigger>
                <TabsTrigger value="orchestration" className="shrink-0">Orchestration</TabsTrigger>
                <TabsTrigger value="notifications" className="shrink-0">Notifications</TabsTrigger>
                <TabsTrigger value="maintenance" className="shrink-0">Maintenance</TabsTrigger>
              </TabsList>
            </div>

            {/* Scrollable tab content */}
            <div className="flex-1 min-h-0 overflow-y-auto px-4 sm:px-6 py-4">
              <TabsContent value="roles" className="mt-0">
                <RolePolicyTab data={rolePolicy.data} loading={rolePolicy.loading} error={rolePolicy.error} />
              </TabsContent>
              <TabsContent value="permissions" className="mt-0">
                <PermissionPolicyTab policy={permissionPolicy} />
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
              <TabsContent value="notifications" className="mt-0">
                <AppNotificationSettings />
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

          <DialogFooter>
            <OverlayActionBar className="flex-row-reverse" actions={
              activeTab === "orchestration" && orchestrationDirty
                ? overlayActions(
                    { id: "save", label: orchestrationRef.current?.saving ? "Saving..." : "Save", onSelect: () => orchestrationRef.current?.save(), disabled: orchestrationRef.current?.saving, primary: true },
                    { id: "reset", label: "Reset to Defaults", onSelect: () => orchestrationRef.current?.reset(), disabled: orchestrationRef.current?.saving },
                    { id: "close", label: "Close", onSelect: () => onOpenChange(false) },
                  )
                : activeTab === "investigation" && investigationDirty
                  ? overlayActions(
                      { id: "save", label: investigationRef.current?.saving ? "Saving..." : "Save", onSelect: () => investigationRef.current?.save(), disabled: investigationRef.current?.saving, primary: true },
                      { id: "reset", label: "Reset to Defaults", onSelect: () => investigationRef.current?.reset(), disabled: investigationRef.current?.saving },
                      { id: "close", label: "Close", onSelect: () => onOpenChange(false) },
                    )
                  : overlayActions({ id: "close", label: "Close", onSelect: () => onOpenChange(false), primary: true })
            } />
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
