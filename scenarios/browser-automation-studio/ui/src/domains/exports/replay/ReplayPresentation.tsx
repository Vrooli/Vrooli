import type { CSSProperties, ReactNode, Ref } from 'react';
import type { ReplayRect } from '@/domains/replay-layout';
import { ReplayStyleFrame } from '@/domains/replay-style';
import type { ReplayLayoutModel } from '@/domains/replay-layout';
import type { ReplayPresentationModel } from './useReplayPresentationModel';

type ReplayPresentationChild = ReactNode | ((layout: ReplayLayoutModel) => ReactNode);

interface ReplayPresentationProps {
  model: ReplayPresentationModel;
  overlayNode?: ReactNode;
  overlayTransformStyle?: CSSProperties;
  viewportContentRect?: ReplayRect;
  presentationStyle?: CSSProperties;
  presentationClassName?: string;
  containerClassName?: string;
  contentClassName?: string;
  captureAreaRef?: Ref<HTMLDivElement>;
  presentationRef?: Ref<HTMLDivElement>;
  viewportRef?: Ref<HTMLDivElement>;
  browserFrameRef?: Ref<HTMLDivElement>;
  children: ReplayPresentationChild;
}

export function ReplayPresentation({
  model,
  overlayNode,
  overlayTransformStyle,
  viewportContentRect,
  presentationStyle,
  presentationClassName,
  containerClassName,
  contentClassName,
  captureAreaRef,
  presentationRef,
  viewportRef,
  browserFrameRef,
  children,
}: ReplayPresentationProps) {
  const content = typeof children === 'function' ? children(model.layout) : children;

  return (
    <ReplayStyleFrame
      backgroundDecor={model.backgroundDecor}
      chromeDecor={model.chromeDecor}
      layout={model.layout}
      overlayNode={overlayNode}
      overlayTransformStyle={overlayTransformStyle}
      viewportContentRect={viewportContentRect}
      presentationStyle={presentationStyle}
      presentationClassName={presentationClassName}
      containerClassName={containerClassName}
      contentClassName={contentClassName}
      captureAreaRef={captureAreaRef}
      presentationRef={presentationRef}
      viewportRef={viewportRef}
      browserFrameRef={browserFrameRef}
    >
      {content}
    </ReplayStyleFrame>
  );
}

export default ReplayPresentation;
