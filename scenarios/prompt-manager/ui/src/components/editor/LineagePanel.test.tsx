import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, fireEvent } from '@/test-utils/renderWithProviders'
import { LineagePanel } from './LineagePanel'

type CapturedProps = { skillId?: string; currentContent?: string }
const captured: { history: CapturedProps; variants: CapturedProps; experiments: CapturedProps } = {
  history: {},
  variants: {},
  experiments: {},
}

vi.mock('./tabs/VersionHistoryTab', () => ({
  VersionHistoryTab: (props: { skillId: string }) => {
    captured.history = { skillId: props.skillId }
    return <div data-testid="mock-history">history:{props.skillId}</div>
  },
}))

vi.mock('./VariantPanel', () => ({
  VariantPanel: (props: { skillId: string; currentContent?: string }) => {
    captured.variants = { skillId: props.skillId, currentContent: props.currentContent }
    return <div data-testid="mock-variants">variants:{props.skillId}</div>
  },
}))

vi.mock('./ExperimentPanel', () => ({
  ExperimentPanel: (props: { skillId: string }) => {
    captured.experiments = { skillId: props.skillId }
    return <div data-testid="mock-experiments">experiments:{props.skillId}</div>
  },
}))

describe('LineagePanel', () => {
  beforeEach(() => {
    captured.history = {}
    captured.variants = {}
    captured.experiments = {}
  })

  it('renders all three tab triggers', () => {
    render(
      <LineagePanel
        skillId="s1"
        currentContent="hello"
        activeTab="history"
        onActiveTabChange={vi.fn()}
      />
    )
    expect(screen.getByTestId('lineage-tab-history')).toBeInTheDocument()
    expect(screen.getByTestId('lineage-tab-variants')).toBeInTheDocument()
    expect(screen.getByTestId('lineage-tab-experiments')).toBeInTheDocument()
  })

  it('forwards skillId and currentContent to child panels', () => {
    render(
      <LineagePanel
        skillId="s42"
        currentContent="body"
        activeTab="history"
        onActiveTabChange={vi.fn()}
      />
    )
    expect(captured.history.skillId).toBe('s42')
    expect(captured.variants.skillId).toBe('s42')
    expect(captured.variants.currentContent).toBe('body')
    expect(captured.experiments.skillId).toBe('s42')
  })

  it('calls onActiveTabChange when a different tab is clicked', () => {
    const onChange = vi.fn()
    render(
      <LineagePanel
        skillId="s1"
        currentContent=""
        activeTab="history"
        onActiveTabChange={onChange}
      />
    )
    fireEvent.mouseDown(screen.getByTestId('lineage-tab-variants'))
    expect(onChange).toHaveBeenCalledWith('variants')
  })

  it('exposes the active tab through aria state', () => {
    const { rerender } = render(
      <LineagePanel
        skillId="s1"
        currentContent=""
        activeTab="history"
        onActiveTabChange={vi.fn()}
      />
    )
    expect(screen.getByTestId('lineage-tab-history')).toHaveAttribute('data-state', 'active')
    expect(screen.getByTestId('lineage-tab-variants')).toHaveAttribute('data-state', 'inactive')

    rerender(
      <LineagePanel
        skillId="s1"
        currentContent=""
        activeTab="experiments"
        onActiveTabChange={vi.fn()}
      />
    )
    expect(screen.getByTestId('lineage-tab-experiments')).toHaveAttribute('data-state', 'active')
  })
})
