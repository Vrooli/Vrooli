import { fireEvent, screen, within } from '@testing-library/react'
import { describe, expect, it, vi } from 'vitest'
import { render } from '@/test-utils/renderWithProviders'
import { ConfirmDialog } from './ConfirmDialog'

vi.mock('@vrooli/react-component-library/OverlayActionBar/1.0.1', () => ({
  OverlayActionBar: ({ actions, className }: {
    actions: Array<{ id: string; label: string; onSelect: () => void; disabled?: boolean; testId?: string }>
    className?: string
  }) => (
    <div className={className} data-rcl-overlay-action-bar role="group" aria-label="Actions">
      {actions.map((action) => (
        <button key={action.id} type="button" data-testid={action.testId} disabled={action.disabled} onClick={action.onSelect}>
          {action.label}
        </button>
      ))}
    </div>
  ),
}))

describe('ConfirmDialog action footer', () => {
  it('uses one shared row and preserves danger, confirm, cancel, and loading behavior', () => {
    const onClose = vi.fn()
    const onConfirm = vi.fn()
    const { rerender } = render(
      <ConfirmDialog
        isOpen
        onClose={onClose}
        onConfirm={onConfirm}
        title="Delete prompt?"
        message="This cannot be undone."
        confirmLabel="Delete"
        variant="danger"
      />,
    )

    const actionGroup = screen.getByRole('group', { name: 'Actions' })
    const actionBar = actionGroup
    const deleteButton = within(actionGroup).getByRole('button', { name: 'Delete' })
    expect(actionBar.hasAttribute('data-rcl-overlay-action-bar')).toBe(true)
    expect(actionBar.className).toContain('prompt-overlay-actions--danger')
    expect(actionBar.querySelectorAll(':scope > button')).toHaveLength(2)
    fireEvent.click(deleteButton)
    expect(onConfirm).toHaveBeenCalledOnce()

    rerender(
      <ConfirmDialog
        isOpen
        onClose={onClose}
        onConfirm={onConfirm}
        title="Delete prompt?"
        message="This cannot be undone."
        confirmLabel="Delete"
        variant="danger"
        isLoading
      />,
    )
    expect(within(screen.getByRole('group', { name: 'Actions' })).getByRole('button', { name: 'Processing...' })).toBeDisabled()
    expect(within(screen.getByRole('group', { name: 'Actions' })).getByRole('button', { name: 'Cancel' })).toBeDisabled()
  })
})
