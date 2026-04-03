/**
 * EvidenceItemCard
 *
 * Renders a single evidence item with type-specific visualization.
 * Supports screenshots (inline thumbnail), API tests (pass/fail list),
 * CLI output (code block), and other types.
 *
 * Unreviewed items appear with a left accent bar and bold title (like an
 * unread message). Clicking "Mark as reviewed" clears the accent.
 */

import { useState } from "react";
import {
  Camera,
  Terminal,
  Globe,
  FileCode,
  Video,
  FileText,
  ChevronDown,
  ChevronRight,
} from "lucide-react";
import type { EvidenceItem, EvidenceType } from "../../services/review-service";
import { buildApiUrl } from "@vrooli/api-base";
import { API_ENDPOINTS } from "../../lib/api-endpoints";
import { MediaLightbox } from "../ui/media-lightbox";
import { selectors } from "../../consts/selectors";
import { CaptureContentViewer } from "./capture-content-viewer";

export interface EvidenceItemCardProps {
  item: EvidenceItem;
  backlogKind: string;
  backlogName: string;
  onVerify: (evidenceId: string, verified: boolean) => void;
}

const typeIcons: Record<EvidenceType, typeof Camera> = {
  screenshot: Camera,
  api_test: Globe,
  cli_output: Terminal,
  config_diff: FileCode,
  workflow_recording: Video,
  custom: FileText,
};

const typeLabels: Record<EvidenceType, string> = {
  screenshot: "Screenshot",
  api_test: "API Test",
  cli_output: "CLI Output",
  config_diff: "Config Diff",
  workflow_recording: "Recording",
  custom: "Custom",
};

export function EvidenceItemCard({
  item,
  backlogKind,
  backlogName,
  onVerify,
}: EvidenceItemCardProps) {
  const [expanded, setExpanded] = useState(false);
  const Icon = typeIcons[item.type] ?? FileText;
  const isUnread = !item.verified;

  return (
    <div
      className={`rounded-md border bg-white dark:bg-slate-900 ${
        isUnread
          ? "border-l-[3px] border-l-violet-500 border-t-slate-200 border-r-slate-200 border-b-slate-200 dark:border-t-slate-700 dark:border-r-slate-700 dark:border-b-slate-700"
          : "border-slate-200 dark:border-slate-700"
      }`}
    >
      {/* Header row */}
      <div className="flex items-center gap-2 px-3 py-2">
        {/* Review checkbox */}
        <input
          type="checkbox"
          checked={item.verified}
          onChange={() => onVerify(item.id, !item.verified)}
          className="h-3.5 w-3.5 flex-shrink-0 accent-cyan-500 cursor-pointer"
          title={item.verified ? "Mark as unreviewed" : "Mark as reviewed"}
          data-testid={selectors.evidence.reviewCheckbox}
        />

        {/* Unread indicator dot */}
        {isUnread && (
          <span className="h-2 w-2 flex-shrink-0 rounded-full bg-violet-500" />
        )}

        {/* Type badge */}
        <span className="flex items-center gap-1 rounded bg-slate-100 px-1.5 py-0.5 text-xs text-slate-500 dark:bg-slate-800 dark:text-slate-400">
          <Icon className="h-3 w-3" />
          {typeLabels[item.type]}
        </span>

        {/* Title — bold when unread */}
        <span
          className={`flex-1 line-clamp-2 text-xs ${
            isUnread
              ? "font-semibold text-slate-800 dark:text-slate-100"
              : "font-normal text-slate-500 dark:text-slate-400"
          }`}
        >
          {item.title}
        </span>

        {/* Expand/collapse for details */}
        <button
          onClick={() => setExpanded((e) => !e)}
          className="flex-shrink-0 text-slate-400 hover:text-slate-600"
        >
          {expanded ? (
            <ChevronDown className="h-3.5 w-3.5" />
          ) : (
            <ChevronRight className="h-3.5 w-3.5" />
          )}
        </button>
      </div>

      {/* Description — always visible */}
      {item.description && (
        <p className="px-3 pb-2 text-xs text-slate-500 dark:text-slate-400 leading-relaxed">
          {item.description}
        </p>
      )}

      {/* Expanded content — type-specific rendering */}
      {expanded && (
        <div className="border-t border-slate-100 px-3 py-2 dark:border-slate-800">
          {/* Type-specific content */}
          {item.type === "screenshot" && item.capture_path && (
            <ScreenshotEvidence
              backlogKind={backlogKind}
              backlogName={backlogName}
              capturePath={item.capture_path}
              beforeCapturePath={item.before_capture_path}
            />
          )}

          {item.type === "api_test" && item.test_results && (
            <TestResultsEvidence results={item.test_results} />
          )}

          {item.type === "cli_output" && item.capture_path && (
            <CLIOutputEvidence
              backlogKind={backlogKind}
              backlogName={backlogName}
              capturePath={item.capture_path}
            />
          )}

          {item.type === "config_diff" && item.capture_path && (
            <ConfigDiffEvidence
              backlogKind={backlogKind}
              backlogName={backlogName}
              capturePath={item.capture_path}
            />
          )}

          {item.type === "workflow_recording" && item.capture_path && (
            <WorkflowRecordingEvidence
              backlogKind={backlogKind}
              backlogName={backlogName}
              capturePath={item.capture_path}
            />
          )}
        </div>
      )}
    </div>
  );
}

// --- Type-specific renderers ---

function ScreenshotEvidence({
  backlogKind,
  backlogName,
  capturePath,
  beforeCapturePath,
}: {
  backlogKind: string;
  backlogName: string;
  capturePath: string;
  beforeCapturePath?: string;
}) {
  const afterUrl = buildApiUrl(
    API_ENDPOINTS.reviewCapture(backlogKind, backlogName, capturePath),
    { appendSuffix: true },
  );

  return (
    <div className="flex gap-2">
      {beforeCapturePath && (
        <div className="flex-1">
          <div className="mb-1 text-xs font-medium text-slate-400">Before</div>
          <img
            src={buildApiUrl(
              API_ENDPOINTS.reviewCapture(backlogKind, backlogName, beforeCapturePath),
              { appendSuffix: true },
            )}
            alt="Before"
            className="max-h-48 rounded border border-slate-200 object-contain dark:border-slate-700"
          />
        </div>
      )}
      <div className="flex-1">
        {beforeCapturePath && (
          <div className="mb-1 text-xs font-medium text-slate-400">After</div>
        )}
        <img
          src={afterUrl}
          alt={capturePath}
          className="max-h-48 rounded border border-slate-200 object-contain dark:border-slate-700"
        />
      </div>
    </div>
  );
}

function TestResultsEvidence({
  results,
}: {
  results: { name: string; passed: boolean; output_summary?: string }[];
}) {
  return (
    <div className="space-y-1">
      {results.map((r, i) => (
        <div key={i} className="flex items-start gap-2 text-xs">
          <span
            className={`mt-0.5 flex-shrink-0 rounded px-1.5 py-0.5 font-mono ${
              r.passed
                ? "bg-emerald-100 text-emerald-700 dark:bg-emerald-900/30 dark:text-emerald-400"
                : "bg-red-100 text-red-700 dark:bg-red-900/30 dark:text-red-400"
            }`}
          >
            {r.passed ? "PASS" : "FAIL"}
          </span>
          <div>
            <div className="text-slate-700 dark:text-slate-200">{r.name}</div>
            {r.output_summary && (
              <div className="text-slate-400 dark:text-slate-500">
                {r.output_summary}
              </div>
            )}
          </div>
        </div>
      ))}
    </div>
  );
}

function CLIOutputEvidence({
  backlogKind,
  backlogName,
  capturePath,
}: {
  backlogKind: string;
  backlogName: string;
  capturePath: string;
}) {
  return (
    <CaptureContentViewer
      backlogKind={backlogKind}
      backlogName={backlogName}
      capturePath={capturePath}
      loadingLabel="Loading output..."
      openLabel="Open full output"
      testId={selectors.evidence.cliOutput}
      renderContent={(content) => (
        <pre className="max-h-64 overflow-auto rounded bg-slate-900 p-2 text-xs font-mono text-slate-300 dark:bg-slate-950">
          {content}
        </pre>
      )}
    />
  );
}

function ConfigDiffEvidence({
  backlogKind,
  backlogName,
  capturePath,
}: {
  backlogKind: string;
  backlogName: string;
  capturePath: string;
}) {
  return (
    <CaptureContentViewer
      backlogKind={backlogKind}
      backlogName={backlogName}
      capturePath={capturePath}
      loadingLabel="Loading diff..."
      openLabel="Open full diff"
      testId={selectors.evidence.configDiff}
      renderContent={(content) => (
        <pre className="max-h-64 overflow-auto rounded bg-slate-900 p-2 text-xs font-mono dark:bg-slate-950">
          {content.split("\n").map((line, i) => (
            <div key={i} className={diffLineClass(line)}>
              {line}
            </div>
          ))}
        </pre>
      )}
    />
  );
}

function diffLineClass(line: string): string {
  if (line.startsWith("@@")) return "text-cyan-400";
  if (line.startsWith("+++") || line.startsWith("---")) return "text-slate-400";
  if (line.startsWith("+")) return "bg-emerald-900/30 text-emerald-300";
  if (line.startsWith("-")) return "bg-red-900/30 text-red-300";
  return "text-slate-300";
}

function WorkflowRecordingEvidence({
  backlogKind,
  backlogName,
  capturePath,
}: {
  backlogKind: string;
  backlogName: string;
  capturePath: string;
}) {
  const [lightboxOpen, setLightboxOpen] = useState(false);
  const videoUrl = buildApiUrl(
    API_ENDPOINTS.reviewCapture(backlogKind, backlogName, capturePath),
    { appendSuffix: true },
  );

  return (
    <div data-testid={selectors.evidence.workflowRecording}>
      <video
        controls
        preload="metadata"
        src={videoUrl}
        className="max-h-48 w-full cursor-pointer rounded border border-slate-200 dark:border-slate-700"
        onClick={(e) => {
          e.preventDefault();
          setLightboxOpen(true);
        }}
      />
      <MediaLightbox
        isOpen={lightboxOpen}
        onClose={() => setLightboxOpen(false)}
        src={videoUrl}
        type="video"
        label={capturePath}
      />
    </div>
  );
}
