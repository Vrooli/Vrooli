/**
 * Drawer to display overall orchestration info on the orchestration page. 
 * Swipes left from right of screen
 */
import { useState } from 'react';
import {
    Close as CloseIcon,
    Info as InfoIcon,
} from '@mui/icons-material';
import {
    IconButton,
    List,
    SwipeableDrawer,
    SxProps,
    Theme,
} from '@mui/material';
import { OrchestrationInfoDialogProps } from '../types';

export const OrchestrationInfoDialog = ({
    sxs,
    routineInfo,
    onUpdate,
    onCancel
}: OrchestrationInfoDialogProps) => {
    const [open, setOpen] = useState(false);

    const toggleOpen = () => setOpen(o => !o);
    const closeMenu = () => setOpen(false);

    return (
        <>
            <IconButton edge="start" color="inherit" aria-label="menu" onClick={toggleOpen}>
                <InfoIcon sx={sxs?.icon} />
            </IconButton>
            <SwipeableDrawer
                anchor="right"
                open={open}
                onOpen={() => {}}
                onClose={closeMenu}
                sx={{
                    '& .MuiDrawer-paper': {
                        background: (t) => t.palette.primary.light,
                        borderLeft: `1px solid ${(t) => t.palette.text.primary}`,
                    }
                }}
            >
                <IconButton onClick={closeMenu} sx={{
                    color: (t) => t.palette.primary.contrastText,
                    borderRadius: 0,
                    borderBottom: `1px solid ${(t) => t.palette.primary.dark}`,
                    justifyContent: 'end',
                    flexDirection: 'row-reverse',
                }}>
                    <CloseIcon fontSize="large" />
                </IconButton>
                <List>
                </List>
            </SwipeableDrawer>
        </>
    );
}