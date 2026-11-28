import { lastResearchWrite, type TopicSummary } from "../api/activity";

/** Topics whose name or title contains the filter, ignoring case. */
export function matching(list: TopicSummary[], filter: string): TopicSummary[] {
  const needle = filter.trim().toLowerCase();
  if (!needle) return list;
  return list.filter((x) => x.topic.includes(needle) || (x.title?.value ?? "").toLowerCase().includes(needle));
}

/** Newest research write first; topics never written to last, by name. */
export function sortByResearchWrite(list: TopicSummary[]): TopicSummary[] {
  const at = (x: TopicSummary) => lastResearchWrite(x)?.at?.value?.seconds ?? -1n;
  return [...list].sort((a, b) => {
    const diff = at(b) - at(a);
    return diff > 0n ? 1 : diff < 0n ? -1 : a.topic.localeCompare(b.topic);
  });
}
