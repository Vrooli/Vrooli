// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
import { useEffect, useMemo, useState } from "react";
import { buildActivityViews, type ActivityView } from "../controllers/activityController";
import { loadActivityFeed, subscribeActivityUpdates } from "../lib/activityStore";

export function useActivityFeed(limit = 8): ActivityView[] {
  const [records, setRecords] = useState(() => loadActivityFeed());

  useEffect(() => {
    const unsubscribe = subscribeActivityUpdates(() => {
      setRecords(loadActivityFeed());
    });
    return () => {
      unsubscribe();
    };
  }, []);

  return useMemo(() => buildActivityViews(records).slice(0, limit), [records, limit]);
}
