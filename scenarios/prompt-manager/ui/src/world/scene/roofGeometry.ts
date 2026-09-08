import { BufferGeometry, Float32BufferAttribute } from 'three'

/** A shared gabled roof, authored in unit dimensions and scaled by the shelter. */
export function roofGeometry(tent = false) {
  const geometry = new BufferGeometry()
  geometry.setAttribute('position', new Float32BufferAttribute([
    -.5,0,-.5, 0,1,-.5, 0,1,.5, -.5,0,-.5, 0,1,.5, -.5,0,.5,
    0,1,-.5, .5,0,-.5, .5,0,.5, 0,1,-.5, .5,0,.5, 0,1,.5,
    -.5,0,-.5, .5,0,-.5, 0,1,-.5,
    ...(tent ? [
      -.5,0,.5, -.19,.62,.5, -.19,0,.5, -.5,0,.5, 0,1,.5, -.19,.62,.5,
      .19,0,.5, .19,.62,.5, .5,0,.5, .19,.62,.5, 0,1,.5, .5,0,.5,
      -.19,.62,.5, 0,1,.5, .19,.62,.5,
    ] : [-.5,0,.5, 0,1,.5, .5,0,.5]),
  ], 3))
  geometry.userData.cameraObstacle = 'triangles'
  geometry.computeVertexNormals()
  return geometry
}

