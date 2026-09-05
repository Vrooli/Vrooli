/**
 * Screen-space label collision: hide lower-priority labels whose projected
 * rectangles overlap a higher-priority one (the map-label rule). Pure and
 * renderer-free so it is tested in Node.
 */

export interface LabelRect {
  id: string
  /** Screen-space centre and size in pixels. */
  x: number
  y: number
  width: number
  height: number
  priority: number
  /** Depth ordering hint: nearer labels win ties. */
  distance: number
}

export interface CollisionOptions {
  paddingPx: number
  /** Maximum labels to show; the rest are hidden by priority. */
  budget: number
  /** Ids that are always shown (the focused actor). */
  pinned?: ReadonlySet<string>
}

function overlaps(a: LabelRect, b: LabelRect, padding: number): boolean {
  return (
    Math.abs(a.x - b.x) * 2 < a.width + b.width + padding * 2 &&
    Math.abs(a.y - b.y) * 2 < a.height + b.height + padding * 2
  )
}

/** Returns the ids that stay visible. Deterministic for a given input order. */
export function resolveCollisions(rects: readonly LabelRect[], options: CollisionOptions): Set<string> {
  const pinned = options.pinned ?? new Set<string>()
  const ordered = [...rects].sort((a, b) => {
    const pa = pinned.has(a.id) ? 1 : 0
    const pb = pinned.has(b.id) ? 1 : 0
    if (pa !== pb) return pb - pa
    if (a.priority !== b.priority) return b.priority - a.priority
    if (a.distance !== b.distance) return a.distance - b.distance
    return a.id.localeCompare(b.id)
  })
  const visible: LabelRect[] = []
  const ids = new Set<string>()
  for (const rect of ordered) {
    if (visible.length >= options.budget && !pinned.has(rect.id)) break
    if (visible.some((shown) => overlaps(rect, shown, options.paddingPx))) continue
    visible.push(rect)
    ids.add(rect.id)
  }
  return ids
}
