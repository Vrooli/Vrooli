import { useState } from "react";
import { Settings, X } from "lucide-react";
import { Button } from "./ui/button";
import { useIsMobile } from "../hooks";
import { useHealthIssueCount } from "../lib/hooks";
import { SettingsTabLayout } from "./SettingsTabLayout";
import { SettingsTabCredentials } from "./SettingsTabCredentials";
import { SettingsTabHealth } from "./SettingsTabHealth";
import { SettingsTabIntegrations } from "./SettingsTabIntegrations";
import { SettingsTabStorage } from "./SettingsTabStorage";
import { SettingsTabGrouping } from "./SettingsTabGrouping";
import { SettingsTabPrecommit } from "./SettingsTabPrecommit";
import type { LayoutPreset, LayoutSection } from "./LayoutSettingsModal";
import type { ChangeGroupAPI, SyncStatusResponse } from "../lib/api";
import type { GroupingRule } from "./FileList";

export type SettingsTab = "layout" | "grouping" | "credentials" | "integrations" | "precommit" | "health" | "storage";

interface SettingsModalProps {
  isOpen: boolean;
  repoDir?: string;
  repoId?: string | null;
  syncStatus?: SyncStatusResponse;
  // Layout props
  preset: LayoutPreset;
  primaryPanel: LayoutSection;
  onChangePreset: (preset: LayoutPreset) => void;
  onChangePrimary: (panel: LayoutSection) => void;
  onResetLayout: () => void;
  // Grouping props
  groupingEnabled: boolean;
  onToggleGrouping: () => void;
  groupingRules: GroupingRule[];
  onChangeGroupingRules: (rules: GroupingRule[]) => void;
  contractGroups?: ChangeGroupAPI[];
  // Common
  onClose: () => void;
  initialTab?: SettingsTab;
}

const tabLabels: Record<SettingsTab, string> = {
  layout: "Layout",
  grouping: "Grouping",
  credentials: "Credentials",
  integrations: "Integrations",
  precommit: "Precommit",
  health: "Health",
  storage: "Storage",
};

export function SettingsModal({
  isOpen,
  repoDir,
  repoId,
  syncStatus,
  preset,
  primaryPanel,
  onChangePreset,
  onChangePrimary,
  onResetLayout,
  groupingEnabled,
  onToggleGrouping,
  groupingRules,
  onChangeGroupingRules,
  contractGroups,
  onClose,
  initialTab = "layout"
}: SettingsModalProps) {
  const isMobile = useIsMobile();
  const [activeTab, setActiveTab] = useState<SettingsTab>(initialTab);
  // Actionable findings only -- see useHealthIssueCount for why informational
  // items are excluded from the badge.
  const healthIssueCount = useHealthIssueCount(repoId);

  if (!isOpen) return null;

  const remoteUrl = syncStatus?.remote_url;
  const hasUpstream = syncStatus?.has_upstream ?? false;

  // Mobile: full-screen modal
  if (isMobile) {
    return (
      <div
        className="fixed inset-0 z-50 flex flex-col bg-slate-950 animate-in slide-in-from-bottom duration-200"
        role="dialog"
        aria-modal="true"
        aria-label="Settings"
      >
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-4 pt-safe">
          <div className="flex items-center gap-3">
            <Settings className="h-5 w-5 text-slate-400" />
            <h2 className="text-base font-semibold text-slate-100">Settings</h2>
          </div>
          <button
            type="button"
            className="h-11 w-11 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800/60 active:bg-slate-700 touch-target"
            onClick={onClose}
            aria-label="Close"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        {/* Tab Navigation */}
        <div className="flex border-b border-slate-800 px-4 overflow-x-auto">
          {(Object.keys(tabLabels) as SettingsTab[]).map((tab) => (
            <button
              key={tab}
              type="button"
              onClick={() => setActiveTab(tab)}
              className={`px-4 py-3 text-sm font-medium border-b-2 transition-colors whitespace-nowrap ${
                activeTab === tab
                  ? "text-blue-400 border-blue-400"
                  : "text-slate-400 border-transparent hover:text-slate-200"
              }`}
            >
              {tabLabels[tab]}
              {tab === "health" && healthIssueCount > 0 && (
                <span
                  className="ml-1.5 inline-flex items-center justify-center rounded-full bg-amber-500/20 px-1.5 py-0.5 text-[10px] font-semibold leading-none text-amber-300"
                  aria-label={`${healthIssueCount} health ${healthIssueCount === 1 ? "issue" : "issues"} need attention`}
                >
                  {healthIssueCount}
                </span>
              )}
            </button>
          ))}
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto px-4 py-6">
          {repoDir && (
            <p className="text-xs text-slate-500 mb-4">Repo: {repoDir}</p>
          )}

          {activeTab === "layout" && (
            <SettingsTabLayout
              preset={preset}
              primaryPanel={primaryPanel}
              onChangePreset={onChangePreset}
              onChangePrimary={onChangePrimary}
              onReset={onResetLayout}
              isMobile={true}
            />
          )}

          {activeTab === "grouping" && (
            <SettingsTabGrouping
              groupingEnabled={groupingEnabled}
              onToggleGrouping={onToggleGrouping}
              rules={groupingRules}
              onChangeRules={onChangeGroupingRules}
              isMobile={true}
              contractGroups={contractGroups}
            />
          )}

          {activeTab === "credentials" && (
            <SettingsTabCredentials
              remoteUrl={remoteUrl}
              hasUpstream={hasUpstream}
              isMobile={true}
              repoId={repoId}
            />
          )}

          {activeTab === "integrations" && (
            <SettingsTabIntegrations
              isMobile={true}
              repoId={repoId}
            />
          )}

          {activeTab === "precommit" && (
            <SettingsTabPrecommit
              isMobile={true}
              repoId={repoId}
            />
          )}

          {activeTab === "health" && (
            <SettingsTabHealth
              isMobile={true}
              repoId={repoId}
            />
          )}

          {activeTab === "storage" && (
            <SettingsTabStorage
              isMobile={true}
              repoId={repoId}
            />
          )}
        </div>

        {/* Footer */}
        <div className="border-t border-slate-800 px-4 py-4 pb-safe">
          <Button
            variant="default"
            size="sm"
            onClick={onClose}
            className="w-full h-12 text-sm touch-target"
          >
            Done
          </Button>
        </div>
      </div>
    );
  }

  // Desktop: centered modal
  return (
    <div
      className="fixed inset-0 z-50 flex items-center justify-center bg-slate-950/80 px-4"
      role="dialog"
      aria-modal="true"
      aria-label="Settings"
    >
      <div className="w-full max-w-2xl max-h-[80vh] flex flex-col rounded-xl border border-slate-800 bg-slate-950 shadow-xl">
        {/* Header */}
        <div className="flex items-center justify-between border-b border-slate-800 px-4 py-3">
          <div className="flex items-center gap-2">
            <Settings className="h-4 w-4 text-slate-400" />
            <div>
              <h2 className="text-sm font-semibold text-slate-100">Settings</h2>
              {repoDir && (
                <p className="text-[11px] text-slate-500 mt-1">Repo: {repoDir}</p>
              )}
            </div>
          </div>
          <button
            type="button"
            className="h-8 w-8 inline-flex items-center justify-center rounded-full border border-slate-700 text-slate-300 hover:bg-slate-800/60"
            onClick={onClose}
            aria-label="Close settings"
          >
            <X className="h-4 w-4" />
          </button>
        </div>

        {/* Tab Navigation */}
        <div className="flex border-b border-slate-800 px-4">
          {(Object.keys(tabLabels) as SettingsTab[]).map((tab) => (
            <button
              key={tab}
              type="button"
              onClick={() => setActiveTab(tab)}
              className={`px-4 py-2 text-xs font-medium border-b-2 transition-colors ${
                activeTab === tab
                  ? "text-blue-400 border-blue-400"
                  : "text-slate-400 border-transparent hover:text-slate-200"
              }`}
            >
              {tabLabels[tab]}
              {tab === "health" && healthIssueCount > 0 && (
                <span
                  className="ml-1.5 inline-flex items-center justify-center rounded-full bg-amber-500/20 px-1.5 py-0.5 text-[10px] font-semibold leading-none text-amber-300"
                  aria-label={`${healthIssueCount} health ${healthIssueCount === 1 ? "issue" : "issues"} need attention`}
                >
                  {healthIssueCount}
                </span>
              )}
            </button>
          ))}
        </div>

        {/* Content */}
        <div className="flex-1 overflow-y-auto px-4 py-4 min-h-[300px]">
          {activeTab === "layout" && (
            <SettingsTabLayout
              preset={preset}
              primaryPanel={primaryPanel}
              onChangePreset={onChangePreset}
              onChangePrimary={onChangePrimary}
              onReset={onResetLayout}
              isMobile={false}
            />
          )}

          {activeTab === "grouping" && (
            <SettingsTabGrouping
              groupingEnabled={groupingEnabled}
              onToggleGrouping={onToggleGrouping}
              rules={groupingRules}
              onChangeRules={onChangeGroupingRules}
              isMobile={false}
              contractGroups={contractGroups}
            />
          )}

          {activeTab === "credentials" && (
            <SettingsTabCredentials
              remoteUrl={remoteUrl}
              hasUpstream={hasUpstream}
              isMobile={false}
              repoId={repoId}
            />
          )}

          {activeTab === "integrations" && (
            <SettingsTabIntegrations
              isMobile={false}
              repoId={repoId}
            />
          )}

          {activeTab === "precommit" && (
            <SettingsTabPrecommit
              isMobile={false}
              repoId={repoId}
            />
          )}

          {activeTab === "health" && (
            <SettingsTabHealth
              isMobile={false}
              repoId={repoId}
            />
          )}

          {activeTab === "storage" && (
            <SettingsTabStorage
              isMobile={false}
              repoId={repoId}
            />
          )}
        </div>

        {/* Footer */}
        <div className="flex items-center justify-end border-t border-slate-800 px-4 py-3">
          <Button variant="outline" size="sm" onClick={onClose} className="h-8 px-3">
            Done
          </Button>
        </div>
      </div>
    </div>
  );
}
