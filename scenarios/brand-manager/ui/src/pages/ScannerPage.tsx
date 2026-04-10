import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { Search, FileSearch, AlertCircle } from "lucide-react";
import { scanScenario, fetchAuditRules, evaluateScenario } from "../lib/api";
import { Button } from "../components/ui/button";
import { Input } from "../components/ui/input";
import { Section } from "../components/ui/section";
import { ErrorAlert } from "../components/ui/error-alert";

// [REQ:BM-REQ-SCAN-CSS] [REQ:BM-REQ-SCAN-JSON] [REQ:BM-REQ-AUDIT-ENDPOINT] [REQ:BM-REQ-DISC-SCAN]

interface ScannerPageProps {
  onNavigate: (path: string) => void;
}

export default function ScannerPage({ onNavigate }: ScannerPageProps) {
  const [scenario, setScenario] = useState("");
  const [scanTarget, setScanTarget] = useState<string | null>(null);

  const { data: scanResult, isLoading: scanning, error: scanError, refetch: rescan } = useQuery({
    queryKey: ["scan", scanTarget],
    queryFn: () => scanScenario(scanTarget ?? ""),
    enabled: !!scanTarget,
  });

  const { data: auditResult, isLoading: auditing, error: auditError } = useQuery({
    queryKey: ["audit", scanTarget],
    queryFn: () => evaluateScenario(scanTarget ?? ""),
    enabled: !!scanTarget,
  });

  const { data: rules } = useQuery({
    queryKey: ["audit-rules"],
    queryFn: fetchAuditRules,
  });

  const handleScan = () => {
    const trimmed = scenario.trim();
    if (trimmed) setScanTarget(trimmed);
  };

  return (
    <div data-testid="scanner-page">
      <div className="flex items-center justify-between mb-6">
        <h1 className="text-2xl font-bold text-slate-50">Brand Scanner</h1>
        <Button variant="outline" size="sm" onClick={() => onNavigate("/brands")} data-testid="back-to-brands">
          Back to Library
        </Button>
      </div>

      <div className="flex items-center gap-3 mb-6">
        <div className="relative flex-1">
          <FileSearch className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-slate-500" />
          <Input
            variant="search"
            placeholder="Enter scenario name to scan..."
            value={scenario}
            onChange={(e) => setScenario(e.target.value)}
            onKeyDown={(e) => e.key === "Enter" && handleScan()}
            data-testid="scanner-input"
          />
        </div>
        <Button onClick={handleScan} disabled={!scenario.trim()} data-testid="scan-btn">
          <Search className="mr-2 h-4 w-4" />
          Scan
        </Button>
      </div>

      {(scanning || auditing) && (
        <div className="text-center text-slate-400 py-12" data-testid="scanner-loading">
          Scanning {scanTarget}...
        </div>
      )}

      {scanError && (
        <ErrorAlert
          error={scanError}
          fallbackMessage="Scan failed."
          fallbackRecovery="Check that the scenario exists and the API is running."
          onRetry={() => rescan()}
          testId="scan-error"
        />
      )}

      {scanResult && (
        <Section title={`Scan Results: ${scanResult.scenario}`} testId="scan-results" className="mb-4">
          <div className="grid grid-cols-3 gap-4 mb-4">
            <div className="text-center p-3 rounded-lg bg-white/5">
              <div className="text-2xl font-bold text-slate-50" data-testid="scan-total">{scanResult.summary.total}</div>
              <div className="text-xs text-slate-500">Total Markers</div>
            </div>
            <div className="text-center p-3 rounded-lg bg-white/5">
              <div className="text-2xl font-bold text-blue-400" data-testid="scan-css">{scanResult.summary.css}</div>
              <div className="text-xs text-slate-500">CSS</div>
            </div>
            <div className="text-center p-3 rounded-lg bg-white/5">
              <div className="text-2xl font-bold text-amber-400" data-testid="scan-json">{scanResult.summary.json}</div>
              <div className="text-xs text-slate-500">JSON</div>
            </div>
          </div>

          {scanResult.findings.length > 0 ? (
            <div className="space-y-2" data-testid="scan-findings">
              {scanResult.findings.map((f, i) => (
                <div key={i} className="flex items-center justify-between text-sm border-b border-white/5 pb-2">
                  <div>
                    <span className="text-slate-200">{f.element}</span>
                    <span className="text-slate-500 ml-2">in {f.file}</span>
                    {f.line && <span className="text-slate-600 ml-1">:{f.line}</span>}
                  </div>
                  <span className={`text-xs px-2 py-0.5 rounded-full ${f.type === "css" ? "bg-blue-500/20 text-blue-300" : "bg-amber-500/20 text-amber-300"}`}>
                    {f.type}
                  </span>
                </div>
              ))}
            </div>
          ) : (
            <p className="text-slate-500 text-sm" data-testid="scan-no-findings">No brand markers found in this scenario.</p>
          )}
        </Section>
      )}

      {auditResult && (
        <Section title="Audit Results" testId="audit-results" className="mb-4">
          <div className="flex items-center gap-2 mb-3">
            {auditResult.pass_all ? (
              <span className="text-emerald-400 text-sm font-medium" data-testid="audit-pass">All checks passed</span>
            ) : (
              <span className="text-red-400 text-sm font-medium flex items-center gap-1" data-testid="audit-fail">
                <AlertCircle className="h-4 w-4" /> Some checks failed
              </span>
            )}
          </div>
          <div className="space-y-2" data-testid="audit-items">
            {auditResult.results.map((r, i) => (
              <div key={i} className="flex items-center justify-between text-sm border-b border-white/5 pb-2">
                <span className="text-slate-200">{r.rule_id}</span>
                <div className="flex items-center gap-2">
                  <span className="text-slate-400 text-xs">{r.message}</span>
                  <span className={`h-2 w-2 rounded-full ${r.passed ? "bg-emerald-500" : "bg-red-500"}`} />
                </div>
              </div>
            ))}
          </div>
        </Section>
      )}

      {auditError && !auditing && (
        <ErrorAlert
          error={auditError}
          fallbackMessage="Audit evaluation failed."
          testId="audit-error"
        />
      )}

      {rules && (
        <Section title="Available Audit Rules" testId="audit-rules-section">
          <div className="space-y-2">
            {rules.rules.map((rule) => (
              <div key={rule.id} className="text-sm border-b border-white/5 pb-2">
                <div className="flex items-center justify-between">
                  <span className="text-slate-200 font-medium">{rule.name}</span>
                  <span className={`text-xs px-2 py-0.5 rounded-full ${rule.severity === "error" ? "bg-red-500/20 text-red-300" : "bg-amber-500/20 text-amber-300"}`}>
                    {rule.severity}
                  </span>
                </div>
                <p className="text-slate-500 text-xs mt-1">{rule.description}</p>
              </div>
            ))}
          </div>
        </Section>
      )}
    </div>
  );
}
