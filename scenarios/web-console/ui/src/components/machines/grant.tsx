import { useState } from "react";
import { useTranslation } from "react-i18next";
import { strings } from "../../consts/strings";
import type { Grant, GrantEffect, PermissionPreset } from "../../api/machines";
import { useBreadthLabel, useEffectLabel } from "./useGrantLabels";

/**
 * Grant rendering.
 *
 * The control plane expands a preset to one scope per app per effect — well
 * over a hundred entries. Rendering that list as the primary answer buries the
 * only fact most operators need, so the shape comes first and the enumeration
 * stays one disclosure away for the person who has to audit it.
 */

const effectTone: Record<GrantEffect, string> = {
  read: "border-sky-400/25 bg-sky-400/10 text-sky-200",
  write: "border-amber-400/25 bg-amber-400/10 text-amber-200",
  destructive: "border-rose-400/25 bg-rose-400/10 text-rose-200",
};

export function EffectChips({
  effects,
  withholds = [],
}: {
  effects: GrantEffect[];
  withholds?: string[];
}) {
  const label = useEffectLabel();
  return (
    <span className="flex flex-wrap items-center gap-1.5">
      {effects.map((effect) => (
        <span
          key={effect}
          data-testid={`machines-effect-${effect}`}
          className={`inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium ${effectTone[effect]}`}
        >
          {label(effect)}
        </span>
      ))}
      {/* Showing what a preset withholds is what makes a narrower default feel
          like a choice rather than something being taken away. */}
      {withholds.map((effect) => (
        <span
          key={`withheld-${effect}`}
          data-testid={`machines-withheld-${effect}`}
          className="inline-flex items-center rounded-full border border-wc-default px-2 py-0.5 text-[11px] text-wc-text-faint line-through decoration-wc-text-faint/70"
        >
          {label(effect as GrantEffect)}
        </span>
      ))}
    </span>
  );
}

/** The one-line reading: what it may do, and how far that reaches. */
export function GrantLine({ grant }: { grant: Grant }) {
  const breadth = useBreadthLabel();
  return (
    <span className="block text-xs leading-5">
      <span className="block text-wc-text-muted">{grant.summary}</span>
      {grant.effects.length > 0 && (
        <span className="block text-wc-text-faint">{breadth(grant.appCount, grant.coversAllApps)}</span>
      )}
    </span>
  );
}

/** The audit view: the concrete scopes, behind a disclosure. */
export function ScopeAudit({ scopes }: { scopes: string[] }) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  if (scopes.length === 0) return null;
  return (
    <div className="mt-3">
      <button
        type="button"
        data-testid="machines-scope-audit-toggle"
        onClick={() => { setOpen((value) => !value); }}
        aria-expanded={open}
        className="min-h-11 text-xs font-medium text-wc-text-secondary underline decoration-dotted underline-offset-4 transition hover:text-wc-text-primary"
      >
        {open ? t(strings.machines.scopeAuditHide) : t(strings.machines.scopeAudit, { count: scopes.length })}
      </button>
      {open && (
        <ul
          data-testid="machines-scope-audit"
          className="mt-2 grid grid-cols-1 gap-x-4 gap-y-0.5 rounded-lg border border-wc-default bg-wc-surface-base/60 p-3 font-mono text-[11px] leading-5 text-wc-text-faint sm:grid-cols-2"
        >
          {scopes.map((scope) => (
            <li key={scope} className="truncate">
              {scope}
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}

/** The block shown under a chosen preset: what it issues and what it holds back. */
export function PresetGrantDetail({ preset }: { preset: PermissionPreset }) {
  const { t } = useTranslation();
  const breadth = useBreadthLabel();
  return (
    <div className="rounded-xl border border-wc-default bg-wc-surface-base/40 p-4">
      <div className="text-[11px] font-semibold uppercase tracking-[0.14em] text-wc-text-faint">
        {t(strings.machines.grantsIssued)}
      </div>
      <div className="mt-2 flex flex-wrap items-center gap-2">
        <EffectChips effects={preset.effects} withholds={preset.withholds} />
        <span className="text-xs text-wc-text-faint">{breadth(preset.appCount, false)}</span>
      </div>
      <ScopeAudit scopes={preset.scopes} />
    </div>
  );
}
