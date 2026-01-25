/**
 * useAssetDisposal - Hook for tracking and disposing Three.js assets.
 * Automatically cleans up assets when component unmounts.
 *
 * Ensures proper memory management by tracking materials, geometries,
 * and textures and disposing them when no longer needed.
 */
// DOC: docs/concepts/3D-WORLD-ARCHITECTURE.md#memory-management

import { useRef, useEffect, useCallback, useMemo } from 'react'
import type { Material, Texture, BufferGeometry } from 'three'
import { useDisposalStore } from '@/stores/disposalStore'
import type { DisposableType, DisposalStats } from '@/types/disposal'

// Counter for generating unique IDs
let assetIdCounter = 0

function generateAssetId(type: DisposableType, owner: string): string {
  return `${owner}-${type}-${++assetIdCounter}`
}

interface UseAssetDisposalOptions {
  /** Component/owner name for tracking */
  owner: string
  /** Whether to auto-dispose on unmount */
  autoDisposeOnUnmount?: boolean
}

interface UseAssetDisposalResult {
  /** Track a material for disposal */
  trackMaterial: (material: Material, id?: string) => string
  /** Track a geometry for disposal */
  trackGeometry: (geometry: BufferGeometry, id?: string) => string
  /** Track a texture for disposal */
  trackTexture: (texture: Texture, id?: string) => string
  /** Dispose a specific tracked asset */
  dispose: (id: string) => boolean
  /** Dispose all assets tracked by this owner */
  disposeAll: () => number
  /** Check if an asset is tracked */
  isTracked: (id: string) => boolean
  /** Get all tracked IDs for this owner */
  getTrackedIds: () => string[]
}

/**
 * Hook for managing Three.js asset disposal.
 *
 * @example
 * ```tsx
 * function MyMesh() {
 *   const { trackMaterial, trackGeometry } = useAssetDisposal({
 *     owner: 'MyMesh'
 *   })
 *
 *   const material = useMemo(() => {
 *     const mat = new MeshStandardMaterial({ color: 'red' })
 *     trackMaterial(mat)
 *     return mat
 *   }, [trackMaterial])
 *
 *   // Material is automatically disposed when component unmounts
 *
 *   return (
 *     <mesh material={material}>
 *       <boxGeometry />
 *     </mesh>
 *   )
 * }
 * ```
 */
export function useAssetDisposal({
  owner,
  autoDisposeOnUnmount = true,
}: UseAssetDisposalOptions): UseAssetDisposalResult {
  // Track IDs created by this instance
  const trackedIdsRef = useRef<Set<string>>(new Set())

  // Store actions (via ref to avoid re-renders)
  const storeRef = useRef(useDisposalStore.getState())

  // Update store ref on mount
  useEffect(() => {
    storeRef.current = useDisposalStore.getState()
  }, [])

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (autoDisposeOnUnmount) {
        // Dispose all assets tracked by this instance
        for (const id of trackedIdsRef.current) {
          storeRef.current.disposeAsset(id)
        }
        trackedIdsRef.current.clear()
      }
    }
  }, [autoDisposeOnUnmount])

  const trackMaterial = useCallback(
    (material: Material, customId?: string): string => {
      const id = customId ?? generateAssetId('material', owner)
      storeRef.current.trackAsset(id, 'material', material, owner)
      trackedIdsRef.current.add(id)
      return id
    },
    [owner]
  )

  const trackGeometry = useCallback(
    (geometry: BufferGeometry, customId?: string): string => {
      const id = customId ?? generateAssetId('geometry', owner)
      storeRef.current.trackAsset(id, 'geometry', geometry, owner)
      trackedIdsRef.current.add(id)
      return id
    },
    [owner]
  )

  const trackTexture = useCallback(
    (texture: Texture, customId?: string): string => {
      const id = customId ?? generateAssetId('texture', owner)
      storeRef.current.trackAsset(id, 'texture', texture, owner)
      trackedIdsRef.current.add(id)
      return id
    },
    [owner]
  )

  const dispose = useCallback((id: string): boolean => {
    const disposed = storeRef.current.disposeAsset(id)
    if (disposed) {
      trackedIdsRef.current.delete(id)
    }
    return disposed
  }, [])

  const disposeAll = useCallback((): number => {
    let count = 0
    for (const id of trackedIdsRef.current) {
      if (storeRef.current.disposeAsset(id)) {
        count++
      }
    }
    trackedIdsRef.current.clear()
    return count
  }, [])

  const isTracked = useCallback((id: string): boolean => {
    return trackedIdsRef.current.has(id)
  }, [])

  const getTrackedIds = useCallback((): string[] => {
    return Array.from(trackedIdsRef.current)
  }, [])

  return {
    trackMaterial,
    trackGeometry,
    trackTexture,
    dispose,
    disposeAll,
    isTracked,
    getTrackedIds,
  }
}

/**
 * Hook for creating a disposable material.
 * Automatically tracks and disposes the material.
 *
 * @example
 * ```tsx
 * function MyMesh() {
 *   const material = useDisposableMaterial(
 *     () => new MeshStandardMaterial({ color: 'red' }),
 *     []
 *   )
 *
 *   return <mesh material={material}><boxGeometry /></mesh>
 * }
 * ```
 */
export function useDisposableMaterial<T extends Material>(
  factory: () => T,
  deps: React.DependencyList,
  owner = 'useDisposableMaterial'
): T {
  const { trackMaterial } = useAssetDisposal({ owner })

  return useMemo(() => {
    const material = factory()
    trackMaterial(material)
    return material
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [trackMaterial, ...deps])
}

/**
 * Hook for creating a disposable geometry.
 */
export function useDisposableGeometry<T extends BufferGeometry>(
  factory: () => T,
  deps: React.DependencyList,
  owner = 'useDisposableGeometry'
): T {
  const { trackGeometry } = useAssetDisposal({ owner })

  return useMemo(() => {
    const geometry = factory()
    trackGeometry(geometry)
    return geometry
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [trackGeometry, ...deps])
}

/**
 * Hook for creating a disposable texture.
 */
export function useDisposableTexture<T extends Texture>(
  factory: () => T,
  deps: React.DependencyList,
  owner = 'useDisposableTexture'
): T {
  const { trackTexture } = useAssetDisposal({ owner })

  return useMemo(() => {
    const texture = factory()
    trackTexture(texture)
    return texture
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [trackTexture, ...deps])
}

/**
 * Hook for getting disposal statistics.
 */
export function useDisposalStats(): DisposalStats {
  return useDisposalStore((state) => state.stats)
}

/**
 * Hook for starting/stopping periodic cleanup.
 */
export function usePeriodicCleanup(enabled = true): void {
  useEffect(() => {
    const store = useDisposalStore.getState()

    if (enabled) {
      store.startPeriodicCleanup()
    }

    return () => {
      store.stopPeriodicCleanup()
    }
  }, [enabled])
}

/**
 * Run a one-time cleanup manually.
 */
export function runCleanup(): number {
  return useDisposalStore.getState().runCleanup()
}

/**
 * Dispose all tracked assets.
 */
export function disposeAllAssets(): number {
  return useDisposalStore.getState().disposeAll()
}
