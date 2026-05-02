import { describe, it, expect, vi } from 'vitest'
import { render, screen, fireEvent } from '@testing-library/react'
import { TopicsValidationPanel } from './TopicsValidationPanel'
import type { TopicValidation } from '@/types/topicsGraph'

function makeValidation(): TopicValidation {
  return {
    findings: [
      {
        rule: 'orphan_input',
        severity: 'error',
        member: { team: 'marketing-crew', member: 'researcher' },
        prefix: 'research-inbox/audience/*',
        detail: 'No producer writes to this prefix',
      },
      {
        rule: 'orphan_output',
        severity: 'warning',
        member: { team: 'marketing-crew', member: 'brand-manager' },
        prefix: 'marketing-canon/*',
        detail: 'Output prefix has no documented consumer',
      },
    ],
    errors: 1,
    warnings: 1,
  }
}

describe('TopicsValidationPanel', () => {
  it('renders error and warning rows grouped by severity', () => {
    render(<TopicsValidationPanel validation={makeValidation()} />)
    expect(screen.getByTestId('topics-validation-panel')).toBeInTheDocument()
    expect(screen.getByText(/Errors \(1\)/)).toBeInTheDocument()
    expect(screen.getByText(/Warnings \(1\)/)).toBeInTheDocument()
    expect(screen.getByTestId('topics-finding-error-orphan_input')).toBeInTheDocument()
    expect(screen.getByTestId('topics-finding-warning-orphan_output')).toBeInTheDocument()
  })

  it('shows clean state when no findings', () => {
    render(
      <TopicsValidationPanel
        validation={{ findings: [], errors: 0, warnings: 0 }}
      />,
    )
    expect(screen.getByText('No findings')).toBeInTheDocument()
  })

  it('calls onSelectMember when a finding is clicked', () => {
    const onSelectMember = vi.fn()
    render(
      <TopicsValidationPanel
        validation={makeValidation()}
        onSelectMember={onSelectMember}
      />,
    )
    fireEvent.click(screen.getByTestId('topics-finding-error-orphan_input'))
    expect(onSelectMember).toHaveBeenCalledWith('marketing-crew', 'researcher')
  })

  it('shows the error and warning counts in the header', () => {
    render(<TopicsValidationPanel validation={makeValidation()} />)
    expect(screen.getByText(/1E · 1W/)).toBeInTheDocument()
  })
})
