/**
 * SettingsDialog - Modal for application settings.
 *
 * Currently provides:
 * - Theme toggle (light/dark/system)
 *
 * Future additions:
 * - Editor preferences
 * - Keyboard shortcut customization
 */

import { Sun, Moon, Monitor } from 'lucide-react'
import { Dialog } from '@/components/shared/Dialog'
import { useTheme } from '@/hooks/use-theme'
import { cn } from '@/lib/utils'
import type { Theme } from '@/types'
import { getShortcutDisplay } from '@/hooks/useKeyboardShortcuts'
import { AISearchStatusPanel } from '@/components/shared/AISearchStatusPanel'
import { useEditorPreferencesStore } from '@/stores/editorPreferencesStore'

interface SettingsDialogProps {
  isOpen: boolean
  onClose: () => void
}

const themeOptions: { value: Theme; label: string; icon: typeof Sun }[] = [
  { value: 'light', label: 'Light', icon: Sun },
  { value: 'dark', label: 'Dark', icon: Moon },
  { value: 'system', label: 'System', icon: Monitor },
]

export function SettingsDialog({ isOpen, onClose }: SettingsDialogProps) {
  const { theme, setTheme } = useTheme()
  const showCodeLineNumbers = useEditorPreferencesStore((state) => state.showCodeLineNumbers)
  const setShowCodeLineNumbers = useEditorPreferencesStore((state) => state.setShowCodeLineNumbers)

  return (
    <Dialog
      isOpen={isOpen}
      onClose={onClose}
      maxWidth="max-w-md"
      titleId="settings-dialog-title"
      descriptionId="settings-dialog-description"
      className="!p-0 bg-card border-border rounded-xl"
      testId="settings-dialog"
    >
      {/* Accessible description (visually hidden) */}
      <p id="settings-dialog-description" className="sr-only">
        Configure application settings including theme and view keyboard shortcuts.
      </p>

      {/* Content */}
      <div className="px-6 py-4 space-y-6">
        <h2 id="settings-dialog-title" className="text-xl font-semibold text-foreground pr-8">
          Settings
        </h2>
        {/* Theme Section */}
        <div>
          <h4 className="text-sm font-medium text-muted-foreground mb-3">Appearance</h4>
          <div className="flex gap-2">
            {themeOptions.map((option) => {
              const Icon = option.icon
              const isSelected = theme === option.value
              return (
                <button
                  key={option.value}
                  type="button"
                  onClick={() => setTheme(option.value)}
                  className={cn(
                    'flex-1 flex flex-col items-center gap-2 px-4 py-3 rounded-lg border transition-all',
                    isSelected
                      ? 'bg-primary/20 border-primary text-foreground'
                      : 'bg-muted/50 border-border text-muted-foreground hover:bg-muted hover:text-foreground'
                  )}
                >
                  <Icon className="h-5 w-5" />
                  <span className="text-xs font-medium">{option.label}</span>
                </button>
              )
            })}
          </div>
        </div>

        {/* Keyboard Shortcuts Section */}
        <div>
          <h4 className="text-sm font-medium text-muted-foreground mb-3">Keyboard Shortcuts</h4>
          <div className="space-y-2 text-sm">
            <ShortcutRow label="Save current skill" shortcut={getShortcutDisplay('save')} />
            <ShortcutRow label="Save all changes" shortcut={getShortcutDisplay('saveAll')} />
            <ShortcutRow label="New skill" shortcut={getShortcutDisplay('new')} />
            <ShortcutRow label="Focus search" shortcut={getShortcutDisplay('search')} />
            <ShortcutRow label="Close / Cancel" shortcut={getShortcutDisplay('escape')} />
            <ShortcutRow label="Open settings" shortcut={getShortcutDisplay('settings')} />
          </div>
        </div>

        {/* Editor Preferences Section */}
        <div>
          <h4 className="text-sm font-medium text-muted-foreground mb-3">Editor Preferences</h4>
          <div className="rounded-lg border border-border bg-muted/30 p-3">
            <div className="flex items-start justify-between gap-3">
              <div className="space-y-1">
                <p className="text-sm text-foreground">Show Code Line Numbers</p>
                <p className="text-xs text-muted-foreground">
                  Display the line-number gutter in markdown code blocks.
                </p>
              </div>
              <button
                type="button"
                role="switch"
                aria-checked={showCodeLineNumbers}
                aria-label="Show code line numbers"
                onClick={() => setShowCodeLineNumbers(!showCodeLineNumbers)}
                className={cn(
                  'relative h-5 w-9 rounded-full transition-colors flex-shrink-0',
                  showCodeLineNumbers ? 'bg-primary' : 'bg-muted-foreground/40'
                )}
              >
                <span
                  className={cn(
                    'absolute top-0.5 left-0.5 h-4 w-4 rounded-full bg-white transition-transform',
                    showCodeLineNumbers && 'translate-x-4'
                  )}
                />
              </button>
            </div>
          </div>
        </div>

        {/* AI Search Section */}
        <div>
          <h4 className="text-sm font-medium text-muted-foreground mb-3">AI Search</h4>
          <AISearchStatusPanel active={isOpen} compact />
        </div>
      </div>

      {/* Footer */}
      <div className="px-6 py-4 border-t border-border">
        <p className="text-xs text-muted-foreground text-center">
          Prompt Manager v1.0
        </p>
      </div>
    </Dialog>
  )
}

function ShortcutRow({ label, shortcut }: { label: string; shortcut: string }) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-muted-foreground">{label}</span>
      <kbd className="px-2 py-1 text-xs font-mono bg-muted border border-border rounded text-foreground">
        {shortcut}
      </kbd>
    </div>
  )
}
