import { useCallback, useMemo, useState } from 'react';
import { Box, IconButton, Menu, Tooltip } from '@mui/material';
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
                <Box sx={{ background: (t) => t.palette.primary.dark }}>
                    <IconButton edge="start" color="inherit" onClick={closeMenu} aria-label="close">
                        <CloseIcon sx={{ fill: 'white', marginLeft: '0.5em' }}/>
                    </IconButton>
                </Box>
                <Box sx={{ padding: 1 }}>
                    <Markdown>{markdown}</Markdown>
                </Box>
            </Box>
        )
    }, [markdown])

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
                                background: (t) => t.palette.background.paper,
                                maxWidth: 'min(100vw, 400px)',
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