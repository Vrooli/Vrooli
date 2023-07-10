// Used to display popular/search results of a particular object type
import { ExpandLessIcon, ExpandMoreIcon } from "@local/shared";
import { Box, Collapse, IconButton, Stack, Typography, useTheme } from "@mui/material";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { useCallback, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { ContentCollapseProps } from "../types";

export function ContentCollapse({
    children,
    helpText,
    id,
    isOpen = true,
    onOpenChange,
    sxs,
    title,
    titleComponent,
    titleKey,
    titleVariables,
    zIndex,
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

    return (
        <Box id={id} sx={{
            color: children ? palette.background.textPrimary : palette.background.textSecondary,
            ...(sxs?.root ?? {}),
        }}>
            {/* Title with help button and collapse */}
            <Stack direction="row" alignItems="center" sx={sxs?.titleContainer ?? {}}>
                <Typography component={titleComponent ?? "h6"} variant="h6">{titleText}</Typography>
                {helpText && <HelpButton
                    markdown={helpText}
                    sx={sxs?.helpButton ?? {}}
                    zIndex={zIndex}
                />}
                <IconButton
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
                </IconButton>
            </Stack>
            {/* Text */}
            <Collapse in={internalIsOpen}>
                {children}
            </Collapse>
        </Box>
    );
}
