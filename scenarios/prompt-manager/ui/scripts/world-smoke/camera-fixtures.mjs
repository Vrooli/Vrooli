/** Browser-only fixtures: borrow rendered geometry, never alter production APIs. */
export async function installFixtureHook(page) {
  await page.addInitScript(() => {
    window.__cameraFixtureRoots = new Set()
    let next = 0
    window.__REACT_DEVTOOLS_GLOBAL_HOOK__ = {
      supportsFiber: true, renderers: new Map(),
      inject(renderer) { this.renderers.set(++next, renderer); return next },
      onCommitFiberRoot(_id, root) { window.__cameraFixtureRoots.add(root) },
      onCommitFiberUnmount() {},
    }
  })
}

export async function insertFixture(page, center, size) {
  return page.evaluate(({ center, size }) => {
    const pending = [...window.__cameraFixtureRoots].map(root => root.current)
    let store
    let visited = 0
    while (pending.length && visited++ < 30000 && !store) {
      const fiber = pending.pop()
      for (const value of [fiber.memoizedProps?.store, fiber.memoizedProps?.value]) {
        if (typeof value?.getState === 'function' && value.getState().scene?.isScene) store = value
      }
      if (fiber.child) pending.push(fiber.child)
      if (fiber.sibling) pending.push(fiber.sibling)
    }
    if (!store) throw new Error('Rendered R3F store not found')
    const { scene, camera } = store.getState()
    let source
    scene.traverse(object => {
      if (!source && object.isInstancedMesh && object.geometry.userData.cameraObstacle === 'box') source = object
    })
    if (!source) throw new Error('Rendered obstacle geometry not found')
    const fixture = new source.constructor(source.geometry, source.material, 1)
    fixture.name = 'browser-camera-obstruction-fixture'
    fixture.frustumCulled = false
    fixture.setMatrixAt(0, source.matrixWorld.clone().identity().makeScale(...size).setPosition(...center))
    scene.add(fixture)
    const halfHeight = camera.near * Math.tan(camera.fov * Math.PI / 360) / camera.zoom
    const radius = Math.hypot(camera.near, halfHeight, halfHeight * camera.aspect)
    const frames = []
    const state = { store, fixture, frames, active: true }
    window.__cameraFixture = state
    const sample = () => {
      if (!state.active) return
      frames.push({ position: camera.position.toArray(), target: window.__worldDiagnostics.cameraTarget,
        route: window.__worldDiagnostics.cameraPoseRoute })
      requestAnimationFrame(sample)
    }
    requestAnimationFrame(sample)
    store.getState().invalidate()
    return { center, size, radius }
  }, { center, size })
}

export async function removeFixture(page) {
  return page.evaluate(() => {
    const state = window.__cameraFixture
    state.active = false
    state.store.getState().scene.remove(state.fixture)
    state.fixture.dispose() // Geometry and material belong to the existing scene.
    state.store.getState().invalidate()
    delete window.__cameraFixture
    return state.frames
  })
}

export function overlaps(position, fixture) {
  return position.every((value, axis) => Math.abs(value - fixture.center[axis]) < fixture.size[axis] / 2 + fixture.radius - 1e-6)
}

/** Check the whole inter-frame eye segment against the expanded fixture box. */
export function segmentOverlaps(from, to, fixture) {
  let enter = 0
  let leave = 1
  for (let axis = 0; axis < 3; axis++) {
    const extent = fixture.size[axis] / 2 + fixture.radius - 1e-6
    const low = fixture.center[axis] - extent
    const high = fixture.center[axis] + extent
    const delta = to[axis] - from[axis]
    if (Math.abs(delta) < 1e-12) {
      if (from[axis] <= low || from[axis] >= high) return false
      continue
    }
    const a = (low - from[axis]) / delta
    const b = (high - from[axis]) / delta
    enter = Math.max(enter, Math.min(a, b))
    leave = Math.min(leave, Math.max(a, b))
    if (enter >= leave) return false
  }
  return enter < leave
}
