import {
    AppBar,
    Box,
    Button,
    Dialog,
    Grid,
    IconButton,
    Toolbar,
    Typography,
} from '@mui/material';
import {
    AddCircle as AddCircleIcon,
    Cancel as CancelIcon,
    Close as CloseIcon,
} from '@mui/icons-material';
import { UpTransition } from 'components';
import { CSSProperties } from 'react';
import { AddDialogBaseProps } from '../types';
import { centeredText } from 'styles';

/**
 * Dialog for displaying any "Add" form
 * @returns 
 */
export const AddDialogBase = ({
    title,
    open = true,
    onSubmit,
    onClose,
    children,
}: AddDialogBaseProps) => {

    let options = (
        <Grid
            container
            spacing={2}
            sx={{
                padding: 2,
                background: (t) => t.palette.primary.main,
            }}
        >
            <Grid item xs={12} sm={6}>
                <Button
                    fullWidth
                    startIcon={<AddCircleIcon />}
                    onClick={onSubmit}
                >Create</Button>
            </Grid>
            <Grid item xs={12} sm={6}>
                <Button
                    fullWidth
                    startIcon={<CancelIcon />}
                    onClick={onClose}
                >Cancel</Button>
            </Grid>
        </Grid>
    );

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
                    <Grid container spacing={0}>
                        <Grid item xs={12} sx={{ ...centeredText }}>
                            <Typography variant="h5">
                                {title}
                            </Typography>
                        </Grid>
                    </Grid>
                </Toolbar>
            </AppBar>
            <Box
                sx={{
                    background: (t) => t.palette.background.default,
                    flex: 'auto',
                    padding: 1,
                    paddingBottom: '15vh',
                    width: '100%',
                    marginTop: 3,
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