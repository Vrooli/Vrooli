import { describe, expect, it } from 'vitest';
import { render, screen, within } from '@testing-library/react';
import ReplayPlayer from './ReplayPlayer';
import type { ReplayFrame } from './types';

describe('ReplayPlayer', () => {
  it('renders a frame with the replay header and screenshot', () => {
    const frames: ReplayFrame[] = [
      {
        id: 'frame-1',
        stepIndex: 0,
        success: true,
        finalUrl: 'https://example.com',
        screenshot: {
          artifactId: 'shot-1',
          url: 'https://example.com/screenshot.png',
          width: 1280,
          height: 720,
        },
      },
    ];

    render(<ReplayPlayer frames={frames} autoPlay={false} loop={false} />);

    expect(screen.getByText('Replay')).toBeInTheDocument();
    // URL may appear in multiple places (address bar, tooltips, etc.)
    expect(screen.getAllByText('https://example.com').length).toBeGreaterThan(0);
    expect(screen.getByAltText('Step 1')).toBeInTheDocument();
  });

  it('sizes the viewport from the layout model', () => {
    const frames: ReplayFrame[] = [
      {
        id: 'frame-1',
        stepIndex: 0,
        success: true,
        screenshot: {
          artifactId: 'shot-1',
          url: 'https://example.com/screenshot.png',
          width: 1280,
          height: 720,
        },
      },
    ];

    render(
      <ReplayPlayer
        frames={frames}
        autoPlay={false}
        loop={false}
        replayStyle={{ browserScale: 0.6 }}
        presentationDimensions={{ width: 1280, height: 720 }}
      />,
    );

    // browserScale 0.6 applied to 1280x720 viewport = 768x432
    const viewport = screen.getByTestId('replay-viewport');
    expect(viewport).toHaveStyle({ width: '768px', height: '432px' });
  });

  it('renders controls outside the presentation frame', () => {
    const frames: ReplayFrame[] = [
      {
        id: 'frame-1',
        stepIndex: 0,
        success: true,
        screenshot: {
          artifactId: 'shot-1',
          url: 'https://example.com/screenshot.png',
          width: 1280,
          height: 720,
        },
      },
    ];

    render(<ReplayPlayer frames={frames} autoPlay={false} loop={false} />);

    const presentation = screen.getByTestId('replay-presentation');
    expect(within(presentation).queryByLabelText('Play replay')).toBeNull();
    expect(screen.getByLabelText('Play replay')).toBeInTheDocument();
  });
});
