/**
 * ViewOverlay - Shared overlay chrome for World View and Graph View.
 *
 * Provides consistent positioning for:
 * - Stats bar (top-left)
 * - Optional left panel content (top-left, below stats)
 * - View toggle, settings, and help buttons (top-right)
 * - Settings and help floating panels
 */

import { useEffect, useMemo, useState, type ReactNode } from 'react'
import { Globe, Network, Settings, HelpCircle, BarChart3, Search, X, Menu, Users } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { selectors } from '@/constants/selectors'
import { useIsMobile } from '@/hooks/useMediaQuery'
import { StatsBar } from './StatsBar'
import { FloatingPanel } from './FloatingPanel'
import { AgentPickerList } from './AgentPickerList'

interface ViewOverlayProps {
  onOpenMobileSidebar?: () => void
  /** Number of pending work items needing attention */
  pendingWorkCount?: number
  /** Number of currently running agents */
  runningAgentCount?: number
  leftPanelContent?: ReactNode
  settingsContent: ReactNode
  settingsTitle?: string
  helpContent: ReactNode
  helpTitle?: string
  homeView?: 'world' | 'graph'
  onHomeViewChange?: (view: 'world' | 'graph') => void
}

export function ViewOverlay({
  onOpenMobileSidebar,
  pendingWorkCount = 0,
  runningAgentCount = 0,
  leftPanelContent,
  settingsContent,
  settingsTitle = 'Settings',
  helpContent,
  helpTitle = 'Help',
  homeView = 'world',
  onHomeViewChange,
}: ViewOverlayProps) {
  const [isSettingsOpen, setIsSettingsOpen] = useState(false)
  const [isHelpOpen, setIsHelpOpen] = useState(false)
  const [activeMobilePanel, setActiveMobilePanel] = useState<'stats' | 'left' | 'agents' | null>(null)

  const graphViewActive = homeView === 'graph'
  const isMobile = useIsMobile()

  const panelAnchorX = useMemo(() => {
    if (typeof window === 'undefined') return 24
    return Math.max(24, window.innerWidth - 680)
  }, [])

  useEffect(() => {
    if (!isMobile && activeMobilePanel !== null) {
      setActiveMobilePanel(null)
    }
  }, [activeMobilePanel, isMobile])

  useEffect(() => {
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key !== 'Escape') return
      if (activeMobilePanel) {
        setActiveMobilePanel(null)
        return
      }
      if (isHelpOpen) {
        setIsHelpOpen(false)
        return
      }
      if (isSettingsOpen) {
        setIsSettingsOpen(false)
      }
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [activeMobilePanel, isHelpOpen, isSettingsOpen])

  const mobilePanelTitle = activeMobilePanel === 'stats' ? 'Stats' : activeMobilePanel === 'agents' ? 'Agents' : 'Queries'

  return (
    <>
      <div className="absolute inset-0 pointer-events-none z-20">
        {/* Top-left: stats + optional left panel content */}
        <div className="absolute top-4 left-4 pointer-events-auto">
          {isMobile ? (
            <div className="flex flex-col items-start gap-1">
              {onOpenMobileSidebar && (
                <div className="relative">
                  <Button
                    variant="outline"
                    size="icon"
                    onClick={onOpenMobileSidebar}
                    className="h-8 w-8 bg-card/80 border-border hover:bg-muted"
                    title="Open sidebar"
                    data-testid={selectors.viewOverlay.mobileSidebarButton}
                  >
                    <Menu className="h-4 w-4" />
                  </Button>
                  {(pendingWorkCount > 0 || runningAgentCount > 0) && (
                    <span className="absolute -top-1.5 -right-1.5 flex h-4 min-w-4 items-center justify-center rounded-full bg-amber-500 text-[9px] font-bold text-white px-1 pointer-events-none">
                      {pendingWorkCount + runningAgentCount}
                    </span>
                  )}
                </div>
              )}
              <Button
                variant="outline"
                size="icon"
                onClick={() => setActiveMobilePanel(activeMobilePanel === 'stats' ? null : 'stats')}
                className="h-8 w-8 bg-card/80 border-border hover:bg-muted"
                title="View stats"
                data-testid={selectors.viewOverlay.mobileStatsButton}
              >
                <BarChart3 className="h-4 w-4" />
              </Button>
              {leftPanelContent && (
                <Button
                  variant="outline"
                  size="icon"
                  onClick={() => setActiveMobilePanel(activeMobilePanel === 'left' ? null : 'left')}
                  className="h-8 w-8 bg-card/80 border-border hover:bg-muted"
                  title="Open queries"
                  data-testid={selectors.viewOverlay.mobileLeftPanelButton}
                >
                  <Search className="h-4 w-4" />
                </Button>
              )}
              <Button
                variant="outline"
                size="icon"
                onClick={() => setActiveMobilePanel(activeMobilePanel === 'agents' ? null : 'agents')}
                className="h-8 w-8 bg-card/80 border-border hover:bg-muted"
                title="Find agent"
                data-testid={selectors.viewOverlay.mobileAgentPickerButton}
              >
                <Users className="h-4 w-4" />
              </Button>
            </div>
          ) : (
            <>
              <StatsBar />
              {leftPanelContent && (
                <div className="mt-2 flex flex-col gap-2 max-w-xs">
                  {leftPanelContent}
                </div>
              )}
            </>
          )}
        </div>

        {/* Top-right: view toggle + settings + help */}
        <div className="absolute top-4 right-4 pointer-events-auto flex flex-col gap-1">
          <Button
            variant="outline"
            size="icon"
            onClick={() => onHomeViewChange?.(graphViewActive ? 'world' : 'graph')}
            className="h-8 w-8 bg-card/80 border-border hover:bg-muted"
            title={graphViewActive ? 'Switch to World View' : 'Switch to Graph View'}
            data-testid={selectors.viewOverlay.viewToggle}
          >
            {graphViewActive ? <Globe className="h-4 w-4" /> : <Network className="h-4 w-4" />}
          </Button>
          <Button
            variant="outline"
            size="icon"
            onClick={() => setIsSettingsOpen(true)}
            className="h-8 w-8 bg-card/80 border-border hover:bg-muted"
            title={settingsTitle}
            data-testid={selectors.viewOverlay.settingsButton}
          >
            <Settings className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            size="icon"
            onClick={() => setIsHelpOpen(true)}
            className="h-8 w-8 bg-card/80 border-border hover:bg-muted"
            title={helpTitle}
            data-testid={selectors.viewOverlay.helpButton}
          >
            <HelpCircle className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {isMobile && activeMobilePanel && (
        <>
          <div
            className="fixed inset-0 z-40 bg-black/50"
            onClick={() => setActiveMobilePanel(null)}
            data-testid={selectors.viewOverlay.mobilePanelBackdrop}
          />
          <div
            className="fixed inset-x-0 bottom-0 z-50 max-h-[70vh] overflow-y-auto rounded-t-2xl border border-border bg-popover text-popover-foreground shadow-2xl"
            role="dialog"
            aria-modal="true"
            aria-label={mobilePanelTitle}
            data-testid={selectors.viewOverlay.mobilePanelSheet}
          >
            <div className="mx-auto mt-2 h-1.5 w-10 rounded-full bg-border" />
            <div className="flex items-center justify-between border-b border-border px-4 py-3">
              <p className="text-sm font-medium">{mobilePanelTitle}</p>
              <button
                type="button"
                onClick={() => setActiveMobilePanel(null)}
                className="rounded p-1 text-muted-foreground hover:bg-muted hover:text-foreground"
                aria-label="Close mobile panel"
              >
                <X className="h-4 w-4" />
              </button>
            </div>
            <div className="p-4">
              {activeMobilePanel === 'stats' ? <StatsBar compact /> : activeMobilePanel === 'agents' ? <AgentPickerList onSelect={() => setActiveMobilePanel(null)} /> : leftPanelContent}
            </div>
          </div>
        </>
      )}

      {/* Settings panel */}
      <FloatingPanel
        isOpen={isSettingsOpen}
        onClose={() => setIsSettingsOpen(false)}
        title={settingsTitle}
        initialPosition={{ x: panelAnchorX, y: 88 }}
        className="max-w-md"
      >
        {settingsContent}
      </FloatingPanel>

      {/* Help panel */}
      <FloatingPanel
        isOpen={isHelpOpen}
        onClose={() => setIsHelpOpen(false)}
        title={helpTitle}
        initialPosition={{ x: panelAnchorX + 24, y: 128 }}
      >
        {helpContent}
      </FloatingPanel>

    </>
  )
}
