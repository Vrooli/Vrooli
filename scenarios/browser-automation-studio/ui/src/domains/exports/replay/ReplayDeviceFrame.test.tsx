import { render, screen } from '@/test-utils';
import { describe, expect, it } from 'vitest';
import { computeReplayLayout } from '@/domains/replay-layout';
import { normalizeReplayStyle } from '@/domains/replay-style/model';
import { resolveReplayStyleTokens } from '@/domains/replay-style/resolver';
import { ReplayStyleFrame } from '@/domains/replay-style/renderer/ReplayStyleFrame';

describe('device frame around the product window', () => {
  it('puts a visible bezel around the source instead of outside the export canvas', () => {
    const style = normalizeReplayStyle({ presentation: { showDesktop: true, showBrowserFrame: false, showDeviceFrame: true }, deviceFrameTheme: 'slim', browserScale: 1 });
    const tokens = resolveReplayStyleTokens(style, { title: 'Browser' });
    const layout = computeReplayLayout({ canvas: { width: 400, height: 700 }, viewport: { width: 160, height: 320 }, browserScale: 1, frameInset: tokens.deviceFrameDecor?.frameInset });
    render(<ReplayStyleFrame {...tokens} layout={layout}><div>Actual source</div></ReplayStyleFrame>);
    const frame = screen.getByTestId('replay-frame');
    const bezel = screen.getByTestId('replay-device-bezel');
    expect(bezel.parentElement).toBe(frame);
    expect(bezel).toHaveStyle({ left: '-6px', top: '-6px' });
    expect(layout.frameRect.x).toBeGreaterThanOrEqual(6);
    expect(layout.frameRect.y).toBeGreaterThanOrEqual(6);
    expect(layout.frameRect.y + layout.frameRect.height + 6).toBeLessThanOrEqual(700);
    expect(layout.viewportRect.height / layout.viewportRect.width).toBe(2);
    expect(screen.getByText('Actual source')).toBeInTheDocument();
  });
});
