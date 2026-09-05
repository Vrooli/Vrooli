import { useState } from "react";
import { useTranslation } from "react-i18next";
import { ArrowLeft } from "lucide-react";
import { IconButton } from "@vrooli/react-component-library/IconButton/3";
import { StatusBadge, type StatusTone } from "@vrooli/react-component-library/StatusBadge/1";
import { Tabs } from "@vrooli/react-component-library/Tabs/1";
import { strings } from "../../consts/strings";
import type { Machine, PermissionPreset } from "../../api/machines";
import { GrantLine } from "../machines/grant";
import { machineIssues, reachabilityDetail, statusBadge } from "../machines/MachineList";
import { machineTestID } from "../machines/testids";
import PermissionsTab from "../machines/PermissionsTab";
import { ConfigurationTab } from "./ConfigurationTab";
import { MachineActivityTab } from "./MachineActivityTab";

/**
 * One machine, four tabs.
 *
 * `Manage` and `Configure` used to be two peer buttons on a 268px card, opening
 * two unrelated full-screen views. They are genuinely two different objects —
 * permission is held by this console, configuration is held by the node over
 * the bridge — but that is a fact about ownership, not about how a person
 * thinks. Both are *settings for this machine*, and the split cost the card its
 * third action slot, which is why `Configure` was clipped.
 *
 * So the card gets one way in and this sheet holds the sections. Nothing here
 * is new function; it is the same two panels given a spine, plus an Overview
 * that answers the questions the card no longer has room for.
 */

export type MachineTab = "overview" | "permissions" | "configuration" | "activity";

const TAB_ORDER: MachineTab[] = ["overview", "permissions", "configuration", "activity"];

export function MachineDetail({
  machine,
  presets,
  initialTab = "overview",
  savingGrant,
  onSaveGrant,
  onBack,
}: {
  machine: Machine;
  presets: PermissionPreset[];
  initialTab?: MachineTab;
  savingGrant: boolean;
  onSaveGrant: (machine: Machine, preset: string) => void;
  onBack: () => void;
}) {
  const { t } = useTranslation();
  const translate = t as (key: string, options?: Record<string, unknown>) => string;
  const [tab, setTab] = useState<MachineTab>(initialTab);
  const badge = statusBadge(machine, translate);
  const issues = machineIssues(machine);
  const isLocal = machine.target.kind === "local";
  const title = isLocal ? t(strings.machines.thisComputer) : machine.target.label;

  const label: Record<MachineTab, string> = {
    overview: t(strings.machines.tabOverview),
    permissions: t(strings.machines.tabPermissions),
    configuration: t(strings.machines.tabConfiguration),
    activity: t(strings.machines.tabActivity),
  };

  return (
    <section
      className="flex min-h-0 flex-1 flex-col"
      data-testid={`machine-detail-${machineTestID(machine.target.id)}`}
      aria-label={title}
    >
      <div className="mx-auto flex w-full max-w-3xl items-center gap-3 px-5 pt-5">
        <IconButton
          data-testid="machine-detail-back"
          onClick={onBack}
          aria-label={t(strings.machines.back)}
          shape="rounded"
        >
          <ArrowLeft aria-hidden />
        </IconButton>
        <div className="min-w-0 flex-1">
          <h2 className="truncate text-lg font-semibold text-wc-text-primary">{title}</h2>
          <p className="truncate text-xs text-wc-text-faint">
            {[machine.target.os, machine.target.arch].filter(Boolean).join(" · ")}
          </p>
        </div>
        <StatusBadge tone={badge.tone as StatusTone}>{badge.label}</StatusBadge>
      </div>

      <div className="mx-auto mt-4 w-full max-w-3xl px-5">
        <Tabs
          ariaLabel={title}
          active={tab}
          onChange={(next) => { setTab(next as MachineTab); }}
          itemTestId={(item) => `machine-detail-tab-${item}`}
          items={TAB_ORDER.map((id) => ({
            id,
            label: label[id],
            // The count rides on the tab that owns it, so the card's "6 need
            // attention" and this strip cannot disagree about where to look.
            badge: id === "configuration" && issues.count > 0 ? issues.count : undefined,
          }))}
        />
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto">
        <div className="mx-auto w-full max-w-3xl px-5 pb-5 pt-4">
          {tab === "overview" && (
            <dl className="grid gap-3 sm:grid-cols-2" data-testid="machine-detail-overview">
              <Fact term={t(strings.machines.overviewPlatform)}>
                {[machine.target.os, machine.target.arch].filter(Boolean).join(" · ") || "—"}
              </Fact>
              <Fact term={t(strings.machines.overviewReachability)}>
                {reachabilityDetail(machine, translate)}
              </Fact>
              <Fact term={t(strings.machines.overviewEndpoint)}>
                <code className="break-all text-[11px]">{machine.target.id}</code>
              </Fact>
              <Fact term={t(strings.machines.overviewPermission)}>
                <GrantLine grant={machine.grant} />
              </Fact>
              {/* The recovery sentence used to sit on the card, where its
                  variable length was one of the things making cards on the same
                  shelf different heights. The card now offers the verb —
                  "Reconnect" — and the sentence explaining it lives here. */}
              {!machine.target.available && machine.target.recovery_action && (
                <div className="rounded-xl border border-amber-400/25 bg-amber-400/10 p-3 sm:col-span-2">
                  <dt className="text-[11px] font-semibold uppercase tracking-[0.14em] text-amber-200/70">
                    {t(strings.machines.reconnect)}
                  </dt>
                  <dd
                    data-testid={`machine-detail-recovery-${machineTestID(machine.target.id)}`}
                    className="mt-1 text-xs leading-5 text-amber-100"
                  >
                    {machine.target.recovery_action}
                  </dd>
                </div>
              )}
            </dl>
          )}

          {tab === "permissions" && (
            <PermissionsTab
              name={title}
              presets={presets}
              initialPreset={machine.grant.preset}
              busy={savingGrant}
              onConfirm={(preset) => { onSaveGrant(machine, preset); }}
              onCancel={onBack}
            />
          )}

          {tab === "configuration" && <ConfigurationTab machine={machine} />}

          {tab === "activity" && <MachineActivityTab machine={machine} />}
        </div>
      </div>
    </section>
  );
}

function Fact({ term, children }: { term: string; children: React.ReactNode }) {
  return (
    <div className="rounded-xl border border-wc-default bg-wc-surface-base/40 p-3">
      <dt className="text-[11px] font-semibold uppercase tracking-[0.14em] text-wc-text-faint">{term}</dt>
      <dd className="mt-1 text-xs leading-5 text-wc-text-primary">{children}</dd>
    </div>
  );
}

export default MachineDetail;
