import { useCallback, useState } from "react";
import type { MouseEvent } from "react";

// Constants for ripple effect
export const RIPPLE_SIZE = 50;
export const RIPPLE_RADIUS = RIPPLE_SIZE / 2;

// Ripple effect interface
export interface Ripple {
    id: number;
    x: number;
    y: number;
    size: number;
    onAnimationEnd: () => void;
}

/**
 * Custom hook for managing ripple effects on interactive components
 * Provides state management and event handlers for click ripples
 */
export const useRippleEffect = (customSize?: number) => {
    const [ripples, setRipples] = useState<Ripple[]>([]);

    // Add a new ripple at the specified coordinates
    const addRipple = useCallback((x: number, y: number) => {
        const newRipple = {
            id: Date.now() + Math.random(), // Ensure uniqueness even for rapid clicks
            x,
            y,
            size: customSize || RIPPLE_SIZE,
            onAnimationEnd: () => {
                setRipples(prev => prev.filter(ripple => ripple.id !== newRipple.id));
            },
        };

        setRipples(prev => [...prev, newRipple]);
    }, [customSize]);

    // Handle click to create ripple effect
    const handleRippleClick = useCallback((event: MouseEvent<HTMLElement>, disabled = false) => {
        if (disabled) return;

        const rect = event.currentTarget.getBoundingClientRect();
        const x = event.clientX - rect.left;
        const y = event.clientY - rect.top;

        addRipple(x, y);
    }, [addRipple]);

    // Remove completed ripples
    const handleRippleComplete = useCallback((id: number) => {
        setRipples(prev => prev.filter(ripple => ripple.id !== id));
    }, []);

    // Clear all ripples (useful for cleanup)
    const clearRipples = useCallback(() => {
        setRipples([]);
    }, []);

    return {
        ripples,
        addRipple,
        handleRippleClick,
        handleRippleComplete,
        clearRipples,
    };
};
