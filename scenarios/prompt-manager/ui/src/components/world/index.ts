/**
 * World component exports.
 *
 * Note: Skill operations are handled via the sidebar.
 */

// Core components
export { WorldCanvas } from './WorldCanvas'
export { WorldScene } from './WorldScene'
export { WorldControls } from './WorldControls'
export { WorldSettingsPopup } from './WorldSettingsPopup'
export { EnvironmentControls } from './EnvironmentControls'
export { DisplayPanel } from './DisplayPanel'

// Agent system
export { AgentProvider, useAgent, useAgentComponent, getAvailableAgents, registerAgent } from './AgentProvider'
export { GeometricAgent, AgentWithAccessories } from './agents'

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
export { NameTag, StatusIcon, ThinkingBubble, SpeechBubble, AgentOverlayGroup } from './overlays'

// Accessories
export {
  BackpackAccessory,
  HeadAccessory,
  HeldItemAccessory,
  ClothingTop,
  ClothingBottom,
  FootwearAccessory,
  useAccessoryLoader,
} from './accessories'

// Interaction system
export { DraggableObject, DragPlane } from './interaction'

// Furniture system
export { FurnitureItem, FurnitureManager, useAddFurniture, useRemoveFurniture } from './furniture'

// Decorations system
export {
  DecorationItem,
  DecorationManager,
  useAddDecoration,
  useRemoveDecoration,
} from './decorations'
