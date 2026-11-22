// Settings dialog with Role Policy, Permissions, Model Pricing, Investigation, Orchestration, Notifications, and Maintenance tabs

import { useCallback, useRef, useState } from "react";
import { AppNotificationSettings } from "@vrooli/react-component-library/AppNotificationSettings/1";
import { OverlayActionBar } from "@vrooli/react-component-library/OverlayActionBar/1";
import { ResponsiveDialog } from "@vrooli/react-component-library/ResponsiveDialog/1";
import { Tabs as RclTabs } from "@vrooli/react-component-library/Tabs/1";
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
      <ResponsiveDialog
        open={open}
        onOpenChange={onOpenChange}
        title="Settings"
        closeLabel="Close"
        size="lg"
        testId="settings-dialog"
        panelClassName="sm:!w-[90vw] xl:!w-[80rem]"
        contentClassName="flex min-h-0 flex-col p-0"
        subheader={<p className="px-4 py-2 text-sm text-muted-foreground sm:px-6">{TAB_DESCRIPTIONS[activeTab]}</p>}
        footer={(
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
        )}
      >
          <RclTabs
            active={activeTab}
            onChange={setActiveTab}
            ariaLabel="Settings categories"
            className="flex min-h-0 flex-1 flex-col [&_[data-rcl-tab-panel]]:min-h-0 [&_[data-rcl-tab-panel]]:flex-1 [&_[data-rcl-tab-panel]]:overflow-y-auto"
            items={[
              { id: "roles", label: "Role Policy" },
              { id: "permissions", label: "Permissions" },
              { id: "pricing", label: "Model Pricing" },
              { id: "investigation", label: "Investigation" },
              { id: "orchestration", label: "Orchestration" },
              { id: "notifications", label: "Notifications" },
              { id: "maintenance", label: "Maintenance" },
            ]}
            testId="settings-dialog-tabs"
            panels={{
              roles: <div className="px-4 py-4 sm:px-6"><RolePolicyTab data={rolePolicy.data} loading={rolePolicy.loading} error={rolePolicy.error} /></div>,
              permissions: <div className="px-4 py-4 sm:px-6"><PermissionPolicyTab policy={permissionPolicy} /></div>,
              pricing: <div className="px-4 py-4 sm:px-6"><ModelPricingTab /></div>,
              investigation: <div className="px-4 py-4 sm:px-6"><InvestigationTab ref={investigationRef} settings={investigationSettings.data} loading={investigationSettings.loading} error={investigationSettings.error} onSave={investigationSettings.updateSettings} onReset={investigationSettings.resetSettings} onDirtyChange={setInvestigationDirty} /></div>,
              orchestration: <div className="px-4 py-4 sm:px-6"><OrchestrationTab ref={orchestrationRef} settings={orchestrationSettings.data} loading={orchestrationSettings.loading} error={orchestrationSettings.error} onSave={orchestrationSettings.updateSettings} onReset={orchestrationSettings.resetSettings} onDirtyChange={setOrchestrationDirty} /></div>,
              notifications: <div className="px-4 py-4 sm:px-6"><AppNotificationSettings /></div>,
              maintenance: <div className="px-4 py-4 sm:px-6"><MaintenanceTab purgePattern={purgePattern} onPurgePatternChange={setPurgePattern} loading={purgeLoading} error={purgeError} onPurgePreview={handlePurgePreview} /></div>,
            }}
          />

      </ResponsiveDialog>

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
