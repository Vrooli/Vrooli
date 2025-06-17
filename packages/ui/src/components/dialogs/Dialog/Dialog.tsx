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

// Drag hook with performance optimizations
function useDragDialog(dialogRef: React.RefObject<HTMLDivElement>, titleRef: React.RefObject<HTMLDivElement>, draggable: boolean, isOpen: boolean) {
    const [isDragging, setIsDragging] = useState(false);
    const [position, setPosition] = useState<{ x: number; y: number } | null>(null);
    
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
        const rect = dialogElement.getBoundingClientRect();
        
        // Store dialog size for performance
        dragStateRef.current.dialogSize = { width: rect.width, height: rect.height };
        
        // Center the dialog
        const centerX = (window.innerWidth - rect.width) / 2;
        const centerY = (window.innerHeight - rect.height) / 2;
        
        setPosition({ x: centerX, y: centerY });
    }, [draggable, isOpen]);

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

        // Handle dragging
        const { position: dragPosition, isDragging } = useDragDialog(dialogRef, titleRef, draggable, isOpen);

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

        // Prevent body scroll when dialog is open
        useEffect(() => {
            if (isOpen) {
                const originalOverflow = document.body.style.overflow;
                document.body.style.overflow = "hidden";
                return () => {
                    document.body.style.overflow = originalOverflow;
                };
            }
        }, [isOpen]);

        // Don't render if not open
        if (!isOpen) return null;

        // Build classes
        const overlayClasses = buildOverlayClasses(enableBackgroundBlur, overlayClassName);
        const overlayPositionClasses = draggable ? 
            "tw-items-start tw-justify-start" : // Don't center when draggable
            buildDialogClasses({
                variant,
                size,
                position,
                className,
            });
        const dialogWrapperClasses = getDialogWrapperClasses(size, draggable, isDragging);
        const contentClasses = buildContentClasses({
            variant,
            className: contentClassName,
        });

        return createPortal(
            <div
                className={cn(overlayClasses, overlayPositionClasses)}
                onClick={handleOverlayClick}
                role="presentation"
            >
                <div
                    ref={combinedRef as any}
                    className={dialogWrapperClasses}
                    style={draggable && dragPosition ? {
                        position: "fixed",
                        left: dragPosition.x,
                        top: dragPosition.y,
                        margin: 0,
                        transform: "none", // Override any transform from base classes
                    } : undefined}
                    role="dialog"
                    aria-modal="true"
                    aria-label={ariaLabel || (typeof title === "string" ? title : undefined)}
                    aria-labelledby={title ? (ariaLabelledBy || titleId) : undefined}
                    aria-describedby={ariaDescribedBy}
                >
                    <div className={contentClasses}>
                        {/* Header with title and close button */}
                        {(title || showCloseButton) && (
                            <div 
                                ref={titleRef}
                                className={cn(
                                    "tw-flex tw-items-center tw-justify-between tw-px-6 tw-pt-4 tw-pb-2",
                                    "tw-flex-shrink-0", // Prevent header from shrinking
                                    draggable && "tw-cursor-move tw-select-none", // Add drag cursor when draggable
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
