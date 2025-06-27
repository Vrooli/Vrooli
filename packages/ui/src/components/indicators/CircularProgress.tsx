import { forwardRef } from "react";
import type { SVGProps } from "react";
import { cn } from "../../utils/tailwind-theme.js";

export interface CircularProgressProps extends Omit<SVGProps<SVGSVGElement>, "size"> {
    /**
     * The size of the spinner in pixels
     * @default 24
     */
    size?: number;
    /**
     * The thickness of the spinner stroke (1-10)
     * @default 3
     */
    thickness?: number;
    /**
     * The variant of the spinner
     * @default "primary"
     */
    variant?: "primary" | "secondary" | "white" | "current";
    /**
     * Whether to use a dual-ring spinner (more complex animation)
     * @default false
     */
    dualRing?: boolean;
    /**
     * Custom class names
     */
    className?: string;
}

// Variant color mappings
const variantClasses = {
    primary: "tw-text-secondary-main",
    secondary: "tw-text-gray-600",
    white: "tw-text-white",
    current: "tw-text-current",
};

/**
 * A modern, animated circular progress spinner built with Tailwind CSS.
 * Features smooth animations and multiple variants.
 */
export const CircularProgress = forwardRef<SVGSVGElement, CircularProgressProps>(
    (
        {
            size = 24,
            thickness = 3,
            variant = "primary",
            dualRing = false,
            className,
            ...props
        },
        ref,
    ) => {
        const strokeWidth = Math.min(Math.max(thickness, 1), 10);
        const radius = (size - strokeWidth) / 2;
        const circumference = radius * 2 * Math.PI;
        
        const spinnerClasses = cn(
            variantClasses[variant],
            "tw-animate-spin",
            className,
        );

        if (dualRing) {
            // Dual ring spinner with more complex animation
            return (
                <svg
                    ref={ref}
                    width={size}
                    height={size}
                    viewBox={`0 0 ${size} ${size}`}
                    fill="none"
                    className={spinnerClasses}
                    {...props}
                >
                    {/* Background ring */}
                    <circle
                        cx={size / 2}
                        cy={size / 2}
                        r={radius}
                        stroke="currentColor"
                        strokeWidth={strokeWidth}
                        opacity={0.25}
                    />
                    {/* Animated ring 1 */}
                    <circle
                        cx={size / 2}
                        cy={size / 2}
                        r={radius}
                        stroke="currentColor"
                        strokeWidth={strokeWidth}
                        strokeDasharray={`${circumference * 0.6} ${circumference * 0.4}`}
                        strokeLinecap="round"
                        className="tw-origin-center tw-animate-[spin_1.4s_linear_infinite]"
                    />
                    {/* Animated ring 2 (counter-rotating) */}
                    <circle
                        cx={size / 2}
                        cy={size / 2}
                        r={radius - strokeWidth - 2}
                        stroke="currentColor"
                        strokeWidth={strokeWidth * 0.8}
                        strokeDasharray={`${(radius - strokeWidth - 2) * Math.PI} ${(radius - strokeWidth - 2) * Math.PI}`}
                        strokeLinecap="round"
                        opacity={0.7}
                        className="tw-origin-center tw-animate-[spin_2s_linear_infinite_reverse]"
                    />
                </svg>
            );
        }

        // Standard single ring spinner with smooth animation
        return (
            <svg
                ref={ref}
                width={size}
                height={size}
                viewBox={`0 0 ${size} ${size}`}
                fill="none"
                className={spinnerClasses}
                {...props}
            >
                {/* Background circle (optional - provides a track) */}
                <circle
                    cx={size / 2}
                    cy={size / 2}
                    r={radius}
                    stroke="currentColor"
                    strokeWidth={strokeWidth}
                    opacity={0.2}
                />
                {/* Animated circle */}
                <circle
                    cx={size / 2}
                    cy={size / 2}
                    r={radius}
                    stroke="currentColor"
                    strokeWidth={strokeWidth}
                    strokeDasharray={circumference}
                    strokeDashoffset={circumference * 0.25}
                    strokeLinecap="round"
                    className="tw-origin-center"
                >
                    <animate
                        attributeName="stroke-dasharray"
                        values={`1,${circumference};${circumference * 0.75},${circumference};${circumference * 0.75},${circumference}`}
                        dur="1.5s"
                        repeatCount="indefinite"
                    />
                    <animate
                        attributeName="stroke-dashoffset"
                        values={`0;-${circumference * 0.25};-${circumference}`}
                        dur="1.5s"
                        repeatCount="indefinite"
                    />
                </circle>
            </svg>
        );
    },
);

CircularProgress.displayName = "CircularProgress";

// Export a simpler dots-based loading indicator as an alternative
export const DotsLoader = ({ size = 24, className }: { size?: number; className?: string }) => {
    const dotSize = size / 8;
    
    return (
        <div className={cn("tw-inline-flex tw-gap-1", className)}>
            {[0, 1, 2].map((i) => (
                <div
                    key={i}
                    className="tw-bg-current tw-rounded-full tw-animate-pulse"
                    style={{
                        width: dotSize,
                        height: dotSize,
                        animationDelay: `${i * 0.15}s`,
                        animationDuration: "1.4s",
                    }}
                />
            ))}
        </div>
    );
};

// Export a modern ring loader with gradient
export const GradientRingLoader = ({ size = 24, className }: { size?: number; className?: string }) => {
    return (
        <div
            className={cn("tw-relative tw-animate-spin", className)}
            style={{ width: size, height: size }}
        >
            <div
                className="tw-absolute tw-inset-0 tw-rounded-full tw-bg-gradient-to-r tw-from-secondary-main tw-via-secondary-light tw-to-transparent"
                style={{
                    mask: `radial-gradient(farthest-side, transparent calc(100% - ${size / 8}px), #000 calc(100% - ${size / 8}px))`,
                    WebkitMask: `radial-gradient(farthest-side, transparent calc(100% - ${size / 8}px), #000 calc(100% - ${size / 8}px))`,
                }}
            />
        </div>
    );
};

// Export the Orbital Spinner - perfect for Vrooli's space theme
export const OrbitalSpinner = ({ size = 40, className }: { size?: number; className?: string }) => {
    const centerOffset = size / 2;
    const orbitRadius = size * 0.35;
    
    return (
        <div className={cn("tw-relative", className)} style={{ width: size, height: size }}>
            {/* Outer ring track */}
            <div 
                className="tw-absolute tw-inset-0 tw-rounded-full tw-border-2" 
                style={{ 
                    borderColor: "rgba(22, 163, 97, 0.2)", // Secondary main with low opacity
                }}
            />
            
            {/* Main rotating ring with gradient */}
            <div className="tw-absolute tw-inset-0 tw-animate-spin" style={{ animationDuration: "2.5s" }}>
                <div 
                    className="tw-h-full tw-w-full tw-rounded-full tw-border-2 tw-border-transparent"
                    style={{
                        borderTopColor: "var(--secondary-main)",
                        borderRightColor: "var(--secondary-light)",
                        filter: "drop-shadow(0 0 6px rgba(22, 163, 97, 0.6))",
                    }}
                />
            </div>
            
            {/* Orbiting particles - representing the 3-tier architecture */}
            {[
                { angle: 0, duration: "1.2s", color: "#0fa", size: 0.15 }, // Cyan (neon from landing)
                { angle: 120, duration: "1.6s", color: "var(--secondary-main)", size: 0.12 },
                { angle: 240, duration: "2s", color: "var(--secondary-light)", size: 0.1 },
            ].map((orbit, i) => (
                <div
                    key={i}
                    className="tw-absolute tw-inset-0 tw-animate-spin"
                    style={{
                        animationDuration: orbit.duration,
                    }}
                >
                    <div
                        className="tw-absolute tw-rounded-full"
                        style={{
                            width: size * orbit.size,
                            height: size * orbit.size,
                            backgroundColor: orbit.color,
                            top: centerOffset - orbitRadius - (size * orbit.size / 2),
                            left: centerOffset - (size * orbit.size / 2),
                            transform: `rotate(${orbit.angle}deg)`,
                            boxShadow: `0 0 ${size * 0.2}px ${orbit.color}, 0 0 ${size * 0.3}px ${orbit.color}`,
                        }}
                    />
                </div>
            ))}
            
            {/* Center glow effect */}
            <div 
                className="tw-absolute tw-rounded-full"
                style={{
                    top: "30%",
                    left: "30%",
                    width: "40%",
                    height: "40%",
                    background: "radial-gradient(circle, rgba(15, 170, 170, 0.3) 0%, transparent 70%)",
                    filter: "blur(4px)",
                }}
            />
        </div>
    );
};
