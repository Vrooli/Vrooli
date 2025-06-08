import Box from "@mui/material/Box";
import Collapse from "@mui/material/Collapse";
import IconButton from "@mui/material/IconButton";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { useTheme } from "@mui/material/styles";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { IconCommon } from "../../icons/Icons.js";
import { HelpButton } from "../buttons/HelpButton.js";
import { type ContentCollapseProps } from "./types.js";

export function ContentCollapse({
    children,
    disableCollapse,
    helpText,
    id,
    isOpen = true,
    onOpenChange,
    sxs,
    title,
    titleComponent,
    titleVariant,
    titleKey,
    titleVariables,
    toTheRight,
}: ContentCollapseProps) {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const [internalIsOpen, setInternalIsOpen] = useState(isOpen);
    useEffect(() => {
        setInternalIsOpen(isOpen);
    }, [isOpen]);

    const toggleOpen = useCallback(() => {
        setInternalIsOpen(!internalIsOpen);
        if (onOpenChange) {
            onOpenChange(!internalIsOpen);
        }
    }, [internalIsOpen, onOpenChange]);

    // Calculate fill color
    const fillColor = sxs?.root?.color ?? (children ? palette.background.textPrimary : palette.background.textSecondary);

    const titleText = useMemo(() => {
        if (titleKey) return t(titleKey, { ...titleVariables, defaultValue: title ?? "" });
        if (title) return title;
        return "";
    }, [title, titleKey, titleVariables, t]);

    const outerBoxStyle = useMemo(function outerBoxStyleMemo() {
        return {
            color: children ? palette.background.textPrimary : palette.background.textSecondary,
            ...sxs?.root,
        };
    }, [children, palette.background.textPrimary, palette.background.textSecondary, sxs?.root]);

    return (
        <Box id={id} sx={outerBoxStyle}>
            {/* Title with help button and collapse */}
            <Stack direction="row" alignItems="center" sx={sxs?.titleContainer}>
                <Typography component={titleComponent ?? "h6"} variant={titleVariant ?? "h6"}>{titleText}</Typography>
                {helpText && <HelpButton
                    markdown={helpText}
                />}
                {!disableCollapse && <IconButton
                    id={`toggle-expand-icon-button-${title}`}
                    aria-label={t(internalIsOpen ? "Collapse" : "Expand")}
                    aria-expanded={internalIsOpen}
                    onClick={toggleOpen}
                >
                    {internalIsOpen ?
                        <IconCommon
                            decorative
                            fill={fillColor}
                            id={`toggle-expand-icon-${title}`}
                            name="ExpandMore"
                        /> :
                        <IconCommon
                            decorative
                            fill={fillColor}
                            id={`toggle-expand-icon-${title}`}
                            name="ExpandLess"
                        />}
                </IconButton>}
                {toTheRight}
            </Stack>
            {/* Text within Collapse, or just children if collapse is disabled */}
            {disableCollapse ? children : (
                <Collapse in={internalIsOpen}>
                    {children}
                </Collapse>
            )}
        </Box>
    );
}
