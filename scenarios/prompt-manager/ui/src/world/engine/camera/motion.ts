import { Camera, Matrix4, Quaternion, Vector3 } from 'three'

/** Shared allocation-free visibility gate; compare against the last accepted pose. */
export class CameraMotionGate {
  readonly position = new Vector3()
  private readonly rotation = new Quaternion()
  private readonly previousPosition = new Vector3()
  private readonly previousRotation = new Quaternion()
  private readonly projection = new Matrix4()
  private initialized = false

  changed(camera: Camera, metres: number, radians: number): boolean {
    camera.getWorldPosition(this.position)
    camera.getWorldQuaternion(this.rotation)
    if (this.initialized && this.previousPosition.distanceToSquared(this.position) <= metres * metres
      && this.previousRotation.angleTo(this.rotation) <= radians && this.projection.equals(camera.projectionMatrix)) return false
    this.initialized = true
    this.previousPosition.copy(this.position)
    this.previousRotation.copy(this.rotation)
    this.projection.copy(camera.projectionMatrix)
    return true
  }
}
