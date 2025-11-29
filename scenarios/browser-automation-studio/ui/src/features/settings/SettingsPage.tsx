import { useMemo, useState, useCallback, lazy, Suspense } from 'react';
import {
  ArrowLeft,
  Settings,
  Film,
  RotateCcw,
  ChevronDown,
  ChevronRight,
  HelpCircle,
  Play,
  Pause,
} from 'lucide-react';
import { useSettingsStore } from '@stores/settingsStore';
import {
  REPLAY_CHROME_OPTIONS,
  REPLAY_BACKGROUND_OPTIONS,
  BACKGROUND_GROUP_ORDER,
  REPLAY_CURSOR_OPTIONS,
  CURSOR_GROUP_ORDER,
  REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS,
  REPLAY_CURSOR_POSITIONS,
  CURSOR_SCALE_MIN,
  CURSOR_SCALE_MAX,
} from '../execution/replay/replayThemeOptions';
import type {
  CursorSpeedProfile,
  CursorPathStyle,
  ReplayFrame,
} from '../execution/ReplayPlayer';
import Tooltip from '@shared/ui/Tooltip';

const ReplayPlayer = lazy(() => import('../execution/ReplayPlayer'));

const SPEED_PROFILE_OPTIONS: Array<{ id: CursorSpeedProfile; label: string; description: string }> = [
  { id: 'linear', label: 'Linear', description: 'Consistent motion between frames' },
  { id: 'easeIn', label: 'Ease In', description: 'Begin slowly, accelerate toward target' },
  { id: 'easeOut', label: 'Ease Out', description: 'Move quickly, settle into target' },
  { id: 'easeInOut', label: 'Ease In/Out', description: 'Smooth acceleration and deceleration' },
  { id: 'instant', label: 'Instant', description: 'Jump directly at the end' },
];

const PATH_STYLE_OPTIONS: Array<{ id: CursorPathStyle; label: string; description: string }> = [
  { id: 'linear', label: 'Linear', description: 'Direct line between positions' },
  { id: 'parabolicUp', label: 'Arc Up', description: 'Arcs upward before target' },
  { id: 'parabolicDown', label: 'Arc Down', description: 'Arcs downward before target' },
  { id: 'cubic', label: 'Cubic', description: 'Smooth S-curve motion' },
  { id: 'pseudorandom', label: 'Organic', description: 'Natural random waypoints' },
];

interface SettingsPageProps {
  onBack: () => void;
}

// Demo frames for the live preview
// Demo frame 1: Modern landing page with navigation, hero section, and feature cards
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
  <!-- Background -->
  <rect fill="url(#heroGrad)" width="1920" height="1080"/>
  <!-- Decorative circles -->
  <circle cx="1600" cy="200" r="300" fill="#3b82f6" opacity="0.05"/>
  <circle cx="200" cy="800" r="250" fill="#6366f1" opacity="0.05"/>
  <!-- Navigation bar -->
  <rect fill="#0f172a" width="1920" height="72" opacity="0.9"/>
  <rect x="80" y="22" width="120" height="28" rx="6" fill="#3b82f6"/>
  <text x="140" y="43" font-family="system-ui" font-size="16" fill="white" text-anchor="middle" font-weight="bold">AutoFlow</text>
  <text x="300" y="43" font-family="system-ui" font-size="14" fill="#94a3b8">Products</text>
  <text x="420" y="43" font-family="system-ui" font-size="14" fill="#94a3b8">Solutions</text>
  <text x="540" y="43" font-family="system-ui" font-size="14" fill="#94a3b8">Pricing</text>
  <text x="660" y="43" font-family="system-ui" font-size="14" fill="#94a3b8">Docs</text>
  <rect x="1680" y="22" width="160" height="40" rx="8" fill="url(#btnGrad)"/>
  <text x="1760" y="48" font-family="system-ui" font-size="14" fill="white" text-anchor="middle" font-weight="600">Get Started</text>
  <!-- Hero section -->
  <text x="960" y="320" font-family="system-ui" font-size="64" fill="white" text-anchor="middle" font-weight="bold">Automate Your Workflow</text>
  <text x="960" y="390" font-family="system-ui" font-size="24" fill="#94a3b8" text-anchor="middle">Build, test, and deploy browser automations in minutes</text>
  <!-- CTA buttons -->
  <rect x="780" y="450" width="180" height="56" rx="12" fill="url(#btnGrad)"/>
  <text x="870" y="486" font-family="system-ui" font-size="18" fill="white" text-anchor="middle" font-weight="600">Start Free</text>
  <rect x="980" y="450" width="180" height="56" rx="12" fill="none" stroke="#475569" stroke-width="2"/>
  <text x="1070" y="486" font-family="system-ui" font-size="18" fill="#94a3b8" text-anchor="middle">Watch Demo</text>
  <!-- Feature cards -->
  <rect x="240" y="600" width="440" height="200" rx="16" fill="#1e293b" stroke="#334155" stroke-width="1"/>
  <circle cx="300" cy="660" r="24" fill="#3b82f6" opacity="0.2"/>
  <text x="300" y="668" font-family="system-ui" font-size="20" fill="#3b82f6" text-anchor="middle">⚡</text>
  <text x="360" y="668" font-family="system-ui" font-size="20" fill="white" font-weight="600">Fast Execution</text>
  <text x="280" y="720" font-family="system-ui" font-size="14" fill="#64748b">Run automations 10x faster with</text>
  <text x="280" y="745" font-family="system-ui" font-size="14" fill="#64748b">our optimized browser engine</text>
  <rect x="740" y="600" width="440" height="200" rx="16" fill="#1e293b" stroke="#334155" stroke-width="1"/>
  <circle cx="800" cy="660" r="24" fill="#22c55e" opacity="0.2"/>
  <text x="800" y="668" font-family="system-ui" font-size="20" fill="#22c55e" text-anchor="middle">✓</text>
  <text x="860" y="668" font-family="system-ui" font-size="20" fill="white" font-weight="600">Visual Builder</text>
  <text x="780" y="720" font-family="system-ui" font-size="14" fill="#64748b">No code required. Build workflows</text>
  <text x="780" y="745" font-family="system-ui" font-size="14" fill="#64748b">with drag-and-drop interface</text>
  <rect x="1240" y="600" width="440" height="200" rx="16" fill="#1e293b" stroke="#334155" stroke-width="1"/>
  <circle cx="1300" cy="660" r="24" fill="#a855f7" opacity="0.2"/>
  <text x="1300" y="668" font-family="system-ui" font-size="20" fill="#a855f7" text-anchor="middle">🔄</text>
  <text x="1360" y="668" font-family="system-ui" font-size="20" fill="white" font-weight="600">Smart Retries</text>
  <text x="1280" y="720" font-family="system-ui" font-size="14" fill="#64748b">Auto-healing selectors and</text>
  <text x="1280" y="745" font-family="system-ui" font-size="14" fill="#64748b">intelligent retry mechanisms</text>
  <!-- Stats bar -->
  <rect x="0" y="880" width="1920" height="120" fill="#0f172a"/>
  <text x="380" y="945" font-family="system-ui" font-size="36" fill="white" text-anchor="middle" font-weight="bold">10M+</text>
  <text x="380" y="980" font-family="system-ui" font-size="14" fill="#64748b" text-anchor="middle">Automations Run</text>
  <text x="760" y="945" font-family="system-ui" font-size="36" fill="white" text-anchor="middle" font-weight="bold">99.9%</text>
  <text x="760" y="980" font-family="system-ui" font-size="14" fill="#64748b" text-anchor="middle">Uptime</text>
  <text x="1140" y="945" font-family="system-ui" font-size="36" fill="white" text-anchor="middle" font-weight="bold">50K+</text>
  <text x="1140" y="980" font-family="system-ui" font-size="14" fill="#64748b" text-anchor="middle">Active Users</text>
  <text x="1520" y="945" font-family="system-ui" font-size="36" fill="white" text-anchor="middle" font-weight="bold">4.9★</text>
  <text x="1520" y="980" font-family="system-ui" font-size="14" fill="#64748b" text-anchor="middle">User Rating</text>
</svg>`;

// Demo frame 2: Signup form with email input focused
const DEMO_FRAME_2_SVG = `<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1920 1080">
  <defs>
    <linearGradient id="bgGrad2" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" style="stop-color:#0f172a"/>
      <stop offset="100%" style="stop-color:#1e1b4b"/>
    </linearGradient>
    <linearGradient id="cardGrad" x1="0%" y1="0%" x2="0%" y2="100%">
      <stop offset="0%" style="stop-color:#1e293b"/>
      <stop offset="100%" style="stop-color:#0f172a"/>
    </linearGradient>
  </defs>
  <!-- Background -->
  <rect fill="url(#bgGrad2)" width="1920" height="1080"/>
  <!-- Decorative elements -->
  <circle cx="200" cy="200" r="400" fill="#3b82f6" opacity="0.03"/>
  <circle cx="1700" cy="900" r="350" fill="#6366f1" opacity="0.03"/>
  <!-- Navigation -->
  <rect fill="#0f172a" width="1920" height="72" opacity="0.95"/>
  <rect x="80" y="22" width="120" height="28" rx="6" fill="#3b82f6"/>
  <text x="140" y="43" font-family="system-ui" font-size="16" fill="white" text-anchor="middle" font-weight="bold">AutoFlow</text>
  <text x="1760" y="43" font-family="system-ui" font-size="14" fill="#94a3b8" text-anchor="middle">Already have an account? Sign in</text>
  <!-- Main card container -->
  <rect x="560" y="160" width="800" height="760" rx="24" fill="url(#cardGrad)" stroke="#334155" stroke-width="1"/>
  <!-- Card header -->
  <text x="960" y="240" font-family="system-ui" font-size="36" fill="white" text-anchor="middle" font-weight="bold">Create your account</text>
  <text x="960" y="285" font-family="system-ui" font-size="16" fill="#64748b" text-anchor="middle">Start automating in less than 2 minutes</text>
  <!-- Social login buttons -->
  <rect x="640" y="330" width="280" height="50" rx="10" fill="#1e293b" stroke="#334155"/>
  <text x="780" y="362" font-family="system-ui" font-size="14" fill="white" text-anchor="middle">Continue with Google</text>
  <rect x="1000" y="330" width="280" height="50" rx="10" fill="#1e293b" stroke="#334155"/>
  <text x="1140" y="362" font-family="system-ui" font-size="14" fill="white" text-anchor="middle">Continue with GitHub</text>
  <!-- Divider -->
  <line x1="640" y1="420" x2="880" y2="420" stroke="#334155" stroke-width="1"/>
  <text x="960" y="425" font-family="system-ui" font-size="14" fill="#64748b" text-anchor="middle">or</text>
  <line x1="1040" y1="420" x2="1280" y2="420" stroke="#334155" stroke-width="1"/>
  <!-- Form fields -->
  <text x="640" y="480" font-family="system-ui" font-size="14" fill="#94a3b8">Full Name</text>
  <rect x="640" y="495" width="640" height="52" rx="10" fill="#0f172a" stroke="#334155"/>
  <text x="660" y="528" font-family="system-ui" font-size="16" fill="#64748b">Enter your name</text>
  <!-- Email field - highlighted as active -->
  <text x="640" y="585" font-family="system-ui" font-size="14" fill="#94a3b8">Email Address</text>
  <rect x="640" y="600" width="640" height="52" rx="10" fill="#0f172a" stroke="#3b82f6" stroke-width="2"/>
  <rect x="640" y="600" width="640" height="52" rx="10" fill="#3b82f6" opacity="0.05"/>
  <text x="660" y="633" font-family="system-ui" font-size="16" fill="white">|</text>
  <text x="640" y="670" font-family="system-ui" font-size="12" fill="#3b82f6">We'll send you a verification link</text>
  <!-- Password field -->
  <text x="640" y="710" font-family="system-ui" font-size="14" fill="#94a3b8">Password</text>
  <rect x="640" y="725" width="640" height="52" rx="10" fill="#0f172a" stroke="#334155"/>
  <text x="660" y="758" font-family="system-ui" font-size="16" fill="#64748b">Create a strong password</text>
  <!-- Submit button -->
  <rect x="640" y="810" width="640" height="56" rx="12" fill="#3b82f6"/>
  <text x="960" y="846" font-family="system-ui" font-size="18" fill="white" text-anchor="middle" font-weight="600">Create Account</text>
  <!-- Terms -->
  <text x="960" y="895" font-family="system-ui" font-size="12" fill="#64748b" text-anchor="middle">By signing up, you agree to our Terms of Service and Privacy Policy</text>
</svg>`;

// Demo frame 3: Success state with confirmation
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
  <!-- Background -->
  <rect fill="url(#bgGrad3)" width="1920" height="1080"/>
  <!-- Decorative elements -->
  <circle cx="960" cy="540" r="500" fill="#22c55e" opacity="0.02"/>
  <circle cx="960" cy="540" r="350" fill="#22c55e" opacity="0.03"/>
  <!-- Navigation -->
  <rect fill="#0f172a" width="1920" height="72" opacity="0.95"/>
  <rect x="80" y="22" width="120" height="28" rx="6" fill="#22c55e"/>
  <text x="140" y="43" font-family="system-ui" font-size="16" fill="white" text-anchor="middle" font-weight="bold">AutoFlow</text>
  <!-- Success icon -->
  <circle cx="960" cy="320" r="80" fill="url(#successGrad)"/>
  <path d="M920 320 L945 345 L1005 285" stroke="white" stroke-width="8" fill="none" stroke-linecap="round" stroke-linejoin="round"/>
  <!-- Success message -->
  <text x="960" y="460" font-family="system-ui" font-size="42" fill="white" text-anchor="middle" font-weight="bold">Account Created Successfully!</text>
  <text x="960" y="510" font-family="system-ui" font-size="18" fill="#94a3b8" text-anchor="middle">Welcome to AutoFlow. We've sent a verification email to</text>
  <text x="960" y="545" font-family="system-ui" font-size="18" fill="#22c55e" text-anchor="middle" font-weight="600">user@example.com</text>
  <!-- Action card -->
  <rect x="610" y="600" width="700" height="180" rx="20" fill="#1e293b" stroke="#22c55e" stroke-width="2" opacity="0.8"/>
  <text x="960" y="660" font-family="system-ui" font-size="20" fill="white" text-anchor="middle" font-weight="600">What's Next?</text>
  <text x="960" y="700" font-family="system-ui" font-size="16" fill="#94a3b8" text-anchor="middle">Check your email and click the verification link to get started.</text>
  <rect x="810" y="730" width="300" height="44" rx="10" fill="url(#successGrad)"/>
  <text x="960" y="759" font-family="system-ui" font-size="16" fill="white" text-anchor="middle" font-weight="600">Go to Dashboard →</text>
  <!-- Progress indicators -->
  <g transform="translate(660, 850)">
    <circle cx="0" cy="0" r="16" fill="#22c55e"/>
    <text x="0" y="5" font-family="system-ui" font-size="12" fill="white" text-anchor="middle" font-weight="bold">1</text>
    <line x1="16" y1="0" x2="200" y2="0" stroke="#22c55e" stroke-width="3"/>
    <circle cx="200" cy="0" r="16" fill="#22c55e"/>
    <text x="200" y="5" font-family="system-ui" font-size="12" fill="white" text-anchor="middle" font-weight="bold">2</text>
    <line x1="216" y1="0" x2="400" y2="0" stroke="#22c55e" stroke-width="3"/>
    <circle cx="400" cy="0" r="16" fill="#22c55e"/>
    <text x="400" y="5" font-family="system-ui" font-size="12" fill="white" text-anchor="middle" font-weight="bold">3</text>
    <line x1="416" y1="0" x2="600" y2="0" stroke="#334155" stroke-width="3" stroke-dasharray="8,4"/>
    <circle cx="600" cy="0" r="16" fill="#1e293b" stroke="#334155" stroke-width="2"/>
    <text x="600" y="5" font-family="system-ui" font-size="12" fill="#64748b" text-anchor="middle">4</text>
    <text x="0" y="40" font-family="system-ui" font-size="12" fill="#94a3b8" text-anchor="middle">Sign Up</text>
    <text x="200" y="40" font-family="system-ui" font-size="12" fill="#94a3b8" text-anchor="middle">Verify</text>
    <text x="400" y="40" font-family="system-ui" font-size="12" fill="#22c55e" text-anchor="middle">Complete</text>
    <text x="600" y="40" font-family="system-ui" font-size="12" fill="#64748b" text-anchor="middle">Start</text>
  </g>
  <!-- Confetti dots -->
  <circle cx="300" cy="300" r="6" fill="#22c55e" opacity="0.6"/>
  <circle cx="1600" cy="400" r="8" fill="#3b82f6" opacity="0.5"/>
  <circle cx="400" cy="700" r="5" fill="#a855f7" opacity="0.4"/>
  <circle cx="1500" cy="250" r="7" fill="#f59e0b" opacity="0.5"/>
  <circle cx="250" cy="500" r="4" fill="#22c55e" opacity="0.3"/>
  <circle cx="1650" cy="650" r="6" fill="#ec4899" opacity="0.4"/>
</svg>`;

const createDemoFrames = (): ReplayFrame[] => [
  {
    id: 'demo-1',
    stepIndex: 0,
    stepType: 'navigate',
    status: 'completed',
    success: true,
    durationMs: 1200,
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
    durationMs: 800,
    finalUrl: 'https://autoflow.app/signup',
    screenshot: {
      artifactId: 'demo-screenshot-2',
      url: `data:image/svg+xml,${encodeURIComponent(DEMO_FRAME_2_SVG)}`,
      width: 1920,
      height: 1080,
    },
    clickPosition: { x: 960, y: 626 },
    highlightRegions: [
      { boundingBox: { x: 640, y: 600, width: 640, height: 52 }, color: 'rgba(59,130,246,0.25)' },
    ],
  },
  {
    id: 'demo-3',
    stepIndex: 2,
    stepType: 'type',
    status: 'completed',
    success: true,
    durationMs: 1500,
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

interface SettingSectionProps {
  title: string;
  tooltip?: string;
  defaultOpen?: boolean;
  children: React.ReactNode;
}

function SettingSection({ title, tooltip, defaultOpen = true, children }: SettingSectionProps) {
  const [isOpen, setIsOpen] = useState(defaultOpen);

  return (
    <div className="border border-gray-700 rounded-lg overflow-hidden">
      <button
        type="button"
        onClick={() => setIsOpen(!isOpen)}
        className="w-full flex items-center justify-between px-4 py-3 bg-gray-800/50 hover:bg-gray-800 transition-colors text-left"
      >
        <div className="flex items-center gap-2">
          <span className="font-medium text-white">{title}</span>
          {tooltip && (
            <Tooltip content={tooltip}>
              <HelpCircle size={14} className="text-gray-500" />
            </Tooltip>
          )}
        </div>
        {isOpen ? <ChevronDown size={16} className="text-gray-400" /> : <ChevronRight size={16} className="text-gray-400" />}
      </button>
      {isOpen && <div className="p-4 bg-gray-900/50 space-y-4">{children}</div>}
    </div>
  );
}

interface OptionGridProps<T extends string> {
  options: Array<{ id: T; label: string; subtitle?: string; description?: string; preview?: React.ReactNode; previewStyle?: React.CSSProperties; previewNode?: React.ReactNode }>;
  value: T;
  onChange: (value: T) => void;
  columns?: 2 | 3 | 4;
}

function OptionGrid<T extends string>({ options, value, onChange, columns = 3 }: OptionGridProps<T>) {
  const gridCols = columns === 2 ? 'grid-cols-2' : columns === 4 ? 'grid-cols-2 sm:grid-cols-4' : 'grid-cols-2 sm:grid-cols-3';

  return (
    <div className={`grid ${gridCols} gap-2`}>
      {options.map((option) => (
        <button
          key={option.id}
          type="button"
          onClick={() => onChange(option.id)}
          className={`flex flex-col items-center p-3 rounded-lg border transition-all text-center ${
            value === option.id
              ? 'border-flow-accent bg-flow-accent/10 ring-1 ring-flow-accent/50'
              : 'border-gray-700 hover:border-gray-600 bg-gray-800/30'
          }`}
        >
          {option.preview && <div className="mb-2">{option.preview}</div>}
          {option.previewNode && <div className="relative w-10 h-10 mb-2 rounded-lg overflow-hidden" style={option.previewStyle}>{option.previewNode}</div>}
          {!option.preview && !option.previewNode && option.previewStyle && (
            <div className="w-10 h-10 mb-2 rounded-lg" style={option.previewStyle} />
          )}
          <span className="text-sm font-medium text-white">{option.label}</span>
          {(option.subtitle || option.description) && (
            <span className="text-xs text-gray-500 mt-0.5">{option.subtitle || option.description}</span>
          )}
        </button>
      ))}
    </div>
  );
}

function SettingsPage({ onBack }: SettingsPageProps) {
  const { replay, setReplaySetting, resetReplaySettings } = useSettingsStore();
  const [isPreviewPlaying, setIsPreviewPlaying] = useState(true);

  const demoFrames = useMemo(() => createDemoFrames(), []);

  const handleReset = useCallback(() => {
    if (window.confirm('Reset all replay settings to defaults?')) {
      resetReplaySettings();
    }
  }, [resetReplaySettings]);

  // Group backgrounds by kind
  const groupedBackgrounds = useMemo(() => {
    return BACKGROUND_GROUP_ORDER.map((group) => ({
      ...group,
      options: REPLAY_BACKGROUND_OPTIONS.filter((bg) => bg.kind === group.id),
    }));
  }, []);

  // Group cursors
  const groupedCursors = useMemo(() => {
    return CURSOR_GROUP_ORDER.map((group) => ({
      ...group,
      options: REPLAY_CURSOR_OPTIONS.filter((c) => c.group === group.id),
    }));
  }, []);

  return (
    <div className="flex flex-col h-screen bg-flow-bg">
      {/* Header */}
      <header className="sticky top-0 z-30 border-b border-gray-800 bg-flow-bg/95 backdrop-blur">
        <div className="px-4 py-4 sm:px-6">
          <div className="flex items-center justify-between">
            <div className="flex items-center gap-4">
              <button
                onClick={onBack}
                className="p-2 text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
                title="Back to Dashboard"
                aria-label="Back to Dashboard"
              >
                <ArrowLeft size={20} />
              </button>
              <div className="flex items-center gap-3">
                <div className="p-2 bg-gray-800 rounded-lg">
                  <Settings size={20} className="text-flow-accent" />
                </div>
                <div>
                  <h1 className="text-xl font-bold text-white">Settings</h1>
                  <p className="text-sm text-gray-400">Configure your replay preferences</p>
                </div>
              </div>
            </div>
            <button
              onClick={handleReset}
              className="flex items-center gap-2 px-3 py-2 text-sm text-gray-400 hover:text-white hover:bg-gray-700 rounded-lg transition-colors"
              title="Reset to defaults"
            >
              <RotateCcw size={16} />
              <span className="hidden sm:inline">Reset</span>
            </button>
          </div>
        </div>
      </header>

      {/* Content */}
      <div className="flex-1 overflow-hidden flex flex-col lg:flex-row">
        {/* Settings Panel */}
        <div className="flex-1 overflow-y-auto p-4 sm:p-6 lg:max-w-2xl">
          <div className="space-y-4">
            {/* Replay Settings Header */}
            <div className="flex items-center gap-3 mb-6">
              <Film size={24} className="text-flow-accent" />
              <div>
                <h2 className="text-lg font-semibold text-white">Replay Appearance</h2>
                <p className="text-sm text-gray-400">Customize how workflow replays look and behave</p>
              </div>
            </div>

            {/* Chrome Theme */}
            <SettingSection title="Browser Chrome" tooltip="The browser window frame style surrounding the screenshot">
              <OptionGrid
                options={REPLAY_CHROME_OPTIONS}
                value={replay.chromeTheme}
                onChange={(v) => setReplaySetting('chromeTheme', v)}
                columns={4}
              />
            </SettingSection>

            {/* Background Theme */}
            <SettingSection title="Background">
              <div className="space-y-4">
                {groupedBackgrounds.map((group) => (
                  <div key={group.id}>
                    <div className="text-xs font-medium text-gray-500 uppercase tracking-wide mb-2">
                      {group.label}
                    </div>
                    <OptionGrid
                      options={group.options}
                      value={replay.backgroundTheme}
                      onChange={(v) => setReplaySetting('backgroundTheme', v)}
                      columns={4}
                    />
                  </div>
                ))}
              </div>
            </SettingSection>

            {/* Cursor Theme */}
            <SettingSection title="Cursor Style">
              <div className="space-y-4">
                {groupedCursors.map((group) => (
                  <div key={group.id}>
                    <div className="text-xs font-medium text-gray-500 uppercase tracking-wide mb-2">
                      {group.label}
                    </div>
                    <OptionGrid
                      options={group.options}
                      value={replay.cursorTheme}
                      onChange={(v) => setReplaySetting('cursorTheme', v)}
                      columns={3}
                    />
                  </div>
                ))}
              </div>
            </SettingSection>

            {/* Cursor Scale */}
            <SettingSection title="Cursor Size" tooltip="Adjust the size of the virtual cursor">
              <div className="flex items-center gap-4">
                <input
                  type="range"
                  min={CURSOR_SCALE_MIN}
                  max={CURSOR_SCALE_MAX}
                  step={0.1}
                  value={replay.cursorScale}
                  onChange={(e) => setReplaySetting('cursorScale', parseFloat(e.target.value))}
                  className="flex-1 accent-flow-accent"
                />
                <span className="text-sm text-gray-400 w-12 text-right">
                  {(replay.cursorScale * 100).toFixed(0)}%
                </span>
              </div>
            </SettingSection>

            {/* Click Animation */}
            <SettingSection title="Click Animation" tooltip="Visual feedback when the cursor clicks">
              <OptionGrid
                options={REPLAY_CURSOR_CLICK_ANIMATION_OPTIONS}
                value={replay.cursorClickAnimation}
                onChange={(v) => setReplaySetting('cursorClickAnimation', v)}
                columns={3}
              />
            </SettingSection>

            {/* Cursor Position */}
            <SettingSection title="Initial Cursor Position" tooltip="Where the cursor starts at the beginning of a replay" defaultOpen={false}>
              <OptionGrid
                options={REPLAY_CURSOR_POSITIONS}
                value={replay.cursorInitialPosition}
                onChange={(v) => setReplaySetting('cursorInitialPosition', v)}
                columns={3}
              />
            </SettingSection>

            {/* Motion Settings */}
            <SettingSection title="Cursor Motion" tooltip="How the cursor moves between positions" defaultOpen={false}>
              <div className="space-y-4">
                <div>
                  <label className="text-sm text-gray-400 mb-2 block">Speed Profile</label>
                  <OptionGrid
                    options={SPEED_PROFILE_OPTIONS}
                    value={replay.cursorSpeedProfile}
                    onChange={(v) => setReplaySetting('cursorSpeedProfile', v)}
                    columns={3}
                  />
                </div>
                <div>
                  <label className="text-sm text-gray-400 mb-2 block">Path Style</label>
                  <OptionGrid
                    options={PATH_STYLE_OPTIONS}
                    value={replay.cursorPathStyle}
                    onChange={(v) => setReplaySetting('cursorPathStyle', v)}
                    columns={3}
                  />
                </div>
              </div>
            </SettingSection>

            {/* Playback Settings */}
            <SettingSection title="Playback" tooltip="Default playback behavior" defaultOpen={false}>
              <div className="space-y-4">
                <div>
                  <label className="text-sm text-gray-400 mb-2 block">
                    Frame Duration: {(replay.frameDuration / 1000).toFixed(1)}s
                  </label>
                  <input
                    type="range"
                    min={800}
                    max={6000}
                    step={100}
                    value={replay.frameDuration}
                    onChange={(e) => setReplaySetting('frameDuration', parseInt(e.target.value, 10))}
                    className="w-full accent-flow-accent"
                  />
                  <div className="flex justify-between text-xs text-gray-500 mt-1">
                    <span>Fast (0.8s)</span>
                    <span>Slow (6s)</span>
                  </div>
                </div>
                <div className="flex items-center justify-between">
                  <label className="text-sm text-gray-300">Auto-play replays</label>
                  <button
                    type="button"
                    role="switch"
                    aria-checked={replay.autoPlay}
                    onClick={() => setReplaySetting('autoPlay', !replay.autoPlay)}
                    className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                      replay.autoPlay ? 'bg-flow-accent' : 'bg-gray-700'
                    }`}
                  >
                    <span
                      className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                        replay.autoPlay ? 'translate-x-6' : 'translate-x-1'
                      }`}
                    />
                  </button>
                </div>
                <div className="flex items-center justify-between">
                  <label className="text-sm text-gray-300">Loop replays</label>
                  <button
                    type="button"
                    role="switch"
                    aria-checked={replay.loop}
                    onClick={() => setReplaySetting('loop', !replay.loop)}
                    className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
                      replay.loop ? 'bg-flow-accent' : 'bg-gray-700'
                    }`}
                  >
                    <span
                      className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                        replay.loop ? 'translate-x-6' : 'translate-x-1'
                      }`}
                    />
                  </button>
                </div>
              </div>
            </SettingSection>
          </div>
        </div>

        {/* Live Preview */}
        <div className="lg:w-1/2 xl:w-3/5 border-t lg:border-t-0 lg:border-l border-gray-800 bg-gray-900/50 flex flex-col min-h-0 overflow-hidden">
          <div className="flex-shrink-0 flex items-center justify-between px-4 py-3 border-b border-gray-800">
            <div className="flex items-center gap-2">
              <Film size={16} className="text-gray-400" />
              <span className="text-sm font-medium text-white">Live Preview</span>
            </div>
            <button
              type="button"
              onClick={() => setIsPreviewPlaying(!isPreviewPlaying)}
              className="flex items-center gap-2 px-3 py-1.5 text-sm text-gray-400 hover:text-white bg-gray-800 hover:bg-gray-700 rounded-lg transition-colors"
            >
              {isPreviewPlaying ? <Pause size={14} /> : <Play size={14} />}
              {isPreviewPlaying ? 'Pause' : 'Play'}
            </button>
          </div>
          <div className="flex-1 min-h-0 p-4 flex items-center justify-center overflow-hidden">
            <div className="w-full h-full max-w-4xl flex items-center justify-center">
              <div className="w-full max-h-full aspect-video overflow-hidden">
                <Suspense
                  fallback={
                    <div className="w-full h-full flex items-center justify-center bg-gray-800 rounded-lg">
                      <span className="text-gray-400">Loading preview...</span>
                    </div>
                  }
                >
                  <ReplayPlayer
                    frames={demoFrames}
                    autoPlay={isPreviewPlaying}
                    loop={replay.loop}
                    chromeTheme={replay.chromeTheme}
                    backgroundTheme={replay.backgroundTheme}
                    cursorTheme={replay.cursorTheme}
                    cursorInitialPosition={replay.cursorInitialPosition}
                    cursorScale={replay.cursorScale}
                    cursorClickAnimation={replay.cursorClickAnimation}
                    cursorDefaultSpeedProfile={replay.cursorSpeedProfile}
                    cursorDefaultPathStyle={replay.cursorPathStyle}
                  />
                </Suspense>
              </div>
            </div>
          </div>
          <div className="flex-shrink-0 px-4 py-3 border-t border-gray-800 text-center">
            <p className="text-xs text-gray-500">
              Preview shows a sample 3-step workflow with your current settings
            </p>
          </div>
        </div>
      </div>
    </div>
  );
}

export default SettingsPage;
