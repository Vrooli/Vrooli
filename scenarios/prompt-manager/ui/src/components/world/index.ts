/**
 * World component exports.
 *
 * Note: Skill selection is now handled via the sidebar in skill selection mode,
 * rather than a separate overlay component.
 */

export { WorldCanvas } from './WorldCanvas'
export { WorldScene } from './WorldScene'
export { WorldControls } from './WorldControls'
export { CombinePanel } from './CombinePanel'
export { AvatarProvider, useAvatar, useAvatarComponent, getAvailableAvatars, registerAvatar } from './AvatarProvider'
export { GeometricAvatar } from './avatars'
