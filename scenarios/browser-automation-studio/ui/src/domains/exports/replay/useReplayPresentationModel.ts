import { useMemo } from 'react';
import type { ReplayStyleConfig, ReplayStyleOverrides } from '@/domains/replay-style';
import {
  normalizeReplayStyle,
  resolveReplayPresentationStyle,
  resolveReplayStyleTokens,
  useResolvedReplayBackground,
} from '@/domains/replay-style';
import { computeReplayLayout, type ReplayFitMode, type ReplayLayoutModel } from '@/domains/replay-layout';
import type { BackgroundDecor, ChromeDecor, CursorDecor } from '@/domains/replay-style';
import type { Dimensions } from './types';

export interface ReplayPresentationModelParams {
  style: ReplayStyleOverrides;
  title: string;
  canvasDimensions: Dimensions;
  viewportDimensions: Dimensions;
  presentationBounds?: Dimensions;
  presentationFit?: ReplayFitMode;
}

export interface ReplayPresentationModel {
  style: ReplayStyleConfig;
  resolvedStyle: ReplayStyleConfig;
  backgroundDecor: BackgroundDecor;
  chromeDecor: ChromeDecor;
  cursorDecor: CursorDecor;
  layout: ReplayLayoutModel;
}

export const useReplayPresentationModel = ({
  style,
  title,
  canvasDimensions,
  viewportDimensions,
  presentationBounds,
  presentationFit,
}: ReplayPresentationModelParams): ReplayPresentationModel => {
  const normalizedStyle = useMemo(() => normalizeReplayStyle(style), [style]);
  const resolvedBackground = useResolvedReplayBackground(normalizedStyle.background);
  const resolvedStyle = useMemo(
    () => resolveReplayPresentationStyle({ ...normalizedStyle, background: resolvedBackground }),
    [normalizedStyle, resolvedBackground],
  );

  const { backgroundDecor, chromeDecor, cursorDecor } = useMemo(
    () => resolveReplayStyleTokens(resolvedStyle, { title }),
    [resolvedStyle, title],
  );

  const layout = useMemo(
    () =>
      computeReplayLayout({
        canvas: canvasDimensions,
        viewport: viewportDimensions,
        browserScale: resolvedStyle.browserScale,
        chromeHeaderHeight: chromeDecor.headerHeight,
        contentInset: backgroundDecor.contentInset,
        container: presentationBounds,
        fit: presentationFit ?? (presentationBounds ? 'contain' : 'none'),
      }),
    [
      canvasDimensions,
      viewportDimensions,
      resolvedStyle.browserScale,
      chromeDecor.headerHeight,
      backgroundDecor.contentInset,
      presentationBounds,
      presentationFit,
    ],
  );

  return {
    style: normalizedStyle,
    resolvedStyle,
    backgroundDecor,
    chromeDecor,
    cursorDecor,
    layout,
  };
};
