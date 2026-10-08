/**
 * scene layer — R3F views of sim state. Imports engine, sim and config.
 * Nothing here holds behaviour state; it reads sim views and writes Three objects.
 */
export { Terrain } from './Terrain'
export { Water } from './Water'
export { Places } from './Places'
export { Props, PropInstances } from './Props'
export { LampLights } from './LampLights'
export { Vegetation } from './Vegetation'
export { Weather } from './Weather'
export { SceneEnvironment } from './Environment'
export { Actors } from './actors'
export { ActorPoseProvider } from './actors/PoseBuffer'
export { WorldStoreContext, useWorldStore } from './WorldStoreContext'
export { Labels } from './labels/Labels'
export { resolveCollisions, type LabelRect } from './labels/collision'
export { clusterLabels, labelWorldSize } from './labels/clusters'
export { RoomHandles } from './editor/RoomHandles'
