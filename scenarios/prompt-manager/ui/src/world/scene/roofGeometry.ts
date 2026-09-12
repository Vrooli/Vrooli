import { BufferGeometry, Float32BufferAttribute } from 'three'

/** A shared gabled roof, authored in unit dimensions and scaled by the shelter. */
export function roofGeometry(tent = false) {
  const geometry = new BufferGeometry()
  const positions = [
    -.5,0,-.5, 0,1,-.5, 0,1,.5, -.5,0,-.5, 0,1,.5, -.5,0,.5,
    0,1,-.5, .5,0,-.5, .5,0,.5, 0,1,-.5, .5,0,.5, 0,1,.5,
    -.5,0,-.5, .5,0,-.5, 0,1,-.5,
    ...(tent ? [
      -.5,0,.5, -.19,.62,.5, -.19,0,.5,
      .19,0,.5, .19,.62,.5, .5,0,.5,
      -.19,.62,.5, 0,1,.5, .19,.62,.5,
    ] : [-.5,0,.5, 0,1,.5, .5,0,.5]),
  ]
  geometry.setAttribute('position', new Float32BufferAttribute(positions, 3))
  const uvs = new Float32Array(positions.length / 3 * 2)
  for (let vertex = 0; vertex < positions.length / 3; vertex += 1) {
    const x = positions[vertex * 3] ?? 0, z = positions[vertex * 3 + 2] ?? 0
    uvs[vertex * 2] = x + .5
    uvs[vertex * 2 + 1] = z + .5
  }
  geometry.setAttribute('uv', new Float32BufferAttribute(uvs, 2))
  geometry.userData.cameraObstacle = 'triangles'
  geometry.computeVertexNormals()
  return geometry
}
