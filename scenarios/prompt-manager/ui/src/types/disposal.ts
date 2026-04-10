/**
 * Asset disposal types for memory management.
 * Provides type-safe tracking and cleanup of Three.js resources.
 */

import type { Material, Texture, BufferGeometry } from 'three'

// WeakRef type for tracking disposable assets
type DisposableObject = Material | Texture | BufferGeometry

/** Types of disposable assets */
export type DisposableType = 'material' | 'geometry' | 'texture' | 'object3d'

/** Tracked asset information */
export interface TrackedAsset {
  /** Unique identifier for the asset */
  id: string
  /** Type of asset */
  type: DisposableType
  /** Reference to the actual Three.js object (uses WeakRef internally in store) */
  ref: { deref(): DisposableObject | undefined }
  /** Component that created this asset */
  owner: string
  /** Timestamp when asset was created */
  createdAt: number
  /** Whether asset has been disposed */
  disposed: boolean
}

/** Disposal statistics */
export interface DisposalStats {
  /** Total tracked assets */
  totalTracked: number
  /** Assets by type */
  byType: Record<DisposableType, number>
  /** Assets disposed in last cleanup */
  lastCleanupCount: number
  /** Time of last cleanup */
  lastCleanupTime: number | null
  /** Total assets disposed ever */
  totalDisposed: number
}

/** Disposal manager configuration */
export interface DisposalConfig {
  /** Whether to auto-track new assets */
  autoTrack: boolean
  /** Whether to auto-cleanup on unmount */
  autoCleanup: boolean
  /** Interval for periodic cleanup (ms, 0 = disabled) */
  cleanupInterval: number
  /** Whether to log disposal operations */
  debug: boolean
}

/** Default disposal configuration */
export const DEFAULT_DISPOSAL_CONFIG: DisposalConfig = {
  autoTrack: true,
  autoCleanup: true,
  cleanupInterval: 30000, // 30 seconds
  debug: false,
}
