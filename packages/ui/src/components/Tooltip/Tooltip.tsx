import React, { useState, useRef, useEffect, useCallback } from "react";
import { createPortal } from "react-dom";

// Global tooltip manager to handle the "skip delay" feature
class TooltipManager {
    private static instance: TooltipManager;
    private isAnyTooltipOpen = false;
    private lastTooltipCloseTime = 0;
    private readonly GRACE_PERIOD = 500; // ms to skip delay after last tooltip closes

    static getInstance() {
        if (!TooltipManager.instance) {
            TooltipManager.instance = new TooltipManager();
        }
        return TooltipManager.instance;
    }

    setTooltipOpen(isOpen: boolean) {
        if (isOpen) {
            this.isAnyTooltipOpen = true;
        } else {
            this.isAnyTooltipOpen = false;
            this.lastTooltipCloseTime = Date.now();
        }
    }

    shouldSkipDelay(): boolean {
        const now = Date.now();
        return this.isAnyTooltipOpen || (now - this.lastTooltipCloseTime < this.GRACE_PERIOD);
    }
}

const tooltipManager = TooltipManager.getInstance();

export type TooltipPlacement = 
    | "top" 
    | "top-start" 
    | "top-end" 
    | "bottom" 
    | "bottom-start" 
    | "bottom-end" 
    | "left" 
    | "left-start" 
    | "left-end" 
    | "right" 
    | "right-start" 
    | "right-end";

export interface TooltipProps {
    children: React.ReactElement;
    title: React.ReactNode;
    placement?: TooltipPlacement;
    arrow?: boolean;
    enterDelay?: number;
    leaveDelay?: number;
    disableHoverListener?: boolean;
    disableFocusListener?: boolean;
    disableTouchListener?: boolean;
    open?: boolean;
    onOpen?: () => void;
    onClose?: () => void;
    className?: string;
    tooltipClassName?: string;
    /** Test ID for testing */
    "data-testid"?: string;
}

export function Tooltip({
    children,
    title,
    placement = "top",
    arrow = true,
    enterDelay = 400,
    leaveDelay = 0,
    disableHoverListener = false,
    disableFocusListener = false,
    disableTouchListener = false,
    open: controlledOpen,
    onOpen,
    onClose,
    className,
    tooltipClassName,
    "data-testid": testId,
}: TooltipProps) {
    const [internalOpen, setInternalOpen] = useState(false);
    const [mounted, setMounted] = useState(false);
    const [position, setPosition] = useState({ top: 0, left: 0 });
    const childRef = useRef<HTMLElement>(null);
    const tooltipRef = useRef<HTMLDivElement>(null);
    const enterTimeoutRef = useRef<NodeJS.Timeout>();
    const leaveTimeoutRef = useRef<NodeJS.Timeout>();
    const unmountTimeoutRef = useRef<NodeJS.Timeout>();

    const isControlled = controlledOpen !== undefined;
    const isOpen = isControlled ? controlledOpen : internalOpen;

    const calculatePosition = useCallback(() => {
        if (!childRef.current || !tooltipRef.current || !isOpen) return;

        const childRect = childRef.current.getBoundingClientRect();
        const tooltipRect = tooltipRef.current.getBoundingClientRect();
        const spacing = 10; // Space between tooltip and element
        const arrowSize = arrow ? 8 : 0;

        let top = 0;
        let left = 0;

        // Calculate base position based on placement
        switch (placement) {
            case "top":
            case "top-start":
            case "top-end":
                top = childRect.top - tooltipRect.height - spacing - arrowSize;
                break;
            case "bottom":
            case "bottom-start":
            case "bottom-end":
                top = childRect.bottom + spacing + arrowSize;
                break;
            case "left":
            case "left-start":
            case "left-end":
                left = childRect.left - tooltipRect.width - spacing - arrowSize;
                break;
            case "right":
            case "right-start":
            case "right-end":
                left = childRect.right + spacing + arrowSize;
                break;
        }

        // Horizontal alignment
        switch (placement) {
            case "top":
            case "bottom":
                left = childRect.left + (childRect.width - tooltipRect.width) / 2;
                break;
            case "top-start":
            case "bottom-start":
                left = childRect.left;
                break;
            case "top-end":
            case "bottom-end":
                left = childRect.right - tooltipRect.width;
                break;
            case "left":
            case "right":
                top = childRect.top + (childRect.height - tooltipRect.height) / 2;
                break;
            case "left-start":
            case "right-start":
                top = childRect.top;
                break;
            case "left-end":
            case "right-end":
                top = childRect.bottom - tooltipRect.height;
                break;
        }

        // Vertical alignment for left/right placements
        if (placement.startsWith("left") || placement.startsWith("right")) {
            left = placement.startsWith("left") 
                ? childRect.left - tooltipRect.width - spacing - arrowSize
                : childRect.right + spacing + arrowSize;
        }

        // Ensure tooltip stays within viewport
        const padding = 8;
        top = Math.max(padding, Math.min(top, window.innerHeight - tooltipRect.height - padding));
        left = Math.max(padding, Math.min(left, window.innerWidth - tooltipRect.width - padding));

        setPosition({ top, left });
    }, [placement, arrow, isOpen]);

    useEffect(() => {
        calculatePosition();
        window.addEventListener("resize", calculatePosition);
        window.addEventListener("scroll", calculatePosition, true);
        
        return () => {
            window.removeEventListener("resize", calculatePosition);
            window.removeEventListener("scroll", calculatePosition, true);
        };
    }, [calculatePosition]);

    const handleOpen = useCallback(() => {
        if (enterTimeoutRef.current) {
            clearTimeout(enterTimeoutRef.current);
        }
        if (leaveTimeoutRef.current) {
            clearTimeout(leaveTimeoutRef.current);
        }
        if (unmountTimeoutRef.current) {
            clearTimeout(unmountTimeoutRef.current);
        }

        // Skip delay if another tooltip was recently open
        const effectiveDelay = tooltipManager.shouldSkipDelay() ? 0 : enterDelay;

        enterTimeoutRef.current = setTimeout(() => {
            setMounted(true);
            if (!isControlled) {
                setInternalOpen(true);
            }
            tooltipManager.setTooltipOpen(true);
            onOpen?.();
        }, effectiveDelay);
    }, [enterDelay, isControlled, onOpen]);

    const handleClose = useCallback(() => {
        if (enterTimeoutRef.current) {
            clearTimeout(enterTimeoutRef.current);
        }
        if (leaveTimeoutRef.current) {
            clearTimeout(leaveTimeoutRef.current);
        }

        leaveTimeoutRef.current = setTimeout(() => {
            if (!isControlled) {
                setInternalOpen(false);
            }
            tooltipManager.setTooltipOpen(false);
            onClose?.();
            
            // Unmount after fade animation completes (300ms)
            unmountTimeoutRef.current = setTimeout(() => {
                setMounted(false);
            }, 300);
        }, leaveDelay);
    }, [leaveDelay, isControlled, onClose]);

    useEffect(() => {
        return () => {
            if (enterTimeoutRef.current) clearTimeout(enterTimeoutRef.current);
            if (leaveTimeoutRef.current) clearTimeout(leaveTimeoutRef.current);
            if (unmountTimeoutRef.current) clearTimeout(unmountTimeoutRef.current);
        };
    }, []);

    // Handle controlled open state
    useEffect(() => {
        if (isControlled) {
            if (controlledOpen) {
                setMounted(true);
                tooltipManager.setTooltipOpen(true);
            } else {
                tooltipManager.setTooltipOpen(false);
                // Unmount after fade animation
                unmountTimeoutRef.current = setTimeout(() => {
                    setMounted(false);
                }, 300);
            }
        }
    }, [isControlled, controlledOpen]);

    const childElement = React.cloneElement(children, {
        ref: childRef,
        ...(disableHoverListener ? {} : {
            onMouseEnter: (e: React.MouseEvent) => {
                children.props.onMouseEnter?.(e);
                handleOpen();
            },
            onMouseLeave: (e: React.MouseEvent) => {
                children.props.onMouseLeave?.(e);
                handleClose();
            },
        }),
        ...(disableFocusListener ? {} : {
            onFocus: (e: React.FocusEvent) => {
                children.props.onFocus?.(e);
                handleOpen();
            },
            onBlur: (e: React.FocusEvent) => {
                children.props.onBlur?.(e);
                handleClose();
            },
        }),
        ...(disableTouchListener ? {} : {
            onTouchStart: (e: React.TouchEvent) => {
                children.props.onTouchStart?.(e);
                handleOpen();
            },
            onTouchEnd: (e: React.TouchEvent) => {
                children.props.onTouchEnd?.(e);
                handleClose();
            },
        }),
        className: `${children.props.className || ""} ${className || ""}`.trim(),
        "data-tooltip-trigger": testId || true,
        "data-tooltip-open": isOpen,
    });

    const getArrowStyles = (isDark: boolean) => {
        const arrowColor = isDark ? "#e5e7eb" : "#1f2937"; // gray-200 : gray-800
        
        const baseStyle: React.CSSProperties = {
            position: "absolute",
            width: 0,
            height: 0,
        };
        
        switch (placement) {
            case "top":
            case "top-start":
            case "top-end":
                return {
                    ...baseStyle,
                    bottom: -8,
                    left: placement === "top" ? "50%" : placement === "top-start" ? 16 : "auto",
                    right: placement === "top-end" ? 16 : "auto",
                    transform: placement === "top" ? "translateX(-50%)" : undefined,
                    borderLeft: "8px solid transparent",
                    borderRight: "8px solid transparent",
                    borderTop: `8px solid ${arrowColor}`,
                };
            case "bottom":
            case "bottom-start":
            case "bottom-end":
                return {
                    ...baseStyle,
                    top: -8,
                    left: placement === "bottom" ? "50%" : placement === "bottom-start" ? 16 : "auto",
                    right: placement === "bottom-end" ? 16 : "auto",
                    transform: placement === "bottom" ? "translateX(-50%)" : undefined,
                    borderLeft: "8px solid transparent",
                    borderRight: "8px solid transparent",
                    borderBottom: `8px solid ${arrowColor}`,
                };
            case "left":
            case "left-start":
            case "left-end":
                return {
                    ...baseStyle,
                    right: -8,
                    top: placement === "left" ? "50%" : placement === "left-start" ? 16 : "auto",
                    bottom: placement === "left-end" ? 16 : "auto",
                    transform: placement === "left" ? "translateY(-50%)" : undefined,
                    borderTop: "8px solid transparent",
                    borderBottom: "8px solid transparent",
                    borderLeft: `8px solid ${arrowColor}`,
                };
            case "right":
            case "right-start":
            case "right-end":
                return {
                    ...baseStyle,
                    left: -8,
                    top: placement === "right" ? "50%" : placement === "right-start" ? 16 : "auto",
                    bottom: placement === "right-end" ? 16 : "auto",
                    transform: placement === "right" ? "translateY(-50%)" : undefined,
                    borderTop: "8px solid transparent",
                    borderBottom: "8px solid transparent",
                    borderRight: `8px solid ${arrowColor}`,
                };
            default:
                return {};
        }
    };

    const tooltipContent = mounted && title && (
        <div
            ref={tooltipRef}
            className={"tw-fixed tw-z-[1500] tw-pointer-events-none"}
            data-testid={testId ? `${testId}-wrapper` : undefined}
            data-placement={placement}
            data-open={isOpen}
            data-mounted={mounted}
            style={{
                top: `${position.top}px`,
                left: `${position.left}px`,
            }}
        >
            <div
                className={`tw-relative tw-px-3 tw-py-2 tw-text-sm tw-rounded-md tw-bg-gray-800 tw-text-white dark:tw-bg-gray-200 dark:tw-text-gray-900 tw-shadow-xl tw-transition-all tw-duration-300 tw-overflow-visible ${
                    isOpen ? "tw-opacity-100 tw-scale-100" : "tw-opacity-0 tw-scale-95"
                } ${tooltipClassName || ""}`}
                style={{ overflow: "visible" }}
                data-testid={testId}
            >
                {title}
                {arrow && (
                    <>
                        {/* Arrow for light mode */}
                        <div 
                            className="tw-block dark:tw-hidden" 
                            style={getArrowStyles(false)} 
                            data-testid={testId ? `${testId}-arrow-light` : undefined}
                        />
                        {/* Arrow for dark mode */}
                        <div 
                            className="tw-hidden dark:tw-block" 
                            style={getArrowStyles(true)} 
                            data-testid={testId ? `${testId}-arrow-dark` : undefined}
                        />
                    </>
                )}
            </div>
        </div>
    );

    return (
        <>
            {childElement}
            {typeof document !== "undefined" && createPortal(tooltipContent, document.body)}
        </>
    );
}
