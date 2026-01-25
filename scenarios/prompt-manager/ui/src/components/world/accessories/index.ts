/**
 * Accessories module exports
 */

export { BackpackAccessory } from './BackpackAccessory'
export { HeadAccessory } from './HeadAccessory'
export { HeldItemAccessory } from './HeldItemAccessory'

export {
  type AccessoryBaseProps,
  type HeadAccessoryProps,
  type BackAccessoryProps,
  type HeldItemProps,
  type AccessoryRenderInfo,
  getDefaultOffset,
} from './types'

export { useSkillBackpack, getBackpackDescription, getBackpackCapacity } from './hooks/useSkillBackpack'
export { useAccessoryLoader, preloadAccessories, hasAccessoryModel, getAvailableAccessoryModels } from './hooks/useAccessoryLoader'
