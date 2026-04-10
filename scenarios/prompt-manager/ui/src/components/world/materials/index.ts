/**
 * Materials module exports
 */

export {
  MATERIAL_PRESETS,
  PHYSICAL_PRESETS,
  getPresetForQuality,
  PRESET_NAMES,
  type StandardPresetName,
  type PhysicalPresetName,
  type PresetName,
  type MaterialPreset,
  type PhysicalMaterialPreset,
  type EmissiveMaterialPreset,
} from './presets'

export {
  useMaterial,
  useMaterialMap,
  useMaterialQuality,
  useMaterialCache,
  useCachedMaterial,
} from './useMaterial'

export { MaterialProvider } from './MaterialProvider'
