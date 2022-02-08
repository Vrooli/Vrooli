import {
    AppBar,
    Box,
    Button,
    Dialog,
    Grid,
    IconButton,
    Slide,
    Stack,
    Toolbar,
    Tooltip,
    Typography,
    useScrollTrigger,
} from '@mui/material';
import {
    Add as AddIcon,
    ChevronLeft as PreviousIcon,
    ChevronRight as NextIcon,
    Close as CloseIcon,
    Edit as EditIcon,
    Restore as CancelIcon,
    Update as SaveIcon,
} from '@mui/icons-material';
import { UpTransition } from 'components';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { BaseObjectDialogProps as BaseObjectDialogProps, ObjectDialogAction, ObjectDialogState } from '../types';

/**
 * Dialog for displaying any "Add" form
 * @returns 
 */
export const BaseObjectDialog = ({
    title,
    open = true,
    hasPrevious,
    hasNext,
    canEdit = false,
    state,
    onAction,
    children,
}: BaseObjectDialogProps) => {
    const [scrollTarget, setScrollTarget] = useState<HTMLElement | undefined>(undefined);
    const scrollTrigger = useScrollTrigger({ target: scrollTarget });

    const onAdd = useCallback(() => onAction(ObjectDialogAction.Add), [onAction]);
    const onCancel = useCallback(() => onAction(ObjectDialogAction.Cancel), [onAction]);
    const onClose = useCallback(() => onAction(ObjectDialogAction.Close), [onAction]);
    const onEdit = useCallback(() => onAction(ObjectDialogAction.Edit), [onAction]);
    const onPrevious = useCallback(() => onAction(ObjectDialogAction.Previous), [onAction]);
    const onNext = useCallback(() => onAction(ObjectDialogAction.Next), [onAction]);
    const onSave = useCallback(() => onAction(ObjectDialogAction.Save), [onAction]);

    /**
     * Determine option buttons to display (besides close, since that's always available). 
     * - If viewing an object (i.e. not adding or updating):
     *      - If canEdit, show edit button
     *      - Else, show no button
     * - If editing, show "Save" and "Cancel" buttons.
     * - If adding, show "Add" button
     */
    const options: JSX.Element = useMemo(() => {
        // Determine edit options to show
        let availableOptions: Array<[string, any, any]> = [];
        switch (state) {
            case ObjectDialogState.View:
                availableOptions = canEdit ? [['Edit', EditIcon, onEdit]] : [];
                break;
            case ObjectDialogState.Edit:
                availableOptions = [['Save', SaveIcon, onSave], ['Cancel', CancelIcon, onCancel]];
                break;
            case ObjectDialogState.Add:
                availableOptions = [['Add', AddIcon, onAdd]];
                break;
        }

        // Determine sizing based on number of options
        const maxGridWidth = `min(100vw, ${200 * availableOptions.length}px)`;
        let gridItemSizes;
        if (availableOptions.length === 1) {
            gridItemSizes = { xs: 12 }
        }
        else if (availableOptions.length === 2) {
            gridItemSizes = { xs: 12, sm: 6 }
        }
        else {
            gridItemSizes = { xs: 12, sm: 6, md: 4 }
        }

        return (
            <Grid container spacing={2} maxWidth={maxGridWidth}>
                {availableOptions.map(([label, Icon, onClick]) => (
                    <Grid key={label} {...gridItemSizes}>
                        <Button
                            fullWidth
                            startIcon={<Icon />}
                            onClick={onClick ? onClick() : undefined}
                        >{label}</Button>
                    </Grid>
                ))}
            </Grid>
        )
    }, [state, canEdit, onAction]);

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
                                {hasPrevious && (
                                    <Tooltip title="Previous" placement="bottom">
                                        <IconButton size="large" onClick={onPrevious} aria-label="previous">
                                            <PreviousIcon sx={{ fill: 'white' }} />
                                        </IconButton>
                                    </Tooltip>
                                )}
                                <Typography variant="h5">
                                    {title}
                                </Typography>
                                {hasNext && (
                                    <Tooltip title="Next" placement="bottom">
                                        <IconButton size="large" onClick={onNext} aria-label="next">
                                            <NextIcon sx={{ fill: 'white' }} />
                                        </IconButton>
                                    </Tooltip>
                                )}
                            </Stack>
                        </Box>
                        {options}
                    </Toolbar>
                </AppBar>
            </Slide>
            <Box
                sx={{
                    background: (t) => t.palette.background.default,
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