import type { KeyboardEvent, MouseEvent, ReactNode } from "react";
import { forwardRef, useCallback, useEffect, useId, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { IconCommon } from "../../../icons/Icons.js";
import { useIsMobile } from "../../../hooks/useIsMobile.js";
import { cn } from "../../../utils/tailwind-theme.js";
import { IconButton } from "../../buttons/IconButton.js";
import "./Dialog.css";
import {
    buildActionsClasses,
    buildContentClasses,
    buildDialogClasses,
    buildOverlayClasses,
    buildTitleClasses,
    getDialogWrapperClasses,
} from "./dialogStyles.js";

// Export types for external use
export type DialogVariant = "default" | "danger" | "success" | "space" | "neon";
export type DialogSize = "sm" | "md" | "lg" | "xl" | "full";
export type DialogPosition = "center" | "top" | "bottom" | "left" | "right";

export interface DialogProps {
    /** Whether the dialog is open */
    isOpen: boolean;
    /** Callback when the dialog should close */
    onClose: () => void;
    /** Dialog content */
    children: ReactNode;
    /** Visual style variant of the dialog */
    variant?: DialogVariant;
    /** Size of the dialog */
    size?: DialogSize;
    /** Position of the dialog */
    position?: DialogPosition;
    /** Title of the dialog */
    title?: ReactNode;
    /** Whether to show close button */
    showCloseButton?: boolean;
    /** Whether clicking outside closes the dialog */
    closeOnOverlayClick?: boolean;
    /** Whether pressing Escape closes the dialog */
    closeOnEscape?: boolean;
    /** Whether to blur the background (default: true) */
    enableBackgroundBlur?: boolean;
    /** Whether the dialog can be dragged (default: false) */
    draggable?: boolean;
    /** Element to anchor/point to with an arrow (like a tooltip) */
    anchorEl?: HTMLElement | null;
    /** Preferred placement of the dialog relative to anchor element. 
     * The dialog will use this placement if there's enough space, 
     * otherwise it will find the best alternative position.
     * Use "auto" to let the dialog choose the best position. */
    anchorPlacement?: "top" | "bottom" | "left" | "right" | "auto";
    /** Whether to highlight the anchor element */
    highlightAnchor?: boolean;
    /** Additional CSS classes for the dialog */
    className?: string;
    /** Additional CSS classes for the overlay */
    overlayClassName?: string;
    /** Additional CSS classes for the content */
    contentClassName?: string;
    /** Whether to disable focus trap */
    disableFocusTrap?: boolean;
    /** ARIA label for the dialog */
    "aria-label"?: string;
    /** ARIA labelledby for the dialog */
    "aria-labelledby"?: string;
    /** ARIA describedby for the dialog */
    "aria-describedby"?: string;
}

// Dialog Title Component
export interface DialogTitleProps {
    children: ReactNode;
    className?: string;
    id?: string;
}

export const DialogTitle = forwardRef<HTMLHeadingElement, DialogTitleProps>(
    ({ children, className, id }, ref) => {
        const generatedId = useId();
        const titleId = id || generatedId;

        return (
            <h2
                ref={ref}
                id={titleId}
                className={buildTitleClasses(className)}
            >
                {children}
            </h2>
        );
    },
);

DialogTitle.displayName = "DialogTitle";

// Dialog Content Component
export interface DialogContentProps {
    children: ReactNode;
    className?: string;
    variant?: DialogVariant;
}

export const DialogContent = forwardRef<HTMLDivElement, DialogContentProps>(
    ({ children, className, variant = "default" }, ref) => {
        return (
            <div
                ref={ref}
                className={cn(
                    "tw-flex-1 tw-overflow-y-auto tw-overflow-x-hidden tw-px-6 tw-py-4",
                    variant === "space" || variant === "neon" ? "tw-text-white" : "tw-text-text-primary",
                    "tw-min-h-0", // Important for flex children to respect overflow
                    className,
                )}
            >
                {children}
            </div>
        );
    },
);

DialogContent.displayName = "DialogContent";

// Dialog Actions Component
export interface DialogActionsProps {
    children: ReactNode;
    className?: string;
    variant?: DialogVariant;
}

export const DialogActions = forwardRef<HTMLDivElement, DialogActionsProps>(
    ({ children, className, variant = "default" }, ref) => {
        return (
            <div
                ref={ref}
                className={buildActionsClasses(variant, className)}
                data-variant={variant}
            >
                {children}
            </div>
        );
    },
);

DialogActions.displayName = "DialogActions";

// Anchor positioning hook
function useAnchorPosition(anchorEl: HTMLElement | null, anchorPlacement: "top" | "bottom" | "left" | "right" | "auto" = "auto", isOpen: boolean, dialogRef: React.RefObject<HTMLDivElement>) {
    const [position, setPosition] = useState<{ x: number; y: number; placement: "top" | "bottom" | "left" | "right" } | null>(null);
    const [isInitialized, setIsInitialized] = useState(false);

    useEffect(() => {
        if (!anchorEl || !isOpen) {
            setPosition(null);
            setIsInitialized(false);
            return;
        }

        const calculatePosition = () => {
            const anchorRect = anchorEl.getBoundingClientRect();
            const viewportWidth = window.innerWidth;
            const viewportHeight = window.innerHeight;
            
            // Get actual dialog dimensions if available, otherwise use estimates
            let dialogWidth = 400;
            let dialogHeight = 300;
            
            if (dialogRef.current) {
                const dialogRect = dialogRef.current.getBoundingClientRect();
                // Only use actual dimensions if dialog is visible (not at 0,0)
                if (dialogRect.width > 0 && dialogRect.height > 0) {
                    dialogWidth = dialogRect.width;
                    dialogHeight = dialogRect.height;
                }
            }
            
            const arrowSize = 12;
            const margin = 20; // Increased margin for better spacing to prevent overlap

            let placement = anchorPlacement;
            let x = 0;
            let y = 0;

            // Calculate available space in each direction
            const spaceTop = anchorRect.top;
            const spaceBottom = viewportHeight - anchorRect.bottom;
            const spaceLeft = anchorRect.left;
            const spaceRight = viewportWidth - anchorRect.right;

            // Check if preferred placement has enough space
            const hasSpaceForPlacement = (checkPlacement: string) => {
                switch (checkPlacement) {
                    case "top":
                        return spaceTop >= dialogHeight + arrowSize + margin;
                    case "bottom":
                        return spaceBottom >= dialogHeight + arrowSize + margin;
                    case "left":
                        return spaceLeft >= dialogWidth + arrowSize + margin;
                    case "right":
                        return spaceRight >= dialogWidth + arrowSize + margin;
                    default:
                        return false;
                }
            };

            // If not auto, check if preferred placement has space
            if (placement !== "auto") {
                if (!hasSpaceForPlacement(placement)) {
                    // Preferred placement doesn't have space, find alternative
                    // Order of fallback based on the preferred placement
                    const fallbackOrder = {
                        top: ["bottom", "right", "left"],
                        bottom: ["top", "right", "left"],
                        left: ["right", "top", "bottom"],
                        right: ["left", "top", "bottom"],
                    };

                    const fallbacks = fallbackOrder[placement] || ["bottom", "top", "right", "left"];
                    
                    // Try fallback positions in order
                    for (const fallbackPlacement of fallbacks) {
                        if (hasSpaceForPlacement(fallbackPlacement)) {
                            placement = fallbackPlacement as typeof placement;
                            break;
                        }
                    }
                    
                    // If no good placement found, keep the preferred one anyway
                    // (it will be constrained to viewport later)
                }
            } else {
                // Auto placement logic - find best fit
                if (spaceBottom >= dialogHeight + arrowSize + margin) {
                    placement = "bottom";
                } else if (spaceTop >= dialogHeight + arrowSize + margin) {
                    placement = "top";
                } else if (spaceRight >= dialogWidth + arrowSize + margin) {
                    placement = "right";
                } else if (spaceLeft >= dialogWidth + arrowSize + margin) {
                    placement = "left";
                } else {
                    placement = "bottom"; // Default fallback
                }
            }

            // Calculate position based on placement
            switch (placement) {
                case "top":
                    x = anchorRect.left + anchorRect.width / 2 - dialogWidth / 2;
                    y = anchorRect.top - dialogHeight - arrowSize - margin;
                    break;
                case "bottom":
                    x = anchorRect.left + anchorRect.width / 2 - dialogWidth / 2;
                    y = anchorRect.bottom + arrowSize + margin;
                    break;
                case "left":
                    x = anchorRect.left - dialogWidth - arrowSize - margin;
                    y = anchorRect.top + anchorRect.height / 2 - dialogHeight / 2;
                    break;
                case "right":
                    x = anchorRect.right + arrowSize + margin;
                    y = anchorRect.top + anchorRect.height / 2 - dialogHeight / 2;
                    break;
            }

            // Constrain to viewport
            x = Math.max(margin, Math.min(x, viewportWidth - dialogWidth - margin));
            y = Math.max(margin, Math.min(y, viewportHeight - dialogHeight - margin));

            setPosition({ x, y, placement });
            setIsInitialized(true);
        };

        // Initial positioning with a slight delay to get accurate dimensions
        const timeoutId = setTimeout(() => {
            calculatePosition();
            // Do a second calculation after a frame to ensure accuracy
            requestAnimationFrame(calculatePosition);
        }, 50);

        // Recalculate on scroll/resize
        const handleUpdate = () => {
            calculatePosition();
        };
        
        window.addEventListener("scroll", handleUpdate, true);
        window.addEventListener("resize", handleUpdate);

        return () => {
            clearTimeout(timeoutId);
            window.removeEventListener("scroll", handleUpdate, true);
            window.removeEventListener("resize", handleUpdate);
        };
    }, [anchorEl, anchorPlacement, isOpen, dialogRef]);

    return position;
}

// Highlight hook
function useHighlightElement(anchorEl: HTMLElement | null, highlightAnchor: boolean, isOpen: boolean) {
    useEffect(() => {
        if (!anchorEl || !highlightAnchor || !isOpen) return;

        // Add highlight class
        anchorEl.classList.add("tw-dialog-anchor-highlight");

        return () => {
            anchorEl.classList.remove("tw-dialog-anchor-highlight");
        };
    }, [anchorEl, highlightAnchor, isOpen]);
}

// Arrow helper functions
function getArrowStyles(placement: "top" | "bottom" | "left" | "right", variant: DialogVariant): React.CSSProperties {
    const isDarkMode = document.documentElement.classList.contains('dark') || 
                      window.matchMedia('(prefers-color-scheme: dark)').matches;
    
    let arrowColor = '#ffffff'; // white for light mode
    if (isDarkMode) {
        arrowColor = '#1f2937'; // dark gray for dark mode
    }
    if (variant === "space" || variant === "neon") {
        arrowColor = '#000000'; // black for special variants
    }
    
    const baseStyle: React.CSSProperties = {
        width: 0,
        height: 0,
        position: 'absolute',
    };
    
    switch (placement) {
        case "bottom":
            // Dialog is below anchor, arrow points up toward anchor
            return {
                ...baseStyle,
                borderLeft: '12px solid transparent',
                borderRight: '12px solid transparent',
                borderBottom: `12px solid ${arrowColor}`,
                filter: 'drop-shadow(0 1px 2px rgba(0, 0, 0, 0.1))',
            };
        case "top":
            // Dialog is above anchor, arrow points down toward anchor
            return {
                ...baseStyle,
                borderLeft: '12px solid transparent',
                borderRight: '12px solid transparent',
                borderTop: `12px solid ${arrowColor}`,
                filter: 'drop-shadow(0 -1px 2px rgba(0, 0, 0, 0.1))',
            };
        case "right":
            // Dialog is to the right of anchor, arrow points left toward anchor
            return {
                ...baseStyle,
                borderTop: '12px solid transparent',
                borderBottom: '12px solid transparent',
                borderRight: `12px solid ${arrowColor}`,
                filter: 'drop-shadow(-1px 0 2px rgba(0, 0, 0, 0.1))',
            };
        case "left":
            // Dialog is to the left of anchor, arrow points right toward anchor
            return {
                ...baseStyle,
                borderTop: '12px solid transparent',
                borderBottom: '12px solid transparent',
                borderLeft: `12px solid ${arrowColor}`,
                filter: 'drop-shadow(1px 0 2px rgba(0, 0, 0, 0.1))',
            };
        default:
            return baseStyle;
    }
}

function getArrowPosition(placement: "top" | "bottom" | "left" | "right", anchorEl: HTMLElement | null): React.CSSProperties {
    if (!anchorEl) return {};
    
    switch (placement) {
        case "bottom":
            // Dialog is below anchor, arrow points up from top edge of dialog
            return {
                top: "-24px", // Move arrow completely outside dialog
                left: "50%",
                transform: "translateX(-50%)",
            };
        case "top":
            // Dialog is above anchor, arrow points down from bottom edge of dialog
            return {
                bottom: "-24px", // Move arrow completely outside dialog
                left: "50%",
                transform: "translateX(-50%)",
            };
        case "right":
            // Dialog is to right of anchor, arrow points left from left edge of dialog
            return {
                left: "-24px", // Move arrow completely outside dialog
                top: "50%",
                transform: "translateY(-50%)",
            };
        case "left":
            // Dialog is to left of anchor, arrow points right from right edge of dialog
            return {
                right: "-24px", // Move arrow completely outside dialog
                top: "50%",
                transform: "translateY(-50%)",
            };
        default:
            return {};
    }
}

// Drag hook with performance optimizations
function useDragDialog(dialogRef: React.RefObject<HTMLDivElement>, titleRef: React.RefObject<HTMLDivElement>, draggable: boolean, isOpen: boolean) {
    const [isDragging, setIsDragging] = useState(false);
    const [position, setPosition] = useState<{ x: number; y: number } | null>(null);
    const [isInitialized, setIsInitialized] = useState(false);
    
    // Use refs for drag state to avoid re-renders during drag
    const dragStateRef = useRef({
        isDragging: false,
        dragOffset: { x: 0, y: 0 },
        dialogSize: { width: 0, height: 0 },
    });

    // Initialize centered position when dialog opens
    useEffect(() => {
        if (!draggable || !isOpen || !dialogRef.current) return;

        const dialogElement = dialogRef.current;
        
        // Use requestAnimationFrame to ensure the dialog is fully rendered
        const initializePosition = () => {
            const rect = dialogElement.getBoundingClientRect();
            
            // Store dialog size for performance
            dragStateRef.current.dialogSize = { width: rect.width, height: rect.height };
            
            // Center the dialog
            const centerX = (window.innerWidth - rect.width) / 2;
            const centerY = (window.innerHeight - rect.height) / 2;
            
            setPosition({ x: centerX, y: centerY });
            setIsInitialized(true);
        };

        if (!isInitialized) {
            requestAnimationFrame(initializePosition);
        }
    }, [draggable, isOpen, isInitialized]);

    useEffect(() => {
        if (!draggable || !titleRef.current || !dialogRef.current || !isOpen) return;

        const titleElement = titleRef.current;
        const dialogElement = dialogRef.current;

        function handleMouseDown(e: globalThis.MouseEvent) {
            if (!position) return;
            
            dragStateRef.current.isDragging = true;
            dragStateRef.current.dragOffset = {
                x: e.clientX - position.x,
                y: e.clientY - position.y,
            };
            
            setIsDragging(true);
            e.preventDefault();
            
            // Add cursor styles for better UX
            document.body.style.cursor = 'move';
            document.body.style.userSelect = 'none';
        }

        function handleMouseMove(e: globalThis.MouseEvent) {
            if (!dragStateRef.current.isDragging) return;

            // Use requestAnimationFrame for smooth updates
            requestAnimationFrame(() => {
                const newX = e.clientX - dragStateRef.current.dragOffset.x;
                const newY = e.clientY - dragStateRef.current.dragOffset.y;

                // Use cached dialog size instead of getBoundingClientRect
                const { width, height } = dragStateRef.current.dialogSize;
                const maxX = window.innerWidth - width;
                const maxY = window.innerHeight - height;

                const constrainedX = Math.max(0, Math.min(newX, maxX));
                const constrainedY = Math.max(0, Math.min(newY, maxY));

                // Direct DOM manipulation for better performance during drag
                if (dialogElement) {
                    dialogElement.style.left = `${constrainedX}px`;
                    dialogElement.style.top = `${constrainedY}px`;
                }
            });
        }

        function handleMouseUp(e: globalThis.MouseEvent) {
            if (!dragStateRef.current.isDragging) return;
            
            dragStateRef.current.isDragging = false;
            setIsDragging(false);
            
            // Restore cursor
            document.body.style.cursor = '';
            document.body.style.userSelect = '';
            
            // Update React state with final position
            const newX = e.clientX - dragStateRef.current.dragOffset.x;
            const newY = e.clientY - dragStateRef.current.dragOffset.y;
            
            const { width, height } = dragStateRef.current.dialogSize;
            const maxX = window.innerWidth - width;
            const maxY = window.innerHeight - height;

            const constrainedX = Math.max(0, Math.min(newX, maxX));
            const constrainedY = Math.max(0, Math.min(newY, maxY));
            
            setPosition({ x: constrainedX, y: constrainedY });
        }

        titleElement.addEventListener("mousedown", handleMouseDown, { passive: false });
        document.addEventListener("mousemove", handleMouseMove, { passive: true });
        document.addEventListener("mouseup", handleMouseUp, { passive: true });

        return () => {
            titleElement.removeEventListener("mousedown", handleMouseDown);
            document.removeEventListener("mousemove", handleMouseMove);
            document.removeEventListener("mouseup", handleMouseUp);
            
            // Cleanup cursor styles
            document.body.style.cursor = '';
            document.body.style.userSelect = '';
        };
    }, [draggable, titleRef, dialogRef, position, isOpen]);

    // Reset position when dialog closes
    useEffect(() => {
        if (!isOpen) {
            setPosition(null);
            setIsDragging(false);
            setIsInitialized(false);
            dragStateRef.current.isDragging = false;
            
            // Cleanup styles
            document.body.style.cursor = '';
            document.body.style.userSelect = '';
        }
    }, [isOpen]);

    return { position, isDragging };
}

// Swipe to close hook for mobile
function useSwipeToClose(
    isEnabled: boolean,
    dialogRef: React.RefObject<HTMLDivElement>,
    titleRef: React.RefObject<HTMLDivElement>,
    onClose: () => void
) {
    useEffect(() => {
        if (!isEnabled || !dialogRef.current || !titleRef.current) return;

        const dialog = dialogRef.current;
        const titleBar = titleRef.current;
        let startY = 0;
        let currentY = 0;
        let isDragging = false;
        let initialTransform = 0;

        const handleTouchStart = (e: TouchEvent) => {
            // Only start swipe if touching the title bar
            if (!titleBar.contains(e.target as Node)) return;
            
            startY = e.touches[0].clientY;
            currentY = startY;
            isDragging = true;
            initialTransform = 0;
            
            // Add transition disable class
            dialog.style.transition = 'none';
        };

        const handleTouchMove = (e: TouchEvent) => {
            if (!isDragging) return;
            
            currentY = e.touches[0].clientY;
            const deltaY = currentY - startY;
            
            // Only allow downward swipe
            if (deltaY > 0) {
                const transform = Math.min(deltaY, 200); // Limit swipe distance
                dialog.style.transform = `translateY(${transform}px)`;
                
                // Add resistance effect
                const opacity = Math.max(0.3, 1 - (deltaY / 300));
                dialog.style.opacity = opacity.toString();
            }
        };

        const handleTouchEnd = () => {
            if (!isDragging) return;
            
            isDragging = false;
            const deltaY = currentY - startY;
            
            // Restore transition
            dialog.style.transition = '';
            
            // If swiped down more than 100px, close the dialog
            if (deltaY > 100) {
                onClose();
            } else {
                // Snap back to original position
                dialog.style.transform = '';
                dialog.style.opacity = '';
            }
        };

        // Add event listeners to the title bar only
        titleBar.addEventListener('touchstart', handleTouchStart, { passive: false });
        document.addEventListener('touchmove', handleTouchMove, { passive: false });
        document.addEventListener('touchend', handleTouchEnd, { passive: true });

        return () => {
            titleBar.removeEventListener('touchstart', handleTouchStart);
            document.removeEventListener('touchmove', handleTouchMove);
            document.removeEventListener('touchend', handleTouchEnd);
            
            // Clean up styles
            if (dialog) {
                dialog.style.transform = '';
                dialog.style.opacity = '';
                dialog.style.transition = '';
            }
        };
    }, [isEnabled, dialogRef, titleRef, onClose]);
}

// Focus trap hook
function useFocusTrap(isOpen: boolean, dialogRef: React.RefObject<HTMLDivElement>, disableFocusTrap?: boolean) {
    useEffect(() => {
        if (!isOpen || disableFocusTrap) return;

        const dialog = dialogRef.current;
        if (!dialog) return;

        const previouslyFocused = document.activeElement as HTMLElement;

        function getFocusableElements() {
            const focusableSelectors = [
                "a[href]",
                "button:not([disabled])",
                "input:not([disabled])",
                "textarea:not([disabled])",
                "select:not([disabled])",
                "[tabindex]:not([tabindex=\"-1\"])",
            ].join(", ");

            return Array.from(dialog.querySelectorAll(focusableSelectors)) as HTMLElement[];
        }

        const focusableElements = getFocusableElements();
        if (focusableElements.length > 0) {
            focusableElements[0].focus();
        }

        function handleKeyDown(e: KeyboardEvent) {
            if (e.key !== "Tab") return;

            const focusableElements = getFocusableElements();
            if (focusableElements.length === 0) return;

            const firstElement = focusableElements[0];
            const lastElement = focusableElements[focusableElements.length - 1];

            if (e.shiftKey && document.activeElement === firstElement) {
                e.preventDefault();
                lastElement.focus();
            } else if (!e.shiftKey && document.activeElement === lastElement) {
                e.preventDefault();
                firstElement.focus();
            }
        }

        dialog.addEventListener("keydown", handleKeyDown as any);

        return () => {
            dialog.removeEventListener("keydown", handleKeyDown as any);
            previouslyFocused?.focus();
        };
    }, [isOpen, dialogRef, disableFocusTrap]);
}

/**
 * A performant, accessible Tailwind CSS dialog component with multiple variants and sizes.
 * 
 * Features:
 * - 5 variants that control action button styling (default, danger, success, space, neon)
 * - 5 size options from small to fullscreen
 * - 5 position options for flexible placement
 * - Smooth animations and transitions
 * - Focus trap for accessibility
 * - Keyboard navigation (Escape to close)
 * - Click outside to close option
 * - Portal rendering for proper z-index handling
 * - Full ARIA support
 * - Theme-aware colors that adapt to light/dark mode
 * 
 * Note: The dialog itself uses theme colors. Variants primarily affect the action buttons,
 * except for "space" and "neon" which also apply special visual effects to the dialog.
 * 
 * @example
 * ```tsx
 * <Dialog isOpen={isOpen} onClose={handleClose} title="Confirm Action" variant="danger">
 *   <DialogContent>
 *     Are you sure you want to delete this item?
 *   </DialogContent>
 *   <DialogActions variant="danger">
 *     <Button variant="ghost" onClick={handleClose}>Cancel</Button>
 *     <Button variant="danger" onClick={handleConfirm}>Delete</Button>
 *   </DialogActions>
 * </Dialog>
 * ```
 */
export const Dialog = forwardRef<HTMLDivElement, DialogProps>(
    (
        {
            isOpen,
            onClose,
            children,
            variant = "default",
            size = "md",
            position = "center",
            title,
            showCloseButton = true,
            closeOnOverlayClick = true,
            closeOnEscape = true,
            enableBackgroundBlur = true,
            draggable = false,
            anchorEl,
            anchorPlacement = "auto",
            highlightAnchor = false,
            className,
            overlayClassName,
            contentClassName,
            disableFocusTrap = false,
            "aria-label": ariaLabel,
            "aria-labelledby": ariaLabelledBy,
            "aria-describedby": ariaDescribedBy,
        },
        ref,
    ) => {
        const dialogRef = useRef<HTMLDivElement>(null);
        const titleRef = useRef<HTMLDivElement>(null);
        const combinedRef = ref || dialogRef;
        const titleId = useId();

        // Detect if mobile viewport for full size behavior (768px is standard mobile breakpoint)
        const isMobile = useIsMobile();
        const effectiveSize = size === "full" && !isMobile ? "xl" : size;
        const effectivePosition = size === "full" && isMobile ? "bottom" : position;

        // Handle anchoring
        const anchorPosition = useAnchorPosition(anchorEl, anchorPlacement, isOpen, dialogRef);
        useHighlightElement(anchorEl, highlightAnchor, isOpen);

        // Handle dragging (disabled when anchored or mobile full screen)
        const { position: dragPosition, isDragging } = useDragDialog(dialogRef, titleRef, draggable && !anchorEl && !(size === "full" && isMobile), isOpen);

        // Handle focus trap
        useFocusTrap(isOpen, dialogRef, disableFocusTrap);

        // Handle swipe to close for mobile full screen
        useSwipeToClose(size === "full" && isMobile, dialogRef, titleRef, onClose);

        // Handle escape key
        useEffect(() => {
            if (!isOpen || !closeOnEscape) return;

            function handleEscape(e: globalThis.KeyboardEvent) {
                if (e.key === "Escape") {
                    onClose();
                }
            }

            document.addEventListener("keydown", handleEscape);
            return () => document.removeEventListener("keydown", handleEscape);
        }, [isOpen, closeOnEscape, onClose]);

        // Handle overlay click - only close if click starts and ends on overlay
        const overlayMouseDownRef = useRef(false);
        
        const handleOverlayMouseDown = useCallback((e: MouseEvent<HTMLDivElement>) => {
            // Mark if the mousedown happened on the overlay (not the dialog)
            overlayMouseDownRef.current = e.target === e.currentTarget;
        }, []);
        
        const handleOverlayClick = useCallback((e: MouseEvent<HTMLDivElement>) => {
            // Only close if:
            // 1. closeOnOverlayClick is enabled
            // 2. The click ended on the overlay (target === currentTarget)
            // 3. The mousedown also started on the overlay
            if (closeOnOverlayClick && e.target === e.currentTarget && overlayMouseDownRef.current) {
                onClose();
            }
            // Reset the ref
            overlayMouseDownRef.current = false;
        }, [closeOnOverlayClick, onClose]);

        // Prevent body scroll when dialog is open (only when background is blurred)
        useEffect(() => {
            // Only prevent scroll when background blur is enabled AND no anchor
            if (isOpen && enableBackgroundBlur && !anchorEl) {
                const originalOverflow = document.body.style.overflow;
                document.body.style.overflow = "hidden";
                return () => {
                    document.body.style.overflow = originalOverflow;
                };
            }
        }, [isOpen, enableBackgroundBlur, anchorEl]);

        // Don't render if not open
        if (!isOpen) return null;

        // Build classes
        const overlayClasses = buildOverlayClasses(enableBackgroundBlur && !anchorEl, overlayClassName);
        const overlayPositionClasses = (draggable && !anchorEl) || anchorEl || (size === "full" && isMobile) ? 
            "tw-items-start tw-justify-start" : // Don't center when draggable, anchored, or mobile full screen
            buildDialogClasses({
                variant,
                size: effectiveSize,
                position: effectivePosition,
                className,
            });
        const dialogWrapperClasses = getDialogWrapperClasses(effectiveSize, draggable && !anchorEl, isDragging, !!anchorEl, size === "full" && isMobile);
        const contentClasses = buildContentClasses({
            variant,
            className: contentClassName,
            isMobileFullScreen: size === "full" && isMobile,
        });

        // Determine final position
        const finalPosition = anchorPosition || dragPosition;

        return createPortal(
            <div
                className={cn(overlayClasses, overlayPositionClasses)}
                onMouseDown={handleOverlayMouseDown}
                onClick={handleOverlayClick}
                role="presentation"
            >
                {/* Arrow pointing to anchor element - positioned absolutely on the overlay */}
                {anchorPosition && anchorEl && finalPosition && (
                    <div 
                        className="tw-pointer-events-none"
                        style={{
                            position: 'fixed',
                            // Calculate arrow position based on dialog position and placement
                            // Remember: placement describes where dialog is relative to anchor
                            left: anchorPosition.placement === 'right' ? finalPosition.x - 12 :  // Dialog is right of anchor, arrow on left edge
                                  anchorPosition.placement === 'left' ? finalPosition.x + (dialogRef.current?.offsetWidth || 400) :  // Dialog is left of anchor, arrow on right edge
                                  finalPosition.x + (dialogRef.current?.offsetWidth || 400) / 2 - 6,  // Top/bottom: center horizontally
                            top: anchorPosition.placement === 'bottom' ? finalPosition.y - 12 :  // Dialog is below anchor, arrow on top edge
                                 anchorPosition.placement === 'top' ? finalPosition.y + (dialogRef.current?.offsetHeight || 300) :  // Dialog is above anchor, arrow on bottom edge
                                 finalPosition.y + (dialogRef.current?.offsetHeight || 300) / 2 - 6,  // Left/right: center vertically
                            zIndex: 9999,
                            ...getArrowStyles(anchorPosition.placement, variant),
                        }}
                        data-testid="dialog-arrow"
                        data-placement={anchorPosition.placement}
                    />
                )}
                
                <div
                    ref={combinedRef as any}
                    className={dialogWrapperClasses}
                    style={finalPosition ? {
                        position: "fixed",
                        left: finalPosition.x,
                        top: finalPosition.y,
                        margin: 0,
                        transform: "none", // Override any transform from base classes
                    } : ((draggable && !anchorEl) || anchorEl) && !(size === "full" && isMobile) ? {
                        // Hide until positioned when draggable OR anchored (but not mobile full screen)
                        opacity: 0,
                        position: "fixed",
                        left: 0,
                        top: 0,
                        margin: 0,
                        transform: "none",
                    } : undefined}
                    role="dialog"
                    aria-modal="true"
                    aria-label={ariaLabel || (typeof title === "string" ? title : undefined)}
                    aria-labelledby={title ? (ariaLabelledBy || titleId) : undefined}
                    aria-describedby={ariaDescribedBy}
                >
                    
                    <div className={cn(
                        contentClasses, 
                        anchorEl && "tw-max-h-[min(80vh,600px)] tw-overflow-hidden tw-flex tw-flex-col",
                        size === "full" && isMobile && "tw-flex tw-flex-col"
                    )}>

                        {/* Header with title and close button */}
                        {(title || showCloseButton) && (
                            <div 
                                ref={titleRef}
                                className={cn(
                                    "tw-flex tw-items-center tw-justify-between tw-px-6 tw-pt-4 tw-pb-2",
                                    "tw-flex-shrink-0", // Prevent header from shrinking
                                    "tw-relative tw-z-10", // Ensure header is above effects
                                    draggable && !anchorEl && !(size === "full" && isMobile) && "tw-cursor-move tw-select-none", // Add drag cursor when draggable and not anchored or mobile full
                                    size === "full" && isMobile && "tw-cursor-grab tw-select-none", // Add swipe indicator for mobile full screen
                                )}
                            >
                                {/* Swipe indicator for mobile full screen */}
                                {size === "full" && isMobile && (
                                    <div className="tw-absolute tw-top-2 tw-left-1/2 tw-transform tw--translate-x-1/2 tw-w-10 tw-h-1 tw-bg-gray-400 tw-rounded-full tw-opacity-60" />
                                )}
                                {title && (
                                    <DialogTitle
                                        id={titleId}
                                        className={variant === "space" || variant === "neon" ? "tw-text-white" : undefined}
                                    >
                                        {title}
                                    </DialogTitle>
                                )}
                                {showCloseButton && (
                                    <IconButton
                                        variant="transparent"
                                        size="sm"
                                        onClick={onClose}
                                        className={cn(
                                            "tw-ml-auto",
                                            !title && "tw-absolute tw-right-4 tw-top-4",
                                            (variant === "space" || variant === "neon") && "tw-text-white hover:tw-text-gray-300",
                                        )}
                                        aria-label="Close dialog"
                                    >
                                        <IconCommon name="Close" />
                                    </IconButton>
                                )}
                            </div>
                        )}

                        {/* Dialog content - ensure it's above effects */}
                        <div className={cn(
                            "tw-relative tw-z-10 tw-flex-1 tw-min-h-0",
                            anchorEl && "tw-overflow-y-auto"
                        )}>
                            {children}
                        </div>

                        {/* Special effects for space variant - moved before content to fix z-index */}
                        {variant === "space" && (
                            <>
                                <div className="tw-dialog-space-stars tw-z-0" />
                                <div className="tw-dialog-space-nebula tw-z-0" />
                            </>
                        )}

                        {/* Special effects for neon variant */}
                        {variant === "neon" && (
                            <div className="tw-dialog-neon-glow tw-z-0" />
                        )}
                    </div>
                </div>
            </div>,
            document.body,
        );
    },
);

Dialog.displayName = "Dialog";

/**
 * Pre-configured dialog components for common use cases
 */
export const DialogFactory = {
    /** Default dialog */
    Default: (props: Omit<DialogProps, "variant">) => (
        <Dialog variant="default" {...props} />
    ),
    /** Danger/destructive dialog */
    Danger: (props: Omit<DialogProps, "variant">) => (
        <Dialog variant="danger" {...props} />
    ),
    /** Success dialog */
    Success: (props: Omit<DialogProps, "variant">) => (
        <Dialog variant="success" {...props} />
    ),
    /** Space-themed dialog */
    Space: (props: Omit<DialogProps, "variant">) => (
        <Dialog variant="space" {...props} />
    ),
    /** Neon glowing dialog */
    Neon: (props: Omit<DialogProps, "variant">) => (
        <Dialog variant="neon" {...props} />
    ),
} as const;
