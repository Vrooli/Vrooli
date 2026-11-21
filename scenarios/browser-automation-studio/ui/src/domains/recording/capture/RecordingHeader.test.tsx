import { fireEvent, render, screen } from '@/test-utils';
import { describe, expect, it, vi } from 'vitest';
import { RecordingHeader } from './RecordingHeader';

vi.mock('../stores/sessionStore', () => ({
  useRecordingConnectionStatus: () => ({ isConnected: true, isWebSocket: true }),
}));

describe('RecordingHeader stop control', () => {
  it('offers start only after the workspace permits recording and disables duplicate requests', () => {
    const start = vi.fn();
    const { rerender } = render(
      <RecordingHeader isRecording={false} mode="recording" onStartRecording={start} canStartRecording={false} />,
    );
    expect(screen.queryByRole('button', { name: 'Start recording' })).not.toBeInTheDocument();

    rerender(
      <RecordingHeader isRecording={false} mode="recording" onStartRecording={start} canStartRecording />,
    );
    fireEvent.click(screen.getByRole('button', { name: 'Start recording' }));
    expect(start).toHaveBeenCalledTimes(1);

    rerender(
      <RecordingHeader isRecording={false} mode="recording" onStartRecording={start} canStartRecording isRecordingStarting />,
    );
    expect(screen.getByRole('button', { name: 'Starting recording' })).toBeDisabled();
  });

  it('exposes stop while recording and disables duplicate requests in flight', () => {
    const stop = vi.fn();
    const { rerender } = render(
      <RecordingHeader isRecording mode="recording" onStopRecording={stop} />,
    );

    const button = screen.getByRole('button', { name: 'Stop recording' });
    fireEvent.click(button);
    expect(stop).toHaveBeenCalledTimes(1);

    rerender(
      <RecordingHeader isRecording mode="recording" onStopRecording={stop} isRecordingStopping />,
    );
    expect(screen.getByRole('button', { name: 'Stopping recording' })).toBeDisabled();
  });

  it('does not expose a recording stop control outside recording mode', () => {
    render(<RecordingHeader isRecording mode="execution" onStopRecording={vi.fn()} />);
    expect(screen.queryByRole('button', { name: /stop recording/i })).not.toBeInTheDocument();
  });
});
