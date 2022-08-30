import { useCallback, useState } from 'react';
import { Box, IconButton, Menu, Tooltip, useTheme } from '@mui/material';
import Markdown from 'markdown-to-jsx';
import { HelpButtonProps } from '../types';
import { TERTIARY_COLOR } from 'utils';
import { CloseIcon, HelpIcon } from '@shared/icons';

export const HelpButton = ({
    id = 'help-details-menu',
    markdown,
    onClick,
    sxRoot,
    sx,
}: HelpButtonProps) => {
    const { palette } = useTheme();
    const [anchorEl, setAnchorEl] = useState(null);
    const open = Boolean(anchorEl);

    const openMenu = useCallback((event) => {
        if (onClick) onClick(event);
        if (!anchorEl) setAnchorEl(event.currentTarget);
    }, [anchorEl, onClick]);
    const closeMenu = () => {
        setAnchorEl(null);
    };

    return (
        <Box
            sx={{
                display: 'inline',
                ...sxRoot,
            }}
        >
            <Tooltip placement='top' title={!open ? "Open Help Menu" : ''}>
                <IconButton
                    onClick={openMenu}
                    sx={{
                        display: 'inline-flex',
                        bottom: '0',
                    }}
                >
                    <HelpIcon fill={TERTIARY_COLOR} {...sx} />
                    <Menu
                        id={id}
                        open={open}
                        disableScrollLock={true}
                        anchorEl={anchorEl}
                        onClose={closeMenu}
                        anchorOrigin={{
                            vertical: 'bottom',
                            horizontal: 'right',
                        }}
                        transformOrigin={{
                            vertical: 'top',
                            horizontal: 'left',
                        }}
                        sx={{
                            '& .MuiPopover-paper': {
                                background: palette.background.paper,
                                maxWidth: 'min(90vw, 400px)',
                            },
                            '& .MuiMenu-list': {
                                padding: 0,
                            },
                            // Style visited, active, and hovered links differently
                            a: {
                                color: palette.mode === 'light' ? '#001cd3' : '#dd86db',
                                '&:visited': {
                                    color: palette.mode === 'light' ? '#001cd3' : '#f551ef',
                                },
                                '&:active': {
                                    color: palette.mode === 'light' ? '#001cd3' : '#f551ef',
                                },
                                '&:hover': {
                                    color: palette.mode === 'light' ? '#5a6ff6' : '#f3d4f2',
                                },
                            },
                        }}
                    >
                        <Box>
                            <Box sx={{
                                background: palette.primary.dark,
                                display: 'flex',
                                flexDirection: 'row-reverse',
                                paddingRight: '0.5rem',
                            }}>
                                <IconButton color="inherit" onClick={closeMenu} aria-label="close">
                                    <CloseIcon fill='white' />
                                </IconButton>
                            </Box>
                            <Box sx={{ padding: 1 }}>
                                <Markdown>{markdown}</Markdown>
                            </Box>
                        </Box>
                    </Menu>
                </IconButton>
            </Tooltip>
        </Box>
    )
}