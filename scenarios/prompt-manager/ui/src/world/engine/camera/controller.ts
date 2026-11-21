export type CameraOwner = 'overview' | 'intro' | 'explore' | 'focus' | 'follow' | 'editing' | 'first-person' | 'third-person'
export interface CameraOwnership {
  owner: CameraOwner
  command: number
  moving: boolean
}
export type CameraCommandCause = 'begin' | 'complete' | 'follow' | 'detach' | 'navigation' | 'generation' | 'reduced-motion' | 'target-removed' | 'edit-start' | 'edit-end'
export interface CameraCommandEvent extends CameraOwnership {
  sequence: number
  at: number
  cause: CameraCommandCause
}

/** One cancellation identity for automatic pose commands and their completion callbacks. */
export class CameraController {
  private command = 0
  private moving = false
  private owner: CameraOwner = 'overview'
  private following = false
  private cleanup: (() => void) | undefined
  private events: CameraCommandEvent[] = []
  private sequence = 0

  constructor(
    private readonly stop: () => void,
    private readonly changed: (state: CameraOwnership, history: CameraCommandEvent[]) => void = () => {},
    private readonly now: () => number = () => performance.now(),
  ) {}

  read(): CameraOwnership { return { owner: this.owner, command: this.command, moving: this.moving } }
  history(): CameraCommandEvent[] { return this.events.map(event => ({ ...event })) }
  private publish(cause: CameraCommandCause) {
    const state = this.read()
    this.events.push({ ...state, cause, sequence: ++this.sequence, at: this.now() })
    if (this.events.length > 32) this.events.shift()
    this.changed(state, this.history())
  }
  private invalidate() {
    ++this.command
    this.moving = false
    const cleanup = this.cleanup
    this.cleanup = undefined
    cleanup?.()
    this.stop()
  }

  begin(owner: 'overview' | 'intro' | 'focus'): number | null {
    if (this.owner === 'editing') return null
    this.invalidate()
    this.following = false
    this.owner = owner
    this.moving = true
    this.publish('begin')
    return this.command
  }

  /** Install listener removal before sending the command, which can finish synchronously. */
  onCancel(command: number, cleanup: () => void): void {
    if (command !== this.command || !this.moving) cleanup()
    else this.cleanup = cleanup
  }

  complete(command: number): boolean {
    if (command !== this.command || !this.moving) return false
    this.moving = false
    const cleanup = this.cleanup
    this.cleanup = undefined
    cleanup?.()
    if (this.owner === 'intro') this.owner = 'overview'
    this.publish('complete')
    return true
  }

  follow(enabled: boolean): void {
    this.following = enabled
    if (this.owner !== 'editing') this.owner = enabled ? 'follow' : this.moving ? this.owner : 'explore'
    this.publish(enabled ? 'follow' : 'detach')
  }

  walking(mode: 'first-person' | 'third-person'): void {
    if (this.owner === 'editing') return
    if (this.moving) this.invalidate()
    this.following = false
    this.owner = mode
    this.publish('navigation')
  }

  navigate(cause: 'navigation' | 'generation' | 'reduced-motion' | 'target-removed' = 'navigation'): void {
    if (this.owner === 'editing') return
    if (this.moving) this.invalidate()
    this.owner = this.following ? 'follow' : 'explore'
    this.publish(cause)
  }

  edit(enabled: boolean): void {
    this.invalidate()
    this.owner = enabled ? 'editing' : this.following ? 'follow' : 'explore'
    this.publish(enabled ? 'edit-start' : 'edit-end')
  }

  dispose(): void { this.invalidate() }
}
