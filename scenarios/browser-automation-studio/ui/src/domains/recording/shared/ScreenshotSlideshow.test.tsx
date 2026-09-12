import { screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { selectors } from '@/constants/selectors';
import { renderWithProviders } from '@/test-utils';
import { ScreenshotSlideshow } from './ScreenshotSlideshow';

describe('ScreenshotSlideshow', () => {
  it('exposes the replay selector contract when a screenshot is displayed', () => {
    renderWithProviders(
      <ScreenshotSlideshow
        currentIndex={0}
        screenshots={[{ url: 'https://example.test/frame.png', stepLabel: 'Navigate' }]}
      />,
    );

    expect(screen.getByTestId(selectors.replay.player)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.replay.screenshot)).toHaveAttribute('src', 'https://example.test/frame.png');
  });
});
