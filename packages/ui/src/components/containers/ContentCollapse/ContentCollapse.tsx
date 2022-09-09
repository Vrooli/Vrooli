// Used to display popular/search results of a particular object type
import { Box, Collapse, IconButton, Stack, Typography, useTheme } from '@mui/material';
import { ContentCollapseProps } from '../types';
import { HelpButton } from 'components';
import { useCallback, useEffect, useState } from 'react';
import { ExpandLessIcon, ExpandMoreIcon } from '@shared/icons';

export function ContentCollapse({
    helpText,
    id,
    isOpen = true,
    onOpenChange,
    sxs,
    title,
    children,
}: ContentCollapseProps) {
    const { palette } = useTheme();

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

    return (
        <Box id={id} sx={{
            padding: 1,
            color: Boolean(children) ? palette.background.textPrimary : palette.background.textSecondary,
            ...(sxs?.root ?? {}),
        }}>
            {/* Title with help button and collapse */}
            <Stack direction="row" alignItems="center" sx={sxs?.titleContainer ?? {}}>
                <Typography variant="h6">{title}</Typography>
                {helpText && <HelpButton markdown={helpText} />}
                <IconButton
                    id={`toggle-expand-icon-button-${title}`}
                    aria-label={internalIsOpen ? 'Collapse' : 'Expand'}
                    onClick={toggleOpen}
                >
                    {internalIsOpen ?
                        <ExpandMoreIcon
                            id={`toggle-expand-icon-${title}`}
                            fill={Boolean(children) ? palette.background.textPrimary : palette.background.textSecondary}
                        /> :
                        <ExpandLessIcon
                            id={`toggle-expand-icon-${title}`}
                            fill={Boolean(children) ? palette.background.textPrimary : palette.background.textSecondary}
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