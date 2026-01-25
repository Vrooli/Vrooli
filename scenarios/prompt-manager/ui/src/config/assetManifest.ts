/**
 * Asset manifest for 3D world resources.
 * Centralizes paths to all loadable assets.
 */

/** Asset categories */
export type AssetCategory = 'model' | 'texture' | 'hdri' | 'audio' | 'animation'

/** Asset metadata */
export interface AssetMeta {
  id: string
  category: AssetCategory
  path: string
  /** Human-readable name */
  name: string
  /** File size in bytes (for preload decisions) */
  size?: number
  /** Whether to preload on app start */
  preload?: boolean
  /** Dependencies (other asset IDs that must load first) */
  dependencies?: string[]
}

/**
 * Model assets (GLTF/GLB files)
 * Note: These are placeholders - actual models will be added later
 */
export const MODEL_ASSETS: Record<string, AssetMeta> = {
  // Accessories - Head
  'hat-basic': {
    id: 'hat-basic',
    category: 'model',
    path: '/assets/models/accessories/hat-basic.glb',
    name: 'Basic Hat',
  },
  'glasses-round': {
    id: 'glasses-round',
    category: 'model',
    path: '/assets/models/accessories/glasses-round.glb',
    name: 'Round Glasses',
  },
  crown: {
    id: 'crown',
    category: 'model',
    path: '/assets/models/accessories/crown.glb',
    name: 'Crown',
  },
  headphones: {
    id: 'headphones',
    category: 'model',
    path: '/assets/models/accessories/headphones.glb',
    name: 'Headphones',
  },

  // Accessories - Back
  backpack: {
    id: 'backpack',
    category: 'model',
    path: '/assets/models/accessories/backpack.glb',
    name: 'Backpack',
  },
  briefcase: {
    id: 'briefcase',
    category: 'model',
    path: '/assets/models/accessories/briefcase.glb',
    name: 'Briefcase',
  },
  folder: {
    id: 'folder',
    category: 'model',
    path: '/assets/models/accessories/folder.glb',
    name: 'Folder',
  },

  // Furniture
  'chair-office': {
    id: 'chair-office',
    category: 'model',
    path: '/assets/models/furniture/chair-office.glb',
    name: 'Office Chair',
  },
  'bench-park': {
    id: 'bench-park',
    category: 'model',
    path: '/assets/models/furniture/bench-park.glb',
    name: 'Park Bench',
  },
  'table-round': {
    id: 'table-round',
    category: 'model',
    path: '/assets/models/furniture/table-round.glb',
    name: 'Round Table',
  },

  // Decorations
  'plant-potted': {
    id: 'plant-potted',
    category: 'model',
    path: '/assets/models/decorations/plant-potted.glb',
    name: 'Potted Plant',
  },
  'lamp-floor': {
    id: 'lamp-floor',
    category: 'model',
    path: '/assets/models/decorations/lamp-floor.glb',
    name: 'Floor Lamp',
  },
}

/**
 * Texture assets
 */
export const TEXTURE_ASSETS: Record<string, AssetMeta> = {
  'noise-perlin': {
    id: 'noise-perlin',
    category: 'texture',
    path: '/assets/textures/noise-perlin.png',
    name: 'Perlin Noise',
    preload: true,
  },
  'matcap-gold': {
    id: 'matcap-gold',
    category: 'texture',
    path: '/assets/textures/matcap-gold.png',
    name: 'Gold Matcap',
  },
  'matcap-silver': {
    id: 'matcap-silver',
    category: 'texture',
    path: '/assets/textures/matcap-silver.png',
    name: 'Silver Matcap',
  },
}

/**
 * HDRI environment maps
 * Note: drei's Environment component has built-in presets, so these are for custom environments
 */
export const HDRI_ASSETS: Record<string, AssetMeta> = {
  'studio-soft': {
    id: 'studio-soft',
    category: 'hdri',
    path: '/assets/hdri/studio-soft.hdr',
    name: 'Soft Studio',
    size: 2000000, // ~2MB
  },
  'outdoor-sunny': {
    id: 'outdoor-sunny',
    category: 'hdri',
    path: '/assets/hdri/outdoor-sunny.hdr',
    name: 'Sunny Outdoor',
    size: 3000000, // ~3MB
  },
}

/**
 * Complete asset manifest
 */
export const ASSET_MANIFEST = {
  models: MODEL_ASSETS,
  textures: TEXTURE_ASSETS,
  hdri: HDRI_ASSETS,
}

/**
 * Get asset path by ID
 */
export function getAssetPath(id: string): string | undefined {
  const allAssets = {
    ...MODEL_ASSETS,
    ...TEXTURE_ASSETS,
    ...HDRI_ASSETS,
  }
  return allAssets[id]?.path
}

/**
 * Get assets to preload
 */
export function getPreloadAssets(): AssetMeta[] {
  const allAssets = [
    ...Object.values(MODEL_ASSETS),
    ...Object.values(TEXTURE_ASSETS),
    ...Object.values(HDRI_ASSETS),
  ]
  return allAssets.filter((asset) => asset.preload)
}

/**
 * Get assets by category
 */
export function getAssetsByCategory(category: AssetCategory): AssetMeta[] {
  const allAssets = [
    ...Object.values(MODEL_ASSETS),
    ...Object.values(TEXTURE_ASSETS),
    ...Object.values(HDRI_ASSETS),
  ]
  return allAssets.filter((asset) => asset.category === category)
}

/**
 * Calculate total size of assets for preloading
 */
export function calculateTotalPreloadSize(): number {
  return getPreloadAssets().reduce((total, asset) => total + (asset.size ?? 0), 0)
}
