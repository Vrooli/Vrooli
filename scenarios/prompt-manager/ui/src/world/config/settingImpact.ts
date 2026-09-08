import type { ActorTuning, CameraTuning, LabelsTuning, LayoutTuning, SimTuning } from './tuning.schema'

export type SettingImpact = 'live' | 'material' | 'geometry' | 'world' | 'asset'

/** Exhaustive surface contract: adding a surface setting requires an impact decision. */
export const layoutSurfaceImpacts: Record<keyof LayoutTuning['surfaces'], SettingImpact> = {
  wallThickness: 'geometry', doorFrameScale: 'geometry', floorLift: 'geometry',
  corridorLift: 'geometry', floorThickness: 'geometry', commonsLift: 'geometry', commonsSegments: 'geometry',
  wallRoughness: 'material', floorRoughness: 'material', corridorRoughness: 'material', commonsRoughness: 'material',
}

export function layoutSurfaceImpact(path: string): SettingImpact | undefined {
  const prefix = 'layout.surfaces.'
  if (!path.startsWith(prefix)) return undefined
  const key = path.slice(prefix.length)
  return Object.prototype.hasOwnProperty.call(layoutSurfaceImpacts, key) ? layoutSurfaceImpacts[key as keyof typeof layoutSurfaceImpacts] : undefined
}

/** Arrays are one setting; nested objects require an explicit decision for every leaf. */
type ImpactTree<T> = { [K in keyof T]: T[K] extends readonly unknown[] ? SettingImpact : T[K] extends object ? ImpactTree<T[K]> : SettingImpact }

export const labelsImpacts: ImpactTree<LabelsTuning> = {
  color: 'material',
  strokeColor: 'material',
  strokePercent: 'material',
  charWidthFactor: 'live',
  refreshEveryFrames: 'live',
  basePxPerUnit: 'geometry',
  pinnedBonus: 'live',
  priorities: {
    failed: 'live',
    working: 'live',
    walkingToDesk: 'live',
    gathered: 'live',
    walkingToTable: 'live',
    socializing: 'live',
    idle: 'live',
  },
  syncSizeEpsilon: 'live',
  renderOrder: 'live',
  budget: 'geometry',
  collapseDistance: 'live',
  fontSize: 'geometry',
  offsetY: 'live',
  roomOffsetY: 'live',
  minScreenPx: 'geometry',
  maxScreenPx: 'geometry',
  paddingPx: 'live',
}

// Bootstrap position and intro duration are initialization-only.
export const cameraImpacts: ImpactTree<Omit<CameraTuning, 'initialPosition' | 'introSeconds'>> = {
  minimumProjectionAspect: 'live',
  minimumFrameFill: 'live',
  input: {
    mouse: {
      left: 'live',
      middle: 'live',
      right: 'live',
      wheel: 'live',
    },
    touch: {
      one: 'live',
      two: 'live',
      three: 'live',
    },
  },
  dollyToCursor: 'live',
  truckSpeed: 'live',
  dollySpeed: 'live',
  cullEpsilonMetres: 'live',
  cullEpsilonRadians: 'live',
  fov: 'live',
  near: 'live',
  far: 'live',
  polarMinDeg: 'live',
  polarMaxDeg: 'live',
  azimuthRangeDeg: 'live',
  minDistance: 'live',
  maxDistance: 'live',
  smoothTime: 'live',
  followEpsilon: 'live',
  followSmoothTime: 'live',
  focusPadding: 'live',
  focusVisibilityTurnsDeg: 'live',
  focusVisibilityHeights: 'live',
  keyOrbitDegPerSec: 'live',
  keyDollyPerSec: 'live',
  frameFill: 'live',
  boundaryHeight: 'live',
  frameHeight: 'live',
}

export const simImpacts: ImpactTree<SimTuning> = {
  tickSeconds: 'live',
  walkSpeed: 'live',
  hurrySpeed: 'live',
  turnRateRadPerSec: 'live',
  arriveRadius: 'live',
  gatherLeadSeconds: 'live',
  gatherWindowSeconds: 'live',
  failedAckSeconds: 'live',
  eventsRing: 'live',
  maxReplansPerTick: 'live',
  pathCacheSize: 'live',
  idle: {
    rollIntervalSeconds: 'live',
    weights: {
      rest: 'live',
      wander: 'live',
      socialize: 'live',
      sit: 'live',
    },
    maxMoversRatio: 'live',
    socializeSeconds: {
      min: 'live',
      max: 'live',
    },
    sitSeconds: {
      min: 'live',
      max: 'live',
    },
    restSeconds: {
      min: 'live',
      max: 'live',
    },
    socializeGap: 'live',
    spacing: 'live',
    spacingAttempts: 'live',
  },
  accelSeconds: 'live',
}

export const layoutImpacts: ImpactTree<LayoutTuning> = {
  lampInsetRatio: 'geometry',
  corridorLampSpacing: 'geometry',
  corridorLampScale: 'geometry',
  cellSize: 'world',
  roomWidth: 'world',
  roomDepth: 'world',
  deskPitch: 'world',
  deskInset: 'world',
  deskSeatOffset: 'world',
  tableRadius: 'world',
  tableSeatRadius: 'world',
  tableSeats: 'world',
  commonsRadius: 'world',
  commonsSeatRadius: 'world',
  commonsSeats: 'world',
  clearingRadius: 'world',
  wallHeight: 'geometry',
  surfaces: layoutSurfaceImpacts,
  boardOffset: 'world',
  outlineRimSamples: 'world',
  siteCandidates: 'world',
  siteRadiusMax: 'world',
  siteSpacing: 'world',
  siteWeightFlat: 'world',
  siteWeightDry: 'world',
  siteWeightNear: 'world',
  siteWeightApart: 'world',
  siteRotationSnapRad: 'world',
  scatterJitter: 'world',
  shoreClearance: 'world',
  stands: {
    frequency: 'world',
    octaves: 'world',
    lacunarity: 'world',
    gain: 'world',
    threshold: 'world',
    softness: 'world',
    contrast: 'world',
    floor: 'world',
  },
  decorSpacingFactor: 'world',
  decorScale: {
    min: 'world',
    max: 'world',
  },
  decorColorJitter: 'world',
  floorplan: {
    corridorWidth: 'world',
    secondaryCorridors: {
      min: 'world',
      max: 'world',
    },
    splitRatio: {
      min: 'world',
      max: 'world',
    },
    maxAspect: 'world',
    roomAreaPerMember: 'world',
    roomMinArea: 'world',
    plateMargin: 'world',
    doorWidth: 'world',
    lobbyRadius: 'world',
    plateAspect: {
      min: 'world',
      max: 'world',
    },
    primaryOffset: 'world',
    secondaryJitter: 'world',
  },
  interior: {
    tableMinMembers: 'world',
    fillerMax: 'world',
  },
}

export const actorImpacts: ImpactTree<ActorTuning> = {
  extras: {
    tierSizes: 'live',
    tierColors: 'live',
    failed: {
      color: 'live',
      intensity: 'live',
    },
    gathered: {
      color: 'live',
      intensity: 'live',
    },
    working: {
      color: 'live',
      intensity: 'live',
    },
    offColor: 'live',
    emotes: {
      start: {
        color: 'live',
        intensity: 'live',
      },
      done: {
        color: 'live',
        intensity: 'live',
      },
      fail: {
        color: 'live',
        intensity: 'live',
      },
      message: {
        color: 'live',
        intensity: 'live',
      },
      gather: {
        color: 'live',
        intensity: 'live',
      },
    },
    spinRate: 'live',
    gearHeight: 'geometry',
    gearDepth: 'geometry',
    gearRoughness: 'material',
    ringThickness: 'geometry',
    ringRadialSegments: 'geometry',
    ringTubularSegments: 'geometry',
    markWidthSegments: 'geometry',
    markHeightSegments: 'geometry',
    markScale: 'live',
    emoteOpacity: 'material',
    emoteShrink: 'live',
  },
  shadow: {
    textureSize: 'material',
    lift: 'live',
    opacity: 'material',
    spread: 'live',
    hopShrink: 'live',
    color: 'material',
    gradient: 'material',
  },
  facing: {
    restSpeed: 'live',
    blendSeconds: 'live',
    maxYawDeg: 'live',
  },
  bodyRadius: 'world',
  breathAmplitude: 'live',
  breathHz: 'live',
  hopHeight: 'live',
  hopHz: 'live',
  squashOnLand: 'live',
  squashRecoverPerSec: 'live',
  wobbleIntensity: 'material',
  blinkIntervalSeconds: {
    min: 'live',
    max: 'live',
  },
  blinkSeconds: 'live',
  emoteSeconds: 'live',
  seatedScale: 'live',
  equipmentTiers: 'live',
  look: {
    minDetailPx: 'live',
    minimumProjectionDepth: 'live',
    bodySquashY: 'live',
    eyeRadius: 'live',
    eyeSpacing: 'live',
    eyeHeight: 'live',
    eyeForward: 'live',
    blinkScaleY: 'live',
    mouthWidth: 'live',
    mouthHeight: 'live',
    mouthForward: 'live',
    mouthDrop: 'live',
    earSize: 'live',
    earHeight: 'live',
    earSpread: 'live',
    equipmentScale: 'live',
    equipmentBack: 'live',
    equipmentHeight: 'live',
    markerHeight: 'live',
    markerRadius: 'live',
    emoteRise: 'live',
    emoteSize: 'live',
    emoteHeight: 'live',
    messageTtlSeconds: 'live',
    eyeColor: 'material',
    mouthColor: 'material',
    eyeRoughness: 'material',
    mouthRoughness: 'material',
    earRoughness: 'material',
    eyeWidthSegments: 'geometry',
    eyeHeightSegments: 'geometry',
    earSegments: 'geometry',
    largeEarScale: 'live',
    earTiltRad: 'live',
    mouthVariantScales: 'live',
    emoteMouthScale: 'live',
  },
  material: {
    color: 'material',
    sheenColor: 'material',
    roughness: 'material',
    clearcoat: 'material',
    clearcoatRoughness: 'material',
    sheen: 'material',
    wobbleScale: 'material',
    wobbleSpeed: 'material',
  },
  mesh: {
    widthSegments: 'geometry',
    heightSegments: 'geometry',
    timeShiftSeconds: 'live',
  },
}

/** Known declarations only: an unknown setting must not silently become a live edit. */
export function tuningSettingImpact(path: string): SettingImpact | undefined {
  const [group, ...keys] = path.split('.')
  let current: unknown = group === 'actor' ? actorImpacts : group === 'layout' ? layoutImpacts : group === 'sim' ? simImpacts : group === 'camera' ? cameraImpacts : group === 'labels' ? labelsImpacts : undefined
  if (!keys.length) return undefined
  for (const key of keys) {
    if (typeof current !== 'object' || current === null || !Object.prototype.hasOwnProperty.call(current, key)) return undefined
    current = (current as Record<string, unknown>)[key]
  }
  return typeof current === 'string' ? current as SettingImpact : undefined
}

/** Project generation inputs from the same declarations used by the workbench. */
function generationInputs(value: Record<string, unknown>, impacts: object): Record<string, unknown> {
  const result: Record<string, unknown> = {}
  for (const [key, impact] of Object.entries(impacts) as Array<[string, unknown]>) {
    if (impact === 'world') result[key] = value[key]
    else if (typeof impact === 'object' && impact !== null) {
      const nested = generationInputs(value[key] as Record<string, unknown>, impact)
      if (Object.keys(nested).length) result[key] = nested
    }
  }
  return result
}

export function generationActorTuning(actor: ActorTuning): Record<string, unknown> {
  return generationInputs(actor, actorImpacts)
}

export function generationLayoutTuning(layout: LayoutTuning): Record<string, unknown> {
  // Preserve the established empty surface namespace in canonical recipes.
  return { surfaces: {}, ...generationInputs(layout, layoutImpacts) }
}
