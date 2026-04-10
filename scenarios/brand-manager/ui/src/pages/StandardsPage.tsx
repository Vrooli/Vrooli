import { useQuery } from "@tanstack/react-query";
import { Shield, ArrowLeft } from "lucide-react";
import { fetchStandards } from "../lib/api";
import { Section } from "../components/ui/section";
import { ErrorAlert } from "../components/ui/error-alert";

// [REQ:BM-REQ-API-STANDARDS] [REQ:BM-REQ-AUDIT-RULES]

interface StandardsPageProps {
  onNavigate: (path: string) => void;
}

export default function StandardsPage({ onNavigate }: StandardsPageProps) {
  const { data: standards, isLoading, error, refetch } = useQuery({
    queryKey: ["standards"],
    queryFn: fetchStandards,
  });

  return (
    <div data-testid="standards-page">
      <button
        onClick={() => onNavigate("/brands")}
        className="flex items-center gap-2 text-sm text-slate-400 hover:text-slate-50 mb-4 transition-colors"
        data-testid="back-to-brands"
      >
        <ArrowLeft className="h-4 w-4" />
        Back to Library
      </button>

      <div className="flex items-center gap-3 mb-6">
        <Shield className="h-6 w-6 text-slate-50" />
        <h1 className="text-2xl font-bold text-slate-50">Brand Standards</h1>
      </div>

      <p className="text-slate-400 text-sm mb-6">
        These rules define the branding standards enforced across all scenarios. They are used by the audit system to validate brand compliance.
      </p>

      {isLoading && (
        <div className="text-center text-slate-400 py-12" data-testid="standards-loading">Loading standards...</div>
      )}

      {error && (
        <ErrorAlert
          error={error}
          fallbackMessage="Failed to load standards."
          onRetry={() => refetch()}
          testId="standards-error"
        />
      )}

      {standards && (
        <Section title="Branding Rules" testId="standards-list">
          <div className="space-y-3">
            {standards.rules.map((rule) => (
              <div key={rule.id} className="p-3 rounded-lg bg-white/5 border border-white/5" data-testid={`standard-${rule.id}`}>
                <div className="flex items-center justify-between mb-1">
                  <span className="text-slate-100 font-medium text-sm">{rule.name}</span>
                  <span className={`text-xs px-2 py-0.5 rounded-full ${
                    rule.severity === "error" ? "bg-red-500/20 text-red-300" :
                    rule.severity === "warning" ? "bg-amber-500/20 text-amber-300" :
                    "bg-slate-500/20 text-slate-300"
                  }`}>
                    {rule.severity}
                  </span>
                </div>
                <p className="text-slate-400 text-xs">{rule.description}</p>
                <p className="text-slate-600 text-xs mt-1">ID: {rule.id}</p>
              </div>
            ))}
          </div>

          {standards.rules.length === 0 && (
            <p className="text-slate-500 text-sm py-4" data-testid="standards-empty">No standards defined.</p>
          )}
        </Section>
      )}
    </div>
  );
}
