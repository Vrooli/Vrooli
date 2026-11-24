/**
 * The small icon vocabulary a template composition may use.
 *
 * A template is persisted JSON, so it cannot import a React component by name.
 * It names one of these instead. The set is deliberately small and product-
 * neutral: the icon supports a label, it never carries meaning alone.
 */
import {
  Activity,
  AlarmClock,
  ArrowRight,
  BarChart3,
  Bell,
  BookOpen,
  Boxes,
  Bus,
  Camera,
  Check,
  ChevronDown,
  ChevronRight,
  Clock,
  Coffee,
  Compass,
  Droplet,
  GitFork,
  Grid3x3,
  Home,
  Layers,
  Leaf,
  Moon,
  Pin,
  Play,
  Plus,
  Receipt,
  Route,
  Search,
  SlidersHorizontal,
  Sparkles,
  Sprout,
  Zap,
  type LucideIcon,
} from "lucide-react";

const ICONS: Record<string, LucideIcon> = {
  activity: Activity,
  alarm: AlarmClock,
  arrow: ArrowRight,
  chart: BarChart3,
  bell: Bell,
  book: BookOpen,
  box: Boxes,
  bus: Bus,
  camera: Camera,
  check: Check,
  chevron: ChevronRight,
  "chevron-down": ChevronDown,
  clock: Clock,
  coffee: Coffee,
  compass: Compass,
  drop: Droplet,
  fork: GitFork,
  grid: Grid3x3,
  home: Home,
  layers: Layers,
  leaf: Leaf,
  moon: Moon,
  pin: Pin,
  play: Play,
  plus: Plus,
  receipt: Receipt,
  route: Route,
  search: Search,
  sliders: SlidersHorizontal,
  spark: Sparkles,
  sprout: Sprout,
  zap: Zap,
};

export function TemplateIcon({ name, size = 16 }: { name: string; size?: number }) {
  const Icon = ICONS[name] ?? Sparkles;
  return <Icon aria-hidden size={size} strokeWidth={1.7} />;
}
