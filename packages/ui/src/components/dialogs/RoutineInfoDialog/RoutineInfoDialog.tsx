/**
 * Drawer to display a routine list item's info on the orchestration page. 
 * Swipes up from bottom of screen
 */
import { useCallback, useState } from 'react';
import {
    Close as CloseIcon,
    Info as InfoIcon,
} from '@mui/icons-material';
import {
    Box,
    IconButton,
    List,
    SwipeableDrawer,
    SxProps,
    Theme,
} from '@mui/material';
import { useLocation } from 'wouter';
import { RoutineInfoDialogProps } from '../types';

export const RoutineInfoDialog = ({
    open,
    routineInfo,
    onClose,
}: RoutineInfoDialogProps) => {

    return (
        <SwipeableDrawer
            anchor="bottom"
            variant='persistent'
            open={open}
            onOpen={() => { }}
            onClose={onClose}
            sx={{
                '& .MuiDrawer-paper': {
                    background: (t) => t.palette.primary.light,
                    borderLeft: `1px solid ${(t) => t.palette.text.primary}`,
                }
            }}
        >
            <Box>
                <IconButton onClick={onClose} sx={{
                    color: (t) => t.palette.primary.contrastText,
                    borderRadius: 0,
                    borderBottom: `1px solid ${(t) => t.palette.primary.dark}`,
                    justifyContent: 'end',
                }}>
                    <CloseIcon fontSize="large" />
                </IconButton>
            </Box>
        </SwipeableDrawer>
    );
}