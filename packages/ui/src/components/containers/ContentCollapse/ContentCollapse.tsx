import { Box, Collapse, IconButton, Stack, Typography, useTheme } from "@mui/material";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { ExpandLessIcon, ExpandMoreIcon } from "icons";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { ContentCollapseProps } from "../types";

export const ContentCollapse = ({
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
}: ContentCollapseProps) => {
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

    return (
        <Box id={id} sx={{
            color: children ? palette.background.textPrimary : palette.background.textSecondary,
            ...(sxs?.root ?? {}),
        }}>
            {/* Title with help button and collapse */}
            <Stack direction="row" alignItems="center" sx={sxs?.titleContainer ?? {}}>
                <Typography component={titleComponent ?? "h6"} variant={titleVariant ?? "h6"}>{titleText}</Typography>
                {helpText && <HelpButton
                    markdown={helpText}
                    sx={sxs?.helpButton ?? {}}
                />}
                {!disableCollapse && <IconButton
                    id={`toggle-expand-icon-button-${title}`}
                    aria-label={t(internalIsOpen ? "Collapse" : "Expand")}
                    onClick={toggleOpen}
                >
                    {internalIsOpen ?
                        <ExpandMoreIcon
                            id={`toggle-expand-icon-${title}`}
                            fill={fillColor}
                        /> :
                        <ExpandLessIcon
                            id={`toggle-expand-icon-${title}`}
                            fill={fillColor}
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
};
