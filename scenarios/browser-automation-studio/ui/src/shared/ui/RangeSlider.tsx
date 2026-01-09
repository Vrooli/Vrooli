import { type ChangeEvent, type CSSProperties } from 'react';
import clsx from 'clsx';
import './RangeSlider.css';

interface RangeSliderProps {
  value: number;
  min: number;
  max: number;
  step?: number;
  onChange: (value: number) => void;
  ariaLabel?: string;
  disabled?: boolean;
  id?: string;
  className?: string;
}

/**
 * RangeSlider - shared, themed slider with filled track and custom thumb.
 */
export function RangeSlider({
  value,
  min,
  max,
  step,
  onChange,
  ariaLabel,
  disabled,
  id,
  className,
}: RangeSliderProps) {
  const safeValue = Number.isFinite(value) ? value : min;
  const clamped = Math.min(Math.max(safeValue, min), max);
  const progress = ((clamped - min) / (max - min)) * 100;

  const style = {
    '--range-progress': `${progress}%`,
  } as CSSProperties;

  const handleChange = (event: ChangeEvent<HTMLInputElement>) => {
    const nextValue = parseFloat(event.target.value);
    onChange(nextValue);
  };

  return (
    <input
      id={id}
      type="range"
      value={clamped}
      min={min}
      max={max}
      step={step}
      aria-label={ariaLabel}
      disabled={disabled}
      onChange={handleChange}
      className={clsx('range-slider__input', className)}
      style={style}
    />
  );
}

export default RangeSlider;
