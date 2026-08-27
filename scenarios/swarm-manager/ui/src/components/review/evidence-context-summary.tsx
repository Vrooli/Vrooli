/**
 * EvidenceContextSummary — Compact, read-only evidence display
 * for use inside the follow-up sheet. Shows the latest round's
 * classification, assessment, and evidence item titles.
 */

import { AlertTriangle, Check, FileText, Image, Terminal, Diff, Video, HelpCircle } from "lucide-react";
import { MarkdownRenderer } from "@vrooli/react-component-library/markdown-renderer/0/0.3.2";
import { selectors } from "../../consts/selectors";
import type { ReviewRound, EvidenceItem } from "../../services/review-service";

function evidenceIcon(type: EvidenceItem["type"]) {
  switch (type) {
    case "screenshot": return Image;
    case "api_test": return FileText;
    case "cli_output": return Terminal;
    case "config_diff": return Diff;
    case "workflow_recording": return Video;
    default: return HelpCircle;
  }
}

export interface EvidenceContextSummaryProps {
  rounds: ReviewRound[];
  selectable?: boolean;
  selectedIds?: Set<string>;
  onToggle?: (evidenceId: string) => void;
}

export function EvidenceContextSummary({ rounds, selectable, selectedIds, onToggle }: EvidenceContextSummaryProps) {
  if (rounds.length === 0) return null;

  const latest = rounds[rounds.length - 1];
  if (!latest) return null;
  const classification = latest.classification;
  const isIssue = classification === "needs_work" || classification === "not_assessable";

  return (
    <div
      className="rounded-md border border-slate-700 bg-slate-800/50 p-3 space-y-2"
      data-testid={selectors.review.evidenceContextSummary}
    >
      <div className="flex items-center gap-2">
        {isIssue
          ? <AlertTriangle className="h-3.5 w-3.5 text-red-400" />
          : <Check className="h-3.5 w-3.5 text-emerald-400" />
        }
        <span className="text-xs font-medium text-slate-300">
          Round {latest.round} — {classification?.replace("_", " ") ?? "pending"}
        </span>
      </div>

      {latest.agent_assessment && (
        <MarkdownRenderer content={latest.agent_assessment} className="prose-sm-slate text-[11px] leading-relaxed text-slate-400" />
      )}

      {latest.evidence.length > 0 && (
        <div className="space-y-1">
          {latest.evidence.map((item) => {
            const Icon = evidenceIcon(item.type);
            const inner = (
              <>
                {selectable && (
                  <input
                    type="checkbox"
                    checked={selectedIds?.has(item.id) ?? false}
                    onChange={() => onToggle?.(item.id)}
                    className="h-3 w-3 accent-cyan-500"
                  />
                )}
                <Icon className="h-3 w-3 shrink-0" />
                <span className="truncate">{item.title}</span>
                {item.verified && (
                  <Check className="h-3 w-3 shrink-0 text-emerald-500" />
                )}
              </>
            );
            return selectable ? (
              <label key={item.id} className="flex items-center gap-1.5 text-[11px] text-slate-500 cursor-pointer hover:text-slate-400">
                {inner}
              </label>
            ) : (
              <div key={item.id} className="flex items-center gap-1.5 text-[11px] text-slate-500">
                {inner}
              </div>
            );
          })}
        </div>
      )}
    </div>
  );
}
