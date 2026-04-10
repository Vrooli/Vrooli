import { useState } from "react";
import { useMutation } from "@tanstack/react-query";
import { FileCode, Check, AlertTriangle, Play } from "lucide-react";
import { fetchApplyPreview, type ApplyPreviewResult } from "../lib/api";
import { Button } from "./ui/button";
import { Input } from "./ui/input";
import { Section } from "./ui/section";

// [REQ:BM-REQ-UI-APPLY]

interface ApplyPreviewProps {
  brandId: string;
}

export function ApplyPreview({ brandId }: ApplyPreviewProps) {
  const [scenarioName, setScenarioName] = useState("");
  const [preview, setPreview] = useState<ApplyPreviewResult | null>(null);

  const previewMutation = useMutation({
    mutationFn: () => fetchApplyPreview(brandId, scenarioName),
    onSuccess: (data) => setPreview(data),
  });

  return (
    <Section testId="apply-preview-section">
      <h2 className="text-sm font-medium text-slate-400 mb-3">Apply Preview</h2>

      <div className="flex gap-2 mb-3">
        <Input
          placeholder="Scenario name..."
          value={scenarioName}
          onChange={(e) => setScenarioName(e.target.value)}
          data-testid="apply-scenario-input"
        />
        <Button
          variant="outline"
          size="sm"
          disabled={!scenarioName.trim() || previewMutation.isPending}
          onClick={() => previewMutation.mutate()}
          data-testid="apply-preview-btn"
        >
          <Play className="h-3 w-3 mr-1" />
          {previewMutation.isPending ? "Checking..." : "Preview"}
        </Button>
      </div>

      {previewMutation.error && (
        <div className="text-red-400 text-xs mb-2" data-testid="apply-preview-error">
          {previewMutation.error instanceof Error ? previewMutation.error.message : "Preview failed"}
        </div>
      )}

      {preview && (
        <div className="rounded-lg border border-white/10 overflow-hidden" data-testid="apply-preview-results">
          <div className="p-3 bg-slate-900/50 border-b border-white/10">
            <p className="text-xs text-slate-400">
              Preview for <span className="text-slate-200 font-medium">{preview.scenario}</span>
              {" "}— Brand v{preview.brand_version}
            </p>
          </div>

          {/* Applied actions */}
          {preview.applied && preview.applied.length > 0 && (
            <div className="p-3">
              <p className="text-xs text-slate-500 mb-2">Changes to be applied:</p>
              <div className="space-y-1.5">
                {preview.applied.map((action, i) => (
                  <div key={i} className="flex items-center gap-2 text-xs" data-testid={`apply-action-${i}`}>
                    <Check className="h-3 w-3 text-emerald-400 shrink-0" />
                    <FileCode className="h-3 w-3 text-slate-500 shrink-0" />
                    <span className="text-slate-300">{action.file}</span>
                    <span className="text-slate-500">({action.type}: {action.element})</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {/* Skipped elements */}
          {preview.skipped && preview.skipped.length > 0 && (
            <div className="p-3 border-t border-white/10">
              <p className="text-xs text-slate-500 mb-2">Skipped:</p>
              <div className="space-y-1">
                {preview.skipped.map((skip, i) => (
                  <div key={i} className="flex items-center gap-2 text-xs">
                    <AlertTriangle className="h-3 w-3 text-amber-400 shrink-0" />
                    <span className="text-slate-400">{skip.element}</span>
                    <span className="text-slate-600">— {skip.reason}</span>
                  </div>
                ))}
              </div>
            </div>
          )}

          {(!preview.applied || preview.applied.length === 0) && (!preview.skipped || preview.skipped.length === 0) && (
            <div className="p-3 text-xs text-slate-500 text-center">
              No changes detected
            </div>
          )}
        </div>
      )}
    </Section>
  );
}
