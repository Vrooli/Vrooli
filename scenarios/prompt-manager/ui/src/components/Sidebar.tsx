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
} from 'lucide-react'
import { Button } from './ui/button'
import { cn } from '@/lib/utils'

interface SidebarProps {
  collapsed: boolean
  onToggle: () => void
  variant?: 'desktop' | 'floating'
  onClose?: () => void
}

const sidebarItems = [
  { icon: Folder, label: 'All Campaigns', active: true },
  { icon: Star, label: 'Favorites', badge: '12' },
  { icon: Clock, label: 'Recent', badge: '3' },
  { icon: TrendingUp, label: 'Popular' },
]

const bottomItems = [
  { icon: Settings, label: 'Settings' },
  { icon: HelpCircle, label: 'Help' },
]

export function Sidebar({ collapsed, onToggle, variant = 'desktop', onClose }: SidebarProps) {
  const handleClose = onClose ?? onToggle

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
          {sidebarItems.map((item, index) => (
            <motion.div
              key={item.label}
              initial={{ opacity: 0, x: -20 }}
              animate={{ opacity: 1, x: 0 }}
              transition={{ delay: index * 0.08 }}
            >
              <Button
                variant={item.active ? 'secondary' : 'ghost'}
                className={cn(
                  'w-full justify-start h-10 px-3',
                  item.active && 'bg-primary/10 text-primary border border-primary/20'
                )}
              >
                <item.icon className="h-4 w-4 mr-3" />
                <span className="flex-1 text-left">{item.label}</span>
                {item.badge && (
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
          ))}
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
            {sidebarItems.map((item, index) => (
              <motion.div
                key={item.label}
                initial={{ opacity: 0, x: -20 }}
                animate={{ opacity: 1, x: 0 }}
                transition={{ delay: index * 0.1 }}
              >
                <Button
                  variant={item.active ? "secondary" : "ghost"}
                  className={cn(
                    "w-full justify-start h-10 px-3",
                    item.active && "bg-primary/10 text-primary border border-primary/20"
                  )}
                >
                  <item.icon className="h-4 w-4 mr-3" />
                  <span className="flex-1 text-left">{item.label}</span>
                  {item.badge && (
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
            ))}
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
          {sidebarItems.map((item, index) => (
            <motion.div
              key={item.label}
              initial={{ opacity: 0, scale: 0.8 }}
              animate={{ opacity: 1, scale: 1 }}
              transition={{ delay: index * 0.1 }}
              className="relative group"
            >
              <Button
                variant={item.active ? "secondary" : "ghost"}
                size="icon"
                className={cn(
                  "w-full h-10",
                  item.active && "bg-primary/10 text-primary"
                )}
              >
                <item.icon className="h-4 w-4" />
              </Button>
              
              {/* Tooltip */}
              <motion.div
                initial={{ opacity: 0, x: -10 }}
                whileHover={{ opacity: 1, x: 0 }}
                className="absolute left-full ml-2 top-1/2 transform -translate-y-1/2 z-50 group-hover:block hidden"
              >
                <div className="bg-popover text-popover-foreground text-sm px-2 py-1 rounded border shadow-lg whitespace-nowrap">
                  {item.label}
                  <div className="absolute left-0 top-1/2 transform -translate-y-1/2 -translate-x-1 w-2 h-2 bg-popover border-l border-b rotate-45" />
                </div>
              </motion.div>
            </motion.div>
          ))}
        </motion.div>
      )}
    </div>
  )
}
