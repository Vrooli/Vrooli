import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { ArrowLeft, Check, Loader2 } from "lucide-react";
import { Button } from "../ui/button";
import { strings } from "../../consts/strings";
import { machineTestID } from "./testids";
import type { PermissionPreset } from "../../api/machines";
import { PresetGrantDetail } from "./grant";

/**
 * Screen 04 — deciding what a machine may do.
 *
 * Presets are the product; scopes are the implementation. The control plane
 * owns what a posture means, so this screen names one and never computes its
 * grant. The narrowest preset is the default selection, which is what makes a
 * narrow default the easy path rather than the expert one.
 */

interface GrantPickerProps {
  name: string;
  presets: PermissionPreset[];
  /** Preselect the posture a machine already holds when changing an existing grant. */
  initialPreset?: string;
  /** Copy differs between linking a new machine and changing an existing one. */
  mode: "link" | "manage";
  busy: boolean;
  onBack: () => void;
  onConfirm: (preset: string) => void;
}

export default function GrantPicker({ name, presets, initialPreset, mode, busy, onBack, onConfirm }: GrantPickerProps) {
  const { t } = useTranslation();
  // The narrowest preset the control plane offers is first, and is the default.
  const [selected, setSelected] = useState(() => initialPreset || presets[0]?.name || "");
  useEffect(() => {
    if (!presets.some((preset) => preset.name === selected)) setSelected(initialPreset || presets[0]?.name || "");
  }, [initialPreset, presets, selected]);

  const chosen = presets.find((preset) => preset.name === selected);

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <div className="mx-auto flex w-full max-w-2xl items-center gap-2 px-5 pt-5">
        <button
          type="button"
          data-testid="machines-grant-back"
          onClick={onBack}
          aria-label={t(strings.machines.back)}
          className="inline-flex h-9 w-9 items-center justify-center rounded-lg text-wc-text-secondary transition hover:bg-wc-surface-input hover:text-wc-text-primary"
        >
          <ArrowLeft className="h-4 w-4" aria-hidden />
        </button>
        <div className="min-w-0">
          <h2 className="truncate text-lg font-semibold text-wc-text-primary">
            {t(strings.machines.permissionsTitle, { name })}
          </h2>
          <p className="truncate text-xs text-wc-text-faint">{t(strings.machines.permissionsSubtitle)}</p>
        </div>
      </div>

      <div className="min-h-0 flex-1 overflow-y-auto">
        <div className="mx-auto w-full max-w-2xl space-y-4 px-5 pb-5 pt-5">
        {presets.length === 0 ? (
          <div data-testid="machines-no-presets" role="alert" className="rounded-xl border border-amber-400/25 bg-amber-400/10 p-4 text-sm leading-6 text-amber-100">
            {t(strings.machines.noPresets)}
          </div>
        ) : (
          <>
            <div role="radiogroup" aria-label={t(strings.machines.grantHeading)} className="space-y-2">
              {presets.map((preset) => {
                const isSelected = preset.name === selected;
                return (
                  <button
                    key={preset.name}
                    type="button"
                    role="radio"
                    aria-checked={isSelected}
                    data-testid={`machines-preset-${machineTestID(preset.name)}`}
                    onClick={() => { setSelected(preset.name); }}
                    className={`flex w-full items-center gap-3 rounded-xl border px-4 py-3 text-start transition ${
                      isSelected
                        ? "border-wc-accent bg-wc-accent/10"
                        : "border-wc-default bg-wc-surface-input hover:border-wc-accent/50"
                    }`}
                  >
                    <span className="min-w-0 flex-1">
                      <span className="block text-sm font-medium text-wc-text-primary">{preset.title}</span>
                      <span className="mt-0.5 block text-xs leading-5 text-wc-text-muted">{preset.description}</span>
                    </span>
                    {isSelected && (
                      <span className="inline-flex shrink-0 items-center gap-1 rounded-full bg-wc-accent/20 px-2 py-0.5 text-[11px] font-medium text-wc-accent">
                        <Check className="h-3 w-3" aria-hidden />
                        {t(strings.machines.chosen)}
                      </span>
                    )}
                  </button>
                );
              })}
            </div>
            {chosen && <PresetGrantDetail preset={chosen} />}
          </>
        )}
        </div>
      </div>

      <footer className="shrink-0 border-t border-wc-default px-5 py-3">
        <div className="mx-auto flex w-full max-w-2xl items-center justify-end gap-2">
          <Button
            data-testid="machines-grant-confirm"
          onClick={() => { onConfirm(selected); }}
          disabled={busy || !chosen}
        >
          {busy && <Loader2 className="me-1.5 h-4 w-4 animate-spin" aria-hidden />}
          {mode === "link"
            ? busy
              ? t(strings.machines.linking)
              : t(strings.machines.linkMachine)
            : busy
              ? t(strings.machines.saving)
              : t(strings.machines.saveGrant)}
        </Button>
      </div>
      </footer>
    </div>
  );
}
