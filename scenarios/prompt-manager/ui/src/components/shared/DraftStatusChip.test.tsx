import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { DraftStatusChip } from './DraftStatusChip'

describe('DraftStatusChip', () => {
  it('renders Draft label when isDraft is true', () => {
    render(<DraftStatusChip isDraft onChange={vi.fn()} />)
    expect(screen.getByText('Draft')).toBeInTheDocument()
  })

  it('renders Published label when isDraft is false', () => {
    render(<DraftStatusChip isDraft={false} onChange={vi.fn()} />)
    expect(screen.getByText('Published')).toBeInTheDocument()
  })

  it('aria-pressed reflects the inverse of isDraft (pressed = published)', () => {
    const { rerender } = render(<DraftStatusChip isDraft onChange={vi.fn()} />)
    expect(screen.getByRole('button')).toHaveAttribute('aria-pressed', 'false')
    rerender(<DraftStatusChip isDraft={false} onChange={vi.fn()} />)
    expect(screen.getByRole('button')).toHaveAttribute('aria-pressed', 'true')
  })

  it('calls onChange with inverted value when clicked', () => {
    const onChange = vi.fn()
    render(<DraftStatusChip isDraft onChange={onChange} />)
    fireEvent.click(screen.getByRole('button'))
    expect(onChange).toHaveBeenCalledWith(false)
  })

  it('does not call onChange when disabled', () => {
    const onChange = vi.fn()
    render(<DraftStatusChip isDraft onChange={onChange} disabled />)
    fireEvent.click(screen.getByRole('button'))
    expect(onChange).not.toHaveBeenCalled()
  })

  it('renders a skeleton when loading', () => {
    const { container } = render(
      <DraftStatusChip isDraft onChange={vi.fn()} isLoading />
    )
    expect(container.querySelector('button')).toBeNull()
  })
})
