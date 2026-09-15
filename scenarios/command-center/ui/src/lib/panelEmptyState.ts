export function panelEmptyMessage(readingId: string): string {
  if (readingId === "app_downloads") return "No bundle app activity recorded in this window.";
  if (readingId === "credit_burn_by_app") return "No credit burn recorded by app in this window.";
  return "No observations recorded in this window.";
}
