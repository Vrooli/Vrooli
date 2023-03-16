// Used to display popular/search results of a particular object type
import { Box, Collapse, IconButton, Stack, Typography, useTheme } from '@mui/material';
import { ExpandLessIcon, ExpandMoreIcon } from '@shared/icons';
import { HelpButton } from 'components/buttons/HelpButton/HelpButton';
import { useCallback, useEffect, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { ContentCollapseProps } from '../types';

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
    const fillColor = sxs?.root?.color ?? (Boolean(children) ? palette.background.textPrimary : palette.background.textSecondary);

    return (
        <Box id={id} sx={{
            color: Boolean(children) ? palette.background.textPrimary : palette.background.textSecondary,
            ...(sxs?.root ?? {}),
        }}>
            {/* Title with help button and collapse */}
            <Stack direction="row" alignItems="center" sx={sxs?.titleContainer ?? {}}>
                <Typography component={titleComponent ?? 'h6'} variant="h6">{t(titleKey!, {...titleVariables, defaultValue: title ?? '' })}</Typography>
                {helpText && <HelpButton markdown={helpText} />}
                <IconButton
                    id={`toggle-expand-icon-button-${title}`}
                    aria-label={t(internalIsOpen ? 'Collapse' : 'Expand')}
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