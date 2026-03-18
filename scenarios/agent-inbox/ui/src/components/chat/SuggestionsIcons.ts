/**
 * Icon mappings for the Suggestions component.
 *
 * Extracted from Suggestions.tsx for modularity and to reduce the main file size.
 */
import {
  Search,
  Bug,
  Sparkles,
  RefreshCw,
  FlaskConical,
  GraduationCap,
  Eye,
  FolderTree,
  Package,
  Lightbulb,
  Gauge,
  FileType,
  Server,
  Layout,
  Wand2,
  Building2,
  ArrowRightLeft,
  Puzzle,
  Route,
  MessageCircleQuestion,
  Shield,
  Accessibility,
} from "lucide-react";

// Icon mapping for dynamic rendering
export const ICON_MAP: Record<string, React.ComponentType<{ className?: string }>> = {
  Search,
  Bug,
  Sparkles,
  RefreshCw,
  FlaskConical,
  GraduationCap,
  Eye,
  FolderTree,
  Package,
  Lightbulb,
  Gauge,
  FileType,
  Server,
  Layout,
  Wand2,
  Building2,
  ArrowRightLeft,
  Puzzle,
  Route,
  MessageCircleQuestion,
  Shield,
  Accessibility,
};

// Mode icons
export const MODE_ICONS: Record<string, React.ComponentType<{ className?: string }>> = {
  Research: Search,
  "Debug/Fix": Bug,
  "Implement Feature": Sparkles,
  Refactor: RefreshCw,
  "Write Tests": FlaskConical,
  "Explain/Teach": GraduationCap,
  "Review/QA": Eye,
};

export function getIconComponent(
  iconName?: string
): React.ComponentType<{ className?: string }> | null {
  if (!iconName) return null;
  return ICON_MAP[iconName] || null;
}
