/**
 * useAccessoryLoader - Hook for loading GLTF accessory models.
 * Handles caching and preloading of accessory assets.
 */

import { useGLTF } from '@react-three/drei'
import { getAssetPath } from '@/config/assetManifest'

/**
 * Result from loading an accessory model
 */
interface AccessoryModelResult {
  /** The loaded GLTF data (using unknown for flexibility) */
  gltf: unknown
  /** Whether the model is still loading */
  isLoading: boolean
  /** Error if loading failed */
  error: Error | null
}

/**
 * Map of accessory types to their model asset IDs
 */
const ACCESSORY_MODEL_MAP: Record<string, string> = {
  // Head accessories
  hat: 'hat-basic',
  glasses: 'glasses-round',
  crown: 'crown',
  headphones: 'headphones',

  // Back accessories
  backpack: 'backpack',
  briefcase: 'briefcase',
  folder: 'folder',
}

/**
 * Hook to load a GLTF model for an accessory.
 *
 * Note: This is a foundation for future GLTF model loading.
 * Currently returns null since actual models aren't available yet.
 *
 * @param accessoryType - Type of accessory to load
 * @returns Model loading result
 *
 * @example
 * ```tsx
 * function HatAccessory({ type }) {
 *   const { gltf, isLoading } = useAccessoryLoader(type)
 *
 *   if (isLoading || !gltf) {
 *     return <FallbackHat /> // Use primitive geometry
 *   }
 *
 *   return <primitive object={gltf.scene.clone()} />
 * }
 * ```
 */
export function useAccessoryLoader(accessoryType: string): AccessoryModelResult {
  const assetId = ACCESSORY_MODEL_MAP[accessoryType]
  // Note: assetPath is computed for future use when actual models are available
  void (assetId ? getAssetPath(assetId) : null)

  // For now, return null since we don't have actual models
  // When models are added, uncomment the useGLTF call
  // const gltf = assetPath ? useGLTF(assetPath) : null

  return {
    gltf: null,
    isLoading: false,
    error: null,
  }
}

/**
 * Preload accessory models for faster loading
 *
 * @param accessoryTypes - Array of accessory types to preload
 */
export function preloadAccessories(accessoryTypes: string[]): void {
  const paths = accessoryTypes
    .map((type) => ACCESSORY_MODEL_MAP[type])
    .filter((id): id is string => !!id)
    .map((id) => getAssetPath(id))
    .filter((path): path is string => !!path)

  paths.forEach((path) => {
    useGLTF.preload(path)
  })
}

/**
 * Check if a model exists for an accessory type
 */
export function hasAccessoryModel(accessoryType: string): boolean {
  const assetId = ACCESSORY_MODEL_MAP[accessoryType]
  return !!assetId && !!getAssetPath(assetId)
}

/**
 * Get all available accessory types with models
 */
export function getAvailableAccessoryModels(): string[] {
  return Object.keys(ACCESSORY_MODEL_MAP).filter(hasAccessoryModel)
}
