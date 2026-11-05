/**
 * Shared stage components for consistent section rendering.
 * These components extract common patterns from section components.
 */

import type { ReactNode } from "react";
import type { LucideIcon } from "lucide-react";
import { Badge } from "../../ui/badge";
import { cn } from "../../../lib/utils";
import type { StatusDisplayConfig } from "../../../lib/status-display";
import { PipelineErrorRecovery } from "../../pipeline/PipelineErrorDisplay";
import type { PipelineErrorInfo } from "../../../store/pipelineTypes";

interface StageStatusOverviewProps {
  icon: LucideIcon;
  title: string;
  description: string;
  statusDisplay: StatusDisplayConfig;
}

/**
 * Status overview card shown at the top of each stage section.
 * Displays stage icon, title, description, and status badge.
 */
export function StageStatusOverview({
  icon: Icon,
  title,
  description,
  statusDisplay,
}: StageStatusOverviewProps) {
  const StatusIcon = statusDisplay.icon;
  return (
    <div className="flex items-center justify-between rounded-lg border border-slate-800 bg-slate-900/50 p-4">
      <div className="flex items-center gap-3">
        <Icon className="h-5 w-5 text-slate-400" />
        <div>
          <p className="text-sm font-medium text-slate-200">{title}</p>
          <p className="text-xs text-slate-400">{description}</p>
        </div>
      </div>
      <Badge
        variant="outline"
        className={cn("flex items-center gap-1.5", statusDisplay.className)}
      >
        <StatusIcon className="h-3 w-3" />
        {statusDisplay.label}
      </Badge>
    </div>
  );
}

interface StagePlaceholderProps {
  scenarioName: string;
  withScenarioText: string;
  withoutScenarioText?: string;
}

/**
 * Placeholder shown when stage has no results and is pending.
 */
export function StagePlaceholder({
  scenarioName,
  withScenarioText,
  withoutScenarioText = "Select a scenario to begin.",
}: StagePlaceholderProps) {
  return (
    <div className="rounded-lg border border-slate-800 bg-slate-950/50 p-4 text-center">
      <p className="text-sm text-slate-400">
        {scenarioName ? withScenarioText : withoutScenarioText}
      </p>
    </div>
  );
}

interface StageErrorProps {
  stageName: string;
  children?: ReactNode;
  /** Structured error info with recovery suggestions */
  errorInfo?: PipelineErrorInfo | null;
  /** Callback when retry button is clicked */
  onRetry?: () => void;
  /** Callback when dismiss button is clicked */
  onDismiss?: () => void;
}

/**
 * Error state shown when stage has failed.
 * Supports both simple error messages and structured error recovery.
 */
export function StageError({
  stageName,
  children,
  errorInfo,
  onRetry,
  onDismiss,
}: StageErrorProps) {
  // Use enhanced error recovery component when error info is available
  if (errorInfo) {
    return (
      <PipelineErrorRecovery
        errorInfo={errorInfo}
        onRetry={onRetry}
        onDismiss={onDismiss}
      />
    );
  }

  // Fallback to simple error display
  return (
    <div className="rounded-lg border border-red-900/50 bg-red-950/20 p-4">
      <p className="text-sm text-red-300">
        {children ??
          `${stageName} stage failed. Check the pipeline logs for details.`}
      </p>
    </div>
  );
}

interface StageAboutProps {
  title: string;
  children: ReactNode;
}

/**
 * About/info card for a stage.
 */
export function StageAbout({ title, children }: StageAboutProps) {
  return (
    <div className="rounded-lg border border-slate-800 bg-slate-950/50 p-3 text-sm text-slate-300">
      <p className="font-semibold text-slate-200">{title}</p>
      {children}
    </div>
  );
}

interface StageDetailCardProps {
  icon: LucideIcon;
  label: string;
  children: ReactNode;
}

/**
 * Detail card for displaying a piece of stage information (e.g., file path).
 */
export function StageDetailCard({
  icon: Icon,
  label,
  children,
}: StageDetailCardProps) {
  return (
    <div className="rounded-lg border border-slate-800 bg-slate-950/50 p-3">
      <div className="flex items-center gap-2 text-xs text-slate-400 mb-1">
        <Icon className="h-3.5 w-3.5" />
        {label}
      </div>
      {children}
    </div>
  );
}

interface StageWarningProps {
  children: ReactNode;
}

/**
 * Warning card for displaying cautions or non-critical issues.
 */
export function StageWarning({ children }: StageWarningProps) {
  return (
    <div className="rounded-lg border border-yellow-900/50 bg-yellow-950/20 p-3 text-sm text-yellow-300">
      {children}
    </div>
  );
}
