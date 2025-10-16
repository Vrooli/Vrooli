import type { CSSProperties, ReactNode } from 'react';
import clsx from 'clsx';
import type { DeviceEmulationViewportBindings } from '@/hooks/useDeviceEmulation';

type DeviceEmulationViewportProps = DeviceEmulationViewportBindings & {
  children: ReactNode;
};

const DeviceEmulationViewport = ({
  displayWidth,
  displayHeight,
  zoomedWidth,
  zoomedHeight,
  zoom,
  colorScheme,
  vision,
  isResponsive,
  onResizePointerDown,
  children,
}: DeviceEmulationViewportProps) => {
  const wrapperStyle: CSSProperties = {
    width: `${Math.round(zoomedWidth)}px`,
    height: `${Math.round(zoomedHeight)}px`,
  };

  const scaleStyle: CSSProperties = {
    width: `${Math.round(displayWidth)}px`,
    height: `${Math.round(displayHeight)}px`,
    transform: `scale(${zoom})`,
    transformOrigin: 'top left',
  };

  return (
    <div className="device-emulation-viewport" style={wrapperStyle}>
      <div className={clsx('device-emulation-viewport__content', `device-emulation-viewport__scheme--${colorScheme}`)}>
        <div className={clsx('device-emulation-viewport__vision', `device-emulation-viewport__vision--${vision}`)}>
          <div className="device-emulation-viewport__scale" style={scaleStyle}>
            {children}
          </div>
        </div>
      </div>
      {isResponsive && (
        <>
          <div
            className="device-emulation-viewport__resize-handle device-emulation-viewport__resize-handle--right"
            data-resize-mode="width"
            onPointerDown={onResizePointerDown}
            aria-hidden
            title="Drag to adjust width"
          />
          <div
            className="device-emulation-viewport__resize-handle device-emulation-viewport__resize-handle--bottom"
            data-resize-mode="height"
            onPointerDown={onResizePointerDown}
            aria-hidden
            title="Drag to adjust height"
          />
          <div
            className="device-emulation-viewport__resize-handle device-emulation-viewport__resize-handle--corner"
            data-resize-mode="both"
            onPointerDown={onResizePointerDown}
            aria-hidden
            title="Drag to resize"
          />
        </>
      )}
    </div>
  );
};

export default DeviceEmulationViewport;
