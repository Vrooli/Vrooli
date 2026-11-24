import { Contrast, Eye, Ruler, ScanLine, Target } from "lucide-react";

import { useTranslation } from "../../i18n";
import type { Template } from "../templates/spec";
import { Chip, Segmented, Select } from "../primitives";
import type { FixtureStatus } from "./FixtureFrame";
import type { Device, InstrumentId, Instruments } from "./InContextStage";
import { VIEWPORTS, type ViewportPreset } from "./viewports";

export function InContextToolbar({
  device,
  onDeviceChange,
  viewport,
  onViewportChange,
  instruments,
  onInstrumentToggle,
  template,
  templates,
  onTemplateChange,
  status,
  onReload,
}: {
  device: Device;
  onDeviceChange: (device: Device) => void;
  viewport: ViewportPreset;
  onViewportChange: (viewport: ViewportPreset) => void;
  instruments: Instruments;
  onInstrumentToggle: (id: InstrumentId) => void;
  template: Template;
  templates: Array<{ id: string; name: string; summary: string }>;
  onTemplateChange: (id: string) => void;
  status: FixtureStatus;
  onReload: () => void;
}) {
  const { t } = useTranslation();
  return (
    <div className="ic-bar">
      <Segmented
        label={t("workbench.component.device", { defaultValue: "Frame" })}
        value={device}
        onChange={onDeviceChange}
        items={[
          { id: "desktop", label: t("workbench.component.desktop", { defaultValue: "Desktop" }) },
          { id: "tablet", label: t("workbench.component.tablet", { defaultValue: "Tablet" }) },
          { id: "phone", label: t("workbench.component.phone", { defaultValue: "Phone" }) },
        ]}
      />
      <Select
        aria-label={t("workbench.component.viewport", { defaultValue: "Viewport width" })}
        className="ic-viewport-select"
        value={viewport.id}
        onChange={(event) => {
          const next = VIEWPORTS.find((entry) => entry.id === event.target.value);
          if (next) onViewportChange(next);
        }}
      >
        {VIEWPORTS.map((entry) => (
          <option key={entry.id} value={entry.id}>
            {t("workbench.component.viewportValue", {
              defaultValue: "{{width}} px",
              width: entry.label,
            })}
          </option>
        ))}
      </Select>
      <div
        className="ic-tools"
        role="group"
        aria-label={t("workbench.component.instruments", { defaultValue: "Instruments" })}
      >
        <ToolButton
          id="guides"
          active={instruments.guides}
          icon={<Ruler aria-hidden size={12} />}
          label={t("workbench.component.instrumentGuides", { defaultValue: "Guides" })}
          onClick={onInstrumentToggle}
        />
        <ToolButton
          id="overflow"
          active={instruments.overflow}
          icon={<ScanLine aria-hidden size={12} />}
          label={t("workbench.component.instrumentOverflow", { defaultValue: "Overflow" })}
          onClick={onInstrumentToggle}
        />
        <ToolButton
          id="targets"
          active={instruments.targets}
          icon={<Target aria-hidden size={12} />}
          label={t("workbench.component.instrumentTargets", { defaultValue: "Targets" })}
          onClick={onInstrumentToggle}
        />
        <ToolButton
          id="contrast"
          active={instruments.contrast}
          icon={<Contrast aria-hidden size={12} />}
          label={t("workbench.component.instrumentContrast", { defaultValue: "Contrast" })}
          onClick={onInstrumentToggle}
        />
        <ToolButton
          id="blur"
          active={instruments.blur}
          icon={<Eye aria-hidden size={12} />}
          label={t("workbench.component.instrumentBlur", { defaultValue: "Blur" })}
          onClick={onInstrumentToggle}
        />
      </div>
      <div className="ic-bar-spacer" />
      <Select
        aria-label={t("workbench.component.template", { defaultValue: "Page template" })}
        className="ic-template-select"
        value={template.id}
        onChange={(event) => onTemplateChange(event.target.value)}
      >
        {templates.map((entry) => (
          <option key={entry.id} value={entry.id}>
            {entry.name} — {entry.summary}
          </option>
        ))}
      </Select>
      <Chip tone={status === "error" ? "red" : status === "loading" ? "amber" : "green"}>
        {status === "loading"
          ? t("workbench.component.harnessConnecting", { defaultValue: "Connecting…" })
          : t("workbench.component.liveHarness", { defaultValue: "Live harness" })}
      </Chip>
      <button
        type="button"
        className="ic-reload"
        aria-label={t("workbench.component.reloadFixture", { defaultValue: "Reload specimen" })}
        onClick={onReload}
      >
        ↻
      </button>
    </div>
  );
}

function ToolButton({
  id,
  active,
  icon,
  label,
  onClick,
}: {
  id: InstrumentId;
  active: boolean;
  icon: React.ReactNode;
  label: string;
  onClick: (id: InstrumentId) => void;
}) {
  return (
    <button
      type="button"
      className={`ic-tool${active ? " is-on" : ""}`}
      aria-pressed={active}
      onClick={() => onClick(id)}
    >
      {icon}
      {label}
    </button>
  );
}
