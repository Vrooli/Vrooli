/** Instance-owned demand-render reasons. Releasing one owner cannot stop another. */
export class AnimationLeases {
  private owners = new Set<symbol>()
  private listeners = new Set<() => void>()
  get count() { return this.owners.size }
  acquire(): () => void {
    const owner = Symbol('animation')
    this.owners.add(owner)
    this.notify()
    return () => { if (this.owners.delete(owner)) this.notify() }
  }
  subscribe(listener: () => void): () => void {
    this.listeners.add(listener)
    return () => { this.listeners.delete(listener) }
  }
  private notify() { for (const listener of this.listeners) listener() }
}
