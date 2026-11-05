import { fireEvent, render, screen } from '@/test-utils/renderWithProviders'
import { describe, expect, it, vi } from 'vitest'
import { ExpandableDescription } from './ExpandableDescription'

describe('ExpandableDescription', () => {
  it('keeps essential copy readable when truncation is explicitly disabled', () => {
    render(
      <ExpandableDescription
        value="A long mission that explains the complete operator outcome and must remain visible on a narrow dashboard."
        onChange={vi.fn()}
        maxLines={0}
      />,
    )

    const description = screen.getByRole('button', { name: 'Edit description' })
    expect(description).not.toHaveClass('line-clamp-2')
    expect(description).toHaveTextContent('must remain visible on a narrow dashboard')
  })

  it('retains keyboard editing for the untruncated variant', () => {
    render(<ExpandableDescription value="Mission" onChange={vi.fn()} maxLines={0} />)

    fireEvent.keyDown(screen.getByRole('button', { name: 'Edit description' }), { key: 'Enter' })

    expect(screen.getByRole('textbox')).toHaveValue('Mission')
  })
})
