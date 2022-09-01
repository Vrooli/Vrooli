/**
 * Shows valid/invalid/incomplete status of some object
 */
import { useCallback, useMemo, useState } from 'react';
import { Box, IconButton, Menu, Stack, Tooltip, Typography, useTheme } from '@mui/material';
import Markdown from 'markdown-to-jsx';
import { StatusButtonProps } from '../types';
import { Status } from 'utils';
import { noSelect } from 'styles';
import { CloseIcon, RoutineIncompleteIcon, RoutineInvalidIcon, RoutineValidIcon } from '@shared/icons';

/**
 * Status indicator and slider change color to represent routine's status
 */
const STATUS_COLOR = {
    [Status.Incomplete]: '#a0b121', // Yellow
    [Status.Invalid]: '#ff6a6a', // Red
    [Status.Valid]: '#00d51e', // Green
}
const STATUS_LABEL = {
    [Status.Incomplete]: 'Incomplete',
    [Status.Invalid]: 'Invalid',
    [Status.Valid]: 'Valid',
}
const STATUS_ICON = {
    [Status.Incomplete]: RoutineIncompleteIcon,
    [Status.Invalid]: RoutineInvalidIcon,
    [Status.Valid]: RoutineValidIcon,
}

export const StatusButton = ({
    status,
    messages,
    sx,
}: StatusButtonProps) => {
    const { palette } = useTheme();

    /**
     * List of status messages converted to markdown. 
     * If one message, no bullet points. If multiple, bullet points.
     */
    const statusMarkdown = useMemo(() => {
        if (messages.length === 0) return 'Routine is valid.';
        if (messages.length === 1) {
            return messages[0];
        }
        return messages.map((s) => {
            return `* ${s}`;
        }).join('\n');
    }, [messages]);

    const StatusIcon = useMemo(() => STATUS_ICON[status], [status]);

    const [anchorEl, setAnchorEl] = useState<HTMLElement | null>(null);
    const open = Boolean(anchorEl);
    const openMenu = useCallback((event) => {
        if (!anchorEl) setAnchorEl(event.currentTarget);
    }, [anchorEl])
    const closeMenu = () => {
        setAnchorEl(null);
    };

    const menu = useMemo(() => (
        <Box>
            <Box sx={{ background: palette.primary.dark }}>
                <IconButton edge="end" color="inherit" onClick={closeMenu} aria-label="close">
                    <CloseIcon fill={palette.primary.contrastText} />
                </IconButton>
            </Box>
            <Box sx={{ padding: 1 }}>
                <Markdown>{statusMarkdown}</Markdown>
            </Box>
        </Box>
    ), [statusMarkdown, palette.primary.dark, palette.primary.contrastText]);

    return (
        <>
            <Tooltip title='Press for details'>
                <Stack
                    direction="row"
                    spacing={1}
                    onClick={openMenu}
                    sx={{
                        ...(noSelect as any),
                        display: 'flex',
                        justifyContent: 'center',
                        alignItems: 'center',
                        cursor: 'pointer',
                        background: STATUS_COLOR[status],
                        color: 'white',
                        padding: '4px',
                        borderRadius: '16px',
                        ...(sx ?? {})
                    }}>
                    <StatusIcon fill='white' />
                    <Typography
                        variant='body2'
                        sx={{
                            // Hide on small screens
                            display: { xs: 'none', sm: 'inline' },
                            paddingRight: '4px',
                        }}>{STATUS_LABEL[status]}</Typography>
                </Stack>
            </Tooltip>
            <Menu
                id='status-menu'
                open={open}
                disableScrollLock={true}
                anchorEl={anchorEl}
                onClose={closeMenu}
                anchorOrigin={{
                    vertical: 'bottom',
                    horizontal: 'center',
                }}
                transformOrigin={{
                    vertical: 'top',
                    horizontal: 'center',
                }}
                sx={{
                    '& .MuiPopover-paper': {
                        background: palette.background.default,
                        maxWidth: 'min(100vw, 400px)',
                    },
                    '& .MuiMenu-list': {
                        padding: 0,
                    }
                }}
            >
                {menu}
            </Menu>
        </>
    )
}