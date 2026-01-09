import type { ReplayFrame } from '@/domains/exports/replay/ReplayPlayer';

const DEMO_FRAME_1_SVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1920 1080">
  <defs>
    <linearGradient id="heroGrad" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" style="stop-color:#1e293b"/>
      <stop offset="100%" style="stop-color:#0f172a"/>
    </linearGradient>
    <linearGradient id="btnGrad" x1="0%" y1="0%" x2="100%" y2="0%">
      <stop offset="0%" style="stop-color:#3b82f6"/>
      <stop offset="100%" style="stop-color:#6366f1"/>
    </linearGradient>
  </defs>
  <rect fill="url(#heroGrad)" width="1920" height="1080"/>
  <circle cx="1600" cy="200" r="300" fill="#3b82f6" opacity="0.05"/>
  <circle cx="200" cy="800" r="250" fill="#6366f1" opacity="0.05"/>
  <rect fill="#0f172a" width="1920" height="72" opacity="0.9"/>
  <rect x="80" y="22" width="120" height="28" rx="6" fill="#3b82f6"/>
  <text x="140" y="43" font-family="system-ui" font-size="16" fill="white" text-anchor="middle" font-weight="bold">AutoFlow</text>
  <text x="300" y="43" font-family="system-ui" font-size="14" fill="#94a3b8">Products</text>
  <text x="420" y="43" font-family="system-ui" font-size="14" fill="#94a3b8">Solutions</text>
  <text x="540" y="43" font-family="system-ui" font-size="14" fill="#94a3b8">Pricing</text>
  <text x="660" y="43" font-family="system-ui" font-size="14" fill="#94a3b8">Docs</text>
  <rect x="1680" y="22" width="160" height="40" rx="8" fill="url(#btnGrad)"/>
  <text x="1760" y="48" font-family="system-ui" font-size="14" fill="white" text-anchor="middle" font-weight="600">Get Started</text>
  <text x="960" y="320" font-family="system-ui" font-size="64" fill="white" text-anchor="middle" font-weight="bold">Automate Your Workflow</text>
  <text x="960" y="390" font-family="system-ui" font-size="24" fill="#94a3b8" text-anchor="middle">Build, test, and deploy browser automations in minutes</text>
  <rect x="780" y="450" width="180" height="56" rx="12" fill="url(#btnGrad)"/>
  <text x="870" y="486" font-family="system-ui" font-size="18" fill="white" text-anchor="middle" font-weight="600">Start Free</text>
  <rect x="980" y="450" width="180" height="56" rx="12" fill="none" stroke="#475569" stroke-width="2"/>
  <text x="1070" y="486" font-family="system-ui" font-size="18" fill="#94a3b8" text-anchor="middle">Watch Demo</text>
</svg>`;

const DEMO_FRAME_2_SVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1920 1080">
  <defs>
    <linearGradient id="bgGrad2" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" style="stop-color:#0f172a"/>
      <stop offset="100%" style="stop-color:#1e1b4b"/>
    </linearGradient>
  </defs>
  <rect fill="url(#bgGrad2)" width="1920" height="1080"/>
  <circle cx="200" cy="200" r="400" fill="#3b82f6" opacity="0.03"/>
  <circle cx="1700" cy="900" r="350" fill="#6366f1" opacity="0.03"/>
  <rect fill="#0f172a" width="1920" height="72" opacity="0.95"/>
  <rect x="80" y="22" width="120" height="28" rx="6" fill="#3b82f6"/>
  <text x="140" y="43" font-family="system-ui" font-size="16" fill="white" text-anchor="middle" font-weight="bold">AutoFlow</text>
  <rect x="560" y="160" width="800" height="760" rx="24" fill="#1e293b" stroke="#334155" stroke-width="1"/>
  <text x="960" y="240" font-family="system-ui" font-size="36" fill="white" text-anchor="middle" font-weight="bold">Create your account</text>
  <text x="960" y="285" font-family="system-ui" font-size="16" fill="#64748b" text-anchor="middle">Start automating in less than 2 minutes</text>
  <text x="640" y="480" font-family="system-ui" font-size="14" fill="#94a3b8">Full Name</text>
  <rect x="640" y="495" width="640" height="52" rx="10" fill="#0f172a" stroke="#334155"/>
  <text x="640" y="585" font-family="system-ui" font-size="14" fill="#94a3b8">Email Address</text>
  <rect x="640" y="600" width="640" height="52" rx="10" fill="#0f172a" stroke="#3b82f6" stroke-width="2"/>
  <rect x="640" y="810" width="640" height="56" rx="12" fill="#3b82f6"/>
  <text x="960" y="846" font-family="system-ui" font-size="18" fill="white" text-anchor="middle" font-weight="600">Create Account</text>
</svg>`;

const DEMO_FRAME_3_SVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1920 1080">
  <defs>
    <linearGradient id="bgGrad3" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" style="stop-color:#0f172a"/>
      <stop offset="100%" style="stop-color:#042f2e"/>
    </linearGradient>
    <linearGradient id="successGrad" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" style="stop-color:#22c55e"/>
      <stop offset="100%" style="stop-color:#16a34a"/>
    </linearGradient>
  </defs>
  <rect fill="url(#bgGrad3)" width="1920" height="1080"/>
  <circle cx="960" cy="540" r="500" fill="#22c55e" opacity="0.02"/>
  <circle cx="960" cy="540" r="350" fill="#22c55e" opacity="0.03"/>
  <rect fill="#0f172a" width="1920" height="72" opacity="0.95"/>
  <rect x="80" y="22" width="120" height="28" rx="6" fill="#22c55e"/>
  <text x="140" y="43" font-family="system-ui" font-size="16" fill="white" text-anchor="middle" font-weight="bold">AutoFlow</text>
  <circle cx="960" cy="320" r="80" fill="url(#successGrad)"/>
  <path d="M920 320 L945 345 L1005 285" stroke="white" stroke-width="8" fill="none" stroke-linecap="round" stroke-linejoin="round"/>
  <text x="960" y="460" font-family="system-ui" font-size="42" fill="white" text-anchor="middle" font-weight="bold">Account Created Successfully!</text>
  <text x="960" y="510" font-family="system-ui" font-size="18" fill="#94a3b8" text-anchor="middle">Welcome to AutoFlow</text>
</svg>`;

export function createDemoFrames(frameDuration: number): ReplayFrame[] {
  return [
    {
      id: 'demo-1',
      stepIndex: 0,
      stepType: 'navigate',
      status: 'completed',
      success: true,
      durationMs: frameDuration,
      finalUrl: 'https://autoflow.app',
      screenshot: {
        artifactId: 'demo-screenshot-1',
        url: `data:image/svg+xml,${encodeURIComponent(DEMO_FRAME_1_SVG)}`,
        width: 1920,
        height: 1080,
      },
      clickPosition: { x: 1760, y: 42 },
    },
    {
      id: 'demo-2',
      stepIndex: 1,
      stepType: 'click',
      status: 'completed',
      success: true,
      durationMs: frameDuration,
      finalUrl: 'https://autoflow.app/signup',
      screenshot: {
        artifactId: 'demo-screenshot-2',
        url: `data:image/svg+xml,${encodeURIComponent(DEMO_FRAME_2_SVG)}`,
        width: 1920,
        height: 1080,
      },
      clickPosition: { x: 960, y: 626 },
    },
    {
      id: 'demo-3',
      stepIndex: 2,
      stepType: 'type',
      status: 'completed',
      success: true,
      durationMs: frameDuration,
      finalUrl: 'https://autoflow.app/welcome',
      screenshot: {
        artifactId: 'demo-screenshot-3',
        url: `data:image/svg+xml,${encodeURIComponent(DEMO_FRAME_3_SVG)}`,
        width: 1920,
        height: 1080,
      },
      assertion: {
        mode: 'exists',
        selector: '.success-message',
        success: true,
        message: 'Account creation successful',
      },
    },
  ];
}
