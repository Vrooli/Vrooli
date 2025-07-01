import { useTheme } from "@mui/material";
import React, { forwardRef, useCallback, useMemo, type ComponentType, type ReactNode } from "react";
import { useTranslation } from "react-i18next";
import { useIsLeftHanded } from "../../../hooks/subscriptions.js";
import { IconCommon } from "../../../icons/Icons.js";
import { IconButton } from "../../buttons/IconButton.js";
import { HelpButton } from "../../buttons/HelpButton.js";
import { Typography } from "../../text/Typography.js";
import { useLocation } from "../../../route/router.js";
import { tryOnClose } from "../../../utils/navigation/urlTools.js";
import { type DialogTitleProps } from "../types.js";

const titleStyle = { stack: { padding: 0 } } as const;

// Skeleton Component for loading state
interface SkeletonProps {
    className?: string;
    width?: string;
    height?: string;
}

const Skeleton: ComponentType<SkeletonProps> = ({ className = "", width = "100%", height = "1rem" }) => (
    <div 
        className={`tw-animate-pulse tw-bg-gray-300 tw-rounded ${className}`}
        style={{ width, height }}
        aria-hidden="true"
    />
);

// Enhanced DialogTitle Props
interface EnhancedDialogTitleProps extends DialogTitleProps {
    /** Show loading skeleton instead of content */
    isLoading?: boolean;
    /** Add subtle shadow */
    shadow?: boolean;
    /** Animate entrance */
    animate?: boolean;
    /** Maximum number of lines for title text (default: 2) */
    maxLines?: number;
    /** Custom content to render instead of structured title/help props */
    children?: ReactNode;
    /** Visual variant - "simple" for minimal styling, "prominent" for primary colored background */
    variant?: "simple" | "prominent";
}

// Memoized action button to prevent unnecessary re-renders
const ActionButton = ({ 
    icon, 
    onClick, 
    ariaLabel, 
    palette, 
    className = "",
    variant = "prominent",
}: {
    icon: string;
    onClick: () => void;
    ariaLabel: string;
    palette: any;
    className?: string;
    variant?: "simple" | "prominent";
}) => {
    const iconColor = variant === "prominent" ? palette.primary.contrastText : palette.text.primary;
    const hoverBg = variant === "prominent" ? "hover:tw-bg-white/10" : "hover:tw-bg-gray-100 dark:hover:tw-bg-gray-800";
    
    return (
        <div className={`tw-flex tw-p-1 tw-transition-all tw-duration-200 tw-rounded ${hoverBg} tw-group ${className}`}>
            <IconButton
                aria-label={ariaLabel}
                onClick={onClick}
                variant="transparent"
                size="sm"
                className="tw-transition-transform tw-duration-200 group-hover:tw-scale-110"
            >
                <IconCommon
                    decorative
                    fill={iconColor}
                    name={icon as any}
                />
            </IconButton>
        </div>
    );
};

export const DialogTitle = forwardRef<HTMLDivElement, EnhancedDialogTitleProps>(({ 
    below,
    id,
    onClose,
    startComponent,
    isLoading = false,
    shadow = true,
    animate = true,
    maxLines = 2,
    children,
    variant = "simple",
    ...titleData
}, ref) => {
    const { t } = useTranslation();
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const isLeftHanded = useIsLeftHanded();

    const handleClose = useCallback(() => {
        tryOnClose(onClose, setLocation);
    }, [onClose, setLocation]);

    // Build class names for the outer container with enhanced styling
    const outerContainerClasses = useMemo(() => {
        const isProminentVariant = variant === "prominent";
        const baseClasses = isProminentVariant 
            ? "tw-sticky tw-top-0 tw-z-[2] tw-text-white tw-bg-primary-dark"
            : "tw-sticky tw-top-0 tw-z-[2] tw-border-b tw-border-gray-200 dark:tw-border-gray-700 tw-bg-background-paper";
        const shadowClasses = shadow && isProminentVariant ? "tw-shadow-lg" : "";
        const animationClasses = animate ? "tw-transition-all tw-duration-300 tw-ease-out" : "";
        
        return `${baseClasses} ${shadowClasses} ${animationClasses}`;
    }, [shadow, animate, variant]);

    // Build class names for the title container with enhanced interactions
    const titleContainerClasses = useMemo(() => {
        const baseClasses = `
            tw-flex tw-items-center tw-gap-2
            tw-p-1 tw-px-3 tw-min-h-[3rem]
            tw-select-none tw-touch-none tw-relative
        `;
        const justifyClasses = variant === "prominent" ? "tw-justify-between" : "tw-justify-start";
        const directionClasses = isLeftHanded ? "tw-flex-row-reverse" : "tw-flex-row";
        const enhancementClasses = animate ? "tw-transition-all tw-duration-300" : "";
        
        return `${baseClasses} ${justifyClasses} ${directionClasses} ${enhancementClasses}`;
    }, [isLeftHanded, animate, variant]);


    // Title text style with line clamping
    const titleTextStyle = useMemo(() => {
        return {
            display: "-webkit-box",
            WebkitLineClamp: maxLines,
            WebkitBoxOrient: "vertical" as const,
            overflow: "hidden",
            textOverflow: "ellipsis",
        };
    }, [maxLines]);

    // Render skeleton loading state
    const renderSkeleton = () => (
        <div className={outerContainerClasses}>
            <div className={titleContainerClasses}>
                {/* Start component skeleton */}
                {startComponent && (
                    <div className="tw-flex tw-items-center">
                        <Skeleton width="1.25rem" height="1.25rem" className="tw-rounded-full" />
                    </div>
                )}
                
                {/* Title skeleton */}
                <div className="tw-flex-1 tw-flex tw-justify-center">
                    <Skeleton 
                        width="12rem" 
                        height="1.5rem" 
                        className="tw-rounded" 
                    />
                </div>
                
                {/* Action buttons skeleton */}
                <div className="tw-flex tw-items-center tw-gap-1">
                    {titleData.help && (
                        <Skeleton width="1.25rem" height="1.25rem" className="tw-rounded" />
                    )}
                    <Skeleton width="1.25rem" height="1.25rem" className="tw-rounded" />
                </div>
            </div>
            
            {/* Below content skeleton */}
            {below && (
                <div className="tw-p-2">
                    <Skeleton width="100%" height="2rem" className="tw-rounded" />
                </div>
            )}
        </div>
    );
    
    // Return skeleton if loading
    if (isLoading) {
        return renderSkeleton();
    }

    // Apply any custom styles from sxs.root
    const style = useMemo(() => {
        return titleData.sxs?.root || undefined;
    }, [titleData.sxs]);

    return (
        <div 
            ref={ref} 
            className={outerContainerClasses}
            style={style}
            role="banner"
            aria-labelledby={id}
        >
            <div
                id={id}
                className={titleContainerClasses}
            >
                {/* Render custom children if provided, otherwise use structured layout */}
                {children ? (
                    <>
                        {/* Custom content */}
                        <div className={variant === "prominent" ? "tw-flex-1 tw-flex tw-items-center tw-justify-center" : "tw-flex-1"}>
                            {children}
                        </div>
                        
                        {/* Close button is always shown if onClose is provided */}
                        {onClose && (
                            <div className="tw-flex tw-items-center tw-gap-1">
                                <ActionButton
                                    icon="Close"
                                    onClick={handleClose}
                                    ariaLabel={t("Close")}
                                    palette={palette}
                                    variant={variant}
                                />
                            </div>
                        )}
                    </>
                ) : (
                    <>
                        {/* Start component */}
                        {startComponent && (
                            <div className="tw-flex tw-items-center">
                                {/* If it's an IconCommon, wrap it in an IconButton for consistency */}
                                {React.isValidElement(startComponent) && startComponent.type === IconCommon ? (
                                    <IconButton
                                        variant="transparent"
                                        size="sm"
                                        className="tw-transition-transform tw-duration-200 hover:tw-scale-105"
                                        onClick={() => {}} // No action by default
                                        style={{ cursor: "default" }} // Remove pointer cursor if no action
                                    >
                                        {startComponent}
                                    </IconButton>
                                ) : (
                                    <div className="tw-transition-transform tw-duration-200 hover:tw-scale-105">
                                        {startComponent}
                                    </div>
                                )}
                            </div>
                        )}
                        
                        {/* Title text with line clamping */}
                        <Typography
                            variant={variant === "prominent" ? "h4" : "h6"}
                            color={variant === "prominent" ? "inherit" : undefined}
                            align={variant === "prominent" ? "center" : "left"}
                            spacing="none"
                            className={variant === "prominent" ? "tw-flex-1" : ""}
                            style={titleTextStyle}
                        >
                            {titleData.title}
                        </Typography>
                        
                        {/* Action buttons row - always stays in same row */}
                        <div className="tw-flex tw-items-center tw-gap-1">
                            {/* Help button */}
                            {titleData.help && (
                                <div className="tw-flex tw-items-center">
                                    <HelpButton
                                        markdown={titleData.help}
                                        size={24}
                                    />
                                </div>
                            )}
                            
                            {/* Close button */}
                            {onClose && (
                                <ActionButton
                                    icon="Close"
                                    onClick={handleClose}
                                    ariaLabel={t("Close")}
                                    palette={palette}
                                    variant={variant}
                                />
                            )}
                        </div>
                    </>
                )}
            </div>
            
            
            {/* Enhanced below content */}
            {below && (
                <div className="tw-transition-all tw-duration-300">
                    {below}
                </div>
            )}
        </div>
    );
});
DialogTitle.displayName = "DialogTitle";
