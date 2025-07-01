import React, { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { cn } from "../../utils/tailwind-theme.js";
import { useThrottle } from "../../hooks/useThrottle.js";
import {
    BASE_SLIDER_STYLES,
    buildFilledTrackClasses,
    buildSliderContainerClasses,
    buildThumbClasses,
    buildTrackClasses,
    calculateSliderPosition,
    calculateValueFromPosition,
    formatSliderValue,
    getCustomSliderStyle,
    getTrackBackgroundStyle,
    SLIDER_CONFIG,
} from "./sliderStyles.js";

export type SliderVariant = 
    | "default" 
    | "primary" 
    | "secondary" 
    | "success" 
    | "warning" 
    | "danger" 
    | "space" 
    | "neon" 
    | "custom";

export type SliderSize = "sm" | "md" | "lg";

export interface SliderMark {
    value: number;
    label?: string;
}

export interface SliderProps extends Omit<React.HTMLAttributes<HTMLDivElement>, "onChange"> {
    /** Current value of the slider */
    value?: number;
    /** Default value for uncontrolled slider */
    defaultValue?: number;
    /** Minimum value */
    min?: number;
    /** Maximum value */
    max?: number;
    /** Step increment */
    step?: number;
    /** Visual style variant */
    variant?: SliderVariant;
    /** Size of the slider */
    size?: SliderSize;
    /** Whether the slider is disabled */
    disabled?: boolean;
    /** Custom color (only applies when variant="custom") */
    color?: string;
    /** Label text displayed above the slider */
    label?: string;
    /** Whether to show the current value as a tooltip */
    showValue?: boolean;
    /** Marks to display along the slider */
    marks?: SliderMark[];
    /** Callback fired when the value changes */
    onChange?: (value: number) => void;
    /** Callback fired when dragging starts */
    onChangeStart?: (value: number) => void;
    /** Callback fired when dragging ends */
    onChangeEnd?: (value: number) => void;
    /** Throttle onChange callbacks in milliseconds (0 = no throttle) */
    throttleMs?: number;
    /** Additional CSS class name */
    className?: string;
    /** Test ID for testing */
    "data-testid"?: string;
}

export const Slider = React.forwardRef<HTMLDivElement, SliderProps>(({
    value: controlledValue,
    defaultValue = 0,
    min = 0,
    max = 100,
    step = 1,
    variant = "default",
    size = "md",
    disabled = false,
    color,
    label,
    showValue = false,
    marks,
    onChange,
    onChangeStart,
    onChangeEnd,
    throttleMs = 0,
    className,
    "data-testid": testId,
    ...props
}, ref) => {
    
    const [internalValue, setInternalValue] = useState(defaultValue);
    const [isDragging, setIsDragging] = useState(false);
    const [showTooltip, setShowTooltip] = useState(false);
    const [visualValue, setVisualValue] = useState(controlledValue ?? defaultValue);
    const sliderRef = useRef<HTMLDivElement>(null);
    const thumbRef = useRef<HTMLDivElement>(null);

    // Determine if slider is controlled
    const isControlled = controlledValue !== undefined;
    const currentValue = isControlled ? controlledValue : internalValue;
    
    // Visual value is used for immediate feedback during dragging
    const displayValue = isDragging ? visualValue : currentValue;

    // Create throttled onChange callback if throttleMs is provided
    const throttledOnChange = useThrottle(
        onChange || (() => {}), 
        throttleMs,
    );

    // Use throttled or regular onChange based on throttleMs
    const effectiveOnChange = useMemo(() => {
        if (throttleMs > 0 && onChange) {
            return throttledOnChange;
        }
        return onChange;
    }, [throttleMs, onChange, throttledOnChange]);

    // Memoized style calculations
    const containerClasses = useMemo(() => 
        buildSliderContainerClasses({ size, disabled, className }), 
        [size, disabled, className],
    );

    const trackClasses = useMemo(() => 
        buildTrackClasses({ size, variant }), 
        [size, variant],
    );

    const filledTrackClasses = useMemo(() => 
        buildFilledTrackClasses({ variant, customColor: color }), 
        [variant, color],
    );

    const thumbClasses = useMemo(() => 
        buildThumbClasses({ size, variant, disabled, isDragging, customColor: color }), 
        [size, variant, disabled, isDragging, color],
    );

    const customStyle = useMemo(() => 
        variant === "custom" ? getCustomSliderStyle(color) : {}, 
        [variant, color],
    );

    // Calculate position percentage
    const positionPercentage = useMemo(() => 
        calculateSliderPosition(displayValue, min, max), 
        [displayValue, min, max],
    );

    // Update value helper
    const updateValue = useCallback((newValue: number) => {
        const clampedValue = Math.max(min, Math.min(max, newValue));
        
        // Always update visual value immediately for responsive feel
        setVisualValue(clampedValue);
        
        if (!isControlled) {
            setInternalValue(clampedValue);
        }
        
        effectiveOnChange?.(clampedValue);
    }, [min, max, isControlled, effectiveOnChange]);

    // Handle track click
    const handleTrackClick = useCallback((event: React.MouseEvent) => {
        if (disabled) return;

        const rect = sliderRef.current?.getBoundingClientRect();
        if (!rect || rect.width === 0) return;

        const position = ((event.clientX - rect.left) / rect.width) * 100;
        if (isNaN(position) || !isFinite(position)) return;
        
        const newValue = calculateValueFromPosition(position, min, max, step);
        updateValue(newValue);
    }, [disabled, min, max, step, updateValue]);

    // Handle mouse/touch events for dragging
    const handlePointerDown = useCallback((event: React.PointerEvent) => {
        if (disabled) return;

        event.preventDefault();
        event.stopPropagation();
        setIsDragging(true);
        setShowTooltip(true);

        const rect = sliderRef.current?.getBoundingClientRect();
        if (!rect || rect.width === 0) return;

        const position = ((event.clientX - rect.left) / rect.width) * 100;
        if (isNaN(position) || !isFinite(position)) return;
        
        const newValue = calculateValueFromPosition(position, min, max, step);
        updateValue(newValue);
        onChangeStart?.(newValue);
    }, [disabled, min, max, step, updateValue, onChangeStart]);

    const handlePointerMove = useCallback((event: PointerEvent) => {
        if (!isDragging || disabled) return;

        event.preventDefault();
        
        const rect = sliderRef.current?.getBoundingClientRect();
        if (!rect || rect.width === 0) return;

        const position = ((event.clientX - rect.left) / rect.width) * 100;
        if (isNaN(position) || !isFinite(position)) return;
        
        const newValue = calculateValueFromPosition(position, min, max, step);
        updateValue(newValue);
    }, [isDragging, disabled, min, max, step, updateValue]);

    const handlePointerUp = useCallback((event: PointerEvent) => {
        if (!isDragging) return;

        setIsDragging(false);
        setShowTooltip(false);
        // Reset visual value to actual value when dragging ends
        setVisualValue(currentValue);
        onChangeEnd?.(visualValue);
    }, [isDragging, currentValue, visualValue, onChangeEnd]);

    // Set up global pointer events for dragging
    useEffect(() => {
        if (isDragging) {
            document.addEventListener("pointermove", handlePointerMove);
            document.addEventListener("pointerup", handlePointerUp);
            
            return () => {
                document.removeEventListener("pointermove", handlePointerMove);
                document.removeEventListener("pointerup", handlePointerUp);
            };
        }
    }, [isDragging, handlePointerMove, handlePointerUp]);

    // Sync visual value with controlled value when not dragging
    useEffect(() => {
        if (!isDragging && isControlled) {
            setVisualValue(controlledValue);
        }
    }, [controlledValue, isDragging, isControlled]);

    // Keyboard navigation
    const handleKeyDown = useCallback((event: React.KeyboardEvent) => {
        if (disabled) return;

        let newValue = currentValue;
        const stepSize = step || 1;
        

        switch (event.key) {
            case "ArrowLeft":
            case "ArrowDown":
                event.preventDefault();
                newValue = currentValue - stepSize;
                break;
            case "ArrowRight":
            case "ArrowUp":
                event.preventDefault();
                newValue = currentValue + stepSize;
                break;
            case "Home":
                event.preventDefault();
                newValue = min;
                break;
            case "End":
                event.preventDefault();
                newValue = max;
                break;
            case "PageDown":
                event.preventDefault();
                newValue = currentValue - stepSize * 10;
                break;
            case "PageUp":
                event.preventDefault();
                newValue = currentValue + stepSize * 10;
                break;
            default:
                return;
        }

        updateValue(newValue);
    }, [disabled, currentValue, step, min, max, updateValue]);

    // Format value for display
    const formatValue = useCallback((val: number) => {
        return formatSliderValue(val, step);
    }, [step]);

    return (
        <div className="tw-w-full">
            {label && (
                <label className={BASE_SLIDER_STYLES.label}>
                    {label}
                </label>
            )}
            
            <div
                ref={ref}
                className={containerClasses}
                style={customStyle}
                data-testid={testId}
                {...props}
            >
                {/* Track */}
                <div
                    ref={sliderRef}
                    className={trackClasses}
                    style={getTrackBackgroundStyle(variant)}
                    onClick={handleTrackClick}
                    data-testid="slider-track"
                >
                    {/* Filled track */}
                    <div
                        className={filledTrackClasses}
                        style={{ 
                            ...(variant === "space" || variant === "neon" 
                                ? { clipPath: `inset(0 ${100 - positionPercentage}% 0 0)` }
                                : { width: `${positionPercentage}%` }
                            ),
                            ...(variant === "custom" && color ? { backgroundColor: color } : {}),
                        }}
                        data-testid="slider-filled-track"
                    />
                </div>

                {/* Thumb */}
                <div
                    ref={thumbRef}
                    className={thumbClasses}
                    style={{ 
                        left: `${positionPercentage}%`,
                        ...(variant === "custom" && color ? { borderColor: color } : {}),
                    }}
                    onPointerDown={handlePointerDown}
                    onKeyDown={handleKeyDown}
                    tabIndex={disabled ? -1 : 0}
                    role="slider"
                    aria-valuemin={min}
                    aria-valuemax={max}
                    aria-valuenow={displayValue}
                    aria-valuetext={formatValue(displayValue)}
                    aria-disabled={disabled}
                    aria-label={label || "Slider"}
                    data-testid="slider-thumb"
                />

                {/* Value tooltip */}
                {showValue && (showTooltip || isDragging) && (
                    <div
                        className={cn(
                            BASE_SLIDER_STYLES.valueDisplay,
                            (showTooltip || isDragging) && "tw-opacity-100",
                        )}
                        style={{ left: `${positionPercentage}%` }}
                    >
                        {formatValue(displayValue)}
                    </div>
                )}

                {/* Marks */}
                {marks && marks.length > 0 && (
                    <div className={BASE_SLIDER_STYLES.marks}>
                        {marks.map((mark, index) => {
                            const markPosition = calculateSliderPosition(mark.value, min, max);
                            return (
                                <div
                                    key={index}
                                    className={BASE_SLIDER_STYLES.mark}
                                    style={{ left: `${markPosition}%`, transform: "translateX(-50%)" }}
                                >
                                    {mark.label || mark.value}
                                </div>
                            );
                        })}
                    </div>
                )}
            </div>
        </div>
    );
});

Slider.displayName = "Slider";

// Add factory components as properties
Slider.Primary = (props: Omit<SliderProps, "variant">) => (
    <Slider variant="primary" {...props} />
);
Slider.Secondary = (props: Omit<SliderProps, "variant">) => (
    <Slider variant="secondary" {...props} />
);
Slider.Success = (props: Omit<SliderProps, "variant">) => (
    <Slider variant="success" {...props} />
);
Slider.Warning = (props: Omit<SliderProps, "variant">) => (
    <Slider variant="warning" {...props} />
);
Slider.Danger = (props: Omit<SliderProps, "variant">) => (
    <Slider variant="danger" {...props} />
);
Slider.Space = (props: Omit<SliderProps, "variant">) => (
    <Slider variant="space" {...props} />
);
Slider.Neon = (props: Omit<SliderProps, "variant">) => (
    <Slider variant="neon" {...props} />
);
Slider.Custom = (props: Omit<SliderProps, "variant"> & { color: string }) => (
    <Slider variant="custom" {...props} />
);

// Factory components for common variants
export const SliderFactory = {
    Primary: (props: Omit<SliderProps, "variant">) => (
        <Slider variant="primary" {...props} />
    ),
    Secondary: (props: Omit<SliderProps, "variant">) => (
        <Slider variant="secondary" {...props} />
    ),
    Success: (props: Omit<SliderProps, "variant">) => (
        <Slider variant="success" {...props} />
    ),
    Warning: (props: Omit<SliderProps, "variant">) => (
        <Slider variant="warning" {...props} />
    ),
    Danger: (props: Omit<SliderProps, "variant">) => (
        <Slider variant="danger" {...props} />
    ),
    Space: (props: Omit<SliderProps, "variant">) => (
        <Slider variant="space" {...props} />
    ),
    Neon: (props: Omit<SliderProps, "variant">) => (
        <Slider variant="neon" {...props} />
    ),
    Custom: (props: Omit<SliderProps, "variant"> & { color: string }) => (
        <Slider variant="custom" {...props} />
    ),
} as const;

export default Slider;
