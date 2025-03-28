import { useTheme, type PaletteColor } from "@mui/material";
import React, { forwardRef } from "react";
import type { IconName as IconCommonName } from "./types/commonIcons.js";
import type { IconName as IconRoutineName } from "./types/routineIcons.js";
import type { IconName as IconServiceName } from "./types/serviceIcons.js";
import type { IconName as IconTextName } from "./types/textIcons.js";
import commonSpriteHref from "/sprites/common-sprite.svg";
import routineSpriteHref from "/sprites/routine-sprite.svg";
import serviceSpriteHref from "/sprites/service-sprite.svg";
import textSpriteHref from "/sprites/text-sprite.svg";

const DEFAULT_SIZE_PX = 24;
const DEFAULT_FILL = "currentColor";
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

    // Handle theme color names
    let fillColor = fill ?? DEFAULT_FILL;
    if (typeof fillColor === "string") {
        // Handle dot notation (e.g., "text.secondary", "grey.500")
        const colorPath = fillColor.split(".");
        let paletteColor: any = theme.palette;

        // Traverse the color path
        for (const key of colorPath) {
            if (paletteColor && typeof paletteColor === "object") {
                paletteColor = paletteColor[key as keyof typeof paletteColor];
            } else {
                break;
            }
        }

        // If we found a valid color value, use it
        if (paletteColor && (typeof paletteColor === "string" || typeof paletteColor === "object")) {
            fillColor = typeof paletteColor === "object" && "main" in paletteColor
                ? (paletteColor as PaletteColor).main
                : paletteColor;
        }
    }

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
    return <IconBase ref={ref} href={href} {...props} />;
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
    return <IconBase ref={ref} href={href} {...props} />;
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
    return <IconBase ref={ref} href={href} {...props} />;
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
    return <IconBase ref={ref} href={href} {...props} />;
});
IconText.displayName = "IconText";

export type CommonIconInfo = { name: IconCommonName; type: "Common" };
export type RoutineIconInfo = { name: IconRoutineName; type: "Routine" };
export type ServiceIconInfo = { name: IconServiceName; type: "Service" };
export type TextIconInfo = { name: IconTextName; type: "Text" };

export type IconInfo = CommonIconInfo | RoutineIconInfo | ServiceIconInfo | TextIconInfo;

type IconProps = Omit<IconBaseProps, "href"> & {
    info: IconInfo;
};
export const Icon = forwardRef<SVGSVGElement, IconProps>(({
    info,
    ...props
}, ref) => {
    if (!info || typeof info !== "object" || !Object.prototype.hasOwnProperty.call(info, "name") || !Object.prototype.hasOwnProperty.call(info, "type")) {
        console.error("Invalid icon info", info, new Error().stack);
        return null;
    }
    const href = `${typeToHrefMap[info.type]}#${info.name}`;
    return <IconBase ref={ref} href={href} {...props} />;
});
Icon.displayName = "Icon";
