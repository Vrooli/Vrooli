import { fireEvent, render, screen } from '@/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { ActionTimeline } from './ActionTimeline';
import type { RecordedAction } from '../types/types';

const actions: RecordedAction[] = [
  {
    id: 'first', sessionId: 'session', sequenceNum: 1, timestamp: '2026-09-30T12:00:00Z',
    actionType: 'input', confidence: 1, url: 'https://example.test', payload: { text: 'one' },
  },
  {
    id: 'second', sessionId: 'session', sequenceNum: 2, timestamp: '2026-09-30T12:00:01Z',
    actionType: 'input', confidence: 1, url: 'https://example.test', payload: { text: 'two' },
  },
];

describe('ActionTimeline journal projection', () => {
  it('renders consecutive actions as separate rows in their recorded order', () => {
    render(<ActionTimeline actions={actions} isRecording={false} />);

    const first = screen.getByText('Type: "one"');
    const second = screen.getByText('Type: "two"');
    expect(first.compareDocumentPosition(second) & Node.DOCUMENT_POSITION_FOLLOWING).toBeTruthy();
    expect(screen.queryByText(/merged actions/i)).not.toBeInTheDocument();
  });

  it('routes delete and selection interactions to the matching raw action index', () => {
    const onDeleteAction = vi.fn();
    const onActionClick = vi.fn();
    render(
      <ActionTimeline
        actions={actions}
        isRecording={false}
        onDeleteAction={onDeleteAction}
        isSelectionMode
        selectedIndices={new Set()}
        onActionClick={onActionClick}
      />,
    );

    fireEvent.click(screen.getAllByTitle('Delete action')[1]!);
    fireEvent.click(screen.getAllByRole('checkbox')[1]!);

    expect(onDeleteAction).toHaveBeenCalledTimes(1);
    expect(onDeleteAction).toHaveBeenCalledWith(1);
    expect(onActionClick).toHaveBeenCalledWith(1, false, false);
  });
});
