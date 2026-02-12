/**
 * ViewOverlay - Shared overlay chrome for World View and Graph View.
 *
 * Provides consistent positioning for:
 * - Stats bar (top-left)
 * - Optional left panel content (top-left, below stats)
 * - View toggle, settings, and help buttons (top-right)
 * - Settings and help modals
 */

import { useState, type ReactNode } from 'react'
import { Globe, Network, Settings, HelpCircle } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { useSelectionStore } from '@/stores/selectionStore'
import { selectors } from '@/constants/selectors'
import { StatsBar } from './StatsBar'
import { OverlayModal } from './OverlayModal'

interface ViewOverlayProps {
  leftPanelContent?: ReactNode
  settingsContent: ReactNode
  settingsTitle?: string
  helpContent: ReactNode
  helpTitle?: string
}

export function ViewOverlay({
  leftPanelContent,
  settingsContent,
  settingsTitle = 'Settings',
  helpContent,
  helpTitle = 'Help',
}: ViewOverlayProps) {
  const [isSettingsOpen, setIsSettingsOpen] = useState(false)
  const [isHelpOpen, setIsHelpOpen] = useState(false)

  const graphViewActive = useSelectionStore((s) => s.graphViewActive)
  const setGraphViewActive = useSelectionStore((s) => s.setGraphViewActive)

  return (
    <>
      <div className="absolute inset-0 pointer-events-none z-20">
        {/* Top-left: stats + optional left panel content */}
        <div className="absolute top-4 left-4 pointer-events-auto">
          <StatsBar />
          {leftPanelContent && (
            <div className="mt-2 flex flex-col gap-2 max-w-xs">
              {leftPanelContent}
            </div>
          )}
        </div>

        {/* Top-right: view toggle + settings + help */}
        <div className="absolute top-4 right-4 pointer-events-auto flex flex-col gap-1">
          <Button
            variant="outline"
            size="icon"
            onClick={() => setGraphViewActive(!graphViewActive)}
            className="h-8 w-8 bg-slate-800/80 border-slate-700 hover:bg-slate-700"
            title={graphViewActive ? 'Switch to World View' : 'Switch to Graph View'}
            data-testid={selectors.viewOverlay.viewToggle}
          >
            {graphViewActive ? <Globe className="h-4 w-4" /> : <Network className="h-4 w-4" />}
          </Button>
          <Button
            variant="outline"
            size="icon"
            onClick={() => setIsSettingsOpen(true)}
            className="h-8 w-8 bg-slate-800/80 border-slate-700 hover:bg-slate-700"
            title={settingsTitle}
            data-testid={selectors.viewOverlay.settingsButton}
          >
            <Settings className="h-4 w-4" />
          </Button>
          <Button
            variant="outline"
            size="icon"
            onClick={() => setIsHelpOpen(true)}
            className="h-8 w-8 bg-slate-800/80 border-slate-700 hover:bg-slate-700"
            title={helpTitle}
            data-testid={selectors.viewOverlay.helpButton}
          >
            <HelpCircle className="h-4 w-4" />
          </Button>
        </div>
      </div>

      {/* Settings modal */}
      <OverlayModal
        isOpen={isSettingsOpen}
        onClose={() => setIsSettingsOpen(false)}
        title={settingsTitle}
        maxWidth="max-w-md"
      >
        {settingsContent}
      </OverlayModal>

      {/* Help modal */}
      <OverlayModal
        isOpen={isHelpOpen}
        onClose={() => setIsHelpOpen(false)}
        title={helpTitle}
      >
        {helpContent}
      </OverlayModal>
    </>
  )
}
