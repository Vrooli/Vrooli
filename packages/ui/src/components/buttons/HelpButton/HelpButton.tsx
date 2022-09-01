import { useCallback, useState } from 'react';
import { Box, IconButton, Menu, Tooltip, useTheme } from '@mui/material';
import Markdown from 'markdown-to-jsx';
import { HelpButtonProps } from '../types';
import { MenuTitle } from 'components/dialogs';
import { HelpIcon } from '@shared/icons';

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
                        verticalAlign: 'top',
                    }}
                >
                    <HelpIcon fill={palette.secondary.main} {...sx} />
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
                        <MenuTitle onClose={closeMenu} />
                        <Box sx={{ padding: 1 }}>
                            <Markdown>{markdown}</Markdown>
                        </Box>
                    </Menu>
                </IconButton>
            </Tooltip>
        </Box>
    )
}