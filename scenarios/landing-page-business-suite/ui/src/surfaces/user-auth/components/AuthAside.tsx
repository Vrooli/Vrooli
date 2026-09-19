import type { ReactNode } from 'react';

export interface AuthAsidePoint {
  icon: ReactNode;
  title: string;
  body: string;
}

interface AuthAsideProps {
  eyebrow: string;
  title: ReactNode;
  lede: string;
  points: AuthAsidePoint[];
}

/** The quiet half of the sign-in screen: what happens, and why it is safe. */
export function AuthAside({ eyebrow, title, lede, points }: AuthAsideProps) {
  return (
    <aside className="auth-aside" aria-label="About signing in">
      <SignInConstellation />
      <p className="eyebrow auth-eyebrow">{eyebrow}</p>
      <h2 className="auth-aside-title">{title}</h2>
      <p className="auth-aside-lede">{lede}</p>
      <ul className="auth-points">
        {points.map((point) => (
          <li key={point.title}>
            <span className="auth-point-icon" aria-hidden="true">{point.icon}</span>
            <span>
              <strong>{point.title}</strong>
              <span>{point.body}</span>
            </span>
          </li>
        ))}
      </ul>
    </aside>
  );
}

/**
 * Three stars for the three moments of sign-in (address, code, inside),
 * joined the way the landing page draws its constellations.
 */
function SignInConstellation() {
  return (
    <svg className="auth-constellation" viewBox="0 0 320 120" role="presentation" aria-hidden="true" focusable="false">
      <defs>
        <radialGradient id="auth-star-glow">
          <stop offset="0" stopColor="#22d3ee" stopOpacity=".55" />
          <stop offset="1" stopColor="#22d3ee" stopOpacity="0" />
        </radialGradient>
        <linearGradient id="auth-line" x1="0" x2="1">
          <stop offset="0" stopColor="#22d3ee" stopOpacity=".15" />
          <stop offset=".5" stopColor="#22d3ee" stopOpacity=".7" />
          <stop offset="1" stopColor="#22d3ee" stopOpacity=".15" />
        </linearGradient>
      </defs>
      <path className="auth-constellation-path" d="M28 84 L124 38 L204 74 L292 28" fill="none" stroke="url(#auth-line)" strokeWidth="1.4" />
      {[[28, 84], [124, 38], [204, 74], [292, 28]].map(([x, y], index) => (
        <g key={index} className="auth-constellation-star" style={{ animationDelay: `${String(index * 0.45)}s` }}>
          <circle cx={x} cy={y} r="16" fill="url(#auth-star-glow)" />
          <circle cx={x} cy={y} r={index === 3 ? 4 : 3} fill="#e6fbff" />
        </g>
      ))}
      <circle cx="70" cy="22" r="1" fill="#9db0c8" opacity=".6" />
      <circle cx="250" cy="104" r="1.2" fill="#9db0c8" opacity=".5" />
      <circle cx="170" cy="14" r=".9" fill="#9db0c8" opacity=".5" />
    </svg>
  );
}
