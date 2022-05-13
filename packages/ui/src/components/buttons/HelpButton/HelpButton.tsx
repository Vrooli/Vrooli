import { useCallback, useMemo, useState } from 'react';
import { Box, IconButton, Menu, Tooltip, useTheme } from '@mui/material';
import { HelpOutline as HelpIcon } from "@mui/icons-material";
import Markdown from 'markdown-to-jsx';
import { HelpButtonProps } from '../types';
import { Close as CloseIcon } from '@mui/icons-material';

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

    const menu = useMemo(() => {
        return (
            <Box>
                <Box sx={{ 
                    background: palette.mode === 'light' ? palette.primary.dark : palette.secondary.dark,
                    display: 'flex',
                    flexDirection: 'row-reverse',
                    paddingRight: '0.5rem',
                }}>
                    <IconButton color="inherit" onClick={closeMenu} aria-label="close">
                        <CloseIcon sx={{ fill: 'white' }}/>
                    </IconButton>
                </Box>
                <Box sx={{ padding: 1 }}>
                    <Markdown>{markdown}</Markdown>
                </Box>
            </Box>
        )
    }, [markdown, palette.mode, palette.primary.dark, palette.secondary.dark])

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
                    <HelpIcon sx={{ fill: 'gb(103 103 104 / 87%)', ...sx }} />
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
                            }
                        }}
                    >
                        {menu}
                    </Menu>
                </IconButton>
            </Tooltip>
        </Box>
    )
}