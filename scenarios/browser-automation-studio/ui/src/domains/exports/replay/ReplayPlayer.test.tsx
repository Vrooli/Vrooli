import { describe, expect, it } from 'vitest';
import { render, screen } from '@testing-library/react';
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
    expect(screen.getByText('https://example.com')).toBeInTheDocument();
    expect(screen.getByAltText('Step 1')).toBeInTheDocument();
  });
});
