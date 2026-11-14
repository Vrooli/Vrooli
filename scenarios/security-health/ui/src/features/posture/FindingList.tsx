import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import type { Finding } from "../../api/validation";
import { compareSeverity, severityMeta } from "./severity";

/**
 * Renders a normalized {@link Finding} list, severity-ordered (ERROR first),
 * each row carrying its severity chip, title, description, remediation, anchor
 * file:line, and producing scanner. Secret findings are reported file:line
 * only by the backend, so nothing here can leak a raw value.
 */
export function FindingList({ findings }: { findings: readonly Finding[] }) {
  const { t } = useTranslation();
  const sorted = [...findings].sort(
    (a, b) => compareSeverity(a.severity, b.severity) || a.ruleId.localeCompare(b.ruleId),
  );

  return (
    <ul data-testid={selectors.posture.findings} className="flex flex-col gap-2">
      {sorted.map((f, i) => {
        const meta = severityMeta(f.severity);
        return (
          <li
            key={`${f.ruleId}:${f.filePath}:${i}`}
            className="rounded-lg border border-app-border bg-app-background/40 p-3"
          >
            <div className="flex flex-wrap items-center gap-2">
              <span className={["rounded px-1.5 py-0.5 text-xs font-semibold uppercase", meta.chipClass].join(" ")}>
                {t(meta.labelKey)}
              </span>
              <span className="font-medium">{f.title}</span>
              <span className="ms-auto rounded bg-app-surface-muted px-1.5 py-0.5 text-xs text-app-muted-foreground">
                {f.scanner} · {f.ruleId}
              </span>
            </div>
            {f.description && <p className="mt-1.5 text-sm text-app-foreground/80">{f.description}</p>}
            {f.filePath && (
              <p className="mt-1 font-mono text-xs text-app-muted-foreground">
                {t(strings.posture.fileLabel)} {f.filePath}
              </p>
            )}
            {f.remediation && (
              <p className="mt-1.5 text-sm text-emerald-300/90">
                <span className="font-semibold">{t(strings.posture.remediationLabel)}: </span>
                {f.remediation}
              </p>
            )}
          </li>
        );
      })}
    </ul>
  );
}
