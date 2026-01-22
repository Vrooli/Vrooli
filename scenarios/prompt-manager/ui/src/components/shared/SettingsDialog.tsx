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

import { Sun, Moon, Monitor, X } from 'lucide-react'
import * as Dialog from '@radix-ui/react-dialog'
import { useTheme } from '@/hooks/use-theme'
import { cn } from '@/lib/utils'
import type { Theme } from '@/types'
import { getShortcutDisplay } from '@/hooks/useKeyboardShortcuts'

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

  return (
    <Dialog.Root open={isOpen} onOpenChange={(open) => !open && onClose()}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-black/50 backdrop-blur-sm data-[state=open]:animate-in data-[state=closed]:animate-out data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0" />
        <Dialog.Content
          aria-describedby="settings-dialog-description"
          className={cn(
            'fixed left-1/2 top-1/2 z-50 -translate-x-1/2 -translate-y-1/2',
            'w-full max-w-md bg-slate-900 border border-white/10 rounded-xl shadow-2xl',
            'data-[state=open]:animate-in data-[state=closed]:animate-out',
            'data-[state=closed]:fade-out-0 data-[state=open]:fade-in-0',
            'data-[state=closed]:zoom-out-95 data-[state=open]:zoom-in-95',
            'data-[state=closed]:slide-out-to-left-1/2 data-[state=closed]:slide-out-to-top-[48%]',
            'data-[state=open]:slide-in-from-left-1/2 data-[state=open]:slide-in-from-top-[48%]'
          )}
        >
          {/* Accessible description (visually hidden) */}
          <Dialog.Description id="settings-dialog-description" className="sr-only">
            Configure application settings including theme and view keyboard shortcuts.
          </Dialog.Description>

          {/* Header */}
          <div className="flex items-center justify-between px-6 py-4 border-b border-white/10">
            <Dialog.Title className="text-lg font-semibold text-white">
              Settings
            </Dialog.Title>
            <Dialog.Close asChild>
              <button
                type="button"
                className="p-1.5 rounded-lg hover:bg-white/10 text-slate-400 hover:text-white transition-colors"
                aria-label="Close"
              >
                <X className="h-5 w-5" />
              </button>
            </Dialog.Close>
          </div>

          {/* Content */}
          <div className="px-6 py-4 space-y-6">
            {/* Theme Section */}
            <div>
              <h4 className="text-sm font-medium text-slate-300 mb-3">Appearance</h4>
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
                          ? 'bg-indigo-600/20 border-indigo-500 text-white'
                          : 'bg-slate-800/50 border-white/10 text-slate-400 hover:bg-slate-800 hover:text-white'
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
              <h4 className="text-sm font-medium text-slate-300 mb-3">Keyboard Shortcuts</h4>
              <div className="space-y-2 text-sm">
                <ShortcutRow label="Save current prompt" shortcut={getShortcutDisplay('save')} />
                <ShortcutRow label="Save all changes" shortcut={getShortcutDisplay('saveAll')} />
                <ShortcutRow label="New prompt" shortcut={getShortcutDisplay('new')} />
                <ShortcutRow label="Focus search" shortcut={getShortcutDisplay('search')} />
                <ShortcutRow label="Close / Cancel" shortcut={getShortcutDisplay('escape')} />
                <ShortcutRow label="Open settings" shortcut={getShortcutDisplay('settings')} />
              </div>
            </div>
          </div>

          {/* Footer */}
          <div className="px-6 py-4 border-t border-white/10">
            <p className="text-xs text-slate-500 text-center">
              Prompt Manager v1.0
            </p>
          </div>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  )
}

function ShortcutRow({ label, shortcut }: { label: string; shortcut: string }) {
  return (
    <div className="flex items-center justify-between">
      <span className="text-slate-400">{label}</span>
      <kbd className="px-2 py-1 text-xs font-mono bg-slate-800 border border-white/10 rounded text-slate-300">
        {shortcut}
      </kbd>
    </div>
  )
}
