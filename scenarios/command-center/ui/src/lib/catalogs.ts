// Runtime helpers for the split instance catalogs. Config files are the
// editable source; this resolver supplies a defensive preview fallback when
// a room has not yet persisted a bind map.
export function resolveSlotBindings(composition: string, signalIds: string[], authored: Record<string, string> = {}): Record<string, string> {
  const available = new Set(signalIds);
  const candidates: Record<string, string[]> = {
    running: ["active_scenarios", "composite_portfolio", "scenario_completeness"], healthy: ["scenario_health", "scenario_completeness"], total: ["total_scenarios"], throughput: ["throughput_stats", "swarm_throughput"], blocking: ["blocking_stats"], posture: ["offer_posture"], visitors: ["visitors"], conversions: ["conversions"], ctaClicks: ["cta_clicks"], arcs: ["traffic_countries"], ladder: ["release_ladder"], funnel: ["funnel_30d"],
  };
  const slots = composition === "orbital-field" ? ["running", "healthy"] : composition === "hive-lattice" ? ["total", "running", "healthy"] : composition === "flow-current" ? ["throughput", "blocking"] : composition === "ledger-river" ? ["posture"] : composition === "signal-constellation" ? ["visitors", "conversions", "ctaClicks"] : composition === "meridian-arc" ? ["arcs"] : composition === "funnel-cascade" ? ["ladder"] : composition === "conversion-funnel" ? ["funnel"] : [];
  return Object.fromEntries(slots.map((name) => [name, authored[name] && available.has(authored[name]) ? authored[name] : candidates[name]?.find((id) => available.has(id)) ?? ""]));
}

export function validateRoomBinding(room: Record<string, unknown>, catalogs: Record<string, Array<Record<string, unknown>>>): string | null {
  const compositionID = typeof room.composition === "string" ? room.composition : "";
  const composition = (catalogs.compositions ?? []).find((entry) => entry.id === compositionID);
  const signals = catalogs.signals ?? [];
  const bind = room.bind && typeof room.bind === "object" ? room.bind as Record<string, unknown> : {};
  const slots = composition?.slots && typeof composition.slots === "object" ? composition.slots as Record<string, unknown> : {};
  for (const [slot, target] of Object.entries(bind)) {
    if (typeof target !== "string") return `Binding for ${slot} must name a signal.`;
    const signal = signals.find((entry) => entry.id === target);
    if (!signal) return `Binding for ${slot} references unknown signal ${target}.`;
    const spec = slots[slot] && typeof slots[slot] === "object" ? slots[slot] as Record<string, unknown> : {};
    if (typeof spec.shape === "string" && signal.shape !== spec.shape) return `Signal ${target} has shape ${String(signal.shape)}; slot ${slot} requires ${spec.shape}.`;
    const columns = spec.columns && typeof spec.columns === "object" ? spec.columns as Record<string, unknown> : {};
    const declared = signal.columns && typeof signal.columns === "object" ? signal.columns as Record<string, unknown> : {};
    for (const [column, definition] of Object.entries(columns)) {
      if (definition && typeof definition === "object" && (definition as Record<string, unknown>).optional !== true && !declared[column]) return `Signal ${target} is missing required column ${column}.`;
    }
  }
  return null;
}
