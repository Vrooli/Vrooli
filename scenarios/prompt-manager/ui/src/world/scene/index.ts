/**
 * scene layer — R3F views of sim state. Imports engine, sim and config.
 * Nothing here holds behaviour state; it reads sim views and writes Three objects.
 */
export { Stage } from './Stage'
export { Places } from './Places'
export { Props, PropInstances } from './Props'
export { Trees } from './Trees'
export { SceneEnvironment } from './Environment'
export { Actors } from './actors'
export { WorldStoreContext, useWorldStore } from './WorldStoreContext'
export { Labels } from './labels/Labels'
export { resolveCollisions, type LabelRect } from './labels/collision'
export { clusterLabels, labelWorldSize } from './labels/clusters'
export { RoomHandles } from './editor/RoomHandles'
