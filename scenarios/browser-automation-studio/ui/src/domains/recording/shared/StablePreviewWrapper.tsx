/**
 * StablePreviewWrapper - Wrapper that keeps children mounted during style toggles
 *
 * This component solves the flickering issue that occurs when toggling between
 * styled (replay presentation) and unstyled preview modes. The previous implementation
 * used conditional rendering that caused React to unmount/remount the children
 * (PlaywrightView), losing canvas state and WebSocket subscriptions.
 *
 * Key design:
 * - Children are ALWAYS rendered in the same DOM position
 * - Decorations (background, chrome, device frame) are added/removed around children
 * - CSS transitions provide smooth visual changes
 * - No tree structure changes that would cause React reconciliation to remount children
 */

import { forwardRef, type ReactNode, type CSSProperties, type Ref } from 'react';
import clsx from 'clsx';
import type { ReplayLayoutModel } from '@/domains/replay-layout';
import type { BackgroundDecor, ChromeDecor, DeviceFrameDecor } from '@/domains/replay-style';

interface StablePreviewWrapperProps {
  /** Whether replay styling is enabled */
  showReplayStyle: boolean;
  /** Layout model for positioning when styled */
  layout: ReplayLayoutModel | null;
  /** Background decoration config */
  backgroundDecor?: BackgroundDecor;
  /** Chrome (browser frame) decoration config */
  chromeDecor?: ChromeDecor;
  /** Device frame decoration config */
  deviceFrameDecor?: DeviceFrameDecor | null;
  /** Content to render (PlaywrightView) - ALWAYS stays mounted */
  children: ReactNode;
  /** Additional class for the outer container */
  className?: string;
  /** Ref for the viewport element (for coordinate mapping) */
  viewportRef?: Ref<HTMLDivElement>;
  /** Optional overlay node (cursor, watermark, etc.) */
  overlayNode?: ReactNode;
}

/**
 * Stable wrapper that keeps children mounted during style toggles.
 *
 * The DOM structure is always:
 * ```
 * <outer-container>
 *   <background-layer />     (conditionally visible)
 *   <frame-container>
 *     <chrome-header />      (conditionally visible)
 *     <viewport-container>   (STABLE - children always here)
 *       {children}
 *     </viewport-container>
 *   </frame-container>
 *   <overlay-layer />        (conditionally visible)
 * </outer-container>
 * ```
 */
export const StablePreviewWrapper = forwardRef<HTMLDivElement, StablePreviewWrapperProps>(
  function StablePreviewWrapper(
    {
      showReplayStyle,
      layout,
      backgroundDecor,
      chromeDecor,
      deviceFrameDecor,
      children,
      className,
      viewportRef,
      overlayNode,
    },
    ref
  ) {
    const chromeHeaderHeight = showReplayStyle && chromeDecor?.header ? chromeDecor.headerHeight : 0;

    // Compute styles based on whether replay style is active
    const outerContainerStyle: CSSProperties = showReplayStyle && layout
      ? {
          width: layout.display.width + (layout.contentInset?.x ?? 0) * 2,
          height: layout.display.height + (layout.contentInset?.y ?? 0) * 2,
        }
      : {
          width: '100%',
          height: '100%',
        };

    const frameContainerStyle: CSSProperties = showReplayStyle && layout
      ? {
          position: 'absolute' as const,
          left: layout.frameRect.x,
          top: layout.frameRect.y,
          width: layout.frameRect.width,
          height: layout.frameRect.height,
        }
      : {
          width: '100%',
          height: '100%',
        };

    const viewportStyle: CSSProperties = showReplayStyle && layout
      ? {
          width: layout.viewportRect.width,
          height: layout.viewportRect.height,
        }
      : {
          width: '100%',
          height: '100%',
        };

    const contentInset = layout?.contentInset ?? { x: 0, y: 0 };
    const contentPadding = showReplayStyle && (contentInset.x > 0 || contentInset.y > 0)
      ? { padding: `${contentInset.y}px ${contentInset.x}px` }
      : undefined;

    return (
      <div
        ref={ref}
        className={clsx(
          'relative transition-all duration-200',
          showReplayStyle && 'overflow-hidden rounded-3xl',
          showReplayStyle && backgroundDecor?.containerClass,
          className
        )}
        style={{
          ...outerContainerStyle,
          ...(showReplayStyle ? backgroundDecor?.containerStyle : undefined),
        }}
      >
        {/* Background layer - only visible when styled */}
        {showReplayStyle && backgroundDecor && (
          <>
            {backgroundDecor.baseLayer}
            {backgroundDecor.overlay}
          </>
        )}

        {/* Device frame overlay (inside screen) - only when styled */}
        {showReplayStyle && deviceFrameDecor?.overlay && !deviceFrameDecor.wrapperStyle && (
          deviceFrameDecor.overlay
        )}

        {/* Content layer */}
        <div
          className={clsx(
            'relative z-[1]',
            showReplayStyle && backgroundDecor?.contentClass
          )}
          style={contentPadding}
        >
          {/* Presentation container */}
          <div
            data-testid="stable-preview-presentation"
            className={clsx(
              'relative',
              showReplayStyle ? '' : 'h-full w-full'
            )}
            style={showReplayStyle && layout ? {
              width: layout.display.width,
              height: layout.display.height,
            } : undefined}
          >
            {/* Frame container (chrome wrapper) */}
            <div
              data-testid="stable-preview-frame"
              className={clsx(
                showReplayStyle && 'absolute',
                showReplayStyle && chromeDecor?.frameClass,
                showReplayStyle && 'flex flex-col overflow-hidden rounded-2xl',
                !showReplayStyle && 'h-full w-full'
              )}
              style={frameContainerStyle}
            >
              {/* Chrome header - only when styled */}
              {showReplayStyle && chromeDecor?.header && (
                <div
                  className="flex-shrink-0 transition-all duration-200"
                  style={{ height: chromeHeaderHeight }}
                >
                  {chromeDecor.header}
                </div>
              )}

              {/* VIEWPORT - CHILDREN ALWAYS RENDERED HERE (STABLE POSITION) */}
              <div
                ref={viewportRef}
                data-testid="stable-preview-viewport"
                className={clsx(
                  'relative overflow-hidden',
                  showReplayStyle && chromeDecor?.contentClass,
                  // Ensure full size when not styled
                  !showReplayStyle && 'h-full w-full'
                )}
                style={viewportStyle}
              >
                {children}
              </div>
            </div>

            {/* Overlay layer - only when styled */}
            {showReplayStyle && overlayNode && layout && (
              <div
                className="pointer-events-none absolute z-[5]"
                style={{
                  left: layout.viewportRect.x,
                  top: layout.viewportRect.y,
                  width: layout.viewportRect.width,
                  height: layout.viewportRect.height,
                }}
              >
                <div className="absolute inset-0">
                  {overlayNode}
                </div>
              </div>
            )}
          </div>
        </div>

        {/* Device frame wrapper overlay (outside screen, e.g., monitor stand) */}
        {showReplayStyle && deviceFrameDecor?.overlay && deviceFrameDecor.wrapperStyle && (
          deviceFrameDecor.overlay
        )}
      </div>
    );
  }
);

export default StablePreviewWrapper;
