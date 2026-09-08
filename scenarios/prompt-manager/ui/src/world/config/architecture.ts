/** Metres and palette shared by authored space templates and their presenters. */
export const architecture = {
  wallHeight: 2.8,
  wallThickness: .18,
  doorWidth: 1.8,
  doorHeight: 2.25,
  floorThickness: .12,
  cutawayHeight: .65,
  shelterWidth: 4.8,
  shelterDepth: 4.6,
  shelterPitch: 6.2,
  shelterCapacity: 4,
  gatheringDepth: 7,
  siteMargin: 1.4,
  fireRadius: .6,
  fireSeatRadius: 2.1,
  fireSeats: 5,
  fireLightHeight: .65,
  fireLightIntensityScale: 2,
  signHeight: 1.65,
  office: {
    sharedWidth: 8,
    roomGap: .35,
    margin: 1,
    windowSill: .85,
    windowHeight: 1.35,
    windowFraction: .55,
    ceilingThickness: .12,
    trimWidth: .08,
    monitorWidth: .65,
    monitorHeight: .4,
    rowPitch: 2.7,
    lampEdgeInset: .4,
    lampDoorClearance: 1.3,
  },
  palette: {
    timber: '#765139', trim: '#dac5a0', roof: '#3c645b', canvas: '#bd8a4d',
    rv: '#e9dcc0', accent: '#477b79', metal: '#343f45', glass: '#83afb7',
    floor: '#c8b48e', wall: '#e6dfcf', path: '#b39a72',
  },
} as const

/** Flames stop before the last embers cool. Wet weather extinguishes exposed
 * fires; the resolved period already includes the quiet-hours light fade.
 */
export function campfireState(lampEmissive: number, quiet: number, wet: boolean) {
  const light = wet ? 0 : Math.max(0, lampEmissive)
  const glow = Math.min(1, light / 2)
  return { light, embers: glow, flame: glow * Math.max(0, Math.min(1, (.78 - quiet) / .65)) }
}
