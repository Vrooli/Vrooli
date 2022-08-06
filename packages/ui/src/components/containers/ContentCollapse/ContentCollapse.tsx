// Used to display popular/search results of a particular object type
import { Box, Collapse, IconButton, Stack, Typography, useTheme } from '@mui/material';
import { ContentCollapseProps } from '../types';
import { HelpButton } from 'components';
import { useCallback, useEffect, useState } from 'react';
import {
    ExpandLess as ExpandLessIcon,
    ExpandMore as ExpandMoreIcon,
} from '@mui/icons-material';

export function ContentCollapse({
    helpText,
    isOpen = true,
    onOpenChange,
    showOnNoText = false,
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
        <Box sx={{
            padding: 1,
            color: Boolean(children) ? palette.background.textPrimary : palette.background.textSecondary
        }}>
            {/* Title with help button and collapse */}
            <Stack direction="row" alignItems="center">
                <Typography variant="h6">{title}</Typography>
                {helpText && <HelpButton markdown={helpText} />}
                <IconButton
                    id={`toggle-expand-icon-button-${title}`}
                    aria-label={internalIsOpen ? 'Collapse' : 'Expand'}
                    color="inherit"
                    onClick={toggleOpen}
                >
                    {internalIsOpen ? <ExpandMoreIcon id={`toggle-expand-icon-${title}`} /> : <ExpandLessIcon id={`toggle-expand-icon-${title}`} />}
                </IconButton>
            </Stack>
            {/* Text */}
            <Collapse in={internalIsOpen}>
                {children}
            </Collapse>
        </Box>
    );
}