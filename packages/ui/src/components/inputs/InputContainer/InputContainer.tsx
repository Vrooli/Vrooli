import React, { forwardRef, useCallback, useRef } from "react";
import { cn } from "../../../utils/tailwind-theme.js";
import type { InputContainerProps } from "./types.js";
import { buildInputContainerClasses } from "./inputContainerStyles.js";

export const InputContainer = forwardRef<HTMLDivElement, InputContainerProps>(
    ({
        variant = "filled",
        size = "md",
        error = false,
        disabled = false,
        fullWidth = false,
        focused = false,
        onClick,
        onFocus,
        onBlur,
        className,
        children,
        label,
        isRequired,
        helperText,
        startAdornment,
        endAdornment,
        htmlFor,
        ...props
    }, ref) => {
        const containerRef = useRef<HTMLDivElement>(null);
        
        // Use provided ref or internal ref
        React.useImperativeHandle(ref, () => containerRef.current as HTMLDivElement);

        const containerClasses = buildInputContainerClasses({
            variant,
            size,
            error,
            disabled,
            fullWidth,
            focused,
            hasStartAdornment: Boolean(startAdornment),
            hasEndAdornment: Boolean(endAdornment),
        });

        const handleContainerClick = useCallback((event: React.MouseEvent<HTMLDivElement>) => {
            // Don't trigger click if clicking on interactive elements
            const target = event.target as HTMLElement;
            if (target.tagName === "INPUT" || target.tagName === "TEXTAREA" || target.tagName === "SELECT") {
                return;
            }
            
            if (onClick) {
                onClick(event);
            }
        }, [onClick]);

        const handleKeyDown = useCallback((event: React.KeyboardEvent<HTMLDivElement>) => {
            if (event.key === "Enter" || event.key === " ") {
                handleContainerClick(event as any);
            }
        }, [handleContainerClick]);

        return (
            <div 
                className={cn("tw-w-full", !fullWidth && "tw-max-w-sm")} 
                data-testid="input-container-wrapper"
                {...props}
            >
                {label && (
                    <label 
                        htmlFor={htmlFor}
                        className={cn(
                            "tw-block tw-text-sm tw-font-medium tw-mb-1 tw-transition-colors tw-duration-200",
                            disabled ? "tw-text-text-secondary tw-opacity-60" : "tw-text-text-primary",
                        )}
                        data-testid="input-container-label"
                    >
                        {label}
                        {isRequired && (
                            <span 
                                className={cn(
                                    "tw-ml-1 tw-transition-colors tw-duration-200",
                                    disabled ? "tw-text-text-secondary tw-opacity-60" : "tw-text-danger-main",
                                )}
                                data-testid="input-container-required-indicator"
                            >
                                *
                            </span>
                        )}
                    </label>
                )}
                
                <div
                    ref={containerRef}
                    className={cn(containerClasses, className)}
                    onClick={handleContainerClick}
                    onFocus={onFocus}
                    onBlur={onBlur}
                    onKeyDown={handleKeyDown}
                    role={onClick ? "button" : undefined}
                    tabIndex={onClick && !disabled ? 0 : undefined}
                    aria-disabled={disabled}
                    data-testid="input-container"
                    data-variant={variant}
                    data-size={size}
                    data-error={error}
                    data-disabled={disabled}
                    data-focused={focused}
                    data-full-width={fullWidth}
                >
                    {startAdornment && (
                        <div 
                            className={cn(
                                "tw-flex tw-items-center tw-justify-center tw-pl-3 tw-flex-shrink-0 tw-transition-colors tw-duration-200",
                                disabled ? "tw-text-text-secondary tw-opacity-60" : "tw-text-text-secondary",
                            )}
                            data-testid="input-container-start-adornment"
                        >
                            {startAdornment}
                        </div>
                    )}
                    
                    <div 
                        className="tw-flex-1 tw-flex tw-items-center tw-min-w-0"
                        data-testid="input-container-content"
                    >
                        {children}
                    </div>
                    
                    {endAdornment && (
                        <div 
                            className={cn(
                                "tw-flex tw-items-center tw-justify-center tw-pr-3 tw-flex-shrink-0 tw-transition-colors tw-duration-200",
                                disabled ? "tw-text-text-secondary tw-opacity-60" : "tw-text-text-secondary",
                            )}
                            data-testid="input-container-end-adornment"
                        >
                            {endAdornment}
                        </div>
                    )}
                </div>
                
                {helperText && (
                    <p 
                        id={htmlFor ? `helper-text-${htmlFor.replace("quantity-box-", "")}` : undefined}
                        className={cn(
                            "tw-mt-1 tw-text-xs tw-transition-colors tw-duration-200",
                            error && !disabled ? "tw-text-danger-main" : "tw-text-text-secondary",
                            disabled && "tw-opacity-60",
                        )}
                        data-testid="input-container-helper-text"
                    >
                        {typeof helperText === "string" ? helperText : JSON.stringify(helperText)}
                    </p>
                )}
            </div>
        );
    },
);

InputContainer.displayName = "InputContainer";
