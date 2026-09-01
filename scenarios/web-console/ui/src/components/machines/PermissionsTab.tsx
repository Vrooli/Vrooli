import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Check } from "lucide-react";
import { RadioGroup } from "@vrooli/react-component-library/RadioGroup/1";
import { Button } from "../ui/button";
import { strings } from "../../consts/strings";
import type { PermissionPreset } from "../../api/machines";
import { PresetGrantDetail } from "./grant";
import { machineTestID } from "./testids";

/**
 * What a machine is allowed to do — the panel formerly reached by `Manage`.
 *
 * Presets are the product; scopes are the implementation. The control plane
 * owns what a posture means, so this screen names one and never computes its
 * grant. The narrowest preset the control plane offers is first, and is the
 * default, which is what makes a narrow default the easy path rather than the
 * expert one.
 *
 * The option cards were hand-rolled `role="radio"` buttons. They are now
 * `RadioGroup variant="card"`, which is the same control with a surface: one
 * radiogroup, one checked radio, arrow-key roving, and a target the size of the
 * sentence rather than the size of the dot.
 */

export interface PermissionsTabProps {
  name: string;
  presets: PermissionPreset[];
  initialPreset?: string;
  busy: boolean;
  onConfirm: (preset: string) => void;
  onCancel: () => void;
  /** Linking a new machine says "Link this machine"; changing one says "Save". */
  mode?: "link" | "manage";
}

export default function PermissionsTab({
  name,
  presets,
  initialPreset,
  busy,
  onConfirm,
  onCancel,
  mode = "manage",
}: PermissionsTabProps) {
  const { t } = useTranslation();
  const [selected, setSelected] = useState(() => initialPreset || presets[0]?.name || "");

  useEffect(() => {
    if (!presets.some((preset) => preset.name === selected)) {
      setSelected(initialPreset || presets[0]?.name || "");
    }
  }, [initialPreset, presets, selected]);

  const chosen = presets.find((preset) => preset.name === selected);
  const dirty = selected !== (initialPreset ?? presets[0]?.name ?? "");

  if (presets.length === 0) {
    return (
      <div
        data-testid="machines-no-presets"
        role="alert"
        className="rounded-xl border border-amber-400/25 bg-amber-400/10 p-4 text-sm leading-6 text-amber-100"
      >
        {t(strings.machines.noPresets)}
      </div>
    );
  }

  return (
    <div className="space-y-4">
      <RadioGroup
        variant="card"
        label={t(strings.machines.permissionsTitle, { name })}
        description={t(strings.machines.permissionsSubtitle)}
        value={selected}
        onValueChange={setSelected}
        options={presets.map((preset) => ({
          value: preset.name,
          label: preset.title,
          description: preset.description,
          testId: `machines-preset-${machineTestID(preset.name)}`,
          badge:
            preset.name === selected ? (
              <span
                data-testid={`machines-preset-chosen-${preset.name}`}
                className="inline-flex items-center gap-1 rounded-full bg-wc-accent/20 px-2 py-0.5 text-[11px] font-medium text-wc-accent"
              >
                <Check className="h-3 w-3" aria-hidden />
                {t(strings.machines.chosen)}
              </span>
            ) : undefined,
        }))}
      />

      {chosen && <PresetGrantDetail preset={chosen} />}

      <div className="flex items-center justify-end gap-2 border-t border-wc-default pt-3">
        <span className="me-auto text-xs text-wc-text-faint">
          {dirty ? "" : t(strings.machines.permissionsUnchanged)}
        </span>
        <Button variant="outline" size="sm" onClick={onCancel} disabled={busy}>
          {t(strings.machines.cancel)}
        </Button>
        <Button
          size="sm"
          data-testid="machines-grant-confirm"
          pending={busy}
          pendingLabel={mode === "link" ? t(strings.machines.linking) : t(strings.machines.saving)}
          disabled={!chosen}
          onClick={() => { onConfirm(selected); }}
        >
          {mode === "link" ? t(strings.machines.linkMachine) : t(strings.machines.savePermissions)}
        </Button>
      </div>
    </div>
  );
}
