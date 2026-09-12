import { AlertTriangle, CheckCircle2, Clock, XCircle } from "lucide-react";

import { AuditStatus, type Audit } from "../../api/audits";
import { formatBytes } from "../../lib/format";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";

type InventorySummary = NonNullable<Audit["live"]>;

const shortHash = (h: string) => (h.length > 12 ? h.slice(0, 12) : h);

type Verdict = "pass" | "drift" | "diff" | "failed" | "running";

function verdictOf(audit: Audit): Verdict {
  if (audit.status === AuditStatus.FAILED) return "failed";
  if (audit.status !== AuditStatus.COMPLETED) return "running";
  if (!audit.comparison) return "diff";
  if (audit.comparison.matches) return "pass";
  return audit.comparison.liveNewerThanSnapshot ? "drift" : "diff";
}

const VERDICT_META: Record<Verdict, { icon: typeof CheckCircle2; tone: string }> = {
  pass: { icon: CheckCircle2, tone: "text-app-success" },
  drift: { icon: AlertTriangle, tone: "text-app-warning" },
  diff: { icon: AlertTriangle, tone: "text-app-danger" },
  failed: { icon: XCircle, tone: "text-app-danger" },
  running: { icon: Clock, tone: "text-app-muted-foreground" },
};

/**
 * Renders a single generic snapshot-audit proof: the verdict (pass / drift /
 * diff / failed), restorability, the specific generic mismatches, the live-vs-
 * snapshot inventory comparison, and per-SQLite integrity. It shows only
 * generic signals — never file contents.
 */
export function AuditReport({
  audit,
  loading,
  error,
}: {
  audit: Audit | undefined;
  loading: boolean;
  error: boolean;
}) {
  const { t } = useTranslation();

  if (loading) {
    return (
      <div data-testid={selectors.audits.report} className="flex items-center gap-2 text-sm text-app-muted-foreground">
        <Clock aria-hidden="true" className="h-4 w-4 animate-pulse" />
        {t(strings.audits.running)}
      </div>
    );
  }
  if (error) {
    return (
      <p data-testid={selectors.audits.report} className="text-sm text-app-danger">
        {t(strings.audits.error)}
      </p>
    );
  }
  if (!audit) return null;

  const verdict = verdictOf(audit);
  const meta = VERDICT_META[verdict];
  const Icon = meta.icon;

  const verdictText = (): string => {
    switch (verdict) {
      case "pass":
        return t(strings.audits.verdictPass);
      case "drift":
        return t(strings.audits.verdictDrift);
      case "diff":
        return t(strings.audits.verdictDiff);
      case "running":
        return t(strings.audits.verdictRunning);
      case "failed":
        return audit.error ? `${t(strings.audits.verdictFailed)} ${audit.error}` : t(strings.audits.verdictFailed);
    }
  };

  return (
    <div
      data-testid={selectors.audits.report}
      className="flex flex-col gap-3 rounded-control border border-app-border p-3"
    >
      <div data-testid={selectors.audits.verdict} className={`flex items-start gap-2 text-sm font-medium ${meta.tone}`}>
        <Icon aria-hidden="true" className="mt-0.5 h-4 w-4 shrink-0" />
        <span>{verdictText()}</span>
      </div>

      <p className="text-xs text-app-muted-foreground">
        {audit.restorable ? `✓ ${t(strings.audits.restorable)}` : `✗ ${t(strings.audits.notRestorable)}`}
      </p>

      {audit.comparison && audit.comparison.mismatches.length > 0 && (
        <div className="flex flex-col gap-1">
          <span className="text-xs font-semibold text-app-foreground">{t(strings.audits.mismatches)}</span>
          <ul className="list-disc ps-5 text-xs text-app-muted-foreground">
            {audit.comparison.mismatches.map((m: string) => (
              <li key={m} className="font-mono">
                {m}
              </li>
            ))}
          </ul>
        </div>
      )}

      <div className="grid grid-cols-2 gap-3 text-xs">
        <InventoryColumn title={t(strings.audits.live)} inv={audit.live} />
        <InventoryColumn title={t(strings.audits.snapshot)} inv={audit.snapshot} />
      </div>
    </div>
  );
}

type SqliteInventory = InventorySummary["sqlite"][number];

function InventoryColumn({ title, inv }: { title: string; inv: InventorySummary | undefined }) {
  const { t } = useTranslation();
  if (!inv) {
    return (
      <div className="flex flex-col gap-1">
        <span className="font-semibold text-app-foreground">{title}</span>
        <span className="text-app-muted-foreground">—</span>
      </div>
    );
  }
  return (
    <div className="flex flex-col gap-0.5">
      <span className="font-semibold text-app-foreground">{title}</span>
      <span className="text-app-muted-foreground">
        {inv.files.toString()} {t(strings.audits.files)} · {inv.directories.toString()} {t(strings.audits.dirs)} ·{" "}
        {inv.symlinks.toString()} {t(strings.audits.symlinks)}
      </span>
      <span className="text-app-muted-foreground">{formatBytes(inv.regularBytes)}</span>
      <span className="font-mono text-app-muted-foreground">
        {t(strings.audits.pathHash)}: {shortHash(inv.pathListSha256) || "—"}
      </span>
      {inv.treeContentSha256 && (
        <span className="font-mono text-app-muted-foreground">
          {t(strings.audits.contentHash)}: {shortHash(inv.treeContentSha256)}
        </span>
      )}
      {inv.sqlite.map((s: SqliteInventory) => (
        <span key={s.path} className="font-mono text-app-muted-foreground">
          {t(strings.audits.sqlite)} {s.path}: {t(strings.audits.integrity)}={s.integrityStatus} · {s.tableCount.toString()}{" "}
          {t(strings.audits.tables)}
        </span>
      ))}
      {inv.unreadablePaths.length > 0 && (
        <span className="text-app-warning">
          {inv.unreadablePaths.length} {t(strings.audits.unreadable)}
        </span>
      )}
    </div>
  );
}
