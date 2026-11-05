import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";

/**
 * useSharedLabels — pre-translated copy for every reusable shared component.
 *
 * Most shared composites (`EmptyState`, `LoadingState`, `ErrorState`,
 * `DataTable`, `SeverityBadge`, `DiffView`, `SplitPane`) accept label
 * strings via props so they stay reusable. Features pull the
 * default copy from this hook. Concentrating the keys here ensures every
 * `strings.shared.*` entry has a single, well-typed callsite — features
 * can override individual labels by spreading `useSharedLabels()` and
 * overwriting fields.
 */
export interface SharedLabels {
  empty: { title: string; description: string };
  loading: { label: string };
  error: { title: string; retry: string };
  dataTable: { empty: string };
  severity: { info: string; low: string; medium: string; high: string; critical: string };
  diff: { added: string; removed: string; unchanged: string };
  splitPane: { resizeHandle: string };
}

export function useSharedLabels(): SharedLabels {
  const { t } = useTranslation();
  return {
    empty: {
      title: t(strings.shared.empty.title),
      description: t(strings.shared.empty.description),
    },
    loading: { label: t(strings.shared.loading.label) },
    error: {
      title: t(strings.shared.error.title),
      retry: t(strings.shared.error.retry),
    },
    dataTable: { empty: t(strings.shared.dataTable.empty) },
    severity: {
      info: t(strings.shared.severity.info),
      low: t(strings.shared.severity.low),
      medium: t(strings.shared.severity.medium),
      high: t(strings.shared.severity.high),
      critical: t(strings.shared.severity.critical),
    },
    diff: {
      added: t(strings.shared.diff.added),
      removed: t(strings.shared.diff.removed),
      unchanged: t(strings.shared.diff.unchanged),
    },
    splitPane: { resizeHandle: t(strings.shared.splitPane.resizeHandle) },
  };
}
