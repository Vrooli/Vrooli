/**
 * Star charts for the presentation sky. Positions are real J2000 right
 * ascension and declination projected onto a tangent plane; `r` is derived from
 * visual magnitude, so the brightest star in a figure really is the brightest
 * point on the page. Decorative only: nothing here is page-owned content.
 */
import type { Mark } from './resources';

export interface ChartStar { id: string; bayer: string; mag: number; x: number; y: number; r: number }
export interface Constellation { viewBox: string; stars: ChartStar[]; edges: [string, string][] }

/** Aquila — the eagle. The web-console product's own constellation. */
export const AQUILA: Constellation = {
  viewBox: '0 0 1000 640',
  stars: [
    { id: 'altair', bayer: 'α Aql', mag: 0.77, x: 412.1, y: 233.0, r: 8.04 },
    { id: 'tarazed', bayer: 'γ Aql', mag: 2.72, x: 438.1, y: 192.7, r: 5.01 },
    { id: 'zeta', bayer: 'ζ Aql', mag: 2.99, x: 672.2, y: 117.8, r: 4.6 },
    { id: 'theta', bayer: 'θ Aql', mag: 3.23, x: 294.5, y: 456.4, r: 4.22 },
    { id: 'delta', bayer: 'δ Aql', mag: 3.36, x: 557.2, y: 365.6, r: 4.02 },
    { id: 'lambda', bayer: 'λ Aql', mag: 3.44, x: 667.4, y: 550.0, r: 3.9 },
    { id: 'alshain', bayer: 'β Aql', mag: 3.71, x: 386.1, y: 289.7, r: 3.48 },
    { id: 'eta', bayer: 'η Aql', mag: 3.9, x: 402.2, y: 414.2, r: 3.18 },
    { id: 'epsilon', bayer: 'ε Aql', mag: 4.02, x: 705.5, y: 90.0, r: 3.0 },
  ],
  edges: [['epsilon', 'zeta'], ['zeta', 'delta'], ['delta', 'lambda'], ['delta', 'altair'], ['tarazed', 'altair'], ['altair', 'alshain'], ['alshain', 'eta'], ['eta', 'theta']],
};

/** Ursa Major's plough. Mizar, another Star line product, sits in the handle. */
export const URSA_MAJOR: Constellation = {
  viewBox: '0 0 1000 640',
  stars: [
    { id: 'alioth', bayer: 'ε UMa', mag: 1.77, x: 354.8, y: 305.0, r: 6.49 },
    { id: 'dubhe', bayer: 'α UMa', mag: 1.79, x: 900.6, y: 102.2, r: 6.46 },
    { id: 'alkaid', bayer: 'η UMa', mag: 1.86, x: 90.0, y: 537.8, r: 6.35 },
    { id: 'mizar', bayer: 'ζ UMa', mag: 2.23, x: 206.8, y: 341.2, r: 5.77 },
    { id: 'merak', bayer: 'β UMa', mag: 2.37, x: 910.0, y: 290.2, r: 5.56 },
    { id: 'phecda', bayer: 'γ UMa', mag: 2.44, x: 652.7, y: 384.3, r: 5.45 },
    { id: 'megrez', bayer: 'δ UMa', mag: 3.31, x: 545.9, y: 267.5, r: 4.1 },
  ],
  edges: [['dubhe', 'merak'], ['merak', 'phecda'], ['phecda', 'megrez'], ['megrez', 'dubhe'], ['megrez', 'alioth'], ['alioth', 'mizar'], ['mizar', 'alkaid']],
};

/** Lyra — the lyre. Vega, the system monitor product's namesake, leads it. */
export const LYRA: Constellation = {
  viewBox: '0 0 1000 640',
  stars: [
    { id: 'vega', bayer: 'α Lyr', mag: 0.03, x: 647.5, y: 147.0, r: 9.19 },
    { id: 'epsilon', bayer: 'ε Lyr', mag: 3.9, x: 551.6, y: 90.0, r: 3.19 },
    { id: 'zeta', bayer: 'ζ Lyr', mag: 4.34, x: 547.2, y: 226.8, r: 3.0 },
    { id: 'delta', bayer: 'δ Lyr', mag: 4.3, x: 419.1, y: 272.9, r: 3.0 },
    { id: 'gamma', bayer: 'γ Lyr', mag: 3.25, x: 352.5, y: 550.0, r: 4.19 },
    { id: 'sheliak', bayer: 'β Lyr', mag: 3.52, x: 476.2, y: 507.3, r: 3.77 },
  ],
  edges: [['vega', 'epsilon'], ['vega', 'zeta'], ['epsilon', 'zeta'], ['zeta', 'delta'], ['delta', 'gamma'], ['gamma', 'sheliak'], ['sheliak', 'zeta']],
};

/**
 * A figure is drawn only where the page's own brand mark identifies one. Other
 * marks get the star field without a figure rather than a borrowed one.
 */
export function constellationForMark(mark: Mark): Constellation | undefined {
  if (mark === 'letter-a') return AQUILA;
  if (mark === 'suite') return URSA_MAJOR;
  if (mark === 'pulse') return LYRA;
  return undefined;
}
