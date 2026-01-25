/**
 * World component exports.
 *
 * Note: Skill selection is now handled via the sidebar in skill selection mode,
 * rather than a separate overlay component.
 */

// Core components
export { WorldCanvas } from './WorldCanvas'
export { WorldScene } from './WorldScene'
export { WorldControls } from './WorldControls'
export { CombinePanel } from './CombinePanel'

// Member system
export { MemberProvider, useMember, useMemberComponent, getAvailableMembers, registerMember } from './MemberProvider'
export { GeometricMember, MemberWithAccessories } from './members'

// Rendering pipeline
export { RenderPipeline, EnvironmentSetup, ShadowSystem, useShadowConfig } from './rendering'
export { BloomEffect, VignetteEffect, SSAOEffect } from './rendering'

// Material system
export {
  MaterialProvider,
  useMaterial,
  useMaterialMap,
  useMaterialQuality,
  useMaterialCache,
  useCachedMaterial,
  MATERIAL_PRESETS,
  PHYSICAL_PRESETS,
} from './materials'

// Overlays
export { NameTag, StatusIcon, ThinkingBubble, SpeechBubble, MemberOverlayGroup } from './overlays'

// Accessories
export {
  BackpackAccessory,
  HeadAccessory,
  HeldItemAccessory,
  useSkillBackpack,
  useAccessoryLoader,
  getBackpackDescription,
  getBackpackCapacity,
} from './accessories'
