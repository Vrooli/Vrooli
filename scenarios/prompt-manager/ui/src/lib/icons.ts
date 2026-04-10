/**
 * Icon utilities for the prompt manager.
 *
 * Provides icon lookup functionality used by IconSelector and other components.
 * Extracted to avoid react-refresh warnings about non-component exports.
 */

import {
  FileText,
  MessageSquare,
  Code,
  Lightbulb,
  Target,
  Zap,
  BookOpen,
  Sparkles,
  Brain,
  Rocket,
  Settings,
  Search,
  PenTool,
  Wand2,
  Bot,
  Terminal,
  Database,
  Cloud,
  Shield,
  Lock,
  Key,
  AlertTriangle,
  CheckCircle,
  HelpCircle,
  Info,
  Star,
  Heart,
  Flag,
  Bookmark,
  Tag,
  Folder,
  File,
  type LucideIcon,
} from 'lucide-react'

/**
 * Available icons with their display names.
 */
export const ICONS: { name: string; icon: LucideIcon }[] = [
  { name: 'file-text', icon: FileText },
  { name: 'message-square', icon: MessageSquare },
  { name: 'code', icon: Code },
  { name: 'lightbulb', icon: Lightbulb },
  { name: 'target', icon: Target },
  { name: 'zap', icon: Zap },
  { name: 'book-open', icon: BookOpen },
  { name: 'sparkles', icon: Sparkles },
  { name: 'brain', icon: Brain },
  { name: 'rocket', icon: Rocket },
  { name: 'settings', icon: Settings },
  { name: 'search', icon: Search },
  { name: 'pen-tool', icon: PenTool },
  { name: 'wand-2', icon: Wand2 },
  { name: 'bot', icon: Bot },
  { name: 'terminal', icon: Terminal },
  { name: 'database', icon: Database },
  { name: 'cloud', icon: Cloud },
  { name: 'shield', icon: Shield },
  { name: 'lock', icon: Lock },
  { name: 'key', icon: Key },
  { name: 'alert-triangle', icon: AlertTriangle },
  { name: 'check-circle', icon: CheckCircle },
  { name: 'help-circle', icon: HelpCircle },
  { name: 'info', icon: Info },
  { name: 'star', icon: Star },
  { name: 'heart', icon: Heart },
  { name: 'flag', icon: Flag },
  { name: 'bookmark', icon: Bookmark },
  { name: 'tag', icon: Tag },
  { name: 'folder', icon: Folder },
  { name: 'file', icon: File },
]

// Create a lookup map for getting icon by name
const ICON_MAP = new Map(ICONS.map((i) => [i.name, i.icon]))

// Default icon when none selected or not found
const DEFAULT_ICON = FileText

/**
 * Get the Lucide icon component for a given icon name.
 */
export function getIcon(name: string): LucideIcon {
  return ICON_MAP.get(name) ?? DEFAULT_ICON
}
