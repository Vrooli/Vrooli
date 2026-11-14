import { useLayoutEffect, useState } from 'react'
import { GLTFLoader } from 'three/examples/jsm/loaders/GLTFLoader.js'
import { MeshoptDecoder } from 'three/examples/jsm/libs/meshopt_decoder.module.js'
import type { Group } from 'three'
import type { PropRecord } from './registry'
import { propUrl } from './registry'
import { preparePropParts, type PropPart } from './geometry'
import { preparedAssets, type PreparedAssetHandle } from './cache'
export type { PropPart } from './geometry'
const EMPTY_PARTS: readonly PropPart[] = []

interface DecodedProp { promise: Promise<Group>; scene?: Group; error?: Error }
// Shipped registry identities share one decoded source between preparation and rendering.
const decoded = new Map<string, DecodedProp>()
function source(record: PropRecord, retry = false): DecodedProp {
  const key = `${record.contentHash}:${propUrl(record)}`
  const existing = decoded.get(key)
  if (existing && !(retry && existing.error)) return existing
  const loader = new GLTFLoader().setMeshoptDecoder(MeshoptDecoder)
  const entry: DecodedProp = { promise: Promise.resolve().then(() => loader.loadAsync(propUrl(record))).then(gltf => {
    entry.scene = gltf.scene
    return gltf.scene
  }).catch((failure: unknown) => {
    entry.error = failure instanceof Error ? failure : new Error(String(failure))
    throw entry.error
  }) }
  decoded.set(key, entry)
  return entry
}

/** Decode and validate using the same source and prepared geometry as visible props. */
export async function preloadProp(record: PropRecord): Promise<void> {
  const scene = await source(record, true).promise
  const handle = preparedAssets.acquire(record.contentHash, () => preparePropParts(scene))
  handle.release()
}

/** Bound concurrent decode starts; cancellation stops further queue admission.
 * Already shared source loads may finish warming the source cache.
 */
export async function preloadProps(records: readonly PropRecord[], signal: AbortSignal): Promise<void> {
  signal.throwIfAborted()
  let next = 0
  let failed = false
  const consume = async () => {
    while (!failed && next < records.length) {
      signal.throwIfAborted()
      const record = records[next++]
      if (!record) continue
      try { await preloadProp(record) } catch (error) { failed = true; throw error }
    }
    signal.throwIfAborted()
  }
  await Promise.all(Array.from({ length: Math.min(4, records.length) }, consume))
}

export function usePropParts(record: PropRecord): readonly PropPart[] {
  const entry = source(record)
  if (entry.error) throw entry.error
  // Suspense waits for the exact decode promise used by precommit preparation.
  // eslint-disable-next-line @typescript-eslint/only-throw-error
  if (!entry.scene) throw entry.promise
  const scene = entry.scene
  const [current, setCurrent] = useState<{ hash: string; handle: PreparedAssetHandle } | null>(null)
  useLayoutEffect(() => {
    const handle = preparedAssets.acquire(record.contentHash, () => preparePropParts(scene))
    setCurrent({ hash: record.contentHash, handle })
    return () => handle.release()
  }, [scene, record.contentHash])
  return current?.hash === record.contentHash ? current.handle.parts : EMPTY_PARTS
}
