/**
 * Tests for SettingsDialog component.
 *
 * Tests cover:
 * - Dialog open/close states
 * - Theme selection UI
 * - Keyboard shortcuts display
 */

import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@/test-utils/renderWithProviders'
import { SettingsDialog } from './SettingsDialog'
import { ThemeProvider } from '@/hooks/use-theme'

const pendingAIStatusRequest = new Promise<never>(() => {})

vi.mock('@/services/skillService', () => ({
  getAISearchStatus: vi.fn(() => pendingAIStatusRequest),
  getAISearchReindexStatus: vi.fn(() => pendingAIStatusRequest),
  reindexAISearch: vi.fn(() => pendingAIStatusRequest),
  cancelAISearchReindex: vi.fn(() => pendingAIStatusRequest),
}))

// Wrapper component that provides theme context
function renderWithTheme(ui: React.ReactElement) {
  return render(<ThemeProvider>{ui}</ThemeProvider>)
}

describe('SettingsDialog', () => {
  describe('visibility', () => {
    it('should not render content when closed', () => {
      renderWithTheme(
        <SettingsDialog isOpen={false} onClose={vi.fn()} />
      )

      expect(screen.queryByText('Settings')).not.toBeInTheDocument()
    })

    it('should render content when open', () => {
      renderWithTheme(
        <SettingsDialog isOpen={true} onClose={vi.fn()} />
      )

      expect(screen.getByText('Settings')).toBeInTheDocument()
    })
  })

  describe('close behavior', () => {
    it('should call onClose when close button is clicked', () => {
      const onClose = vi.fn()
      renderWithTheme(
        <SettingsDialog isOpen={true} onClose={onClose} />
      )

      const closeButton = screen.getByLabelText('Close dialog')
      fireEvent.click(closeButton)

      expect(onClose).toHaveBeenCalledTimes(1)
    })
  })

  describe('theme section', () => {
    it('should display theme options', () => {
      renderWithTheme(
        <SettingsDialog isOpen={true} onClose={vi.fn()} />
      )

      expect(screen.getByText('Appearance')).toBeInTheDocument()
      expect(screen.getByText('Light')).toBeInTheDocument()
      expect(screen.getByText('Dark')).toBeInTheDocument()
      expect(screen.getByText('System')).toBeInTheDocument()
    })

    it('should have three theme option buttons', () => {
      renderWithTheme(
        <SettingsDialog isOpen={true} onClose={vi.fn()} />
      )

      const lightButton = screen.getByText('Light').closest('button')
      const darkButton = screen.getByText('Dark').closest('button')
      const systemButton = screen.getByText('System').closest('button')

      expect(lightButton).toBeInTheDocument()
      expect(darkButton).toBeInTheDocument()
      expect(systemButton).toBeInTheDocument()
    })
  })

  describe('keyboard shortcuts section', () => {
    it('should display keyboard shortcuts heading', () => {
      renderWithTheme(
        <SettingsDialog isOpen={true} onClose={vi.fn()} />
      )

      expect(screen.getByText('Keyboard Shortcuts')).toBeInTheDocument()
    })

    it('should display shortcut labels', () => {
      renderWithTheme(
        <SettingsDialog isOpen={true} onClose={vi.fn()} />
      )

      expect(screen.getByText('Save current skill')).toBeInTheDocument()
      expect(screen.getByText('Save all changes')).toBeInTheDocument()
      expect(screen.getByText('New skill')).toBeInTheDocument()
      expect(screen.getByText('Focus search')).toBeInTheDocument()
      expect(screen.getByText('Close / Cancel')).toBeInTheDocument()
      expect(screen.getByText('Open settings')).toBeInTheDocument()
    })

    it('should display shortcut key combinations', () => {
      renderWithTheme(
        <SettingsDialog isOpen={true} onClose={vi.fn()} />
      )

      // Check for Esc and comma which are always the same
      expect(screen.getByText('Esc')).toBeInTheDocument()
      expect(screen.getByText(',')).toBeInTheDocument()
    })
  })

  describe('editor preferences section', () => {
    it('should display editor preferences heading and line number setting', () => {
      renderWithTheme(
        <SettingsDialog isOpen={true} onClose={vi.fn()} />
      )

      expect(screen.getByText('Editor Preferences')).toBeInTheDocument()
      expect(screen.getByText('Show Code Line Numbers')).toBeInTheDocument()
    })

    it('should toggle code line numbers setting', () => {
      renderWithTheme(
        <SettingsDialog isOpen={true} onClose={vi.fn()} />
      )

      const toggle = screen.getByRole('switch', { name: 'Show code line numbers' })
      expect(toggle).toHaveAttribute('aria-checked', 'true')

      fireEvent.click(toggle)
      expect(toggle).toHaveAttribute('aria-checked', 'false')
    })
  })

  describe('footer', () => {
    it('should display version information', () => {
      renderWithTheme(
        <SettingsDialog isOpen={true} onClose={vi.fn()} />
      )

      expect(screen.getByText('Prompt Manager v1.0')).toBeInTheDocument()
    })
  })
})
