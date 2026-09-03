/** @vrooliComponentSource react-component-library:StatusBadge */
import { useEffect, useState } from "react";
import {
  CheckCircle2,
  ChevronRight,
  CircleAlert,
  Play,
  ScanSearch,
  ShieldCheck,
} from "lucide-react";
import { Link } from "react-router-dom";

import type {
  ComponentTestArtifact,
  ComponentTestReport,
  ComponentTestResult,
} from "../../api/componentTests";
import { API_BASE } from "../../api/client";
import { Button } from "@vrooli/react-component-library/Button/2";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { assetSearchForTab } from "../../routes";
import type { ComponentExperience } from "../../api/components";
import { OverlayCanvas } from "../../../../library/components/OverlayCanvas/versions/1.0.7/OverlayCanvas";
import {
  EvidenceCarousel,
  type EvidenceItem,
} from "../../../../library/components/EvidenceCarousel/versions/1.0.14/EvidenceCarousel";

type VerdictTone = "success" | "danger" | "warning" | "neutral";

export function tone(verdict: string): VerdictTone {
  return verdict === "passed"
    ? "success"
    : verdict === "failed"
      ? "danger"
      : verdict === "blocked" || verdict === "unmeasured"
        ? "warning"
        : "neutral";
}

export function verdictLabel(verdict: string) {
  return verdict === "passed"
    ? "Passed"
    : verdict === "failed"
      ? "Needs attention"
      : verdict === "blocked"
        ? "Blocked"
        : verdict === "unmeasured"
          ? "Unmeasured"
          : "Inconclusive";
}

export const EVIDENCE_KINDS: ReadonlyArray<{
  id: string;
  label: string;
  aliases: readonly string[];
}> = [
  { id: "story-sheet", label: "Story sheet", aliases: ["bas-story-sheet"] },
  { id: "screenshot", label: "Screenshot", aliases: ["screenshot", "bas-screenshot"] },
  {
    id: "accessibility-tree",
    label: "Accessibility Tree",
    aliases: ["accessibility", "bas-accessibility"],
  },
  // BAS returns computed styles and bounds inline in its accessibility snapshot,
  // so these are intentionally mapped to the accessibility artifact rather than
  // presented as falsely unavailable.
  {
    id: "computed-style",
    label: "Computed Styles",
    aliases: ["computed-style", "computed_style", "accessibility", "bas-accessibility"],
  },
  {
    id: "layout-box",
    label: "Layout Box",
    aliases: ["layout-box", "layout_box", "accessibility", "bas-accessibility"],
  },
  { id: "console", label: "Console", aliases: ["console", "console_logs", "bas-console_logs"] },
  { id: "performance", label: "Performance", aliases: ["performance", "bas-performance"] },
] as const;

function StageRow({ result }: { result: ComponentTestResult }) {
  const isPassing = result.verdict === "passed";
  return (
    <li className="grid gap-space-2xs rounded-control border border-app-border bg-app-background p-space-xs sm:grid-cols-[auto_minmax(0,1fr)_auto] sm:items-start">
      <span
        className={`mt-space-3xs flex h-control-compact w-control-compact items-center justify-center rounded-pill ${isPassing ? "bg-app-success/10 text-app-success" : "bg-app-danger/10 text-app-danger"}`}
        aria-hidden
      >
        {isPassing ? (
          <CheckCircle2 className="h-icon-sm w-icon-sm" />
        ) : (
          <CircleAlert className="h-icon-sm w-icon-sm" />
        )}
      </span>
      <div className="min-w-0">
        <div className="flex flex-wrap items-center gap-space-2xs">
          <h5 className="font-medium capitalize">{result.stage.replace(/_/g, " ")}</h5>
          <span className="font-mono text-xs text-app-muted-foreground">
            {result.assetLibraryId}@{result.version}
          </span>
        </div>
        {(result.ruleSource || result.ruleDeclaredIn) && (
          <p className="mt-space-3xs text-xs text-app-muted-foreground">
            Rule: <span className="font-mono">{result.ruleSource || "unknown"}</span>
            {result.ruleDeclaredIn && (
              <>
                {" · "}
                <code>{result.ruleDeclaredIn}</code>
              </>
            )}
          </p>
        )}
        {result.message && (
          <p className="mt-space-3xs text-xs text-app-muted-foreground">{result.message}</p>
        )}
        {result.remediation && (
          <p className="mt-space-2xs rounded-control border border-app-warning/30 bg-app-warning/10 px-space-2xs py-space-2xs text-xs text-app-foreground">
            <span className="font-medium text-app-warning">Recommended next step:</span>{" "}
            {result.remediation}
          </p>
        )}
      </div>
      <StatusBadge tone={tone(result.verdict)}>{verdictLabel(result.verdict)}</StatusBadge>
    </li>
  );
}

export type CaptureItem = EvidenceItem & {
  label: string;
  available: boolean;
  artifact?: ComponentTestArtifact;
};

export function storyCaptureLabel(storyID: string): string {
  if (storyID.startsWith("review-sheet:")) {
    return `Review sheet · ${storyID.slice("review-sheet:".length).split(",").join(", ")}`;
  }
  return storyID;
}

export function browserVisibleArtifactUrl(reference: string): string {
  const rclBase = API_BASE.replace(/\/$/, "");
  if (reference.startsWith("/embedded/")) return `${rclBase}${reference}`;

  // Historical reports persisted BAS's loopback URL before the embedded
  // proxy route existed. Rewrite those records at read time so opening old
  // evidence never asks a hosted RCL page for loopback-network permission.
  try {
    const parsed = new URL(reference);
    if (
      (parsed.hostname === "127.0.0.1" || parsed.hostname === "localhost") &&
      parsed.pathname.startsWith("/api/v1/artifacts/")
    ) {
      return `${rclBase}/embedded/browser-automation-studio${parsed.pathname}${parsed.search}`;
    }
  } catch {
    // Non-URL references are internal provenance identifiers, not browser URLs.
  }
  return reference;
}

const MAX_EVIDENCE_BYTES = 8_000_000;

export async function readBoundedEvidenceText(
  response: Response,
  label: string,
  maxBytes = MAX_EVIDENCE_BYTES,
): Promise<string> {
  const contentLength = Number(response.headers.get("content-length") ?? 0);
  const tooLarge = () =>
    new Error(`${label} is too large to render safely (over ${Math.round(maxBytes / 1024)} KB).`);
  if (contentLength > maxBytes) throw tooLarge();

  // Do not fall back to response.text() when a proxy omits Content-Length.
  // Large Chrome traces can otherwise be fully materialized before the UI
  // notices that rendering them would freeze the evidence workspace.
  if (!response.body) {
    const text = await response.text();
    if (new TextEncoder().encode(text).byteLength > maxBytes) throw tooLarge();
    return text;
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let bytes = 0;
  let text = "";
  try {
    while (true) {
      const { done, value } = await reader.read();
      if (done) {
        text += decoder.decode();
        return text;
      }
      bytes += value.byteLength;
      if (bytes > maxBytes) {
        await reader.cancel();
        throw tooLarge();
      }
      text += decoder.decode(value, { stream: true });
    }
  } finally {
    reader.releaseLock();
  }
}

export function summarizePerformance(value: unknown): {
  summary: Record<string, unknown>;
  truncated: boolean;
} {
  if (!value || typeof value !== "object" || Array.isArray(value)) {
    return { summary: { value }, truncated: false };
  }
  const record = value as Record<string, unknown>;
  const traceEvents = Array.isArray(record.traceEvents) ? record.traceEvents : [];
  const names = new Map<string, number>();
  let minTimestamp = Number.POSITIVE_INFINITY;
  let maxTimestamp = Number.NEGATIVE_INFINITY;
  traceEvents.forEach((event) => {
    if (!event || typeof event !== "object") return;
    const item = event as Record<string, unknown>;
    const name = typeof item.name === "string" ? item.name : "unnamed";
    names.set(name, (names.get(name) ?? 0) + 1);
    if (typeof item.ts === "number") {
      minTimestamp = Math.min(minTimestamp, item.ts);
      maxTimestamp = Math.max(
        maxTimestamp,
        item.ts + (typeof item.dur === "number" ? item.dur : 0),
      );
    }
  });
  const topEvents = [...names.entries()]
    .sort((a, b) => b[1] - a[1])
    .slice(0, 12)
    .map(([name, count]) => ({ name, count }));
  return {
    summary: {
      capture: "Performance trace",
      traceEventCount: traceEvents.length,
      durationMs:
        Number.isFinite(minTimestamp) && Number.isFinite(maxTimestamp)
          ? Math.round((maxTimestamp - minTimestamp) / 1000)
          : null,
      webVitals: record.metadata,
      mostFrequentEvents: topEvents,
    },
    truncated: traceEvents.length > 0,
  };
}

function PerformanceArtifactSummary({ value }: { value: Record<string, unknown> }) {
  const events = Array.isArray(value.mostFrequentEvents) ? value.mostFrequentEvents : [];
  const max = Math.max(...events.map((event) => (event as { count?: number }).count ?? 0), 1);
  return (
    <div className="space-y-space-sm p-space-sm">
      <div className="grid gap-space-xs sm:grid-cols-3">
        {[
          ["Trace events", value.traceEventCount ?? "—"],
          ["Captured duration", value.durationMs == null ? "—" : `${value.durationMs} ms`],
          ["Payload", "Summarized"],
        ].map(([title, metric]) => (
          <div
            key={String(title)}
            className="rounded-control border border-app-border bg-app-surface p-space-xs"
          >
            <p className="text-xs text-app-muted-foreground">{String(title)}</p>
            <p className="mt-space-3xs text-lg font-semibold">{String(metric)}</p>
          </div>
        ))}
      </div>
      <div className="rounded-control border border-app-border bg-app-surface p-space-xs">
        <div className="flex items-center justify-between gap-space-xs">
          <h4 className="text-xs font-semibold uppercase tracking-wide">Most frequent events</h4>
          <span className="text-xs text-app-muted-foreground">Top 12</span>
        </div>
        <div className="mt-space-xs space-y-space-2xs">
          {events.length ? (
            events.map((event, index) => {
              const item = event as { name?: string; count?: number };
              const count = item.count ?? 0;
              return (
                <div key={`${item.name ?? "event"}-${index}`} className="text-xs">
                  <div className="flex justify-between gap-space-xs">
                    <span className="truncate font-mono">{item.name ?? "Unnamed event"}</span>
                    <span className="text-app-muted-foreground">{count}</span>
                  </div>
                  <div className="mt-space-3xs h-1.5 overflow-hidden rounded-pill bg-app-surface-muted">
                    <div
                      className="h-full rounded-pill bg-app-primary"
                      style={{ width: `${Math.max(4, (count / max) * 100)}%` }}
                    />
                  </div>
                </div>
              );
            })
          ) : (
            <p className="text-xs text-app-muted-foreground">No trace events were recorded.</p>
          )}
        </div>
      </div>
      <p className="text-xs text-app-muted-foreground">
        The raw Chrome trace remains available as the durable artifact; this view is summarized to
        keep the evidence viewer responsive.
      </p>
    </div>
  );
}

function StructuredArtifact({ reference, label }: { reference: string; label: string }) {
  const [state, setState] = useState<{
    status: "loading" | "ready" | "error";
    text: string;
    value?: Record<string, unknown>;
  }>({
    status: "loading",
    text: "",
  });

  useEffect(() => {
    const controller = new AbortController();
    setState({ status: "loading", text: "" });
    void fetch(browserVisibleArtifactUrl(reference), {
      signal: controller.signal,
      credentials: "include",
    })
      .then(async (response) => {
        if (!response.ok) throw new Error(`Evidence request returned ${response.status}.`);
        return readBoundedEvidenceText(response, label);
      })
      .then((text) => {
        let formatted = text;
        let structuredValue: Record<string, unknown> | undefined;
        try {
          const parsed = JSON.parse(text) as unknown;
          const result =
            label === "Performance"
              ? summarizePerformance(parsed)
              : { summary: parsed, truncated: false };
          formatted = JSON.stringify(result.summary, null, 2);
          if (label === "Performance" && result.summary && typeof result.summary === "object") {
            structuredValue = result.summary as Record<string, unknown>;
          }
          if (result.truncated) {
            formatted +=
              "\n\nTrace events are summarized above to keep the evidence viewer responsive.";
          }
        } catch {
          // Markdown and plain-text artifacts are already display-ready.
        }
        setState({
          status: "ready",
          text: formatted,
          value: structuredValue,
        });
      })
      .catch((error: unknown) => {
        if (error instanceof DOMException && error.name === "AbortError") return;
        setState({
          status: "error",
          text: error instanceof Error ? error.message : "The evidence could not be loaded.",
        });
      });
    return () => controller.abort();
  }, [reference]);

  return (
    <div className="min-h-[18rem] overflow-auto p-space-sm">
      {state.status === "loading" ? (
        <p className="text-xs text-app-muted-foreground">Loading {label}…</p>
      ) : state.status === "error" ? (
        <p role="alert" className="text-xs text-app-danger">
          {state.text}
        </p>
      ) : label === "Performance" && state.value ? (
        <PerformanceArtifactSummary value={state.value} />
      ) : (
        <pre className="whitespace-pre-wrap break-words rounded-control border border-app-border bg-app-surface p-space-sm font-mono text-xs leading-relaxed text-app-foreground">
          {state.text}
        </pre>
      )}
    </div>
  );
}

function ScreenshotArtifact({
  reference,
  alt = "Captured component screenshot",
  showOverlay,
  overlaySubjects,
  overlayMessage,
}: {
  reference: string;
  alt?: string;
  showOverlay: boolean;
  overlaySubjects: Array<{
    id: string;
    label: string;
    x: number;
    y: number;
    width: number;
    height: number;
  }>;
  overlayMessage: string;
}) {
  const [status, setStatus] = useState<"loading" | "ready" | "error">("loading");

  return (
    <div className="relative flex min-h-[18rem] items-center justify-center overflow-auto bg-app-surface-muted p-space-sm">
      {status === "loading" ? (
        <div
          role="status"
          className="absolute inset-0 flex items-center justify-center text-xs text-app-muted-foreground"
        >
          Loading captured screenshot…
        </div>
      ) : null}
      {status === "error" ? (
        <div role="alert" className="max-w-md text-center">
          <p className="text-sm font-medium">Screenshot could not be displayed</p>
          <p className="mt-space-3xs text-xs text-app-muted-foreground">
            The capture exists, but its embedded artifact route did not return an image. Rerun the
            component test to publish a fresh capture.
          </p>
        </div>
      ) : null}
      <img
        src={reference}
        alt={alt}
        className={`max-h-[42rem] max-w-full object-contain ${status === "ready" ? "opacity-100" : "opacity-0"}`}
        onLoad={() => setStatus("ready")}
        onError={() => setStatus("error")}
      />
      {status === "ready" && showOverlay && overlaySubjects.length ? (
        <div className="pointer-events-none absolute inset-space-sm">
          <OverlayCanvas subjects={overlaySubjects} message={overlayMessage} />
        </div>
      ) : null}
    </div>
  );
}

function EvidenceWorkspace({
  items,
  storySheets,
  storyChoices,
  selectedStoryID,
  onSelectStory,
  selectedId,
  onSelect,
  hasBASArtifacts,
  overlaySubjects,
  overlayMessage,
}: {
  items: CaptureItem[];
  storySheets: ComponentTestArtifact[];
  storyChoices: Array<{ id: string; label: string }>;
  selectedStoryID: string;
  onSelectStory: (storyID: string) => void;
  selectedId?: string;
  onSelect: (item: CaptureItem) => void;
  hasBASArtifacts: boolean;
  overlaySubjects: Array<{
    id: string;
    label: string;
    x: number;
    y: number;
    width: number;
    height: number;
  }>;
  overlayMessage: string;
}) {
  const [showOverlay, setShowOverlay] = useState(true);
  const evidenceItems: EvidenceItem[] = items.map((item) => ({
    id: item.id,
    kind: item.id,
    label: item.label,
    status: item.available ? "available" : "missing",
    reference: item.artifact?.reference,
  }));

  return (
    <div className="space-y-space-xs">
      {storySheets.length ? (
        <section
          aria-label="Captured story sheets"
          data-testid="captured-story-sheet-gallery"
          className="space-y-space-2xs rounded-control border border-app-border bg-app-surface-muted p-space-xs"
        >
          <div>
            <h4 className="text-xs font-semibold uppercase tracking-wide">All story sheets</h4>
            <p className="mt-space-3xs text-xs text-app-muted-foreground">
              Composite BAS captures are shown together so no story group is hidden behind a
              selector.
            </p>
          </div>
          <div className="grid gap-space-xs xl:grid-cols-2">
            {storySheets.map((sheet, index) => (
              <figure
                key={`${sheet.reference}-${sheet.storyId ?? index}`}
                className="overflow-hidden rounded-control border border-app-border bg-app-surface"
              >
                <figcaption className="border-b border-app-border px-space-xs py-space-2xs text-xs font-medium text-app-foreground">
                  {storyCaptureLabel(sheet.storyId || sheet.label || `Story sheet ${index + 1}`)}
                </figcaption>
                {sheet.reference ? (
                  <ScreenshotArtifact
                    reference={browserVisibleArtifactUrl(sheet.reference)}
                    alt={`Captured story sheet: ${storyCaptureLabel(sheet.storyId || sheet.label || `Story sheet ${index + 1}`)}`}
                    showOverlay={false}
                    overlaySubjects={[]}
                    overlayMessage=""
                  />
                ) : (
                  <div className="p-space-sm text-xs text-app-muted-foreground">
                    Capture reference unavailable.
                  </div>
                )}
              </figure>
            ))}
          </div>
        </section>
      ) : null}
      {storyChoices.length ? (
        <label className="flex flex-wrap items-center gap-space-2xs text-xs text-app-muted-foreground">
          <span className="font-medium text-app-foreground">Captured story</span>
          <select
            aria-label="Captured story"
            className="min-w-0 rounded-control border border-app-border bg-app-surface px-space-xs py-space-2xs text-app-foreground"
            value={selectedStoryID}
            onChange={(event) => onSelectStory(event.target.value)}
          >
            {storyChoices.map((story) => (
              <option key={story.id} value={story.id}>
                {story.label}
              </option>
            ))}
          </select>
        </label>
      ) : null}
      {!hasBASArtifacts ? (
        <div className="rounded-control border border-app-warning/30 bg-app-warning/5 p-space-xs text-xs text-app-muted-foreground">
          <span className="font-medium text-app-foreground">
            No BAS capture bundle in this report.
          </span>{" "}
          This report may predate browser evidence publishing. Run the component tests again to
          collect inspectable captures.
        </div>
      ) : null}
      <EvidenceCarousel
        items={evidenceItems}
        selectedId={selectedId}
        onSelect={(item: EvidenceItem) => {
          const selected = items.find((candidate) => candidate.id === item.id);
          if (selected) onSelect(selected);
        }}
        renderControls={(item: EvidenceItem) =>
          item.id === "screenshot" && item.status === "available" ? (
            <label className="flex items-center gap-space-2xs text-xs text-app-muted-foreground">
              <input
                type="checkbox"
                checked={showOverlay}
                onChange={(event) => setShowOverlay(event.target.checked)}
              />
              Show measured overlay
            </label>
          ) : null
        }
        renderContent={(item: EvidenceItem) => {
          const capture = items.find((candidate) => candidate.id === item.id);
          if (
            !capture?.available ||
            !capture.artifact?.reference ||
            (!capture.artifact.reference.startsWith("http") &&
              !capture.artifact.reference.startsWith("/embedded/"))
          ) {
            return (
              <div className="flex min-h-[18rem] items-center justify-center p-space-md">
                <div className="max-w-md text-center">
                  <p className="text-sm font-medium">{item.label ?? item.kind} is not captured</p>
                  <p className="mt-space-3xs text-xs text-app-muted-foreground">
                    This evidence type was not published by the selected test report. Run the
                    current contract to collect it.
                  </p>
                </div>
              </div>
            );
          }
          const reference = browserVisibleArtifactUrl(capture.artifact.reference);
          if (item.id === "screenshot" || item.id === "story-sheet") {
            return (
              <ScreenshotArtifact
                reference={reference}
                showOverlay={showOverlay}
                overlaySubjects={overlaySubjects}
                overlayMessage={overlayMessage}
              />
            );
          }
          return (
            <StructuredArtifact
              reference={capture.artifact.reference}
              label={item.label ?? item.kind}
            />
          );
        }}
      />
    </div>
  );
}

function Report({ report }: { report: ComponentTestReport }) {
  const passed = report.results.filter((result) => result.verdict === "passed").length;
  const attention = report.results.length - passed;
  const hasIssues = attention > 0;
  return (
    <article
      data-testid="component-test-report"
      className="overflow-hidden rounded-panel border border-app-border bg-app-surface shadow-sm"
    >
      <header
        className={`border-b p-space-sm ${hasIssues ? "border-app-warning/30 bg-app-warning/5" : "border-app-success/30 bg-app-success/5"}`}
      >
        <div className="flex flex-wrap items-start justify-between gap-space-xs">
          <div className="flex items-start gap-space-xs">
            <span
              className={`flex h-control-md w-control-md items-center justify-center rounded-control ${hasIssues ? "bg-app-warning/10 text-app-warning" : "bg-app-success/10 text-app-success"}`}
              aria-hidden
            >
              {hasIssues ? (
                <CircleAlert className="h-icon-md w-icon-md" />
              ) : (
                <ShieldCheck className="h-icon-md w-icon-md" />
              )}
            </span>
            <div>
              <p className="text-sm font-semibold">{verdictLabel(report.verdict)}</p>
              <p className="mt-space-3xs text-xs text-app-muted-foreground">
                Declared behavior for {report.rootLibraryId || "this component"}
                {report.rootVersion ? `@${report.rootVersion}` : ""}
                {report.includeClosure ? " and its dependency closure" : ""}.
              </p>
            </div>
          </div>
          <StatusBadge tone={tone(report.verdict)}>{report.verdict}</StatusBadge>
        </div>
        <div className="mt-space-xs flex flex-wrap gap-space-2xs text-xs">
          <span className="rounded-pill bg-app-surface px-space-xs py-space-3xs text-app-muted-foreground">
            <strong className="text-app-foreground">{passed}</strong> checks passed
          </span>
          {hasIssues && (
            <span className="rounded-pill bg-app-surface px-space-xs py-space-3xs text-app-muted-foreground">
              <strong className="text-app-foreground">{attention}</strong> need attention
            </span>
          )}
          <span className="rounded-pill bg-app-surface px-space-xs py-space-3xs font-mono text-app-muted-foreground">
            {report.id}
          </span>
        </div>
      </header>
      <div className="space-y-space-sm p-space-sm">
        <section
          data-testid="tests-category-views"
          aria-label="Test categories"
          className="grid gap-space-xs md:grid-cols-2"
        >
          {(["integrity", "behavior", "experience", "cost"] as const).map((category) => {
            const results = report.results.filter((result) => {
              const stage = result.stage.toLowerCase();
              if (category === "integrity")
                return (
                  stage.includes("closure") ||
                  stage.includes("source") ||
                  stage.includes("contract")
                );
              if (category === "behavior")
                return stage.includes("declared") || stage.includes("behavior");
              if (category === "experience")
                return stage.includes("experience") || stage.includes("claim");
              return (
                stage.includes("performance") || stage.includes("cost") || stage.includes("console")
              );
            });
            const passedCount = results.filter((result) => result.verdict === "passed").length;
            const attentionCount = results.length - passedCount;
            return (
              <section
                key={category}
                data-testid={`tests-category-${category}`}
                className="rounded-control border border-app-border bg-app-background p-space-xs"
              >
                <div className="flex items-center justify-between gap-space-2xs">
                  <h4 className="text-sm font-semibold capitalize">{category}</h4>
                  <StatusBadge
                    tone={
                      attentionCount === 0
                        ? "success"
                        : results.some((result) => result.verdict === "failed")
                          ? "danger"
                          : "warning"
                    }
                  >
                    {results.length ? (attentionCount ? "Needs review" : "Passed") : "Unmeasured"}
                  </StatusBadge>
                </div>
                {results.length ? (
                  <p className="mt-space-2xs text-xs text-app-muted-foreground">
                    <strong className="text-app-foreground">{passedCount}</strong> passed
                    {attentionCount ? (
                      <>
                        , <strong className="text-app-foreground">{attentionCount}</strong> need
                        attention
                      </>
                    ) : null}
                    .
                  </p>
                ) : (
                  <p className="mt-space-2xs text-xs text-app-muted-foreground">
                    No capture is available for this category.
                  </p>
                )}
              </section>
            );
          })}
        </section>
        <section aria-labelledby="test-results-heading">
          <details>
            <summary id="test-results-heading" className="cursor-pointer text-sm font-semibold">
              Show all results ({report.results.length})
            </summary>
            <ul className="mt-space-2xs space-y-space-2xs">
              {[...report.results]
                .sort((a, b) => Number(a.verdict === "passed") - Number(b.verdict === "passed"))
                .map((result, index) => (
                  <StageRow
                    key={`${result.stage}-${result.assetLibraryId}-${index}`}
                    result={result}
                  />
                ))}
            </ul>
          </details>
        </section>
      </div>
    </article>
  );
}

function TestHistorySkeleton() {
  return (
    <div
      data-testid="component-test-history-skeleton"
      role="status"
      aria-live="polite"
      aria-label="Loading test history"
      className="animate-pulse rounded-panel border border-app-border bg-app-surface p-space-sm"
    >
      <div className="flex items-start justify-between gap-space-sm">
        <div className="space-y-space-2xs">
          <span className="block h-icon-sm w-field-compact rounded-pill bg-app-surface-muted" />
          <span className="block h-icon-xs w-panel-compact max-w-full rounded-pill bg-app-surface-muted" />
        </div>
        <span className="block h-icon-lg w-avatar-sm rounded-pill bg-app-surface-muted" />
      </div>
      <div className="mt-space-md space-y-space-2xs">
        <span className="block h-icon-xs w-avatar-md rounded-pill bg-app-surface-muted" />
        <span className="block h-control-2xl rounded-control bg-app-surface-muted" />
        <span className="block h-control-2xl rounded-control bg-app-surface-muted" />
      </div>
      <span className="sr-only">Loading test history…</span>
    </div>
  );
}

export type ComponentTestPanelViewProps = {
  version: string;
  includeClosure: boolean;
  setIncludeClosure: (value: boolean) => void;
  failedClaims: ComponentExperience["claims"];
  setSelectedClaimID: (value: string) => void;
  setSelectedCaptureKind: (value: string) => void;
  latest?: ComponentTestReport;
  reports: { data?: ComponentTestReport[]; isLoading: boolean; isError: boolean };
  selected: { isError: boolean };
  run: {
    isError: boolean;
    error: unknown;
    isPending: boolean;
    mutate: () => void;
  };
  activeStoryID: string;
  storySheets: ComponentTestArtifact[];
  storyChoices: Array<{ id: string; label: string }>;
  selectStory: (storyID: string) => void;
  activeClaimID: string;
  activeClaim?: ComponentExperience["claims"][number];
  activeEvidence?: ComponentExperience["evidence"][number];
  measurement?: NonNullable<ComponentExperience["evidence"][number]["measurement"]>;
  overlaySubjects: Array<{
    id: string;
    label: string;
    x: number;
    y: number;
    width: number;
    height: number;
  }>;
  captureItems: CaptureItem[];
  hasBASArtifacts: boolean;
  selectedCapture?: CaptureItem;
};

export function ComponentTestPanelView({
  version,
  includeClosure,
  setIncludeClosure,
  failedClaims,
  setSelectedClaimID,
  setSelectedCaptureKind,
  latest,
  reports,
  selected,
  run,
  activeStoryID,
  storySheets,
  storyChoices,
  selectStory,
  activeClaimID,
  activeClaim,
  activeEvidence,
  measurement,
  overlaySubjects,
  captureItems,
  hasBASArtifacts,
  selectedCapture,
}: ComponentTestPanelViewProps) {
  return (
    <section
      data-testid="component-test-panel"
      className="space-y-space-sm text-sm text-app-foreground"
      aria-label="Component tests"
    >
      <header className="flex flex-wrap items-start justify-between gap-space-xs rounded-panel border border-app-border bg-app-surface-muted p-space-sm">
        <div>
          <p className="text-xs font-semibold uppercase tracking-wide text-app-primary">
            Quality evidence
          </p>
          <h3 className="mt-space-3xs text-lg font-semibold">Component tests</h3>
          <p className="mt-space-3xs max-w-xl text-xs text-app-muted-foreground">
            Validate declared behavior for version {version || "unselected"}, including the pinned
            dependency closure. Each run is retained as reviewable evidence.
          </p>
        </div>
        <label className="flex items-center gap-space-2xs text-xs text-app-muted-foreground">
          <input
            type="checkbox"
            checked={includeClosure}
            onChange={(event) => setIncludeClosure(event.target.checked)}
          />
          Include dependency closure
        </label>
        <Button size="sm" onClick={() => run.mutate()} disabled={run.isPending || !version}>
          <Play aria-hidden className="h-icon-sm w-icon-sm" />
          {run.isPending ? "Running checks…" : "Run component tests"}
        </Button>
      </header>
      {run.isError && (
        <p
          role="alert"
          className="rounded-control border border-app-danger/30 bg-app-danger/10 p-space-xs text-xs text-app-danger"
        >
          {run.error instanceof Error
            ? run.error.message
            : "The component test could not be started."}
        </p>
      )}
      {selected.isError && (
        <p
          role="alert"
          className="rounded-control border border-app-warning/30 bg-app-warning/10 p-space-xs text-xs text-app-warning"
        >
          The requested report is unavailable. Choose a report from history or run the current
          contract.
        </p>
      )}
      {reports.isError && (
        <p
          role="alert"
          className="rounded-control border border-app-danger/30 bg-app-danger/10 p-space-xs text-xs text-app-danger"
        >
          Test history could not be loaded. Retry this page; any new test result remains available
          here.
        </p>
      )}
      {reports.isLoading ? (
        <TestHistorySkeleton />
      ) : latest ? null : reports.isError ? null : (
        <EmptyState
          className="border border-dashed border-app-border bg-app-surface-muted p-space-md text-xs"
          title="No component test evidence yet"
          description="Run the declared contract to create a durable, shareable result."
        />
      )}
      <section
        data-testid="claim-overlay-panel"
        aria-label="Claim overlay and evidence"
        className="overflow-hidden rounded-panel border border-app-border bg-app-surface"
      >
        <header className="flex flex-wrap items-start justify-between gap-space-xs border-b border-app-border bg-app-surface-muted p-space-sm">
          <div className="flex items-start gap-space-xs">
            <span className="flex h-control-md w-control-md shrink-0 items-center justify-center rounded-control bg-app-primary/10 text-app-primary">
              <ScanSearch className="h-icon-md w-icon-md" aria-hidden />
            </span>
            <div>
              <h3 className="text-sm font-semibold">Experience evidence</h3>
              <p className="mt-space-3xs max-w-2xl text-xs text-app-muted-foreground">
                Review the observation, inspect its measured subjects, and open the exact BAS
                capture that supports the result.
              </p>
            </div>
          </div>
          <StatusBadge tone={failedClaims.length ? "danger" : "success"}>
            {failedClaims.length
              ? `${failedClaims.length} claim${failedClaims.length === 1 ? "" : "s"} need attention`
              : "All claims passed"}
          </StatusBadge>
        </header>
        {failedClaims.length ? (
          <div className="grid gap-0 lg:grid-cols-[minmax(12rem,0.3fr)_minmax(0,1fr)]">
            <nav
              aria-label="Claims needing attention"
              className="border-b border-app-border bg-app-surface-muted p-space-xs lg:border-b-0 lg:border-r"
            >
              <p className="px-space-2xs pb-space-2xs text-xs font-semibold uppercase tracking-wide text-app-muted-foreground">
                Claims needing attention
              </p>
              <div role="tablist" aria-label="Failed claims" className="space-y-space-2xs">
                {failedClaims.map((claim) => {
                  const selected = claim.id === activeClaimID;
                  return (
                    <button
                      key={claim.id}
                      type="button"
                      role="tab"
                      aria-selected={selected}
                      onClick={() => setSelectedClaimID(claim.id)}
                      className={`flex w-full items-start gap-space-2xs rounded-control border p-space-xs text-left text-xs transition focus:outline-none focus:ring-2 focus:ring-app-primary ${selected ? "border-app-primary bg-app-primary/10" : "border-transparent hover:border-app-border hover:bg-app-surface"}`}
                    >
                      <CircleAlert
                        className="mt-0.5 h-icon-sm w-icon-sm shrink-0 text-app-danger"
                        aria-hidden
                      />
                      <span className="min-w-0">
                        <span className="block font-medium">{claim.id}</span>
                        <span className="mt-space-3xs block text-app-muted-foreground">
                          {claim.statement}
                        </span>
                      </span>
                    </button>
                  );
                })}
              </div>
            </nav>
            <div className="space-y-space-sm p-space-sm">
              {activeClaim && activeEvidence && measurement ? (
                <>
                  <div>
                    <p className="text-sm font-medium">{activeClaim.statement}</p>
                    <p className="mt-space-3xs text-xs text-app-muted-foreground">
                      {activeEvidence.exampleName || activeEvidence.stateId || "Selected scenario"}
                      {activeEvidence.viewport ? ` · ${activeEvidence.viewport}` : ""}
                    </p>
                  </div>
                  <dl className="grid gap-space-2xs sm:grid-cols-3">
                    {[
                      ["Observed", measurement.observed ?? "—"],
                      ["Required", measurement.required ?? "—"],
                      ["Unit", measurement.unit || "—"],
                    ].map(([label, value]) => (
                      <div
                        key={label}
                        className="rounded-control border border-app-border bg-app-surface-muted p-space-xs"
                      >
                        <dt className="text-xs text-app-muted-foreground">{label}</dt>
                        <dd className="mt-space-3xs text-sm font-semibold">{value}</dd>
                      </div>
                    ))}
                  </dl>
                  <OverlayCanvas
                    subjects={overlaySubjects}
                    message={`${measurement.metric || activeClaim.id} measured overlay`}
                  />
                </>
              ) : (
                <div
                  data-testid="claim-overlay-unmeasured"
                  className="rounded-control border border-app-warning/30 bg-app-warning/5 p-space-sm"
                >
                  <p className="text-sm font-medium">Measurement unavailable</p>
                  <p className="mt-space-3xs text-xs text-app-muted-foreground">
                    This claim failed, but the run did not produce subject bounds for an overlay.
                  </p>
                </div>
              )}
              <section
                aria-labelledby="capture-inspector-heading"
                className="border-t border-app-border pt-space-sm"
              >
                <EvidenceWorkspace
                  items={captureItems}
                  storySheets={storySheets}
                  storyChoices={storyChoices}
                  selectedStoryID={activeStoryID}
                  onSelectStory={selectStory}
                  selectedId={selectedCapture?.id}
                  onSelect={(item) => setSelectedCaptureKind(item.id)}
                  hasBASArtifacts={hasBASArtifacts}
                  overlaySubjects={overlaySubjects}
                  overlayMessage={`${measurement?.metric || activeClaim?.id || "Claim"} measured overlay`}
                />
              </section>
            </div>
          </div>
        ) : (
          <div data-testid="claim-overlay-empty" className="space-y-space-sm p-space-sm">
            <div className="flex items-start gap-space-xs rounded-control border border-app-success/30 bg-app-success/5 p-space-sm">
              <CheckCircle2
                className="mt-0.5 h-icon-md w-icon-md shrink-0 text-app-success"
                aria-hidden
              />
              <div>
                <p className="text-sm font-medium">No failed claims to inspect</p>
                <p className="mt-space-3xs text-xs text-app-muted-foreground">
                  All experience claims passed. The capture bundle is still available for review
                  below.
                </p>
              </div>
            </div>
            <section aria-labelledby="capture-inspector-heading-empty">
              <EvidenceWorkspace
                items={captureItems}
                storySheets={storySheets}
                storyChoices={storyChoices}
                selectedStoryID={activeStoryID}
                onSelectStory={selectStory}
                selectedId={selectedCapture?.id}
                onSelect={(item) => setSelectedCaptureKind(item.id)}
                hasBASArtifacts={hasBASArtifacts}
                overlaySubjects={overlaySubjects}
                overlayMessage="Measured claim overlay"
              />
            </section>
          </div>
        )}
      </section>
      {latest ? <Report report={latest} /> : null}
      {reports.data && reports.data.length > 1 && (
        <section aria-labelledby="test-history-heading">
          <h4 id="test-history-heading" className="text-sm font-semibold">
            Run history
          </h4>
          <p className="mt-space-3xs text-xs text-app-muted-foreground">
            Open a prior run without losing the current component context.
          </p>
          <ul className="mt-space-2xs space-y-space-3xs">
            {reports.data.slice(1).map((report) => (
              <li key={report.id}>
                <Link
                  to={assetSearchForTab("overview", report.id)}
                  className="flex items-center justify-between gap-space-xs rounded-control border border-app-border bg-app-surface px-space-xs py-space-2xs text-sm transition hover:bg-app-surface-muted focus:outline-none focus:ring-2 focus:ring-app-primary"
                  aria-label={`Open component test report ${report.id}`}
                >
                  <span className="flex min-w-0 items-center gap-space-2xs">
                    <StatusBadge tone={tone(report.verdict)}>
                      {verdictLabel(report.verdict)}
                    </StatusBadge>
                    <span className="truncate font-mono text-xs text-app-muted-foreground">
                      {report.id}
                    </span>
                  </span>
                  <ChevronRight
                    aria-hidden
                    className="h-icon-sm w-icon-sm shrink-0 text-app-muted-foreground"
                  />
                </Link>
              </li>
            ))}
          </ul>
        </section>
      )}
    </section>
  );
}
