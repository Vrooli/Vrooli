import type { WorldState } from './model'

/**
 * Deterministic digest of the parts of the state that behaviour can change.
 * Two runs from the same seed and signal script must produce the same digest.
 */
export function hashState(state: WorldState): string {
  const parts: string[] = [String(state.tick), state.time.toFixed(3), String(state.rngState), String(state.revision), terrainDigest(state)]
  for (const id of state.actorOrder) {
    const a = state.actors[id]
    if (!a) continue
    parts.push(
      `${a.id}:${a.state}:${a.position[0].toFixed(4)},${a.position[1].toFixed(4)}:${a.facing.toFixed(4)}:${a.seatId ?? ''}:${a.idle.activity}:${a.anim.hopPhase.toFixed(4)}:${a.anim.squash.toFixed(4)}`,
    )
  }
  parts.push(String(state.events.length), String(state.nextSeq))
  return fnv(parts.join('|'))
}

export function terrainDigest(state: WorldState): string {
  const field = state.terrain
  const parts = [`${field.radius}:${field.cellSize}:${field.cols}:${field.rows}`]
  for (let index = 0; index < field.height.length; index += 1) {
    parts.push(`${field.height[index]?.toFixed(5)}:${field.moisture[index]?.toFixed(5)}`)
  }
  return fnv(parts.join('|'))
}

function fnv(input: string): string {
  let h = 0x811c9dc5
  for (let i = 0; i < input.length; i += 1) {
    h ^= input.charCodeAt(i)
    h = Math.imul(h, 0x01000193)
  }
  return (h >>> 0).toString(16).padStart(8, '0')
}
