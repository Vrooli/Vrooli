// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
import type { ActivityView } from "../../../shared/controllers/activityController";

export type ActivityFeedProps = {
  items: ActivityView[];
};

const toneClass = (tone: ActivityView["tone"]) => {
  if (tone === "good") return "ko-pill-good";
  if (tone === "poor") return "ko-warning-pill-error";
  return "ko-warning-pill-warning";
};

export function ActivityFeed({ items }: ActivityFeedProps) {
  if (!items.length) {
    return (
      <div className="ko-panel p-6 text-center">
        <p className="ko-muted">No recent activity yet.</p>
        <p className="ko-text-sm ko-subtle mt-1">Run a search or start a healing job to populate the feed.</p>
      </div>
    );
  }

  return (
    <div className="ko-stack-xs">
      {items.map((item) => (
        <div key={item.id} className="ko-card p-4">
          <div className="flex items-start justify-between gap-3">
            <div>
              <p className="ko-text-sm font-semibold">{item.title}</p>
              {item.description && <p className="ko-text-xs ko-subtle mt-1">{item.description}</p>}
              {item.metaLabel && <p className="ko-text-xs ko-subtle mt-1">{item.metaLabel}</p>}
            </div>
            <span className={`ko-pill ${toneClass(item.tone)}`}>{item.statusLabel}</span>
          </div>
          <p className="ko-text-xs ko-muted mt-2">{item.timestampLabel}</p>
        </div>
      ))}
    </div>
  );
}
