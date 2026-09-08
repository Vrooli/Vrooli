import { describe, expect, it } from 'vitest'
import { BoxGeometry, Group, Mesh, Vector3 } from 'three'
import { createObstacleSweep } from './obstacles'
import { Walker, WALK } from './walking'

const flat = () => ({ height: 0, walkable: true })
describe('grounded operator movement', () => {
  it('walks and runs in metres per second, without diagonal acceleration or frame-rate drift', () => {
    for (const fps of [5, 10, 30, 60, 120]) {
      const straight = new Walker(flat, () => 1), diagonal = new Walker(flat, () => 1), running = new Walker(flat, () => 1)
      for (let frame = 0; frame < fps; frame++) { straight.step(1 / fps, 1, 0, false); diagonal.step(1 / fps, 1, 1, false); running.step(1 / fps, 1, 0, true) }
      expect(straight.position.length()).toBeCloseTo(WALK.speed, 7)
      expect(diagonal.position.length()).toBeCloseTo(WALK.speed, 7)
      expect(running.position.length()).toBeCloseTo(WALK.runSpeed, 7)
      const stopped = straight.position.clone()
      straight.step(1, 0, 0, false)
      expect(straight.position.equals(stopped)).toBe(true)
    }
  })
  it('cannot tunnel through a narrow blocked navigation band at low FPS', () => {
    const body = new Walker((_x, z) => ({ height: 0, walkable: z > -.5 || z < -.7 }), () => 1)
    body.step(.2, 1, 0, true)
    expect(body.position.z).toBeGreaterThan(-.5)
    expect(body.blocked).toBe(true)
  })
  it('rejects cliffs, follows gentle ground, and finds a safe entry near a blocked target', () => {
    const cliff = new Walker((_x, z) => ({ height: z < -.2 ? 2 : 0, walkable: true }), () => 1)
    cliff.step(.2, 1, 0, true)
    expect(cliff.position.z).toBeGreaterThanOrEqual(-.2)
    const slope = new Walker((_x, z) => ({ height: -z * .2, walkable: true }), () => 1)
    slope.step(.2, 1, 0, false)
    expect(slope.position.y).toBeCloseTo(-slope.position.z * .2)
    const spawn = new Walker((x, z) => ({ height: 3, walkable: Math.hypot(x, z) >= 1 }), () => 1)
    expect(spawn.spawn(0, 0)).toBe(true)
    expect(spawn.position.y).toBe(3)
    expect(Math.hypot(spawn.position.x, spawn.position.z)).toBeGreaterThanOrEqual(1)
    expect(new Walker(() => ({ height: 0, walkable: false }), () => 1).spawn(0, 0)).toBe(false)
  })
  it('sweeps the body against real structural boxes and retracts the third-person boom', () => {
    const root = new Group(), wall = new Mesh(new BoxGeometry(10, 4, .1))
    wall.geometry.userData.cameraObstacle = 'box'
    wall.position.set(0, 2, -1)
    root.add(wall)
    const body = new Walker(flat, createObstacleSweep(root))
    body.step(.2, 1, 0, true)
    expect(body.position.z).toBeGreaterThan(-.7)
    expect(body.blocked).toBe(true)
    wall.position.z = 1
    body.position.set(0, 0, 0)
    const third = body.view(true)
    expect(third.eye.z).toBeGreaterThan(0)
    expect(third.eye.z).toBeLessThan(.8)
    expect(body.view(false).eye.distanceTo(new Vector3(0, WALK.eyeHeight, 0))).toBe(0)
    wall.geometry.dispose()
  })
  it('bounds pitch, preserves yaw across turns, and does not replay tab suspension', () => {
    const body = new Walker(flat, () => 1)
    body.look(10000, 10000, 1, false)
    expect(body.pitch).toBe(-1.4)
    expect(body.yaw).toBeGreaterThan(Math.PI * 2)
    body.step(120, 1, 0, true)
    expect(body.position.length()).toBeCloseTo(WALK.runSpeed * .05)
  })
  it('uses rendered furniture bounds even where a navigation cell was carved for agent seating', () => {
    const root = new Group(), props = new Group(), seat = new Mesh(new BoxGeometry(2, .5, .5))
    props.userData.walkObstacle = true
    seat.position.set(0, .25, -1)
    props.add(seat); root.add(props)
    const body = new Walker(flat, createObstacleSweep(root, undefined, true))
    body.step(.2, 1, 0, true)
    expect(body.position.z).toBeGreaterThan(-.5)
    expect(body.blocked).toBe(true)
    seat.geometry.dispose()
  })
  it('protects the whole capsule from thin furniture edges between sphere sample heights', () => {
    const root = new Group(), edge = new Mesh(new BoxGeometry(.2, .02, .1))
    edge.geometry.userData.cameraObstacle = 'box'
    edge.position.set(.3, .625, -.8)
    root.add(edge)
    const body = new Walker(flat, createObstacleSweep(root))
    body.step(.2, 1, 0, true)
    expect(body.position.z).toBeGreaterThan(-.55)
    expect(body.blocked).toBe(true)
    edge.geometry.dispose()
  })
  it('does not spawn inside furniture merely because movement toward an exit is allowed', () => {
    const root = new Group(), box = new Mesh(new BoxGeometry(2, 2, 2))
    box.geometry.userData.cameraObstacle = 'box'
    box.position.set(0, 1, 0)
    root.add(box)
    const body = new Walker(flat, createObstacleSweep(root))
    expect(body.safe(.8, .8)).toBeNull()
    expect(body.spawn(.8, .8)).toBe(true)
    expect(body.safe(body.position.x, body.position.z)).not.toBeNull()
    box.geometry.dispose()
  })
})


describe('operator jumping', () => {
  it('has consistent airtime and height at low and high frame rates, and cannot double jump', () => {
    for (const fps of [5, 30, 60, 120]) {
      const body = new Walker(flat, () => 1)
      expect(body.jump()).toBe(true)
      expect(body.jump()).toBe(false)
      for (let i = 0; i < fps / 5; i++) body.step(1 / fps, 1, 0, false)
      expect(body.position.y).toBeCloseTo(.76, 6)
      expect(body.position.z).toBeCloseTo(-.7, 6)
      expect(body.validPosition()).toBe(true)
      expect(body.view(false).eye.y).toBeCloseTo(.76 + WALK.eyeHeight)
      expect(body.view(true).target.y).toBeCloseTo(.76 + WALK.eyeHeight)
      for (let i = 0; i < fps; i++) body.step(1 / fps, 0, 0, false)
      expect(body.position.y).toBe(0)
      expect(body.grounded).toBe(true)
      expect(body.jump()).toBe(true)
    }
  })

  it('stops at a thin ceiling and lands without clipping at low FPS', () => {
    const root = new Group(), ceiling = new Mesh(new BoxGeometry(10, .02, 10))
    ceiling.geometry.userData.cameraObstacle = 'box'
    ceiling.position.y = 2.2
    root.add(ceiling)
    const body = new Walker(flat, createObstacleSweep(root))
    body.jump()
    for (let i = 0; i < 8; i++) {
      body.step(.2, 0, 0, false)
      expect(body.position.y + WALK.height).toBeLessThanOrEqual(2.19)
      expect(body.validPosition()).toBe(true)
    }
    expect(body.position.y).toBe(0)
    expect(body.grounded).toBe(true)
    ceiling.geometry.dispose()
  })
})


it('can clear a low obstacle while airborne and lands safely beyond it', () => {
  const root = new Group(), obstacle = new Mesh(new BoxGeometry(2, .25, .15))
  obstacle.geometry.userData.cameraObstacle = 'box'
  obstacle.position.set(0, .125, -1)
  root.add(obstacle)
  const body = new Walker(flat, createObstacleSweep(root))
  body.jump()
  for (let i = 0; i < 60; i++) {
    body.step(1 / 60, 1, 0, false)
    expect(body.validPosition()).toBe(true)
  }
  expect(body.position.z).toBeCloseTo(-WALK.speed)
  expect(body.position.y).toBe(0)
  expect(body.grounded).toBe(true)
  obstacle.geometry.dispose()
})
