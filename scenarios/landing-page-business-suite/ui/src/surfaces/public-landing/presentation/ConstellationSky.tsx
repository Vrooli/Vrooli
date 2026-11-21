/**
 * The presentation's night sky: a parallax star field behind the whole page and,
 * in the hero, the page's own constellation resolving into the product.
 *
 * Entirely decorative and aria-hidden — it renders no page-owned content. All
 * motion is CSS, gated on `prefers-reduced-motion`, so a reduced-motion viewer
 * and every screenshot capture get the finished composition with nothing running.
 */
import { useEffect, useMemo, useRef } from 'react';
import type { Constellation } from './constellations';

/** Star field depth. Nearer layers are fewer, larger, and move further. */
const LAYERS = [
  { n: 90, r: [0.5, 1.0], o: [0.16, 0.34], depth: 0.10, tw: [6, 11] },
  { n: 52, r: [0.9, 1.6], o: [0.26, 0.52], depth: 0.22, tw: [5, 9] },
  { n: 22, r: [1.4, 2.3], o: [0.40, 0.78], depth: 0.38, tw: [4, 7] },
] as const;

/** Deterministic: the sky is identical across reloads, captures and viewers. */
function lcg(seed: number) {
  let s = seed;
  return () => { s = (s * 1664525 + 1013904223) % 4294967296; return s / 4294967296; };
}

interface Speck { left: string; top: string; size: string; o: string; tw: string; delay: string }

function layerSpecks(index: number): Speck[] {
  const layer = LAYERS[index];
  if (!layer) return [];
  const rnd = lcg(20260916 + index * 7919);
  const specks: Speck[] = [];
  for (let i = 0; i < layer.n; i++) {
    const size = layer.r[0] + rnd() * (layer.r[1] - layer.r[0]);
    specks.push({
      left: `${(rnd() * 100).toFixed(2)}%`, top: `${(rnd() * 100).toFixed(2)}%`,
      size: `${size.toFixed(2)}px`,
      o: (layer.o[0] + rnd() * (layer.o[1] - layer.o[0])).toFixed(2),
      tw: `${(layer.tw[0] + rnd() * (layer.tw[1] - layer.tw[0])).toFixed(1)}s`,
      delay: `-${(rnd() * 8).toFixed(1)}s`,
    });
  }
  return specks;
}

/** Fixed behind the page. One sky, not a decoration per section. */
export function StarField() {
  const layers = useMemo(() => LAYERS.map((_, index) => layerSpecks(index)), []);
  const refs = useRef<(HTMLDivElement | null)[]>([]);

  useEffect(() => {
    // eslint-disable-next-line @typescript-eslint/no-unnecessary-condition -- Test DOMs may omit matchMedia.
    const reduced = window.matchMedia?.('(prefers-reduced-motion: reduce)') as MediaQueryList | undefined;
    if (reduced?.matches) return;
    let frame = 0;
    let scrolled = 0;
    let px = 0;
    let py = 0;
    const onScroll = () => { scrolled = window.scrollY; };
    const onPointer = (event: PointerEvent) => {
      px = event.clientX / window.innerWidth - 0.5;
      py = event.clientY / window.innerHeight - 0.5;
    };
    const step = () => {
      LAYERS.forEach((layer, index) => {
        const node = refs.current[index];
        if (!node) return;
        node.style.transform = `translate3d(${(px * -26 * layer.depth).toFixed(2)}px, ${(scrolled * -layer.depth + py * -20 * layer.depth).toFixed(2)}px, 0)`;
      });
      frame = requestAnimationFrame(step);
    };
    window.addEventListener('scroll', onScroll, { passive: true });
    window.addEventListener('pointermove', onPointer, { passive: true });
    frame = requestAnimationFrame(step);
    return () => {
      cancelAnimationFrame(frame);
      window.removeEventListener('scroll', onScroll);
      window.removeEventListener('pointermove', onPointer);
    };
  }, []);

  return <div className="sky" aria-hidden="true">
    <div className="sky-ground" />
    {layers.map((specks, index) => <div key={index} className="star-layer" ref={node => { refs.current[index] = node; }}>
      {specks.map((speck, i) => <i key={i} style={{
        left: speck.left, top: speck.top, width: speck.size, height: speck.size,
        ['--speck-o' as string]: speck.o, ['--speck-tw' as string]: speck.tw, animationDelay: speck.delay,
      }} />)}
    </div>)}
  </div>;
}

const STAR_T0 = 0.30, STAR_STEP = 0.075, EDGE_T0 = 0.95, EDGE_STEP = 0.085;

/**
 * The hero figure. Stars ignite brightest-first, edges draw between them, and the
 * whole layer then converges toward the product exhibit below it.
 */
export function ConstellationFigure({ chart }: { chart: Constellation }) {
  const points = useMemo(() => new Map(chart.stars.map(star => [star.id, star])), [chart]);
  return <div className="hero-sky" aria-hidden="true">
    <svg className="hero-chart" viewBox={chart.viewBox} role="presentation" focusable="false">
      <defs>
        {/* The glow follows the page's own accent so each product's figure
            ignites in its brand color, not a borrowed one. */}
        <radialGradient id="presentation-star-glow">
          <stop offset="0" style={{ stopColor: 'var(--accent, #22d3ee)' }} stopOpacity=".62" />
          <stop offset=".42" style={{ stopColor: 'var(--accent, #22d3ee)' }} stopOpacity=".20" />
          <stop offset="1" style={{ stopColor: 'var(--accent, #22d3ee)' }} stopOpacity="0" />
        </radialGradient>
      </defs>
      {chart.edges.map(([from, to], index) => {
        const a = points.get(from);
        const b = points.get(to);
        if (!a || !b) throw new Error(`Unresolved constellation edge: ${from}-${to}`);
        const length = Math.hypot(b.x - a.x, b.y - a.y).toFixed(1);
        return <path key={`${from}-${to}`} className="chart-edge"
          style={{ ['--chart-d' as string]: `${(EDGE_T0 + index * EDGE_STEP).toFixed(2)}s`, ['--chart-len' as string]: length }}
          d={`M${String(a.x)} ${String(a.y)} L${String(b.x)} ${String(b.y)}`} />;
      })}
      {chart.stars.map((star, index) => {
        const delay = `${(STAR_T0 + index * STAR_STEP).toFixed(2)}s`;
        return <g key={star.id}>
          <circle className="chart-halo" style={{ ['--chart-d' as string]: delay }}
            cx={star.x} cy={star.y} r={(star.r * 3.4).toFixed(1)} fill="url(#presentation-star-glow)" />
          <circle className="chart-core" style={{ ['--chart-d' as string]: delay }}
            cx={star.x} cy={star.y} r={(star.r * 0.56).toFixed(2)} />
          {star.mag < 3.5 && <text style={{ ['--chart-d' as string]: `${(STAR_T0 + index * STAR_STEP + 0.6).toFixed(2)}s` }}
            x={(star.x + star.r * 1.5 + 6).toFixed(1)} y={(star.y + 3.5).toFixed(1)}>{star.bayer}</text>}
        </g>;
      })}
    </svg>
  </div>;
}
