import { motion } from 'framer-motion'
import { Search, Sparkles, Sun, Moon, Monitor, PanelLeftOpen } from 'lucide-react'
import { useTheme } from '@/hooks/use-theme'
import { Button } from './ui/button'
import { Input } from './ui/input'
import type { Campaign } from '@/types'

interface HeaderProps {
  searchQuery: string
  onSearchChange: (query: string) => void
  selectedCampaign: Campaign | null
  showSidebarToggle?: boolean
  onToggleSidebar?: () => void
}

export function Header({ searchQuery, onSearchChange, selectedCampaign, showSidebarToggle, onToggleSidebar }: HeaderProps) {
  const { theme, toggleTheme } = useTheme()

  const getThemeIcon = () => {
    switch (theme) {
      case 'light': return <Sun className="h-4 w-4" />
      case 'dark': return <Moon className="h-4 w-4" />
      default: return <Monitor className="h-4 w-4" />
    }
  }

  return (
    <motion.header
      initial={{ opacity: 0, y: -20 }}
      animate={{ opacity: 1, y: 0 }}
      className="border-b border-border/50 bg-background/80 backdrop-blur-xl"
    >
      <div className="flex items-center justify-between px-6 py-4">
        {/* Left side - Logo and breadcrumb */}
        <div className="flex items-center space-x-3">
          {showSidebarToggle && (
            <motion.div
              className="lg:hidden"
              whileHover={{ scale: 1.05 }}
              whileTap={{ scale: 0.95 }}
            >
              <Button
                variant="ghost"
                size="icon"
                onClick={onToggleSidebar}
                className="h-10 w-10"
                aria-label="Open navigation panel"
              >
                <PanelLeftOpen className="h-5 w-5" />
              </Button>
            </motion.div>
          )}
          <motion.div
            className="flex items-center space-x-3"
            whileHover={{ scale: 1.02 }}
          >
            <div className="relative">
              <motion.div
                animate={{ rotate: [0, 360] }}
                transition={{ duration: 20, repeat: Infinity, ease: "linear" }}
                className="w-8 h-8 rounded-lg bg-gradient-to-br from-purple-500 to-pink-500 flex items-center justify-center"
              >
                <Sparkles className="h-4 w-4 text-white" />
              </motion.div>
              <motion.div
                className="absolute inset-0 rounded-lg bg-gradient-to-br from-purple-500 to-pink-500 opacity-20"
                animate={{ 
                  scale: [1, 1.2, 1],
                  opacity: [0.2, 0.1, 0.2]
                }}
                transition={{ duration: 2, repeat: Infinity, ease: "easeInOut" }}
              />
            </div>
            <div>
              <h1 className="text-xl font-bold bg-gradient-to-r from-gray-900 to-gray-600 dark:from-white dark:to-gray-300 bg-clip-text text-transparent">
                Prompt Manager
              </h1>
              {selectedCampaign && (
                <motion.p
                  initial={{ opacity: 0, x: -10 }}
                  animate={{ opacity: 1, x: 0 }}
                  className="text-sm text-muted-foreground"
                >
                  {selectedCampaign.name}
                </motion.p>
              )}
            </div>
          </motion.div>
        </div>

        {/* Center - Search */}
        <div className="flex-1 max-w-xl mx-3 md:mx-8">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 transform -translate-y-1/2 text-muted-foreground h-4 w-4" />
            <Input
              type="text"
              placeholder="Search prompts, campaigns, or tags..."
              value={searchQuery}
              onChange={(e) => onSearchChange(e.target.value)}
              className="pl-10 bg-background/50 border-border/50 focus:border-primary/50 focus:bg-background transition-all duration-200"
            />
            {searchQuery && (
              <motion.div
                initial={{ opacity: 0, scale: 0.8 }}
                animate={{ opacity: 1, scale: 1 }}
                className="absolute right-3 top-1/2 transform -translate-y-1/2"
              >
                <Button
                  variant="ghost"
                  size="sm"
                  onClick={() => onSearchChange('')}
                  className="h-6 w-6 p-0 text-muted-foreground hover:text-foreground"
                >
                  ×
                </Button>
              </motion.div>
            )}
          </div>
        </div>

        {/* Right side - Actions */}
        <div className="flex items-center space-x-2">
          <motion.div whileHover={{ scale: 1.05 }} whileTap={{ scale: 0.95 }}>
            <Button
              variant="ghost"
              size="icon"
              onClick={toggleTheme}
              className="relative overflow-hidden"
            >
              {getThemeIcon()}
              <motion.div
                className="absolute inset-0 bg-gradient-to-r from-transparent via-white/20 to-transparent"
                initial={{ x: "-100%" }}
                whileHover={{ x: "100%" }}
                transition={{ duration: 0.6 }}
              />
            </Button>
          </motion.div>
          
          <motion.div whileHover={{ scale: 1.05 }} whileTap={{ scale: 0.95 }}>
            <Button variant="outline" size="sm" className="bg-background/50">
              Settings
            </Button>
          </motion.div>
        </div>
      </div>

      {/* Search results indicator */}
      {searchQuery.length > 2 && (
        <motion.div
          initial={{ opacity: 0, height: 0 }}
          animate={{ opacity: 1, height: "auto" }}
          exit={{ opacity: 0, height: 0 }}
          className="px-6 py-2 bg-muted/30 border-t border-border/30"
        >
          <p className="text-sm text-muted-foreground">
            Searching for "{searchQuery}"...
          </p>
        </motion.div>
      )}
    </motion.header>
  )
}
