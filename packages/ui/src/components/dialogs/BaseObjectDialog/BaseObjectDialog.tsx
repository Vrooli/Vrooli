import {
    AppBar,
    Box,
    Dialog,
    IconButton,
    Slide,
    Toolbar,
    Typography,
    useScrollTrigger,
    useTheme,
} from '@mui/material';
import {
    Close as CloseIcon,
} from '@mui/icons-material';
import { useCallback, useState } from 'react';
import { BaseObjectDialogProps, ObjectDialogAction } from '../types';

/**
 * Dialog for displaying any "Add" form
 * @returns 
 */
export const BaseObjectDialog = ({
    children,
    onAction,
    open = true,
    title = '',
    zIndex,
}: BaseObjectDialogProps) => {
    const { palette } = useTheme();

    const [scrollTarget, setScrollTarget] = useState<HTMLElement | undefined>(undefined);
    const scrollTrigger = useScrollTrigger({ target: scrollTarget });

    const onClose = useCallback(() => onAction(ObjectDialogAction.Close), [onAction]);

    return (
        <Dialog
            fullScreen
            open={open}
            onClose={onClose}
            sx={{
                zIndex,
            }}
        >
            {/* TODO hide not working */}
            <Slide appear={false} direction="down" in={!scrollTrigger}>
                <AppBar ref={node => {
                    if (node) {
                        setScrollTarget(node);
                    }
                }}>
                    <Toolbar sx={{
                        background: palette.primary.dark,
                        color: palette.primary.contrastText,
                        width: '100vw',
                    }}>
                        {/* Title */}
                        <Typography variant="h5" sx={{ marginLeft: 'auto' }}>
                            {title}
                        </Typography>
                        {/* Close icon */}
                        <IconButton
                            edge="end"
                            color="inherit"
                            onClick={onClose}
                            aria-label="close"
                            sx={{
                                marginLeft: 'auto'
                            }}
                        >
                            <CloseIcon />
                        </IconButton>
                    </Toolbar>
                </AppBar>
            </Slide>
            <Box
                sx={{
                    background: palette.mode === 'light' ? '#c2cadd' : palette.background.default,
                    flex: 'auto',
                    padding: 0,
                    paddingTop: { xs: '56px', sm: '64px' },
                    width: '100%',
                }}
            >
                {children}
            </Box>
        </Dialog>
    );
}