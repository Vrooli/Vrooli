import { useTheme } from "@mui/material";
import React, { forwardRef } from "react";
import { resolveThemeColor } from "../utils/themeColors.js";
import type { IconName as IconCommonName } from "./types/commonIcons.js";
import type { IconName as IconRoutineName } from "./types/routineIcons.js";
import type { IconName as IconServiceName } from "./types/serviceIcons.js";
import type { IconName as IconTextName } from "./types/textIcons.js";

// Check for global (for mocha tests) first, then fallback to globalThis (for browser)
const commonSpriteHref = (typeof global !== "undefined" && global.commonSpriteHref) ||
    (typeof globalThis !== "undefined" && globalThis.commonSpriteHref) ||
    "/sprites/common-sprite.svg";
const routineSpriteHref = (typeof global !== "undefined" && global.routineSpriteHref) ||
    (typeof globalThis !== "undefined" && globalThis.routineSpriteHref) ||
    "/sprites/routine-sprite.svg";
const serviceSpriteHref = (typeof global !== "undefined" && global.serviceSpriteHref) ||
    (typeof globalThis !== "undefined" && globalThis.serviceSpriteHref) ||
    "/sprites/service-sprite.svg";
const textSpriteHref = (typeof global !== "undefined" && global.textSpriteHref) ||
    (typeof globalThis !== "undefined" && globalThis.textSpriteHref) ||
    "/sprites/text-sprite.svg";

const DEFAULT_SIZE_PX = 24;
const DEFAULT_DECORATIVE = true;
const typeToHrefMap: { [key in IconInfo["type"]]: string } = {
    Common: commonSpriteHref,
    Routine: routineSpriteHref,
    Service: serviceSpriteHref,
    Text: textSpriteHref,
};

type IconBaseProps = Omit<React.SVGProps<SVGSVGElement>, "fill" | "size"> & {
    /** 
     * Set to false when icon conveys meaning and needs to be announced by screen readers
     * @default true
     */
    decorative?: boolean | null | undefined;
    /** 
     * The color to fill the icon with. Can be any valid CSS color value.
     * @default "currentColor"
     */
    fill?: string | null | undefined;
    href: string;
    size?: number | null | undefined;
}

const IconBase = forwardRef<SVGSVGElement, IconBaseProps>(({
    "aria-label": ariaLabel,
    "aria-hidden": ariaHidden,
    decorative,
    fill,
    href,
    size,
    style,
    ...props
}, ref) => {
    const theme = useTheme();

    // Use the new utility function to resolve the color
    const fillColor = resolveThemeColor(theme, fill);

    // Only add aria-hidden if:
    // 1. Icon is decorative AND
    // 2. aria-hidden wasn't explicitly set to false
    const shouldBeHidden = (decorative ?? DEFAULT_DECORATIVE) && ariaHidden !== false;

    return (
        <svg
            ref={ref}
            width={size ?? DEFAULT_SIZE_PX}
            height={size ?? DEFAULT_SIZE_PX}
            fill={fillColor}
            style={style}
            data-icon-type="base"
            {...props}
            aria-hidden={shouldBeHidden ? "true" : undefined}
            // Only add aria-label if icon is not decorative
            aria-label={!(decorative ?? DEFAULT_DECORATIVE) ? ariaLabel : undefined}
        >
            <use href={href} />
        </svg>
    );
});
IconBase.displayName = "IconBase";

type IconCommonProps = Omit<IconBaseProps, "href"> & {
    name: IconCommonName;
}
export const IconCommon = forwardRef<SVGSVGElement, IconCommonProps>(({
    name,
    ...props
}, ref) => {
    const href = `${commonSpriteHref}#${name}`;
    return <IconBase ref={ref} href={href} data-icon-type="common" data-icon-name={name} {...props} />;
});
IconCommon.displayName = "IconCommon";

type IconRoutineProps = Omit<IconBaseProps, "href"> & {
    name: IconRoutineName;
}
export const IconRoutine = forwardRef<SVGSVGElement, IconRoutineProps>(({
    name,
    ...props
}, ref) => {
    const href = `${routineSpriteHref}#${name}`;
    return <IconBase ref={ref} href={href} data-icon-type="routine" data-icon-name={name} {...props} />;
});
IconRoutine.displayName = "IconRoutine";

type IconServiceProps = Omit<IconBaseProps, "href"> & {
    name: IconServiceName;
}
export const IconService = forwardRef<SVGSVGElement, IconServiceProps>(({
    name,
    ...props
}, ref) => {
    const href = `${serviceSpriteHref}#${name}`;
    return <IconBase ref={ref} href={href} data-icon-type="service" data-icon-name={name} {...props} />;
});
IconService.displayName = "IconService";

type IconTextProps = Omit<IconBaseProps, "href"> & {
    name: IconTextName;
}
export const IconText = forwardRef<SVGSVGElement, IconTextProps>(({
    name,
    ...props
}, ref) => {
    const href = `${textSpriteHref}#${name}`;
    return <IconBase ref={ref} href={href} data-icon-type="text" data-icon-name={name} {...props} />;
});
IconText.displayName = "IconText";

export type CommonIconInfo = { name: IconCommonName; type: "Common" };
export type RoutineIconInfo = { name: IconRoutineName; type: "Routine" };
export type ServiceIconInfo = { name: IconServiceName; type: "Service" };
export type TextIconInfo = { name: IconTextName; type: "Text" };

export type IconInfo = CommonIconInfo | RoutineIconInfo | ServiceIconInfo | TextIconInfo;

type IconProps = Omit<IconBaseProps, "href"> & {
    info: IconInfo;
    /** Override the size prop with a specific value. Takes precedence over size prop. */
    sizeOverride?: number | null | undefined;
};
export const Icon = forwardRef<SVGSVGElement, IconProps>(({
    info,
    sizeOverride,
    size,
    ...props
}, ref) => {
    if (!info || typeof info !== "object" || !Object.prototype.hasOwnProperty.call(info, "name") || !Object.prototype.hasOwnProperty.call(info, "type")) {
        console.error("Invalid icon info", info, new Error().stack);
        return null;
    }
    const href = `${typeToHrefMap[info.type]}#${info.name}`;
    // Use sizeOverride if provided, otherwise use size
    const finalSize = sizeOverride !== null && sizeOverride !== undefined ? sizeOverride : size;
    return <IconBase ref={ref} href={href} size={finalSize} data-icon-type={info.type.toLowerCase()} data-icon-name={info.name} {...props} />;
});
Icon.displayName = "Icon";

type IconFaviconProps = {
    /** The URL to fetch the favicon for */
    href: string;
    /** The size of the icon in pixels */
    size?: number | null | undefined;
    /** The color to fill the fallback icon with */
    fill?: string | null | undefined;
    /** Whether the icon is decorative (for accessibility) */
    decorative?: boolean | null | undefined;
    /** Additional style properties */
    style?: React.CSSProperties;
    /** CSS class name */
    className?: string;
    /** ARIA label for accessibility */
    "aria-label"?: string;
    /** ARIA hidden attribute */
    "aria-hidden"?: boolean;
    /** Custom fallback icon info. If not provided, uses the Website icon */
    fallbackIcon?: IconInfo;
    /** Test ID for testing */
    "data-testid"?: string;
};

export const IconFavicon = forwardRef<HTMLElement, IconFaviconProps>(({
    href,
    size = DEFAULT_SIZE_PX,
    style,
    className,
    "aria-label": ariaLabel,
    "aria-hidden": ariaHidden,
    decorative = DEFAULT_DECORATIVE,
    fill,
    fallbackIcon,
    "data-testid": testId,
    ...props
}, ref) => {
    // Extract the domain from the URL
    let domain;
    try {
        const urlObj = new URL(href);
        domain = urlObj.hostname; // e.g., "www.example.com"
    } catch (e) {
        console.error(`[IconFavicon] Invalid URL: ${href}`, e);
        domain = null; // Handle non-website URLs like "mailto:"
    }

    // Create a favicon state outside conditional to avoid React Hook rules issues
    const [currentSource, setCurrentSource] = React.useState(0);

    // If we have a valid domain, render an img element
    if (domain) {
        const finalSize = size ?? DEFAULT_SIZE_PX;
        const protocol = href.startsWith("https") ? "https://" : "http://";

        // Create an array of favicon sources to try in order, from highest to lowest quality
        const faviconSources = [
            // Apple touch icon (often 180x180 or 192x192)
            `${protocol}${domain}/apple-touch-icon.png`,
            // Microsoft tile image (often 144x144)
            `${protocol}${domain}/mstile-144x144.png`,
            // Other common high-res favicon sizes
            `${protocol}${domain}/favicon-196x196.png`,
            `${protocol}${domain}/favicon-128x128.png`,
            `${protocol}${domain}/favicon-96x96.png`,
            // Standard favicon at root path
            `${protocol}${domain}/favicon.ico`,
            // DuckDuckGo icon service
            `https://icons.duckduckgo.com/ip3/${domain}.ico`,
            // Google favicon service (as final fallback)
            `https://www.google.com/s2/favicons?domain=${domain}&sz=64`,
        ];

        // Use a memoized function for handling errors to avoid recreation on each render
        const handleIconError = React.useCallback(() => {
            if (currentSource < faviconSources.length - 1) {
                setCurrentSource(currentSource + 1);
            }
        }, [currentSource, faviconSources.length]);

        return (
            <img
                ref={ref as React.Ref<HTMLImageElement>}
                src={faviconSources[currentSource]}
                width={finalSize}
                height={finalSize}
                style={{
                    objectFit: "contain",
                    ...style,
                }}
                className={className}
                aria-hidden={decorative && ariaHidden !== false ? "true" : undefined}
                alt={!decorative ? ariaLabel : ""}
                onError={handleIconError}
                data-testid={testId}
                data-icon-type="favicon"
                data-favicon-domain={domain}
                {...props}
            />
        );
    }

    // For invalid URLs, render the fallback icon using IconBase
    const fallbackHref = fallbackIcon
        ? `${typeToHrefMap[fallbackIcon.type]}#${fallbackIcon.name}`
        : `${commonSpriteHref}#Website`;

    return (
        <IconBase
            ref={ref as React.Ref<SVGSVGElement>}
            href={fallbackHref}
            size={size}
            style={style}
            className={className}
            aria-label={ariaLabel}
            aria-hidden={ariaHidden}
            decorative={decorative}
            fill={fill}
            data-testid={testId}
            data-icon-type="favicon-fallback"
            {...props}
        />
    );
});
IconFavicon.displayName = "IconFavicon";
