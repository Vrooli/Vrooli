import type { KeyboardEvent, MouseEvent, ReactNode } from "react";
import { forwardRef, useCallback, useEffect, useId, useRef, useState } from "react";
import { createPortal } from "react-dom";
import { IconCommon } from "../../../icons/Icons.js";
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
    /** Placement of the dialog relative to anchor element */
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

            // Auto placement logic
            if (placement === "auto") {
                const spaceTop = anchorRect.top;
                const spaceBottom = viewportHeight - anchorRect.bottom;
                const spaceLeft = anchorRect.left;
                const spaceRight = viewportWidth - anchorRect.right;

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
function getArrowClasses(placement: "top" | "bottom" | "left" | "right", anchorEl: HTMLElement | null, variant: DialogVariant) {
    const baseClasses = "tw-w-0 tw-h-0 tw-z-10";
    
    switch (placement) {
        case "bottom":
            // Dialog is below anchor, arrow points up toward anchor
            return cn(baseClasses, "tw-dialog-arrow-up", 
                     variant === "space" && "tw-dialog-arrow-space",
                     variant === "neon" && "tw-dialog-arrow-neon");
        case "top":
            // Dialog is above anchor, arrow points down toward anchor
            return cn(baseClasses, "tw-dialog-arrow-down",
                     variant === "space" && "tw-dialog-arrow-space",
                     variant === "neon" && "tw-dialog-arrow-neon");
        case "right":
            // Dialog is to the right of anchor, arrow points left toward anchor
            return cn(baseClasses, "tw-dialog-arrow-left",
                     variant === "space" && "tw-dialog-arrow-space",
                     variant === "neon" && "tw-dialog-arrow-neon");
        case "left":
            // Dialog is to the left of anchor, arrow points right toward anchor
            return cn(baseClasses, "tw-dialog-arrow-right",
                     variant === "space" && "tw-dialog-arrow-space",
                     variant === "neon" && "tw-dialog-arrow-neon");
        default:
            return baseClasses;
    }
}

function getArrowPosition(placement: "top" | "bottom" | "left" | "right", anchorEl: HTMLElement | null): React.CSSProperties {
    if (!anchorEl) return {};
    
    switch (placement) {
        case "bottom":
            // Dialog is below anchor, arrow at top of dialog
            return {
                top: "-12px",
                left: "50%",
                transform: "translateX(-50%)",
            };
        case "top":
            // Dialog is above anchor, arrow at bottom of dialog
            return {
                bottom: "-12px",
                left: "50%",
                transform: "translateX(-50%)",
            };
        case "right":
            // Dialog is to right of anchor, arrow at left of dialog
            return {
                left: "-12px",
                top: "50%",
                transform: "translateY(-50%)",
            };
        case "left":
            // Dialog is to left of anchor, arrow at right of dialog
            return {
                right: "-12px",
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

        // Handle anchoring
        const anchorPosition = useAnchorPosition(anchorEl, anchorPlacement, isOpen, dialogRef);
        useHighlightElement(anchorEl, highlightAnchor, isOpen);

        // Handle dragging (disabled when anchored)
        const { position: dragPosition, isDragging } = useDragDialog(dialogRef, titleRef, draggable && !anchorEl, isOpen);

        // Handle focus trap
        useFocusTrap(isOpen, dialogRef, disableFocusTrap);

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

        // Handle overlay click
        const handleOverlayClick = useCallback((e: MouseEvent<HTMLDivElement>) => {
            if (closeOnOverlayClick && e.target === e.currentTarget) {
                onClose();
            }
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
        const overlayPositionClasses = (draggable && !anchorEl) || anchorEl ? 
            "tw-items-start tw-justify-start" : // Don't center when draggable or anchored
            buildDialogClasses({
                variant,
                size,
                position,
                className,
            });
        const dialogWrapperClasses = getDialogWrapperClasses(size, draggable && !anchorEl, isDragging, !!anchorEl);
        const contentClasses = buildContentClasses({
            variant,
            className: contentClassName,
        });

        // Determine final position
        const finalPosition = anchorPosition || dragPosition;

        return createPortal(
            <div
                className={cn(overlayClasses, overlayPositionClasses)}
                onClick={handleOverlayClick}
                role="presentation"
            >
                <div
                    ref={combinedRef as any}
                    className={dialogWrapperClasses}
                    style={finalPosition ? {
                        position: "fixed",
                        left: finalPosition.x,
                        top: finalPosition.y,
                        margin: 0,
                        transform: "none", // Override any transform from base classes
                    } : ((draggable && !anchorEl) || anchorEl) ? {
                        // Hide until positioned when draggable OR anchored
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
                    <div className={cn(contentClasses, anchorEl && "tw-max-h-full tw-overflow-hidden tw-flex tw-flex-col")}>
                        {/* Arrow pointing to anchor element */}
                        {anchorPosition && (
                            <div 
                                className={cn(
                                    "tw-absolute tw-pointer-events-none",
                                    getArrowClasses(anchorPosition.placement, anchorEl, variant)
                                )}
                                style={getArrowPosition(anchorPosition.placement, anchorEl)}
                            />
                        )}

                        {/* Header with title and close button */}
                        {(title || showCloseButton) && (
                            <div 
                                ref={titleRef}
                                className={cn(
                                    "tw-flex tw-items-center tw-justify-between tw-px-6 tw-pt-4 tw-pb-2",
                                    "tw-flex-shrink-0", // Prevent header from shrinking
                                    draggable && !anchorEl && "tw-cursor-move tw-select-none", // Add drag cursor when draggable and not anchored
                                )}
                            >
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

                        {/* Dialog content */}
                        {children}

                        {/* Special effects for space variant */}
                        {variant === "space" && (
                            <>
                                <div className="tw-dialog-space-stars" />
                                <div className="tw-dialog-space-nebula" />
                            </>
                        )}

                        {/* Special effects for neon variant */}
                        {variant === "neon" && (
                            <div className="tw-dialog-neon-glow" />
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
