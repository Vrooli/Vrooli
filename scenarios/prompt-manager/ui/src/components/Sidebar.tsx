import { motion } from 'framer-motion'
import {
  PanelLeftClose,
  PanelLeftOpen,
  Folder,
  Star,
  Clock,
  TrendingUp,
  Settings,
  HelpCircle,
  X,
  type LucideIcon,
} from 'lucide-react'
import { Button } from './ui/button'
import { cn } from '@/lib/utils'

export type ViewFilter = 'campaigns' | 'favorites' | 'recent' | 'popular'

interface SidebarItem {
  key: ViewFilter
  icon: LucideIcon
  label: string
  badge?: number
}

interface SidebarProps {
  collapsed: boolean
  onToggle: () => void
  variant?: 'desktop' | 'floating'
  onClose?: () => void
  activeFilter: ViewFilter
  onFilterChange: (filter: ViewFilter) => void
  counts?: {
    favorites?: number
    recent?: number
    popular?: number
  }
}

const bottomItems = [
  { icon: Settings, label: 'Settings' },
  { icon: HelpCircle, label: 'Help' },
]

export function Sidebar({
  collapsed,
  onToggle,
  variant = 'desktop',
  onClose,
  activeFilter,
  onFilterChange,
  counts = {}
}: SidebarProps) {
  const handleClose = onClose ?? onToggle

  const sidebarItems: SidebarItem[] = [
    { key: 'campaigns', icon: Folder, label: 'All Campaigns' },
    { key: 'favorites', icon: Star, label: 'Favorites', badge: counts.favorites },
    { key: 'recent', icon: Clock, label: 'Recent', badge: counts.recent },
    { key: 'popular', icon: TrendingUp, label: 'Popular', badge: counts.popular },
  ]

  const renderNavItem = (item: SidebarItem, index: number) => {
    const isActive = activeFilter === item.key
    return (
      <motion.div
        key={item.key}
        initial={{ opacity: 0, x: -20 }}
        animate={{ opacity: 1, x: 0 }}
        transition={{ delay: index * 0.08 }}
      >
        <Button
          variant={isActive ? 'secondary' : 'ghost'}
          onClick={() => onFilterChange(item.key)}
          className={cn(
            'w-full justify-start h-10 px-3',
            isActive && 'bg-primary/10 text-primary border border-primary/20'
          )}
        >
          <item.icon className="h-4 w-4 mr-3" />
          <span className="flex-1 text-left">{item.label}</span>
          {item.badge !== undefined && item.badge > 0 && (
            <motion.span
              initial={{ scale: 0 }}
              animate={{ scale: 1 }}
              className="ml-auto bg-primary/20 text-primary text-xs px-2 py-0.5 rounded-full"
            >
              {item.badge}
            </motion.span>
          )}
        </Button>
      </motion.div>
    )
  }

  if (variant === 'floating') {
    return (
      <div className="border-b border-border/30 bg-card/80 backdrop-blur-sm">
        <div className="flex items-center justify-between px-4 py-3">
          <motion.span
            initial={{ opacity: 0, x: -8 }}
            animate={{ opacity: 1, x: 0 }}
            className="text-sm font-semibold uppercase tracking-wide text-muted-foreground"
          >
            Navigation
          </motion.span>
          <motion.div whileHover={{ scale: 1.05 }} whileTap={{ scale: 0.95 }}>
            <Button
              variant="ghost"
              size="icon"
              onClick={handleClose}
              className="h-8 w-8 rounded-full"
            >
              <X className="h-4 w-4" />
            </Button>
          </motion.div>
        </div>

        <motion.nav
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="px-4 pb-4 space-y-2"
        >
          {sidebarItems.map((item, index) => renderNavItem(item, index))}
        </motion.nav>

        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="px-4 pb-4 space-y-1 border-t border-border/30 pt-4"
        >
          {bottomItems.map((item) => (
            <Button
              key={item.label}
              variant="ghost"
              className="w-full justify-start h-10 px-3"
            >
              <item.icon className="h-4 w-4 mr-3" />
              {item.label}
            </Button>
          ))}
        </motion.div>
      </div>
    )
  }

  return (
    <div className="flex flex-col h-full">
      {/* Toggle button */}
      <div className="p-4 border-b border-border/30">
        <motion.div whileHover={{ scale: 1.05 }} whileTap={{ scale: 0.95 }}>
          <Button
            variant="ghost"
            size="icon"
            onClick={onToggle}
            className="w-full justify-center"
          >
            {collapsed ? (
              <PanelLeftOpen className="h-4 w-4" />
            ) : (
              <PanelLeftClose className="h-4 w-4" />
            )}
          </Button>
        </motion.div>
      </div>

      {/* Navigation items */}
      {!collapsed && (
        <motion.nav
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          exit={{ opacity: 0 }}
          className="flex-1 p-4 space-y-2"
        >
          <div className="space-y-1">
            {sidebarItems.map((item, index) => renderNavItem(item, index))}
          </div>
        </motion.nav>
      )}

      {/* Bottom section */}
      {!collapsed && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="p-4 border-t border-border/30 space-y-1"
        >
          {bottomItems.map((item) => (
            <Button
              key={item.label}
              variant="ghost"
              className="w-full justify-start h-10 px-3"
            >
              <item.icon className="h-4 w-4 mr-3" />
              {item.label}
            </Button>
          ))}
        </motion.div>
      )}

      {/* Collapsed navigation */}
      {collapsed && (
        <motion.div
          initial={{ opacity: 0 }}
          animate={{ opacity: 1 }}
          className="flex-1 p-2 space-y-2"
        >
          {sidebarItems.map((item, index) => {
            const isActive = activeFilter === item.key
            return (
              <motion.div
                key={item.key}
                initial={{ opacity: 0, scale: 0.8 }}
                animate={{ opacity: 1, scale: 1 }}
                transition={{ delay: index * 0.1 }}
                className="relative group"
              >
                <Button
                  variant={isActive ? 'secondary' : 'ghost'}
                  size="icon"
                  onClick={() => onFilterChange(item.key)}
                  className={cn(
                    'w-full h-10',
                    isActive && 'bg-primary/10 text-primary'
                  )}
                >
                  <item.icon className="h-4 w-4" />
                </Button>

                {/* Tooltip */}
                <div className="absolute left-full ml-2 top-1/2 transform -translate-y-1/2 z-50 hidden group-hover:block pointer-events-none">
                  <div className="bg-popover text-popover-foreground text-sm px-2 py-1 rounded border shadow-lg whitespace-nowrap">
                    {item.label}
                    <div className="absolute left-0 top-1/2 transform -translate-y-1/2 -translate-x-1 w-2 h-2 bg-popover border-l border-b rotate-45" />
                  </div>
                </div>
              </motion.div>
            )
          })}
        </motion.div>
      )}
    </div>
  )
}
