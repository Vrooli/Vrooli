import {
    AppBar,
    Box,
    Button,
    Dialog,
    Grid,
    IconButton,
    Stack,
    Toolbar,
    Tooltip,
    Typography,
} from '@mui/material';
import {
    ChevronLeft as ChevronLeftIcon,
    ChevronRight as ChevronRightIcon,
    Close as CloseIcon,
    Edit as EditIcon,
    Restore as RestoreIcon,
    Update as UpdateIcon,
} from '@mui/icons-material';
import { UpTransition } from 'components';
import { CSSProperties, useMemo } from 'react';
import { ViewDialogBaseProps } from '../types';

/**
 * Dialog for displaying any "Add" form
 * @returns 
 */
export const ViewDialogBase = ({
    title,
    open = true,
    canEdit = false,
    isEditing = false,
    onEdit,
    onSave,
    onRevert,
    onClose,
    children,
    hasPrevious,
    hasNext,
    onPrevious,
    onNext,
}: ViewDialogBaseProps) => {

    const options = useMemo(() => {
        // Determine edit options to show
        let availableOptions: Array<[string, any, any]> = [];
        if (isEditing) {
            availableOptions.push(
                ['Revert', RestoreIcon, onRevert],
                ['Save', UpdateIcon, onSave]
            );
        }
        else if (canEdit) {
            availableOptions.push(['Edit', EditIcon, onEdit]);
        }
        // If no options, don't show anything
        if (availableOptions.length === 0) return null;

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
            <Box p={2} sx={{ background: (t) => t.palette.primary.main }}>
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
            </Box>
        )

    }, [canEdit, isEditing])

    return (
        <Dialog
            fullScreen
            open={open}
            onClose={onClose}
            TransitionComponent={UpTransition}
        >
            <AppBar sx={{ position: 'relative' } as CSSProperties}>
                <Toolbar>
                    <IconButton edge="start" color="inherit" onClick={onClose} aria-label="close">
                        <CloseIcon />
                    </IconButton>
                    <Box sx={{ width: '100%', alignItems: 'center' }}>
                        <Stack direction="row" spacing={1} justifyContent="center" alignItems="center">
                            {hasPrevious && (
                                <Tooltip title="Previous" placement="bottom">
                                    <IconButton size="large" onClick={onPrevious} aria-label="previous">
                                        <ChevronLeftIcon sx={{ fill: 'white' }} />
                                    </IconButton>
                                </Tooltip>
                            )}
                            <Typography variant="h5">
                                {title}
                            </Typography>
                            {hasNext && (
                                <Tooltip title="Next" placement="bottom">
                                    <IconButton size="large" onClick={onNext} aria-label="next">
                                        <ChevronRightIcon sx={{ fill: 'white' }} />
                                    </IconButton>
                                </Tooltip>
                            )}
                        </Stack>
                    </Box>
                </Toolbar>
            </AppBar>
            <Box
                sx={{
                    background: (t) => t.palette.background.default,
                    flex: 'auto',
                    padding: 0,
                    paddingBottom: '15vh',
                    width: '100%',
                }}
            >
                {children}
                <Box
                    sx={{
                        background: (t) => t.palette.primary.main,
                        position: 'fixed',
                        bottom: '0',
                        width: '-webkit-fill-available',
                        zIndex: 1,
                    }}
                >
                    {options}
                </Box>
            </Box>
        </Dialog>
    );
}