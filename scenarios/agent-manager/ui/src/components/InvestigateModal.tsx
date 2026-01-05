import { useState } from "react";
import { AlertCircle, Search } from "lucide-react";
import { Button } from "./ui/button";
import {
  Dialog,
  DialogBody,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "./ui/dialog";
import { Label } from "./ui/label";
import { Textarea } from "./ui/textarea";
import type { AnalysisType, CreateInvestigationRequest, ReportSections } from "../types";

interface InvestigateModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  selectedRunIds: string[];
  onInvestigate: (request: CreateInvestigationRequest) => Promise<void>;
  loading?: boolean;
  error?: string | null;
}

export function InvestigateModal({
  open,
  onOpenChange,
  selectedRunIds,
  onInvestigate,
  loading = false,
  error = null,
}: InvestigateModalProps) {
  const [analysisType, setAnalysisType] = useState<AnalysisType>({
    error_diagnosis: true,
    efficiency_analysis: false,
    tool_usage_patterns: false,
  });

  const [reportSections, setReportSections] = useState<ReportSections>({
    root_cause_evidence: true,
    recommendations: true,
    metrics_summary: true,
  });

  const [customContext, setCustomContext] = useState("");

  const handleSubmit = async () => {
    await onInvestigate({
      run_ids: selectedRunIds,
      analysis_type: analysisType,
      report_sections: reportSections,
      custom_context: customContext.trim() || undefined,
    });
  };

  const hasAnalysisType = analysisType.error_diagnosis || analysisType.efficiency_analysis || analysisType.tool_usage_patterns;
  const hasReportSections = reportSections.root_cause_evidence || reportSections.recommendations || reportSections.metrics_summary;

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <DialogHeader onClose={() => onOpenChange(false)}>
          <DialogTitle className="flex items-center gap-2">
            <Search className="h-5 w-5" />
            Investigate {selectedRunIds.length} Run{selectedRunIds.length !== 1 ? "s" : ""}
          </DialogTitle>
          <DialogDescription>
            Analyze the selected runs to identify issues, patterns, and recommendations.
          </DialogDescription>
        </DialogHeader>

        <DialogBody className="space-y-6">
          {/* Analysis Type */}
          <div className="space-y-3">
            <Label className="text-sm font-semibold">Analysis Type</Label>
            <div className="space-y-2">
              <CheckboxItem
                id="error_diagnosis"
                checked={analysisType.error_diagnosis}
                onChange={(checked) =>
                  setAnalysisType({ ...analysisType, error_diagnosis: checked })
                }
                label="Error Diagnosis"
                description="Identify root causes of failures and errors"
              />
              <CheckboxItem
                id="efficiency_analysis"
                checked={analysisType.efficiency_analysis}
                onChange={(checked) =>
                  setAnalysisType({ ...analysisType, efficiency_analysis: checked })
                }
                label="Efficiency Analysis"
                description="Analyze token usage, duration, and cost patterns"
              />
              <CheckboxItem
                id="tool_usage_patterns"
                checked={analysisType.tool_usage_patterns}
                onChange={(checked) =>
                  setAnalysisType({ ...analysisType, tool_usage_patterns: checked })
                }
                label="Tool Usage Patterns"
                description="Examine tool calls and their effectiveness"
              />
            </div>
          </div>

          {/* Report Sections */}
          <div className="space-y-3">
            <Label className="text-sm font-semibold">Report Sections</Label>
            <div className="space-y-2">
              <CheckboxItem
                id="root_cause_evidence"
                checked={reportSections.root_cause_evidence}
                onChange={(checked) =>
                  setReportSections({ ...reportSections, root_cause_evidence: checked })
                }
                label="Root Cause + Evidence"
                description="Detailed analysis with supporting evidence from events"
              />
              <CheckboxItem
                id="recommendations"
                checked={reportSections.recommendations}
                onChange={(checked) =>
                  setReportSections({ ...reportSections, recommendations: checked })
                }
                label="Recommendations"
                description="Actionable suggestions to fix identified issues"
              />
              <CheckboxItem
                id="metrics_summary"
                checked={reportSections.metrics_summary}
                onChange={(checked) =>
                  setReportSections({ ...reportSections, metrics_summary: checked })
                }
                label="Metrics Summary"
                description="Quantitative analysis of run performance"
              />
            </div>
          </div>

          {/* Custom Context */}
          <div className="space-y-2">
            <Label htmlFor="customContext">Additional Context (optional)</Label>
            <Textarea
              id="customContext"
              value={customContext}
              onChange={(e) => setCustomContext(e.target.value)}
              placeholder="Provide any additional context for the investigation..."
              rows={3}
            />
            <p className="text-xs text-muted-foreground">
              Help focus the investigation by describing what you suspect or want to understand.
            </p>
          </div>

          {error && (
            <div className="flex items-start gap-2 rounded-md border border-destructive/50 bg-destructive/10 px-3 py-2 text-sm text-destructive">
              <AlertCircle className="h-4 w-4 mt-0.5" />
              <span>{error}</span>
            </div>
          )}
        </DialogBody>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)} disabled={loading}>
            Cancel
          </Button>
          <Button
            onClick={handleSubmit}
            disabled={loading || !hasAnalysisType || !hasReportSections}
            className="gap-2"
          >
            {loading ? (
              "Starting..."
            ) : (
              <>
                <Search className="h-4 w-4" />
                Start Investigation
              </>
            )}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

interface CheckboxItemProps {
  id: string;
  checked: boolean;
  onChange: (checked: boolean) => void;
  label: string;
  description?: string;
}

function CheckboxItem({ id, checked, onChange, label, description }: CheckboxItemProps) {
  return (
    <label
      htmlFor={id}
      className="flex items-start gap-3 rounded-md border border-border p-3 cursor-pointer hover:bg-muted/50 transition-colors"
    >
      <input
        type="checkbox"
        id={id}
        checked={checked}
        onChange={(e) => onChange(e.target.checked)}
        className="mt-0.5 h-4 w-4 rounded border-border text-primary focus:ring-primary focus:ring-offset-0"
      />
      <div className="flex-1">
        <span className="text-sm font-medium">{label}</span>
        {description && (
          <p className="text-xs text-muted-foreground">{description}</p>
        )}
      </div>
    </label>
  );
}
