/**
 * @libraryId react-component-library:VersionRow
 * @displayName VersionRow
 * @description A compact version-history row composed from published data-display assets.
 * @version 1.0.4
 * @tags ["component","data-display","versions","token-bound"]
 * @warning Managed by React Component Library. Preserve this header when editing adopted copies.
 */
/** @vrooliComponentSource data-display.version-row */
import { useState } from "react";
import { AspectRatio } from "@vrooli/react-component-library/AspectRatio/1";
import { BoundedMeter } from "@vrooli/react-component-library/BoundedMeter/1";
import { CopyableText } from "@vrooli/react-component-library/CopyableText/1";
import {
  CaptureGrid,
  type CaptureCell,
} from "@vrooli/react-component-library/CaptureGrid/1";
import {
  FindingList,
  type Finding,
} from "@vrooli/react-component-library/FindingList/1";
import { ProgressiveImage } from "@vrooli/react-component-library/ProgressiveImage/1";
import { RelativeTime } from "@vrooli/react-component-library/RelativeTime/1";
import { StatusBadge } from "@vrooli/react-component-library/StatusBadge/1";
import { StyleSheet } from "@vrooli/react-component-library/StyleSheet/1";
import { TrendSpark } from "@vrooli/react-component-library/TrendSpark/1";

export interface VersionRowProps {
  version: string;
  sha: string;
  status: string;
  createdAt?: string;
  releasedAt?: string;
  sourcePath?: string;
  requiredTokens?: string[];
  lifecycleState?: string;
  gatePassCount?: number;
  gateFailCount?: number;
  testRuns?: number;
  testPassRate?: number;
  fileCount?: number;
  linesOfCode?: number;
  dependencyCount?: number;
  adoptionCurrent?: number;
  adoptionPeak?: number;
  trend?: number[];
  captures?: CaptureCell[];
  thumbnail?: { src: string; alt: string };
  findings?: Finding[];
  adopters?: VersionAdopter[];
  previousVersion?: string;
  diffSummary?: VersionDiffSummary;
  selected?: boolean;
  onSelect?: () => void;
  onExpandedChange?: (expanded: boolean) => void;
  className?: string;
}

export interface VersionAdopter {
  scenario: string;
  adoptedVersion?: string;
  forkStatus?: string;
  statusDetail?: string;
}

export interface VersionDiffSummary {
  fromVersion: string;
  additions: number;
  removals: number;
  note?: string;
}

const styles = `
[data-rcl-version-row] { display: grid; gap: var(--space-2xs, 8px); min-inline-size: 0; padding: var(--space-xs, 12px); border: var(--border-hairline, 1px) solid var(--color-border, #cbd5e1); border-radius: var(--radius-control, 0.375rem); background: var(--color-surface-raised, #ffffff); color: var(--color-foreground, #0f172a); text-align: start; }
[data-rcl-version-row][data-selected="true"] { border-color: var(--color-accent, #0891b2); box-shadow: 0 0 0 2px color-mix(in srgb, var(--color-accent, #0891b2) 18%, transparent); }
[data-rcl-version-row-main], [data-rcl-version-row-meta] { display: flex; align-items: center; gap: var(--space-xs, 12px); min-inline-size: 0; flex-wrap: wrap; }
[data-rcl-version-row-main] { justify-content: space-between; }
[data-rcl-version-row-meta], [data-rcl-version-row-details] { color: var(--color-muted-foreground, #64748b); font: var(--text-caption, 600 var(--text-caption-size) / var(--text-caption-line) var(--font-sans)); }
[data-rcl-version-row-version] { color: var(--color-foreground, #0f172a); font: var(--text-body, 400 var(--text-body-size) / var(--text-body-line) var(--font-sans)); }
[data-rcl-version-row-details] { display: grid; grid-template-columns: repeat(auto-fit, minmax(8rem, 1fr)); gap: var(--space-2xs, 8px) var(--space-sm, 16px); margin: 0; padding-block-start: var(--space-xs, 12px); border-block-start: var(--border-hairline, 1px) solid var(--color-border, #cbd5e1); }
[data-rcl-version-row-details] div { min-inline-size: 0; }
[data-rcl-version-row-details] dt { color: var(--color-muted-foreground, #64748b); font-size: .7rem; }
[data-rcl-version-row-details] dd { margin: 0; color: var(--color-foreground, #0f172a); overflow-wrap: anywhere; }
[data-rcl-version-row] > button { display: block; inline-size: 100%; border: 0; padding: 0; background: transparent; color: inherit; text-align: inherit; cursor: pointer; }
[data-rcl-version-row-expand] { justify-self: start; min-block-size: var(--tap-target-min, 44px); border: 0; padding: 0; background: transparent; color: var(--color-primary, #2563eb); font: var(--text-label, 500 var(--text-label-size) / var(--text-label-line) var(--font-sans)); cursor: pointer; }
[data-rcl-version-row-expanded] { display: grid; gap: var(--space-sm, 16px); min-inline-size: 0; padding-block-start: var(--space-sm, 16px); border-block-start: var(--border-hairline, 1px) solid var(--color-border, #cbd5e1); }
[data-rcl-version-row-expanded-grid] { display: grid; grid-template-columns: repeat(auto-fit, minmax(15rem, 1fr)); gap: var(--space-sm, 16px); min-inline-size: 0; }
[data-rcl-version-row-panel] { display: grid; gap: var(--space-xs, 12px); min-inline-size: 0; padding: var(--space-xs, 12px); border: var(--border-hairline, 1px) solid var(--color-border, #cbd5e1); border-radius: var(--radius-control, 0.375rem); background: var(--color-surface-muted, #f1f5f9); }
[data-rcl-version-row-panel] h4 { margin: 0; color: var(--color-foreground, #0f172a); font: var(--text-label, 500 var(--text-label-size) / var(--text-label-line) var(--font-sans)); }
[data-rcl-version-row-thumbnail] { max-inline-size: 12rem; }
[data-rcl-version-row-adopters] { display: grid; gap: var(--space-2xs, 8px); margin: 0; padding: 0; list-style: none; }
[data-rcl-version-row-adopter] { display: flex; align-items: baseline; justify-content: space-between; gap: var(--space-xs, 12px); min-inline-size: 0; color: var(--color-foreground, #0f172a); }
[data-rcl-version-row-adopter] span:first-child { overflow-wrap: anywhere; }
`;

export function VersionRow({
  version,
  sha,
  status,
  createdAt,
  releasedAt,
  sourcePath,
  requiredTokens = [],
  lifecycleState,
  gatePassCount = 0,
  gateFailCount = 0,
  testRuns = 0,
  testPassRate = 0,
  fileCount = 0,
  linesOfCode = 0,
  dependencyCount = 0,
  adoptionCurrent = 0,
  adoptionPeak = 0,
  trend = [adoptionCurrent, adoptionPeak],
  captures = [],
  thumbnail,
  findings = [],
  adopters = [],
  previousVersion,
  diffSummary,
  selected = false,
  onSelect,
  onExpandedChange,
  className,
}: VersionRowProps) {
  const [expanded, setExpanded] = useState(false);
  const health =
    testRuns > 0 ? `${Math.round(testPassRate * 100)}% pass` : "unknown";
  const main = (
    <div {...{ "data-rcl-version-row-main": "" }}>
      <strong {...{ "data-rcl-version-row-version": "" }}>
        v{version || "(no @version)"}
      </strong>
      <StatusBadge
        tone={
          (lifecycleState || status) === "released"
            ? "success"
            : (lifecycleState || status) === "deprecated"
              ? "warning"
              : "neutral"
        }
      >
        {lifecycleState || status || "unknown"}
      </StatusBadge>
    </div>
  );
  return (
    <div data-testid="data-display-version-row" className={className}>
      <StyleSheet name="version-row" css={styles} />
      <div
        {...{ "data-rcl-version-row": "" }}
        data-selected={selected ? "true" : undefined}
      >
        {onSelect ? (
          <button
            type="button"
            aria-label={`View version ${version}`}
            onClick={onSelect}
          >
            {main}
          </button>
        ) : (
          main
        )}
        <div {...{ "data-rcl-version-row-meta": "" }}>
          <CopyableText value={sha.slice(0, 12)} />
          {createdAt ? <RelativeTime value={createdAt} /> : null}
          <span>
            {adoptionCurrent} current · {adoptionPeak} peak
          </span>
          <TrendSpark
            values={trend}
            label={`${version} adoption trend`}
            tone="success"
          />
        </div>
        <BoundedMeter
          label="Health"
          value={testRuns > 0 ? testPassRate * 100 : 0}
          max={100}
          valueText={health}
          status={
            testRuns > 0 ? `${testRuns} test runs` : "No test runs recorded"
          }
          tone={
            testRuns === 0
              ? "neutral"
              : testPassRate >= 0.9
                ? "success"
                : "warning"
          }
          testId={`version-health-meter-${version}`}
        />
        <dl {...{ "data-rcl-version-row-details": "" }}>
          <div>
            <dt>Lifecycle</dt>
            <dd>{lifecycleState || status || "unknown"}</dd>
          </div>
          <div>
            <dt>Health</dt>
            <dd data-testid={`version-health-${version}`}>{health}</dd>
          </div>
          <div>
            <dt>Gates</dt>
            <dd>
              {gatePassCount} pass · {gateFailCount} fail
            </dd>
          </div>
          <div>
            <dt>Tests</dt>
            <dd>{testRuns} runs</dd>
          </div>
          <div>
            <dt>Adoption</dt>
            <dd>
              {adoptionCurrent} current · {adoptionPeak} peak
            </dd>
          </div>
          <div>
            <dt>Footprint</dt>
            <dd>
              {fileCount} files · {linesOfCode} LOC · {dependencyCount} deps
            </dd>
          </div>
          <div>
            <dt>Released</dt>
            <dd>
              {releasedAt ? (
                <RelativeTime value={releasedAt} />
              ) : (
                "not released"
              )}
            </dd>
          </div>
          <div>
            <dt>Source</dt>
            <dd>{sourcePath || "unknown"}</dd>
          </div>
        </dl>
        {requiredTokens.length > 0 && (
          <div data-testid={`version-required-tokens-${version}`}>
            <StatusBadge tone="warning">
              Requires {requiredTokens.join(", ")}
            </StatusBadge>
          </div>
        )}
        <button
          type="button"
          {...{ "data-rcl-version-row-expand": "" }}
          aria-expanded={expanded}
          aria-controls={`version-row-expanded-${version}`}
          onClick={() =>
            setExpanded((value) => {
              const next = !value;
              onExpandedChange?.(next);
              return next;
            })
          }
        >
          {expanded ? "Hide version details" : "Show version details"}
        </button>
        {expanded && (
          <div
            id={`version-row-expanded-${version}`}
            {...{ "data-rcl-version-row-expanded": "" }}
            data-testid={`version-expanded-${version}`}
          >
            <div {...{ "data-rcl-version-row-expanded-grid": "" }}>
              <section
                {...{ "data-rcl-version-row-panel": "" }}
                aria-label={`${version} captured states`}
              >
                <h4>Captured states</h4>
                {thumbnail ? (
                  <div {...{ "data-rcl-version-row-thumbnail": "" }}>
                    <AspectRatio ratio="4 / 3">
                      <ProgressiveImage
                        src={thumbnail.src}
                        alt={thumbnail.alt}
                        ratio="1"
                        loading="lazy"
                      />
                    </AspectRatio>
                  </div>
                ) : null}
                <CaptureGrid
                  cells={
                    captures.length
                      ? captures
                      : [
                          {
                            id: `${version}-missing`,
                            viewport: "All states",
                            theme: "light",
                            status: "missing",
                          },
                        ]
                  }
                />
              </section>
              <section
                {...{ "data-rcl-version-row-panel": "" }}
                aria-label={`${version} failing checks`}
              >
                <h4>Failing checks</h4>
                <FindingList findings={findings} />
              </section>
              <section
                {...{ "data-rcl-version-row-panel": "" }}
                aria-label={`${version} adopters`}
              >
                <h4>Adopted by</h4>
                {adopters.length ? (
                  <ul {...{ "data-rcl-version-row-adopters": "" }}>
                    {adopters.map((adopter) => (
                      <li
                        key={`${adopter.scenario}-${adopter.adoptedVersion ?? version}`}
                        {...{ "data-rcl-version-row-adopter": "" }}
                      >
                        <span>{adopter.scenario}</span>
                        <StatusBadge
                          tone={adopter.forkStatus ? "warning" : "success"}
                        >
                          {adopter.forkStatus || "clean"}
                        </StatusBadge>
                      </li>
                    ))}
                  </ul>
                ) : (
                  <span>No recorded adopters.</span>
                )}
              </section>
              <section
                {...{ "data-rcl-version-row-panel": "" }}
                aria-label={`${version} change summary`}
              >
                <h4>
                  Changed from {previousVersion || "the previous version"}
                </h4>
                {diffSummary ? (
                  <div data-testid={`version-diff-summary-${version}`}>
                    <span>
                      +{diffSummary.additions} additions · −
                      {diffSummary.removals} removals
                    </span>
                    {diffSummary.note ? <p>{diffSummary.note}</p> : null}
                  </div>
                ) : (
                  <span data-testid={`version-diff-summary-${version}`}>
                    No comparison has been run.
                  </span>
                )}
              </section>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
