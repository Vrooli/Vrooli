/**
 * Brief Preview Component
 *
 * Compact preview of the current brief for the dashboard.
 * Shows key summary and provides quick access to full briefs page.
 *
 * [REQ:LD-BRIEF-MORNING] - Morning brief preview
 * [REQ:LD-BRIEF-EVENING] - Evening review preview
 */
import { Sun, Moon, ChevronRight, AlertCircle } from "lucide-react";
import { Link } from "react-router-dom";
import type { Brief } from "../../lib/api";
import { Card } from "../ui";

interface BriefPreviewProps {
  brief: Brief | null;
  isLoading: boolean;
  error?: Error | null;
}

export default function BriefPreview({ brief, isLoading, error }: BriefPreviewProps) {
  if (isLoading) {
    return (
      <Card padding="lg" data-testid="brief-preview">
        <div className="animate-pulse space-y-3">
          <div className="flex items-center gap-2">
            <div className="w-5 h-5 bg-white/10 rounded" />
            <div className="h-5 bg-white/10 rounded w-24" />
          </div>
          <div className="h-4 bg-white/10 rounded w-full" />
          <div className="h-4 bg-white/10 rounded w-3/4" />
        </div>
      </Card>
    );
  }

  if (error) {
    return (
      <Card padding="lg" data-testid="brief-preview">
        <div className="flex items-center gap-3 text-red-400">
          <AlertCircle className="w-5 h-5 flex-shrink-0" />
          <span className="text-sm">Unable to load brief</span>
        </div>
      </Card>
    );
  }

  if (!brief) {
    return (
      <Card padding="lg" data-testid="brief-preview">
        <div className="text-center text-gray-400 py-4">
          <p className="text-sm">No brief available</p>
        </div>
      </Card>
    );
  }

  const isMorning = brief.type === "morning";
  const Icon = isMorning ? Sun : Moon;
  const title = isMorning ? "Morning Brief" : "Evening Review";
  const iconColor = isMorning ? "text-yellow-400" : "text-indigo-400";

  // Get first 2 sections for preview (safely handle null/undefined sections)
  const sections = brief.sections ?? [];
  const previewSections = sections.slice(0, 2);
  const hasMoreSections = sections.length > 2;

  return (
    <Link to="/briefs" data-testid="brief-preview">
      <Card interactive padding="lg" className="group">
        {/* Header */}
        <div className="flex items-center justify-between mb-3">
          <div className="flex items-center gap-2">
            <Icon className={`w-5 h-5 ${iconColor}`} />
            <span className="font-medium text-white">{title}</span>
          </div>
          <ChevronRight className="w-4 h-4 text-gray-400 group-hover:text-white transition-colors" />
        </div>

        {/* Summary (truncated) */}
        <p className="text-sm text-gray-300 line-clamp-2 mb-3">
          {brief.summary}
        </p>

        {/* Preview sections */}
        {previewSections.length > 0 && (
          <div className="space-y-2 text-sm">
            {previewSections.map((section) => (
              <div key={section.domain} className="flex items-center justify-between">
                <span className="text-gray-400">{section.display_name}</span>
                <span className="text-gray-500">
                  {section.event_count} event{section.event_count !== 1 ? "s" : ""}
                </span>
              </div>
            ))}
            {hasMoreSections && (
              <div className="text-xs text-gray-500 pt-1">
                +{sections.length - 2} more domains
              </div>
            )}
          </div>
        )}

        {/* Score badge (if available) */}
        {brief.score !== undefined && brief.score !== null && (
          <div className="mt-3 pt-3 border-t border-white/10 flex items-center justify-between">
            <span className="text-xs text-gray-400">Lifestyle Score</span>
            <span className="text-lg font-bold text-white">{brief.score}</span>
          </div>
        )}

        {/* Call to action */}
        <div className="mt-3 text-xs text-blue-400 group-hover:text-blue-300 transition-colors">
          View full brief →
        </div>
      </Card>
    </Link>
  );
}
