import { useMemo } from "react";
import { Popover } from "./ui/popover";
import { Plus, CheckSquare, Square, AlertTriangle, ShieldCheck, FileText, Shield, Camera } from "lucide-react";
import { useTestExecutions, useTidinessScore, useTidinessIssues, useAuditorViolations, useVisualCaptures, useVisualCaptureDetail } from "../lib/hooks";
import { testFailureContextItems, codeQualityContextItems, changeSummaryContextItem, scenarioQualityContextItem, ruleViolationContextItems, rulesSummaryContextItem, screenshotContextItem } from "../lib/agentContext";
import type { AgentContextItem, RepoFileStats } from "../lib/api";

interface ContextPickerProps {
  scenarioSlug: string;
  repoId?: string | null;
  testGenieAvailable: boolean;
  tidinessAvailable: boolean;
  auditorAvailable: boolean;
  visualCaptureAvailable: boolean;
  fileStats?: RepoFileStats;
  contextItems: AgentContextItem[];
  onAddContext: (item: AgentContextItem) => void;
  onRemoveContext: (id: string) => void;
}

export function ContextPickerPopover({
  scenarioSlug,
  repoId,
  testGenieAvailable,
  tidinessAvailable,
  auditorAvailable,
  visualCaptureAvailable,
  fileStats,
  contextItems,
  onAddContext,
  onRemoveContext,
}: ContextPickerProps) {
  const testExecs = useTestExecutions(scenarioSlug, testGenieAvailable, repoId);
  const tidinessScore = useTidinessScore(scenarioSlug, tidinessAvailable, repoId);
  const tidinessIssues = useTidinessIssues(scenarioSlug, undefined, tidinessAvailable, repoId);

  const changeSummary = useMemo(() => {
    if (!fileStats) return null;
    return changeSummaryContextItem(fileStats);
  }, [fileStats]);

  const testItems = useMemo(() => {
    if (!testExecs.data?.items?.length) return [];
    const latest = testExecs.data.items[0];
    if (!latest?.phases) return [];
    return testFailureContextItems(latest.phases);
  }, [testExecs.data]);

  const qualityItem = useMemo(() => {
    if (!tidinessScore.data) return null;
    return scenarioQualityContextItem(tidinessScore.data);
  }, [tidinessScore.data]);

  const codeItems = useMemo(() => {
    if (!tidinessIssues.data?.length) return [];
    return codeQualityContextItems(tidinessIssues.data);
  }, [tidinessIssues.data]);

  const auditorViolations = useAuditorViolations(scenarioSlug, auditorAvailable, repoId);
  const violationsData = auditorViolations.data;

  const ruleViolationItems = useMemo(() => {
    if (!violationsData?.length) return [];
    return ruleViolationContextItems(violationsData);
  }, [violationsData]);

  const rulesSummary = useMemo(() => {
    if (!violationsData?.length) return null;
    return rulesSummaryContextItem(violationsData);
  }, [violationsData]);

  // Screenshots — fetch latest capture and its detail
  const captures = useVisualCaptures(scenarioSlug, visualCaptureAvailable, repoId);
  const latestCapture = captures.data?.snapshots?.[0] ?? null;
  const captureDetail = useVisualCaptureDetail(
    latestCapture?.id ?? "",
    scenarioSlug,
    visualCaptureAvailable && !!latestCapture,
    repoId,
  );

  const screenshotItems = useMemo(() => {
    if (!latestCapture || !captureDetail.data?.screenshots?.length) return [];
    return captureDetail.data.screenshots.map((file) =>
      screenshotContextItem(latestCapture, file),
    );
  }, [latestCapture, captureDetail.data]);

  const isAttached = (id: string) => contextItems.some((c) => c.id === id);

  const toggle = (item: AgentContextItem) => {
    if (isAttached(item.id)) {
      onRemoveContext(item.id);
    } else {
      onAddContext(item);
    }
  };

  const hasAnyItems = !!(changeSummary || testItems.length || qualityItem || codeItems.length || rulesSummary || ruleViolationItems.length || screenshotItems.length);

  return (
    <Popover
      direction="up"
      align="start"
      trigger={
        <span className="h-8 w-8 flex items-center justify-center rounded border border-slate-700 text-slate-400 hover:text-slate-200 hover:border-slate-600 transition-colors">
          <Plus className="h-4 w-4" />
        </span>
      }
    >
      <div className="max-h-80 overflow-y-auto p-2 w-72">
        {!hasAnyItems && (
          <p className="text-xs text-slate-500 p-2 text-center">No context available yet</p>
        )}

        {/* Changes section */}
        {changeSummary && (
          <Section icon={<FileText className="h-3 w-3" />} title="Changes">
            <CheckItem
              item={changeSummary}
              checked={isAttached(changeSummary.id)}
              onToggle={toggle}
            />
          </Section>
        )}

        {/* Test failures */}
        {testGenieAvailable && (
          <Section icon={<AlertTriangle className="h-3 w-3" />} title="Test Failures">
            {testItems.length > 0 ? (
              testItems.map((item) => (
                <CheckItem
                  key={item.id}
                  item={item}
                  checked={isAttached(item.id)}
                  onToggle={toggle}
                />
              ))
            ) : (
              <p className="text-[11px] text-slate-600 px-2 py-1">No failures</p>
            )}
          </Section>
        )}

        {/* Code quality */}
        {tidinessAvailable && (
          <Section icon={<ShieldCheck className="h-3 w-3" />} title="Code Quality">
            {qualityItem && (
              <CheckItem
                item={qualityItem}
                checked={isAttached(qualityItem.id)}
                onToggle={toggle}
              />
            )}
            {codeItems.length > 0 ? (
              codeItems.slice(0, 10).map((item) => (
                <CheckItem
                  key={item.id}
                  item={item}
                  checked={isAttached(item.id)}
                  onToggle={toggle}
                />
              ))
            ) : !qualityItem ? (
              <p className="text-[11px] text-slate-600 px-2 py-1">No data</p>
            ) : null}
            {codeItems.length > 10 && (
              <p className="text-[11px] text-slate-500 px-2 py-1">
                +{codeItems.length - 10} more issues
              </p>
            )}
          </Section>
        )}

        {/* Rules */}
        {auditorAvailable && (
          <Section icon={<Shield className="h-3 w-3" />} title="Rules">
            {rulesSummary && (
              <CheckItem
                item={rulesSummary}
                checked={isAttached(rulesSummary.id)}
                onToggle={toggle}
              />
            )}
            {ruleViolationItems.length > 0 ? (
              ruleViolationItems.slice(0, 10).map((item) => (
                <CheckItem
                  key={item.id}
                  item={item}
                  checked={isAttached(item.id)}
                  onToggle={toggle}
                />
              ))
            ) : !rulesSummary ? (
              <p className="text-[11px] text-slate-600 px-2 py-1">No data</p>
            ) : null}
            {ruleViolationItems.length > 10 && (
              <p className="text-[11px] text-slate-500 px-2 py-1">
                +{ruleViolationItems.length - 10} more violations
              </p>
            )}
          </Section>
        )}

        {/* Screenshots */}
        {visualCaptureAvailable && (
          <Section icon={<Camera className="h-3 w-3" />} title="Screenshots">
            {screenshotItems.length > 0 ? (
              screenshotItems.slice(0, 10).map((item) => (
                <CheckItem
                  key={item.id}
                  item={item}
                  checked={isAttached(item.id)}
                  onToggle={toggle}
                />
              ))
            ) : (
              <p className="text-[11px] text-slate-600 px-2 py-1">No captures yet</p>
            )}
            {screenshotItems.length > 10 && (
              <p className="text-[11px] text-slate-500 px-2 py-1">
                +{screenshotItems.length - 10} more screenshots
              </p>
            )}
          </Section>
        )}
      </div>
    </Popover>
  );
}

function Section({ icon, title, children }: { icon: React.ReactNode; title: string; children: React.ReactNode }) {
  return (
    <div className="mb-2 last:mb-0">
      <div className="flex items-center gap-1.5 px-2 py-1 text-[11px] font-medium text-slate-400 uppercase tracking-wider">
        {icon}
        {title}
      </div>
      {children}
    </div>
  );
}

function CheckItem({ item, checked, onToggle }: { item: AgentContextItem; checked: boolean; onToggle: (item: AgentContextItem) => void }) {
  return (
    <button
      type="button"
      onClick={() => onToggle(item)}
      className="w-full flex items-center gap-2 px-2 py-1.5 rounded text-left hover:bg-slate-800/60 transition-colors"
    >
      {checked ? (
        <CheckSquare className="h-3.5 w-3.5 text-blue-400 shrink-0" />
      ) : (
        <Square className="h-3.5 w-3.5 text-slate-600 shrink-0" />
      )}
      <span className="text-xs text-slate-300 truncate">{item.label}</span>
    </button>
  );
}
