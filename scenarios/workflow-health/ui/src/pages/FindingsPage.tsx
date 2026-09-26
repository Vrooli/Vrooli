import { selectors } from "../consts/selectors";
import { strings } from "../consts/strings";
import { useTranslation } from "../i18n";
import { findings, uiText } from "./workflowData";

const severityClass = {
  Critical: "bg-red-100 text-red-800",
  High: "bg-orange-100 text-orange-800",
  Medium: "bg-amber-100 text-amber-800",
  Low: "bg-slate-100 text-slate-800",
} as const;

export function FindingsPage() {
  const { t } = useTranslation();

  return (
    <section
      data-testid={selectors.pages.findings}
      aria-labelledby="findings-heading"
      className="flex flex-col gap-5"
    >
      <div>
        <h2 id="findings-heading" className="text-3xl font-semibold">
          {t(strings.pages.findings.title)}
        </h2>
        <p className="mt-2 max-w-3xl text-sm text-app-muted-foreground">
          {t(strings.pages.findings.description)}
        </p>
      </div>

      <section className="rounded-panel border border-app-border bg-app-surface p-4">
        <div className="overflow-x-auto">
          <table className="min-w-full text-left text-sm">
            <thead className="text-xs uppercase text-app-muted-foreground">
              <tr>
                {uiText.findings.headers.map((header) => (
                  <th key={header} className="px-2 py-2 font-semibold">
                    {header}
                  </th>
                ))}
              </tr>
            </thead>
            <tbody>
              {findings.map((finding) => (
                <tr key={finding.id} className="border-t border-app-border align-top">
                  <td className="px-2 py-3 font-mono text-xs">{finding.id}</td>
                  <td className="px-2 py-3">
                    <span className={`rounded-full px-2.5 py-1 text-xs font-medium ${severityClass[finding.severity]}`}>
                      {finding.severity}
                    </span>
                  </td>
                  <td className="px-2 py-3">{finding.asset}</td>
                  <td className="px-2 py-3">{finding.summary}</td>
                  <td className="px-2 py-3 text-app-muted-foreground">{finding.remediation}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      </section>
    </section>
  );
}
