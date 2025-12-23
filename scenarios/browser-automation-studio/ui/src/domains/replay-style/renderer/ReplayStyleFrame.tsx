import { useEffect, useRef, type CSSProperties, type Ref, type ReactNode } from 'react';
import clsx from 'clsx';
import type { BackgroundDecor, ChromeDecor } from '../catalog';
import type { ReplayLayoutModel, ReplayRect } from '@/domains/replay-layout';
import { OverlayRegistry, OverlayRegistryContext } from '@/domains/replay-positioning';

interface ReplayStyleFrameProps {
  backgroundDecor: BackgroundDecor;
  chromeDecor: ChromeDecor;
  layout: ReplayLayoutModel;
  presentationStyle?: CSSProperties;
  presentationClassName?: string;
  containerStyle?: CSSProperties;
  overlayNode?: ReactNode;
  containerClassName?: string;
  contentClassName?: string;
  captureAreaRef?: Ref<HTMLDivElement>;
  presentationRef?: Ref<HTMLDivElement>;
  viewportRef?: Ref<HTMLDivElement>;
  browserFrameRef?: Ref<HTMLDivElement>;
  viewportContentRect?: ReplayRect;
  overlayTransformStyle?: CSSProperties;
  children: ReactNode;
}

export function ReplayStyleFrame({
  backgroundDecor,
  chromeDecor,
  layout,
  presentationStyle,
  presentationClassName,
  containerStyle,
  overlayNode,
  containerClassName,
  contentClassName,
  captureAreaRef,
  presentationRef,
  viewportRef,
  browserFrameRef,
  viewportContentRect,
  overlayTransformStyle,
  children,
}: ReplayStyleFrameProps) {
  const chromeHeaderHeight = chromeDecor.header ? chromeDecor.headerHeight : 0;
  const contentInset = layout.contentInset ?? { x: 0, y: 0 };
  const contentStyle =
    contentInset.x > 0 || contentInset.y > 0
      ? { padding: `${contentInset.y}px ${contentInset.x}px` }
      : undefined;
  const registryRef = useRef<OverlayRegistry | null>(null);
  if (!registryRef.current) {
    registryRef.current = new OverlayRegistry();
  }

  useEffect(() => {
    const registry = registryRef.current;
    if (!registry) {
      return;
    }
    registry.setRect('overlay-root', {
      x: 0,
      y: 0,
      width: layout.viewportRect.width,
      height: layout.viewportRect.height,
    });
    const contentRect = viewportContentRect ?? {
      x: 0,
      y: 0,
      width: layout.viewportRect.width,
      height: layout.viewportRect.height,
    };
    registry.setRect('browser-content', contentRect);
    registry.setRect('browser-frame', {
      x: 0,
      y: -chromeHeaderHeight,
      width: layout.viewportRect.width,
      height: layout.viewportRect.height + chromeHeaderHeight,
    });
  }, [
    chromeHeaderHeight,
    layout.viewportRect.height,
    layout.viewportRect.width,
    viewportContentRect,
  ]);

  return (
    <div
      className={clsx(
        'relative overflow-hidden rounded-3xl',
        backgroundDecor.containerClass,
        containerClassName,
      )}
      style={{ ...backgroundDecor.containerStyle, ...containerStyle }}
    >
      {backgroundDecor.baseLayer}
      {backgroundDecor.overlay}
      <div
        className={clsx('relative z-[1]', backgroundDecor.contentClass, contentClassName)}
        style={contentStyle}
      >
        <div
          ref={captureAreaRef}
          className="space-y-3"
        >
          <div
            ref={presentationRef}
            data-testid="replay-presentation"
            className={clsx('relative', presentationClassName)}
            style={{
              width: `${layout.display.width}px`,
              height: `${layout.display.height}px`,
              ...(presentationStyle ?? {}),
            }}
          >
            <div
              ref={browserFrameRef}
              data-testid="replay-frame"
              className="absolute"
              style={{
                left: `${layout.frameRect.x}px`,
                top: `${layout.frameRect.y}px`,
                width: `${layout.frameRect.width}px`,
                height: `${layout.frameRect.height}px`,
              }}
            >
              <div className={clsx('flex h-full w-full flex-col overflow-hidden rounded-2xl', chromeDecor.frameClass)}>
                {chromeDecor.header && (
                  <div className="flex-shrink-0" style={{ height: `${chromeHeaderHeight}px` }}>
                    {chromeDecor.header}
                  </div>
                )}
                <div
                  ref={viewportRef}
                  data-testid="replay-viewport"
                  className={clsx('relative overflow-hidden', chromeDecor.contentClass)}
                  style={{
                    width: `${layout.viewportRect.width}px`,
                    height: `${layout.viewportRect.height}px`,
                  }}
                >
                  {children}
                </div>
              </div>
            </div>
            {overlayNode && (
              <OverlayRegistryContext.Provider value={registryRef.current}>
                <div
                  className="absolute z-[5]"
                  style={{
                    left: `${layout.viewportRect.x}px`,
                    top: `${layout.viewportRect.y}px`,
                    width: `${layout.viewportRect.width}px`,
                    height: `${layout.viewportRect.height}px`,
                  }}
                >
                  <div
                    style={{
                      position: 'absolute',
                      inset: 0,
                      ...(overlayTransformStyle ?? {}),
                    }}
                  >
                    {overlayNode}
                  </div>
                </div>
              </OverlayRegistryContext.Provider>
            )}
          </div>
        </div>
      </div>
    </div>
  );
}

export default ReplayStyleFrame;
