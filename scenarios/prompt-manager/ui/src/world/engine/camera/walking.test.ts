import { readFileSync } from 'node:fs'
import { GLTFLoader } from 'three/examples/jsm/loaders/GLTFLoader.js'
import { MeshoptDecoder } from 'three/examples/jsm/libs/meshopt_decoder.module.js'
import { preparePropParts } from '../assets/geometry'
import registry from '../assets/registry.generated.json'
import park from '../../config/scenes/park.json'
import { describe, expect, it } from 'vitest'
import { BoxGeometry, Group, Mesh, InstancedMesh, Matrix4, Vector3 } from 'three'
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
  it('enters walking mode on a raised floor without relocating the visitor outside the room', () => {
    const root = new Group(), floor = new Mesh(new BoxGeometry(12, .12, 12))
    floor.geometry.userData.cameraObstacle = 'box'
    root.add(floor)
    const body = new Walker(flat, createObstacleSweep(root))
    expect(body.spawn(0, 0)).toBe(true)
    expect(body.position.x).toBe(0)
    expect(body.position.z).toBe(0)
    expect(body.position.y).toBeCloseTo(.06, 4)
    expect(body.validPosition()).toBe(true)
    floor.geometry.dispose()
  })
  it('steps across a small floor lip under a doorway with less than the maximum step headroom', () => {
    const root = new Group(), floor = new Mesh(new BoxGeometry(12, .12, 6)), lintel = new Mesh(new BoxGeometry(1.8, .55, .18))
    floor.position.z = -4
    lintel.position.set(0, 2.525, -1)
    floor.geometry.userData.cameraObstacle = lintel.geometry.userData.cameraObstacle = 'box'
    root.add(floor, lintel)
    for (const fps of [5, 60]) {
      const body = new Walker(flat, createObstacleSweep(root))
      body.position.y = .021
      for (let frame = 0; frame < fps; frame++) body.step(1 / fps, 1, 0, false)
      expect(body.position.z).toBeLessThan(-2)
      expect(body.position.y).toBeCloseTo(.06, 4)
    }
    floor.geometry.dispose(); lintel.geometry.dispose()
  })
  it('clears a short obstacle on a raised floor without confusing its support height with a terrain cliff', () => {
    const root = new Group(), floor = new Mesh(new BoxGeometry(12, .12, 12)), box = new Mesh(new BoxGeometry(.6, .4, .6))
    box.position.set(0, .26, -1)
    floor.geometry.userData.cameraObstacle = box.geometry.userData.cameraObstacle = 'box'
    root.add(floor, box)
    for (const fps of [5, 60]) {
      const body = new Walker(flat, createObstacleSweep(root))
      expect(body.spawn(0, 0)).toBe(true)
      let peak = 0
      for (let frame = 0; frame < fps; frame++) { body.step(1 / fps, 1, 0, false); peak = Math.max(peak, body.position.y) }
      expect(peak).toBeGreaterThan(.45)
      expect(body.position.z).toBeLessThan(-2)
      expect(body.position.y).toBeCloseTo(.06, 4)
    }
    floor.geometry.dispose(); box.geometry.dispose()
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


it('steps over low boxes and stays above their surface at every frame', () => {
  for (const fps of [5, 60, 120]) {
    const root = new Group(), box = new Mesh(new BoxGeometry(2, .25, .8))
    box.geometry.userData.cameraObstacle = 'box'
    box.position.set(0, .125, -1)
    root.add(box)
    const body = new Walker(flat, createObstacleSweep(root))
    let highest = 0
    for (let i = 0; i < fps; i++) {
      body.step(1 / fps, 1, 0, false)
      highest = Math.max(highest, body.position.y)
      expect(body.validPosition()).toBe(true)
    }
    expect(highest).toBeGreaterThan(.2)
    expect(body.position.z).toBeCloseTo(-WALK.speed)
    expect(body.position.y).toBeCloseTo(0)
    box.geometry.dispose()
  }
})

it('cannot step through a low ceiling or climb objects above step height', () => {
  for (const ceiling of [true, false]) {
    const root = new Group(), box = new Mesh(new BoxGeometry(2, ceiling ? .25 : .5, .8))
    box.geometry.userData.cameraObstacle = 'box'
    box.position.set(0, ceiling ? .125 : .25, -1)
    root.add(box)
    if (ceiling) {
      const roof = new Mesh(new BoxGeometry(4, .1, 4))
      roof.geometry.userData.cameraObstacle = 'box'
      roof.position.y = 1.95
      root.add(roof)
    }
    const body = new Walker(flat, createObstacleSweep(root))
    for (let i = 0; i < 60; i++) body.step(1 / 60, 1, 0, false)
    expect(body.position.z).toBeGreaterThan(-.6)
    expect(body.validPosition()).toBe(true)
    root.children.forEach(mesh => (mesh as Mesh).geometry.dispose())
  }
})


it('walks across the shipped park log, in both directions and at rotated placements', async () => {
  const record = registry.props['park/log_seat']
  const bytes = readFileSync('public/assets/world/' + record.path)
  const asset = await new GLTFLoader().setMeshoptDecoder(MeshoptDecoder).parseAsync(new Uint8Array(bytes).buffer, '')
  const parts = preparePropParts(asset.scene)
  try {
    for (const rotation of [0, Math.PI / 4, Math.PI / 2]) for (const fps of [5, 60]) for (const sign of [-1, 1]) {
      const root = new Group()
      root.userData.walkObstacle = true
      for (const part of parts) {
        const mesh = new InstancedMesh(part.geometry, part.material, 1)
        const matrix = new Matrix4().makeRotationY(rotation).scale(new Vector3().setScalar(park.propScale))
        matrix.setPosition(0, -(record.bounds.min[1] ?? 0) * park.propScale, 0)
        mesh.setMatrixAt(0, matrix)
        root.add(mesh)
      }
      const body = new Walker(flat, createObstacleSweep(root, undefined, true))
      body.position.set(0, 0, sign * 1.5)
      body.yaw = sign < 0 ? Math.PI : 0
      let high = 0
      for (let i = 0; i < fps; i++) {
        body.step(1 / fps, 1, 0, false)
        high = Math.max(high, body.position.y)
        expect(body.validPosition()).toBe(true)
      }
      expect(body.position.z * sign).toBeLessThan(-1.5)
      expect(high).toBeGreaterThan(.3)
      expect(body.position.y).toBeCloseTo(0)
      root.children.forEach(mesh => (mesh as InstancedMesh).dispose())
    }
  } finally {
    parts.forEach(part => part.geometry.dispose())
    asset.scene.traverse(object => {
      if (object instanceof Mesh) {
        object.geometry.dispose()
        const materials = Array.isArray(object.material) ? object.material : [object.material]
        materials.forEach(material => material.dispose())
      }
    })
  }
})
