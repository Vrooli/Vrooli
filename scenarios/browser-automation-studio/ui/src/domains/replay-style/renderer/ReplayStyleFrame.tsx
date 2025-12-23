import type { ReactNode, CSSProperties, Ref } from 'react';
import clsx from 'clsx';
import type { BackgroundDecor, ChromeDecor } from '../catalog';

interface ReplayStyleFrameProps {
  backgroundDecor: BackgroundDecor;
  chromeDecor: ChromeDecor;
  frameScale?: number;
  frameStyle?: CSSProperties;
  showInterfaceChrome?: boolean;
  header?: ReactNode;
  footer?: ReactNode;
  watermarkNode?: ReactNode;
  containerClassName?: string;
  contentClassName?: string;
  captureAreaRef?: Ref<HTMLDivElement>;
  browserFrameRef?: Ref<HTMLDivElement>;
  children: ReactNode;
}

export function ReplayStyleFrame({
  backgroundDecor,
  chromeDecor,
  frameScale = 1,
  frameStyle,
  showInterfaceChrome = false,
  header,
  footer,
  watermarkNode,
  containerClassName,
  contentClassName,
  captureAreaRef,
  browserFrameRef,
  children,
}: ReplayStyleFrameProps) {
  return (
    <div
      className={clsx(
        'relative overflow-hidden rounded-3xl',
        backgroundDecor.containerClass,
        containerClassName,
      )}
      style={backgroundDecor.containerStyle}
    >
      {backgroundDecor.baseLayer}
      {backgroundDecor.overlay}
      {watermarkNode}
      <div className={clsx('relative z-[1]', backgroundDecor.contentClass, contentClassName)}>
        {header}
        <div
          ref={captureAreaRef}
          className={clsx('space-y-3', showInterfaceChrome && 'mt-4')}
        >
          <div
            ref={browserFrameRef}
            className="mx-auto w-full"
            style={{ width: `${frameScale * 100}%`, ...(frameStyle ?? {}) }}
          >
            <div className={clsx('overflow-hidden rounded-2xl', chromeDecor.frameClass)}>
              {chromeDecor.header}
              <div className={clsx('relative overflow-hidden', chromeDecor.contentClass)}>
                {children}
              </div>
            </div>
          </div>
          {footer}
        </div>
      </div>
    </div>
  );
}

export default ReplayStyleFrame;
