import { useId } from "react";
import { generateStarField, zodiacForDate, type ZodiacName } from "../visual/proceduralSkyModel";

type Constellation = {
  label: string;
  stars: ReadonlyArray<readonly [number, number, number?]>;
  lines: ReadonlyArray<readonly [number, number]>;
};

const STARS = generateStarField();

// Constellation line figures are not standardized by the IAU. These restrained
// diagrams follow the familiar Western silhouettes without presenting any one
// tradition's optional line set as an official astronomical boundary.
const CONSTELLATIONS: Record<ZodiacName, Constellation> = {
  aries: { label: "Aries", stars: [[13, 60, 1.1], [34, 45, 1.4], [60, 39, 1.8], [78, 49, 1.2]], lines: [[0, 1], [1, 2], [2, 3]] },
  taurus: { label: "Taurus", stars: [[10, 20], [39, 48, 1.8], [52, 58, 1.3], [64, 43, 1.5], [92, 17], [43, 72], [58, 78]], lines: [[0, 1], [1, 2], [2, 3], [3, 4], [1, 5], [2, 6]] },
  gemini: { label: "Gemini", stars: [[27, 12, 1.8], [67, 14, 1.7], [32, 35], [62, 37], [25, 61], [67, 62], [14, 86], [38, 84], [57, 86], [82, 84]], lines: [[0, 2], [2, 4], [4, 6], [4, 7], [1, 3], [3, 5], [5, 8], [5, 9], [2, 3]] },
  cancer: { label: "Cancer", stars: [[48, 13, 1.4], [49, 40, 1.8], [22, 66], [70, 61], [87, 82]], lines: [[0, 1], [1, 2], [1, 3], [3, 4]] },
  leo: { label: "Leo", stars: [[16, 57], [27, 35], [43, 23, 1.3], [52, 42, 1.8], [42, 63], [66, 72], [89, 59, 1.5]], lines: [[0, 1], [1, 2], [2, 3], [3, 4], [4, 0], [4, 5], [5, 6]] },
  virgo: { label: "Virgo", stars: [[8, 34], [29, 44], [47, 56, 1.3], [66, 44], [90, 31], [59, 72, 1.9], [45, 88], [78, 73]], lines: [[0, 1], [1, 2], [2, 3], [3, 4], [2, 5], [5, 6], [5, 7]] },
  libra: { label: "Libra", stars: [[23, 31, 1.5], [72, 28, 1.4], [84, 68], [48, 84], [14, 64]], lines: [[0, 1], [1, 2], [2, 3], [3, 4], [4, 0], [0, 3]] },
  scorpio: { label: "Scorpio", stars: [[9, 19], [25, 31], [36, 50, 1.8], [46, 65], [61, 72], [75, 67], [86, 54], [91, 38], [82, 29]], lines: [[0, 1], [1, 2], [2, 3], [3, 4], [4, 5], [5, 6], [6, 7], [7, 8]] },
  sagittarius: { label: "Sagittarius", stars: [[18, 59], [38, 35], [65, 37], [80, 58], [58, 67], [37, 63], [52, 49, 1.7], [73, 19], [90, 14]], lines: [[0, 1], [1, 2], [2, 3], [3, 4], [4, 5], [5, 0], [1, 6], [6, 4], [2, 7], [7, 8]] },
  capricorn: { label: "Capricorn", stars: [[10, 30], [31, 47, 1.3], [51, 74], [76, 83], [91, 57], [76, 39], [45, 24]], lines: [[0, 1], [1, 2], [2, 3], [3, 4], [4, 5], [5, 6], [6, 0], [1, 6]] },
  aquarius: { label: "Aquarius", stars: [[12, 21], [29, 15], [43, 26, 1.6], [58, 17], [73, 29], [57, 44], [69, 59], [60, 75], [77, 88], [42, 66]], lines: [[0, 1], [1, 2], [2, 3], [3, 4], [2, 5], [5, 6], [6, 7], [7, 8], [5, 9]] },
  pisces: { label: "Pisces", stars: [[10, 24], [23, 14], [37, 23], [25, 34], [47, 49], [62, 62], [78, 77], [91, 65], [86, 48], [72, 45]], lines: [[0, 1], [1, 2], [2, 3], [3, 0], [3, 4], [4, 5], [5, 6], [6, 7], [7, 8], [8, 9], [9, 6]] },
};

function ConstellationFigure({ zodiac }: { zodiac: ZodiacName }) {
  const figure = CONSTELLATIONS[zodiac];
  return <g className="sky-constellation" data-constellation={zodiac}>
    <g transform="translate(345 105) scale(3.1)">
      {figure.lines.map(([from, to], index) => <line key={`line-${index}`} x1={figure.stars[from]?.[0]} y1={figure.stars[from]?.[1]} x2={figure.stars[to]?.[0]} y2={figure.stars[to]?.[1]} />)}
      {figure.stars.map(([x, y, weight = 1], index) => <circle key={`star-${index}`} cx={x} cy={y} r={1.05 * weight} />)}
    </g>
    <text className="sky-constellation-label" x="500" y="430" textAnchor="middle">{figure.label}</text>
  </g>;
}

export function ProceduralSky({ showConstellation = false, date = new Date() }: { showConstellation?: boolean; date?: Date }) {
  const id = useId().replace(/:/g, "");
  const zodiac = zodiacForDate(date);
  const milkyGradient = `milky-gradient-${id}`;
  const milkyBlur = `milky-blur-${id}`;

  return <div className="procedural-sky" aria-hidden="true">
    <svg className="procedural-night-sky" viewBox="0 0 1000 700" preserveAspectRatio="xMidYMid slice">
      <defs>
        <linearGradient id={milkyGradient} x1="0" y1="1" x2="1" y2="0">
          <stop offset="0" stopColor="#e8d8c3" stopOpacity="0" />
          <stop offset=".22" stopColor="#d8d9e9" stopOpacity=".42" />
          <stop offset=".58" stopColor="#d7e1ee" stopOpacity=".31" />
          <stop offset="1" stopColor="#aebed8" stopOpacity="0" />
        </linearGradient>
        <filter id={milkyBlur} x="-25%" y="-25%" width="150%" height="150%"><feGaussianBlur stdDeviation="18" /></filter>
      </defs>
      <g className="sky-milky-way" filter={`url(#${milkyBlur})`}>
        <path className="sky-milky-way-haze" d="M -100 690 C 210 520 615 285 1100 -40" stroke={`url(#${milkyGradient})`} />
        <path className="sky-milky-way-core" d="M -100 690 C 210 520 615 285 1100 -40" stroke={`url(#${milkyGradient})`} />
        <path className="sky-milky-way-rift" d="M -100 675 C 225 505 620 300 1100 -32" />
      </g>
      <g className="sky-stars">
        {STARS.map((star, index) => <circle className={`sky-star sky-star-${star.color}${star.radius > 1.45 ? " sky-star-bright" : ""}`} key={index} cx={star.x} cy={star.y} r={star.radius} opacity={Math.min(star.opacity, 1)} />)}
      </g>
      {showConstellation && <ConstellationFigure zodiac={zodiac} />}
    </svg>
    <span className="scene-comet" />
    <span className="scene-balloon">
      <svg viewBox="0 0 32 48" focusable="false">
        <path className="scene-balloon-envelope" d="M16 2C7.8 2 3 8.8 3 16.2c0 8.6 6.1 14.2 10.3 18h5.4C22.9 30.4 29 24.8 29 16.2 29 8.8 24.2 2 16 2Z" />
        <path className="scene-balloon-panel" d="M16 3.5c-3.5 4.8-4.4 16.1-1.6 29.7h3.2C20.4 19.6 19.5 8.3 16 3.5Z" />
        <path className="scene-balloon-rope" d="m12.8 34 1.1 6m5.3-6-1.1 6" />
        <path className="scene-balloon-basket" d="M12.5 40h7l-1 5h-5Z" />
      </svg>
    </span>
  </div>;
}
