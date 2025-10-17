import { useId } from 'react';
import type { ChangeEvent } from 'react';
import { Eye, MonitorSmartphone, Palette, RotateCcw, Undo2, ZoomIn } from 'lucide-react';
import type {
  DeviceEmulationToolbarBindings,
  DevicePresetId,
  DeviceVisionMode,
  DeviceColorScheme,
  DeviceZoomLevel,
} from '@/hooks/useDeviceEmulation';

const DeviceEmulationToolbar = ({
  presets,
  selectedPresetId,
  displayWidth,
  displayHeight,
  zoomLevels,
  zoom,
  colorScheme,
  vision,
  isResponsive,
  maxResponsiveWidth,
  maxResponsiveHeight,
  onPresetChange,
  onDimensionChange,
  onZoomChange,
  onColorSchemeChange,
  onVisionChange,
  onRotate,
  onReset,
}: DeviceEmulationToolbarBindings) => {
  const idPrefix = useId();
  const presetSelectId = `${idPrefix}-preset`;
  const widthInputId = `${idPrefix}-width`;
  const heightInputId = `${idPrefix}-height`;
  const zoomSelectId = `${idPrefix}-zoom`;
  const schemeSelectId = `${idPrefix}-scheme`;
  const visionSelectId = `${idPrefix}-vision`;

  const handlePresetChange = (event: ChangeEvent<HTMLSelectElement>) => {
    onPresetChange(event.target.value as DevicePresetId);
  };

  const handleWidthChange = (event: ChangeEvent<HTMLInputElement>) => {
    const nextValue = Number.parseInt(event.target.value, 10);
    if (Number.isFinite(nextValue)) {
      onDimensionChange('width', nextValue);
    }
  };

  const handleHeightChange = (event: ChangeEvent<HTMLInputElement>) => {
    const nextValue = Number.parseInt(event.target.value, 10);
    if (Number.isFinite(nextValue)) {
      onDimensionChange('height', nextValue);
    }
  };

  const handleZoomChange = (event: ChangeEvent<HTMLSelectElement>) => {
    const nextValue = Number.parseFloat(event.target.value);
    if (Number.isFinite(nextValue)) {
      onZoomChange(nextValue as DeviceZoomLevel);
    }
  };

  const handleSchemeChange = (event: ChangeEvent<HTMLSelectElement>) => {
    onColorSchemeChange(event.target.value as DeviceColorScheme);
  };

  const handleVisionChange = (event: ChangeEvent<HTMLSelectElement>) => {
    onVisionChange(event.target.value as DeviceVisionMode);
  };

  return (
    <div className="device-emulation-toolbar" role="group" aria-label="Device emulation controls">
      <label className="device-emulation-toolbar__control" htmlFor={presetSelectId}>
        <MonitorSmartphone aria-hidden size={14} />
        <select
          id={presetSelectId}
          value={selectedPresetId}
          onChange={handlePresetChange}
          className="device-emulation-toolbar__select"
        >
          {presets.map(preset => (
            <option key={preset.id} value={preset.id}>
              {preset.label}
            </option>
          ))}
        </select>
      </label>
      <div className="device-emulation-toolbar__dimensions" aria-live="polite">
        <label className="device-emulation-toolbar__dimensions-input" htmlFor={widthInputId}>
          <input
            id={widthInputId}
            type="number"
            inputMode="numeric"
            min={1}
            max={maxResponsiveWidth ?? undefined}
            value={Math.round(displayWidth)}
            onChange={handleWidthChange}
            disabled={!isResponsive}
          />
        </label>
        <span className="device-emulation-toolbar__dimensions-separator" aria-hidden>x</span>
        <label className="device-emulation-toolbar__dimensions-input" htmlFor={heightInputId}>
          <input
            id={heightInputId}
            type="number"
            inputMode="numeric"
            min={1}
            max={maxResponsiveHeight ?? undefined}
            value={Math.round(displayHeight)}
            onChange={handleHeightChange}
            disabled={!isResponsive}
          />
        </label>
      </div>
      <label className="device-emulation-toolbar__control" htmlFor={zoomSelectId}>
        <ZoomIn aria-hidden size={14} />
        <select
          id={zoomSelectId}
          value={zoom}
          onChange={handleZoomChange}
          className="device-emulation-toolbar__select"
        >
          {zoomLevels.map(level => (
            <option key={level} value={level}>{`${Math.round(level * 100)}%`}</option>
          ))}
        </select>
      </label>
      <label className="device-emulation-toolbar__control" htmlFor={schemeSelectId}>
        <Palette aria-hidden size={14} />
        <select
          id={schemeSelectId}
          value={colorScheme}
          onChange={handleSchemeChange}
          className="device-emulation-toolbar__select"
        >
          <option value="system">System</option>
          <option value="light">Light</option>
          <option value="dark">Dark</option>
        </select>
      </label>
      <label className="device-emulation-toolbar__control" htmlFor={visionSelectId}>
        <Eye aria-hidden size={14} />
        <select
          id={visionSelectId}
          value={vision}
          onChange={handleVisionChange}
          className="device-emulation-toolbar__select"
        >
          <option value="none">Normal</option>
          <option value="blur">Blurry</option>
          <option value="grayscale">Grayscale</option>
          <option value="protanopia">Protanopia</option>
          <option value="deuteranopia">Deuteranopia</option>
          <option value="tritanopia">Tritanopia</option>
        </select>
      </label>
      <button
        type="button"
        className="device-emulation-toolbar__icon-btn"
        onClick={onRotate}
        aria-label="Rotate viewport"
        title="Rotate viewport"
      >
        <RotateCcw aria-hidden size={14} />
      </button>
      <button
        type="button"
        className="device-emulation-toolbar__icon-btn"
        onClick={onReset}
        aria-label="Reset device emulation"
        title="Reset device emulation"
      >
        <Undo2 aria-hidden size={14} />
      </button>
    </div>
  );
};

export default DeviceEmulationToolbar;
