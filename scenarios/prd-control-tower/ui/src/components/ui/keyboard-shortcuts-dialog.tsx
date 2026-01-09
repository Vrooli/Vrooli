import { useEffect, useState } from 'react'
import { Command } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from './dialog'

interface Shortcut {
  keys: string[]
  description: string
  context?: string
}

const SHORTCUTS: Shortcut[] = [
  { keys: ['⌘', 'K'], description: 'Focus search', context: 'Catalog page' },
  { keys: ['Esc'], description: 'Clear search and unfocus', context: 'Catalog page' },
  { keys: ['⌘', 'S'], description: 'Save draft', context: 'Draft editor' },
  { keys: ['?'], description: 'Show keyboard shortcuts', context: 'Global' },
]

export function KeyboardShortcutsDialog() {
  const [open, setOpen] = useState(false)

  useEffect(() => {
    const handleKeyDown = (event: KeyboardEvent) => {
      // Show shortcuts dialog when ? is pressed (without Shift it's /)
      if (event.key === '?' && !event.metaKey && !event.ctrlKey && !event.altKey) {
        // Only trigger if we're not in an input/textarea
        const target = event.target as HTMLElement
        if (target.tagName !== 'INPUT' && target.tagName !== 'TEXTAREA' && !target.isContentEditable) {
          event.preventDefault()
          setOpen(true)
        }
      }

      // Close with Escape
      if (event.key === 'Escape' && open) {
        setOpen(false)
      }
    }

    window.addEventListener('keydown', handleKeyDown)
    return () => window.removeEventListener('keydown', handleKeyDown)
  }, [open])

  return (
    <>
      {/* Floating help button */}
      <button
        onClick={() => setOpen(true)}
        className="fixed bottom-6 left-6 sm:bottom-8 sm:left-8 z-40 flex h-12 w-12 items-center justify-center rounded-full bg-slate-800 text-white shadow-lg transition-all hover:scale-110 hover:shadow-xl hover:bg-slate-900 focus:outline-none focus:ring-4 focus:ring-slate-300 focus:ring-offset-2 active:scale-95"
        title="Keyboard shortcuts (?)"
        aria-label="Show keyboard shortcuts"
      >
        <Command size={20} strokeWidth={2.5} />
      </button>

      <Dialog open={open} onOpenChange={setOpen}>
        <DialogContent className="max-w-lg">
          <DialogHeader>
            <DialogTitle className="flex items-center gap-2 text-xl">
              <Command size={20} className="text-violet-600" />
              Keyboard Shortcuts
            </DialogTitle>
            <DialogDescription>
              Press <kbd className="mx-1 inline-flex h-5 items-center gap-1 rounded border border-slate-200 bg-slate-50 px-1.5 text-[11px] font-semibold text-slate-600">?</kbd> anytime to view this panel
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            {SHORTCUTS.map((shortcut, index) => (
              <div
                key={index}
                className="flex items-center justify-between gap-4 rounded-lg border border-slate-100 bg-gradient-to-r from-white to-slate-50/50 p-3 transition hover:border-violet-200 hover:shadow-sm"
              >
                <div className="flex-1">
                  <p className="text-sm font-medium text-slate-900">{shortcut.description}</p>
                  {shortcut.context && (
                    <p className="text-xs text-slate-500 mt-0.5">{shortcut.context}</p>
                  )}
                </div>
                <div className="flex items-center gap-1 shrink-0">
                  {shortcut.keys.map((key, keyIndex) => (
                    <kbd
                      key={keyIndex}
                      className="inline-flex h-7 min-w-[1.75rem] items-center justify-center rounded border border-slate-200 bg-slate-50 px-2 text-xs font-semibold text-slate-700 shadow-sm"
                    >
                      {key}
                    </kbd>
                  ))}
                </div>
              </div>
            ))}
          </div>

          <div className="rounded-lg border border-violet-100 bg-violet-50/50 p-3 text-sm">
            <p className="flex items-start gap-2 text-slate-700">
              <span className="shrink-0 text-violet-600">💡</span>
              <span>
                <strong className="text-slate-900">Tip:</strong> On Windows/Linux, use <kbd className="mx-1 inline-flex h-5 items-center gap-1 rounded border border-slate-200 bg-white px-1.5 text-[11px] font-semibold text-slate-600">Ctrl</kbd> instead of <kbd className="mx-1 inline-flex h-5 items-center gap-1 rounded border border-slate-200 bg-white px-1.5 text-[11px] font-semibold text-slate-600">⌘</kbd>
              </span>
            </p>
          </div>
        </DialogContent>
      </Dialog>
    </>
  )
}
