/**
 * TimelineTab Component Tests
 */

import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { TimelineTab, type TimelineTabProps } from './TimelineTab';
import type { TimelineItem } from '../types/timeline-unified';

// Mock UnifiedTimeline to avoid testing its internals
vi.mock('../timeline/TimelineEventCard', () => ({
  UnifiedTimeline: vi.fn(({ items, isLoading, isLive, mode, isSelectionMode }) => (
    <div data-testid="unified-timeline">
      <span data-testid="item-count">{items.length}</span>
      <span data-testid="is-loading">{isLoading ? 'loading' : 'ready'}</span>
      <span data-testid="is-live">{isLive ? 'live' : 'paused'}</span>
      <span data-testid="mode">{mode}</span>
      <span data-testid="selection-mode">{isSelectionMode ? 'selecting' : 'normal'}</span>
    </div>
  )),
}));

describe('TimelineTab', () => {
  const defaultProps: TimelineTabProps = {
    actions: [],
    isRecording: false,
    isLoading: false,
    isReplaying: false,
    hasUnstableSelectors: false,
    onClearRequested: vi.fn(),
    onCreateWorkflow: vi.fn(),
    isSelectionMode: false,
    selectedIndices: new Set(),
    onToggleSelectionMode: vi.fn(),
    onActionClick: vi.fn(),
    onSelectAll: vi.fn(),
    onSelectNone: vi.fn(),
  };

  const mockTimelineItems: TimelineItem[] = [
    {
      id: 'item-1',
      type: 'action',
      timestamp: new Date('2024-01-01'),
      actionType: 'click',
      source: 'recording',
    },
    {
      id: 'item-2',
      type: 'action',
      timestamp: new Date('2024-01-01'),
      actionType: 'input',
      source: 'recording',
    },
  ];

  beforeEach(() => {
    vi.clearAllMocks();
  });

  describe('rendering', () => {
    it('should render with empty state', () => {
      render(<TimelineTab {...defaultProps} />);

      expect(screen.getByText('0 steps')).toBeInTheDocument();
      expect(screen.getByTestId('unified-timeline')).toBeInTheDocument();
    });

    it('should show correct step count from timelineItems', () => {
      render(<TimelineTab {...defaultProps} timelineItems={mockTimelineItems} />);

      expect(screen.getByText('2 steps')).toBeInTheDocument();
    });

    it('should show correct step count from actions when no timelineItems', () => {
      const actions = [
        { id: '1', type: 'click' },
        { id: '2', type: 'input' },
        { id: '3', type: 'scroll' },
      ] as never[];

      render(<TimelineTab {...defaultProps} actions={actions} />);

      expect(screen.getByText('3 steps')).toBeInTheDocument();
    });

    it('should use itemCountOverride when provided', () => {
      render(<TimelineTab {...defaultProps} timelineItems={mockTimelineItems} itemCountOverride={10} />);

      expect(screen.getByText('10 steps')).toBeInTheDocument();
    });

    it('should show singular "step" for single item', () => {
      render(<TimelineTab {...defaultProps} timelineItems={[mockTimelineItems[0]]} />);

      expect(screen.getByText('1 step')).toBeInTheDocument();
    });
  });

  describe('selection mode', () => {
    it('should toggle selection mode button', async () => {
      const user = userEvent.setup();
      const onToggleSelectionMode = vi.fn();

      render(
        <TimelineTab
          {...defaultProps}
          timelineItems={mockTimelineItems}
          onToggleSelectionMode={onToggleSelectionMode}
        />
      );

      const selectButton = screen.getByRole('button', { name: /Select steps/i });
      await user.click(selectButton);

      expect(onToggleSelectionMode).toHaveBeenCalled();
    });

    it('should disable select button when no items', () => {
      render(<TimelineTab {...defaultProps} />);

      const selectButton = screen.getByRole('button', { name: /Select steps/i });
      expect(selectButton).toBeDisabled();
    });

    it('should show All/None buttons when in selection mode', () => {
      render(<TimelineTab {...defaultProps} timelineItems={mockTimelineItems} isSelectionMode={true} />);

      // In selection mode, All and None buttons appear (with aria-labels)
      expect(screen.getByRole('button', { name: 'Select all' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Deselect all' })).toBeInTheDocument();
    });

    it('should call onSelectAll when All button clicked', async () => {
      const user = userEvent.setup();
      const onSelectAll = vi.fn();

      render(
        <TimelineTab
          {...defaultProps}
          timelineItems={mockTimelineItems}
          isSelectionMode={true}
          onSelectAll={onSelectAll}
        />
      );

      await user.click(screen.getByRole('button', { name: 'Select all' }));

      expect(onSelectAll).toHaveBeenCalled();
    });

    it('should call onSelectNone when None button clicked', async () => {
      const user = userEvent.setup();
      const onSelectNone = vi.fn();

      render(
        <TimelineTab
          {...defaultProps}
          timelineItems={mockTimelineItems}
          isSelectionMode={true}
          onSelectNone={onSelectNone}
        />
      );

      await user.click(screen.getByRole('button', { name: 'Deselect all' }));

      expect(onSelectNone).toHaveBeenCalled();
    });

    it('should show selection count badge when items selected', () => {
      render(
        <TimelineTab
          {...defaultProps}
          timelineItems={mockTimelineItems}
          isSelectionMode={true}
          selectedIndices={new Set([0, 1])}
        />
      );

      expect(screen.getByText('2 selected')).toBeInTheDocument();
    });
  });

  describe('clear button', () => {
    it('should show Clear button in recording mode with items', () => {
      render(<TimelineTab {...defaultProps} mode="recording" timelineItems={mockTimelineItems} />);

      expect(screen.getByRole('button', { name: /Clear timeline/i })).toBeInTheDocument();
    });

    it('should not show Clear button in selection mode', () => {
      render(
        <TimelineTab
          {...defaultProps}
          mode="recording"
          timelineItems={mockTimelineItems}
          isSelectionMode={true}
        />
      );

      expect(screen.queryByRole('button', { name: /Clear timeline/i })).not.toBeInTheDocument();
    });

    it('should not show Clear button when no items', () => {
      render(<TimelineTab {...defaultProps} mode="recording" />);

      expect(screen.queryByRole('button', { name: /Clear timeline/i })).not.toBeInTheDocument();
    });

    it('should call onClearRequested when clicked', async () => {
      const user = userEvent.setup();
      const onClearRequested = vi.fn();

      render(
        <TimelineTab
          {...defaultProps}
          mode="recording"
          timelineItems={mockTimelineItems}
          onClearRequested={onClearRequested}
        />
      );

      await user.click(screen.getByRole('button', { name: /Clear timeline/i }));

      expect(onClearRequested).toHaveBeenCalled();
    });
  });

  describe('status indicators', () => {
    it('should show "Review selectors" badge when hasUnstableSelectors', () => {
      render(
        <TimelineTab {...defaultProps} timelineItems={mockTimelineItems} hasUnstableSelectors={true} />
      );

      expect(screen.getByText('Review selectors')).toBeInTheDocument();
    });

    it('should not show "Review selectors" when in selection mode', () => {
      render(
        <TimelineTab
          {...defaultProps}
          timelineItems={mockTimelineItems}
          hasUnstableSelectors={true}
          isSelectionMode={true}
        />
      );

      expect(screen.queryByText('Review selectors')).not.toBeInTheDocument();
    });

    it('should show "Executing" badge in execution mode', () => {
      render(<TimelineTab {...defaultProps} mode="execution" />);

      expect(screen.getByText('Executing')).toBeInTheDocument();
    });
  });

  describe('AI Navigation button', () => {
    it('should show AI Navigation button when callback provided', () => {
      render(
        <TimelineTab
          {...defaultProps}
          mode="recording"
          timelineItems={mockTimelineItems}
          onAINavigation={() => {}}
        />
      );

      expect(screen.getByRole('button', { name: /AI Navigation/i })).toBeInTheDocument();
    });

    it('should not show AI Navigation button when callback not provided', () => {
      render(<TimelineTab {...defaultProps} mode="recording" timelineItems={mockTimelineItems} />);

      expect(screen.queryByRole('button', { name: /AI Navigation/i })).not.toBeInTheDocument();
    });

    it('should not show AI Navigation button in execution mode', () => {
      render(
        <TimelineTab
          {...defaultProps}
          mode="execution"
          timelineItems={mockTimelineItems}
          onAINavigation={() => {}}
        />
      );

      expect(screen.queryByRole('button', { name: /AI Navigation/i })).not.toBeInTheDocument();
    });

    it('should call onAINavigation when clicked', async () => {
      const user = userEvent.setup();
      const onAINavigation = vi.fn();

      render(
        <TimelineTab
          {...defaultProps}
          mode="recording"
          timelineItems={mockTimelineItems}
          onAINavigation={onAINavigation}
        />
      );

      await user.click(screen.getByRole('button', { name: /AI Navigation/i }));

      expect(onAINavigation).toHaveBeenCalled();
    });
  });

  describe('Create Workflow button', () => {
    it('should show Create Workflow button when items exist', () => {
      render(<TimelineTab {...defaultProps} mode="recording" timelineItems={mockTimelineItems} />);

      expect(screen.getByRole('button', { name: /Create Workflow/i })).toBeInTheDocument();
    });

    it('should not show Create Workflow button when no items', () => {
      render(<TimelineTab {...defaultProps} mode="recording" />);

      expect(screen.queryByRole('button', { name: /Create Workflow/i })).not.toBeInTheDocument();
    });

    it('should not show Create Workflow button in execution mode', () => {
      render(<TimelineTab {...defaultProps} mode="execution" timelineItems={mockTimelineItems} />);

      expect(screen.queryByRole('button', { name: /Create Workflow/i })).not.toBeInTheDocument();
    });

    it('should disable Create Workflow when replaying', () => {
      render(
        <TimelineTab {...defaultProps} mode="recording" timelineItems={mockTimelineItems} isReplaying={true} />
      );

      expect(screen.getByRole('button', { name: /Create Workflow/i })).toBeDisabled();
    });

    it('should disable Create Workflow when loading', () => {
      render(
        <TimelineTab {...defaultProps} mode="recording" timelineItems={mockTimelineItems} isLoading={true} />
      );

      expect(screen.getByRole('button', { name: /Create Workflow/i })).toBeDisabled();
    });

    it('should show selection count in button text when items selected', () => {
      render(
        <TimelineTab
          {...defaultProps}
          mode="recording"
          timelineItems={mockTimelineItems}
          isSelectionMode={true}
          selectedIndices={new Set([0, 1])}
        />
      );

      expect(screen.getByText(/Create Workflow \(2 steps\)/i)).toBeInTheDocument();
    });

    it('should show "Select steps" message when in selection mode with nothing selected', () => {
      render(
        <TimelineTab
          {...defaultProps}
          mode="recording"
          timelineItems={mockTimelineItems}
          isSelectionMode={true}
          selectedIndices={new Set()}
        />
      );

      expect(screen.getByText('Select steps to create workflow')).toBeInTheDocument();
    });

    it('should disable Create Workflow in selection mode with nothing selected', () => {
      render(
        <TimelineTab
          {...defaultProps}
          mode="recording"
          timelineItems={mockTimelineItems}
          isSelectionMode={true}
          selectedIndices={new Set()}
        />
      );

      expect(screen.getByRole('button', { name: /Create Workflow/i })).toBeDisabled();
    });

    it('should call onCreateWorkflow when clicked', async () => {
      const user = userEvent.setup();
      const onCreateWorkflow = vi.fn();

      render(
        <TimelineTab
          {...defaultProps}
          mode="recording"
          timelineItems={mockTimelineItems}
          onCreateWorkflow={onCreateWorkflow}
        />
      );

      await user.click(screen.getByRole('button', { name: /Create Workflow/i }));

      expect(onCreateWorkflow).toHaveBeenCalled();
    });

    it('should show helper text when not in selection mode', () => {
      render(<TimelineTab {...defaultProps} mode="recording" timelineItems={mockTimelineItems} />);

      // Check the helper text paragraph exists
      const helperText = screen.getByText(/to choose specific steps/i);
      expect(helperText).toBeInTheDocument();
      expect(helperText.textContent).toContain('Use');
      expect(helperText.textContent).toContain('Select');
    });
  });

  describe('UnifiedTimeline integration', () => {
    it('should pass correct props to UnifiedTimeline', () => {
      render(
        <TimelineTab
          {...defaultProps}
          timelineItems={mockTimelineItems}
          isLoading={true}
          isLive={true}
          mode="execution"
          isSelectionMode={true}
        />
      );

      expect(screen.getByTestId('item-count')).toHaveTextContent('2');
      expect(screen.getByTestId('is-loading')).toHaveTextContent('loading');
      expect(screen.getByTestId('is-live')).toHaveTextContent('live');
      expect(screen.getByTestId('mode')).toHaveTextContent('execution');
      expect(screen.getByTestId('selection-mode')).toHaveTextContent('selecting');
    });

    it('should use isRecording as isLive when isLive not provided', () => {
      render(<TimelineTab {...defaultProps} timelineItems={mockTimelineItems} isRecording={true} />);

      expect(screen.getByTestId('is-live')).toHaveTextContent('live');
    });
  });

  describe('className prop', () => {
    it('should apply custom className', () => {
      const { container } = render(<TimelineTab {...defaultProps} className="custom-class" />);

      expect(container.firstChild).toHaveClass('custom-class');
    });
  });
});
