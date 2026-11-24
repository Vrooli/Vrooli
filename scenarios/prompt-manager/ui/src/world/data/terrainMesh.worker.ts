import { runCooperatively } from '../sim/cooperative'
import { terrainMeshSteps, type TerrainMeshInput } from '../sim/terrain/mesh'
import type { TerrainMeshReply } from './terrainMesh'

const scope = self as unknown as {
  postMessage(reply: TerrainMeshReply, transfers?: ArrayBuffer[]): void
  onmessage: ((event: MessageEvent<TerrainMeshInput>) => void) | null
}
scope.onmessage = event => {
  void runCooperatively(terrainMeshSteps(event.data)).then(mesh => {
    scope.postMessage({ kind: 'complete', mesh }, [mesh.vertices.buffer, mesh.normals.buffer, mesh.colours.buffer, mesh.indices.buffer] as ArrayBuffer[])
  }).catch((error: unknown) => scope.postMessage({ kind: 'failed', message: String(error) }))
}
