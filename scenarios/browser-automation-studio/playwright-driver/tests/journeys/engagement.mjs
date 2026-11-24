/** True only when the selected isolated instance is the active BAS engagement. */
export function isActiveJourneyEngagement(item, scenario, slug, instance) {
  if (!item || item.Scenario !== scenario || item.Slug !== slug) return false;
  if (item.Mode === instance) return true;
  return item.Variant === instance && item.ShadowInstanceKey === `${scenario}@${instance}`;
}
