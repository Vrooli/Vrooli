import { useCallback, useEffect, useRef, useState } from "react";
import { Move, RotateCcw } from "lucide-react";

import { useTranslation } from "../../i18n";
import type { CatalogAsset, WorkbenchStory } from "../data";
import { Chip, Segmented, Select } from "../primitives";
import { TemplateRenderer } from "../templates/TemplateRenderer";
import { DEFAULT_TEMPLATE } from "../templates/library";
import { resolveTemplate, templateShelf, useLocalTemplates } from "../templates/local";
import type { SlotPlacement } from "../templates/spec";
import { FixtureFrame, type FixtureStatus } from "./FixtureFrame";

export type Device = "desktop" | "tablet" | "phone";

const DEVICE_WIDTH: Record<Device, number> = {
  desktop: 1180,
  tablet: 834,
  phone: 390,
};

export interface InContextStageProps {
  asset: CatalogAsset;
  story?: WorkbenchStory;
  version: string;
  templateId: string;
  onTemplateChange: (id: string) => void;
  device: Device;
  onDeviceChange: (device: Device) => void;
  onOverflowChange?: (overflow: boolean) => void;
  onPlacementChange?: (placed: boolean) => void;
}

function placementKey(assetId: string, templateId: string): string {
  return `rcl.workbench.context.${assetId}.${templateId}.v1`;
}

function readPlacement(assetId: string, templateId: string): SlotPlacement | undefined {
  try {
    const raw = window.localStorage.getItem(placementKey(assetId, templateId));
    if (!raw) return undefined;
    const parsed = JSON.parse(raw) as SlotPlacement;
    if (typeof parsed.x !== "number" || typeof parsed.y !== "number") return undefined;
    if (typeof parsed.w !== "number" || typeof parsed.h !== "number") return undefined;
    return parsed;
  } catch {
    return undefined;
  }
}

export function InContextStage({
  asset,
  story,
  version,
  templateId,
  onTemplateChange,
  device,
  onDeviceChange,
  onOverflowChange,
  onPlacementChange,
}: InContextStageProps) {
  const { t } = useTranslation();
  const locals = useLocalTemplates();
  const templates = templateShelf(locals);
  const template = resolveTemplate(templateId, locals) ?? DEFAULT_TEMPLATE;
  const hostRef = useRef<HTMLDivElement>(null);
  const [status, setStatus] = useState<FixtureStatus>("loading");
  const [reloadKey, setReloadKey] = useState(0);
  const [placement, setPlacement] = useState<SlotPlacement | undefined>(() =>
    readPlacement(asset.id, template.id),
  );

  useEffect(() => {
    setPlacement(readPlacement(asset.id, template.id));
  }, [asset.id, template.id]);

  useEffect(() => {
    const host = hostRef.current;
    if (!host) return;
    const check = () => onOverflowChange?.(host.scrollWidth > host.clientWidth + 1);
    check();
    const observer = new ResizeObserver(check);
    observer.observe(host);
    return () => observer.disconnect();
  }, [device, onOverflowChange, templateId]);

  useEffect(() => {
    onPlacementChange?.(Boolean(placement));
  }, [onPlacementChange, placement]);

  const updatePlacement = useCallback(
    (_nodeId: string, next: SlotPlacement) => {
      setPlacement(next);
      try {
        window.localStorage.setItem(placementKey(asset.id, template.id), JSON.stringify(next));
      } catch {
        /* local only */
      }
    },
    [asset.id, template.id],
  );

  const resetPlacement = () => {
    setPlacement(undefined);
    try {
      window.localStorage.removeItem(placementKey(asset.id, template.id));
    } catch {
      /* ignore */
    }
  };

  const width = DEVICE_WIDTH[device];

  return (
    <div className="wb-context-stage">
      <div className="wb-row" style={{ flexWrap: "wrap", marginBottom: 12, width: "100%", maxWidth: width }}>
        <Segmented
          label={t("workbench.component.device", { defaultValue: "Device" })}
          value={device}
          onChange={onDeviceChange}
          items={[
            { id: "desktop", label: t("workbench.component.desktop", { defaultValue: "Desktop" }) },
            { id: "tablet", label: t("workbench.component.tablet", { defaultValue: "Tablet" }) },
            { id: "phone", label: t("workbench.component.phone", { defaultValue: "Phone" }) },
          ]}
        />
        <div style={{ flex: 1 }} />
        <Select
          aria-label={t("workbench.component.template", { defaultValue: "Page template" })}
          value={template.id}
          onChange={(event) => onTemplateChange(event.target.value)}
          style={{ width: "auto", minWidth: 190 }}
        >
          {templates.map((entry) => (
            <option key={entry.id} value={entry.id}>
              {entry.name} — {entry.tagline}
            </option>
          ))}
        </Select>
        <Chip tone="ice">{template.slotHint}</Chip>
        <button
          type="button"
          className="wb-iconbtn"
          aria-label={t("workbench.component.reloadFixture", { defaultValue: "Reload specimen" })}
          onClick={() => setReloadKey((value) => value + 1)}
        >
          ↻
        </button>
        {placement ? (
          <button
            type="button"
            className="wb-btn quiet sm"
            onClick={resetPlacement}
            title={t("workbench.component.resetPlacementHint", {
              defaultValue: "Return the component to the slot the template designed.",
            })}
          >
            <RotateCcw aria-hidden size={12} />
            {t("workbench.component.resetPlacement", { defaultValue: "Reset position" })}
          </button>
        ) : null}
      </div>

      <div className="wb-device" style={{ width }}>
        <div className="wb-device-bar">
          <span className="wb-device-dot" />
          <span className="wb-device-dot" />
          <span className="wb-device-dot" />
          <span className="wb-device-url">{`app.vrooli.local${template.id === "aperture" ? "/overview" : ""}`}</span>
          <span className="wb-tag">{device}</span>
        </div>
        <div className="wb-device-host" ref={hostRef} style={{ height: device === "phone" ? 720 : 760 }}>
          <TemplateRenderer
            template={template}
            spec={template.spec}
            mode="preview"
            slotEditable={Boolean(story)}
            slotPlacement={placement}
            onSlotPlacementChange={updatePlacement}
            slot={
              story ? (
                <FixtureFrame
                  assetId={asset.id}
                  storyId={story.id}
                  version={version}
                  reloadKey={reloadKey}
                  height={placement?.h ?? 56}
                  title={t("workbench.component.contextTitle", {
                    defaultValue: "{{name}} in {{template}}",
                    name: story.name || story.id,
                    template: template.name,
                  })}
                  onStatus={setStatus}
                />
              ) : undefined
            }
          />
        </div>
      </div>
      <p className="wb-specimen-caption" style={{ maxWidth: width, width: "100%" }}>
        <Move aria-hidden size={12} style={{ verticalAlign: "-2px", marginRight: 4 }} />
        {t("workbench.component.contextCaption", {
          defaultValue:
            "{{template}} · {{idea}} The slot is live: drag the component inside the page, resize it from the corner, or reset it to the designed slot.",
          template: template.name,
          idea: template.idea,
        })}
        {status === "error" ? ` ${t("workbench.component.slotFailed", { defaultValue: "The specimen failed to render." })}` : ""}
      </p>
    </div>
  );
}
