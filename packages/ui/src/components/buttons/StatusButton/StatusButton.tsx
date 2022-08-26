/**
 * Shows valid/invalid/incomplete status of some object
 */
import { useCallback, useMemo, useState } from 'react';
import { Box, Chip, IconButton, Menu, Tooltip, useTheme } from '@mui/material';
import Markdown from 'markdown-to-jsx';
import { StatusButtonProps } from '../types';
import {
    Mood as ValidIcon,
    MoodBad as InvalidIcon,
    SentimentDissatisfied as IncompleteIcon,
} from '@mui/icons-material';
import { Status } from 'utils';
import { noSelect } from 'styles';
import { CloseIcon } from 'assets/img';

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
    [Status.Incomplete]: IncompleteIcon,
    [Status.Invalid]: InvalidIcon,
    [Status.Valid]: ValidIcon,
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

    const [anchorEl, setAnchorEl] = useState(null);
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
        <Box sx={{ 
            display: 'flex',
            justifyContent: 'center',
            alignItems: 'center',
            ...(sx ?? {})
        }}>
            <Tooltip title='Press for details'>
                <Chip
                    icon={<StatusIcon sx={{ fill: 'white' }} />}
                    label={STATUS_LABEL[status]}
                    onClick={openMenu}
                    sx={{
                        ...noSelect,
                        background: STATUS_COLOR[status],
                        color: 'white',
                        cursor: 'pointer',
                        // Hide label on small screens
                        '& .MuiChip-label': {
                            display: { xs: 'none', sm: 'block' },
                        },
                        // Hiding label messes up spacing with icon
                        '& .MuiSvgIcon-root': {
                            marginLeft: '4px',
                            marginRight: { xs: '4px', sm: '-4px' },
                        },
                    }}
                />
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
        </Box>
    )
}