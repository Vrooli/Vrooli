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
}

/**
 * Custom hook for managing ripple effects on interactive components
 * Provides state management and event handlers for click ripples
 */
export const useRippleEffect = () => {
    const [ripples, setRipples] = useState<Ripple[]>([]);

    // Handle click to create ripple effect
    const handleRippleClick = useCallback((event: MouseEvent<HTMLElement>, disabled: boolean = false) => {
        if (disabled) return;

        const rect = event.currentTarget.getBoundingClientRect();
        const x = event.clientX - rect.left;
        const y = event.clientY - rect.top;

        const newRipple: Ripple = {
            id: Date.now() + Math.random(), // Ensure uniqueness even for rapid clicks
            x,
            y,
        };

        setRipples(prev => [...prev, newRipple]);
    }, []);

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
        handleRippleClick,
        handleRippleComplete,
        clearRipples,
    };
};