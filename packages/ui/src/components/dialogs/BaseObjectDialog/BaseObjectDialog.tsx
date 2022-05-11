import {
    AppBar,
    Box,
    Dialog,
    IconButton,
    Slide,
    Stack,
    Toolbar,
    Tooltip,
    Typography,
    useScrollTrigger,
    useTheme,
} from '@mui/material';
import {
    ChevronLeft as PreviousIcon,
    ChevronRight as NextIcon,
    Close as CloseIcon,
} from '@mui/icons-material';
import { UpTransition } from 'components';
import { useCallback, useState } from 'react';
import { BaseObjectDialogProps as BaseObjectDialogProps, ObjectDialogAction } from '../types';

/**
 * Dialog for displaying any "Add" form
 * @returns 
 */
export const BaseObjectDialog = ({
    children,
    hasNext,
    hasPrevious,
    onAction,
    open = true,
    title,
}: BaseObjectDialogProps) => {
    const { palette } = useTheme();

    const [scrollTarget, setScrollTarget] = useState<HTMLElement | undefined>(undefined);
    const scrollTrigger = useScrollTrigger({ target: scrollTarget });

    const onClose = useCallback(() => onAction(ObjectDialogAction.Close), [onAction]);
    const onPrevious = useCallback(() => onAction(ObjectDialogAction.Previous), [onAction]);
    const onNext = useCallback(() => onAction(ObjectDialogAction.Next), [onAction]);

    return (
        <Dialog
            fullScreen
            open={open}
            onClose={onClose}
            TransitionComponent={UpTransition}
        >
            {/* TODO hide not working */}
            <Slide appear={false} direction="down" in={!scrollTrigger}>
                <AppBar ref={node => {
                    if (node) {
                        setScrollTarget(node);
                    }
                }}>
                    <Toolbar>
                        <IconButton edge="start" color="inherit" onClick={onClose} aria-label="close">
                            <CloseIcon />
                        </IconButton>
                        <Box sx={{ width: '100%', alignItems: 'center' }}>
                            <Stack direction="row" spacing={1} justifyContent="center" alignItems="center">
                                {/* Navigate to previous */}
                                {hasPrevious && (
                                    <Tooltip title="Previous" placement="bottom">
                                        <IconButton size="large" onClick={onPrevious} aria-label="previous">
                                            <PreviousIcon sx={{ fill: 'white' }} />
                                        </IconButton>
                                    </Tooltip>
                                )}
                                {/* Title */}
                                <Typography variant="h5">
                                    {title}
                                </Typography>
                                {/* Navigate to next */}
                                {hasNext && (
                                    <Tooltip title="Next" placement="bottom">
                                        <IconButton size="large" onClick={onNext} aria-label="next">
                                            <NextIcon sx={{ fill: 'white' }} />
                                        </IconButton>
                                    </Tooltip>
                                )}
                            </Stack>
                        </Box>
                    </Toolbar>
                </AppBar>
            </Slide>
            <Box
                sx={{
                    background: palette.background.default,
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