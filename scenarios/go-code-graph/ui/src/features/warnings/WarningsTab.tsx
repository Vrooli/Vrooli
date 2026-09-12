import { CodeGraphWarningKind } from "@vrooli/proto-types/common/v1/code_graph_pb";
import type { CodeGraphWarning } from "../../api/graph";

import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { EmptyState } from "../../components/EmptyState";
import { SeverityBadge, type SeverityLevel } from "../../components/SeverityBadge";

type WarnKindKey = (typeof strings.warnings.kind)[keyof typeof strings.warnings.kind];

interface KindMeta {
  readonly labelKey: WarnKindKey;
  readonly level: SeverityLevel;
}

const UNSPECIFIED_META: KindMeta = {
  labelKey: strings.warnings.kind.unspecified,
  level: "info",
};

const KIND_META: Partial<Record<CodeGraphWarningKind, KindMeta>> = {
  [CodeGraphWarningKind.PARSE_ERROR]: {
    labelKey: strings.warnings.kind.parse_error,
    level: "high",
  },
  [CodeGraphWarningKind.UNRESOLVED_IMPORT]: {
    labelKey: strings.warnings.kind.unresolved_import,
    level: "medium",
  },
  [CodeGraphWarningKind.TYPE_CHECK_FAILURE]: {
    labelKey: strings.warnings.kind.type_check_failure,
    level: "medium",
  },
  [CodeGraphWarningKind.AMBIGUOUS_DECLARATION]: {
    labelKey: strings.warnings.kind.ambiguous_declaration,
    level: "low",
  },
};

function metaFor(kind: CodeGraphWarningKind): KindMeta {
  return KIND_META[kind] ?? UNSPECIFIED_META;
}

export interface WarningsTabProps {
  warnings: readonly CodeGraphWarning[];
  includeVendor: boolean;
  onToggleVendor: (next: boolean) => void;
}

/**
 * Diagnostics tab. Renders parse warnings + unresolved imports from the last
 * Extract response, with a vendor-filter toggle that re-runs extraction with
 * the include-vendor option (the parent owns the re-run via params).
 */
export function WarningsTab({ warnings, includeVendor, onToggleVendor }: WarningsTabProps) {
  const { t } = useTranslation();

  return (
    <div data-testid={selectors.features.warnings.root} className="flex flex-col gap-3">
      <div className="flex flex-col gap-1">
        <h3 className="text-lg font-semibold">{t(strings.warnings.title)}</h3>
        <p className="text-sm text-app-muted-foreground">{t(strings.warnings.description)}</p>
      </div>

      <label className="flex items-center gap-2 text-sm">
        <input
          type="checkbox"
          data-testid={selectors.features.warnings.vendorToggle}
          checked={includeVendor}
          onChange={(e) => onToggleVendor(e.target.checked)}
          className="h-4 w-4 rounded border-app-border"
        />
        <span>{t(strings.warnings.vendorToggle)}</span>
        <span className="text-xs text-app-muted-foreground">{t(strings.warnings.vendorHint)}</span>
      </label>

      <p
        data-testid={selectors.features.warnings.summary}
        className="text-xs uppercase tracking-wide text-app-muted-foreground"
      >
        {t(strings.warnings.count, { count: warnings.length })}
      </p>

      {warnings.length === 0 ? (
        <div data-testid={selectors.features.warnings.empty}>
          <EmptyState title={t(strings.warnings.empty)} />
        </div>
      ) : (
        <ul data-testid={selectors.features.warnings.list} className="flex flex-col gap-2">
          {warnings.map((warning, index) => {
            const meta = metaFor(warning.kind);
            return (
              <li
                key={`${warning.kind}-${warning.file}-${index}`}
                data-testid={selectors.features.warnings.item({ index })}
                className="flex flex-col gap-1 rounded-panel border border-app-border bg-app-surface p-3 text-sm backdrop-blur-sm"
              >
                <div className="flex items-center gap-2">
                  <SeverityBadge level={meta.level} label={t(meta.labelKey)} />
                  <span className="text-xs text-app-muted-foreground">
                    {t(strings.warnings.fileLabel)}{" "}
                    {warning.file.length > 0 ? (
                      <span className="font-mono text-app-foreground">{warning.file}</span>
                    ) : (
                      t(strings.warnings.projectLevel)
                    )}
                  </span>
                </div>
                {warning.message.length > 0 ? (
                  <p className="text-app-foreground">{warning.message}</p>
                ) : null}
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}
