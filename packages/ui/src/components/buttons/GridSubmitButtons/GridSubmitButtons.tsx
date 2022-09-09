/**
 * Prompts user to select which link the new node should be added on
 */
import { Box, Button, Grid, Popover, useTheme } from '@mui/material';
import { CancelIcon, CreateIcon, SaveIcon } from '@shared/icons';
import Markdown from 'markdown-to-jsx';
import { useCallback, useMemo, useState } from 'react';
import { GridSubmitButtonsProps } from '../types';

const capitalizeFirstLetter = (str: string) => { return str.charAt(0).toUpperCase() + str.slice(1); };

export const GridSubmitButtons = ({
    disabledCancel,
    disabledSubmit,
    errors,
    isCreate,
    onCancel,
    onSetSubmitting,
    onSubmit,
}: GridSubmitButtonsProps) => {
    const { palette } = useTheme();

    // Errors popup
    const [errorAnchorEl, setErrorAnchorEl] = useState<any | null>(null);
    const isErrorOpen = Boolean(errorAnchorEl);
    const openError = useCallback((ev: React.MouseEvent | React.TouchEvent) => {
        ev.preventDefault();
        setErrorAnchorEl(ev.currentTarget ?? ev.target)
    }, []);
    const closeError = useCallback(() => {
        setErrorAnchorEl(null);
        if (typeof onSetSubmitting === 'function') {
            onSetSubmitting(false);
        }
    }, [onSetSubmitting]);

    // Errors as a markdown list
    const errorMessage = useMemo<string>(() => {
        return errors ? Object.entries(errors).map(([key, value]) => `- ${capitalizeFirstLetter(key)}: ${value}`).join('\n') : '';
    }, [errors]);

    const handleSubmit = useCallback((ev: React.MouseEvent | React.TouchEvent) => {
        // If formik invalid, display errors in popup
        if (errors && Object.keys(errors).length > 0) openError(ev);
        else if (!disabledSubmit && typeof onSubmit === 'function') onSubmit();
    }, [errors, openError, disabledSubmit, onSubmit]);

    const isSubmitDisabled = useMemo(() => (errors && Object.keys(errors).length > 0) || (disabledSubmit === true), [errors, disabledSubmit]);

    return (
        <>
            {/* Errors popup */}
            <Popover
                open={isErrorOpen}
                anchorEl={errorAnchorEl}
                onClose={closeError}
                anchorOrigin={{
                    vertical: 'top',
                    horizontal: 'center',
                }}
                transformOrigin={{
                    vertical: 'bottom',
                    horizontal: 'center',
                }}
                sx={{
                    '& .MuiPopover-paper': {
                        padding: 1,
                        overflow: 'unset',
                        background: palette.background.paper,
                        color: palette.background.textPrimary,
                    },
                    // Remove horizontal spacing for list items
                    '& ul': {
                        paddingInlineStart: '20px',
                        margin: '8px',
                    }
                }}
            >
                <Markdown>{errorMessage}</Markdown>
                {/* Triangle placed below popper */}
                <Box sx={{
                    width: '0',
                    height: '0',
                    borderLeft: '10px solid transparent',
                    borderRight: '10px solid transparent',
                    borderTop: `10px solid ${palette.background.paper}`,
                    position: 'absolute',
                    bottom: '-10px',
                    left: '50%',
                    transform: 'translateX(-50%)',
                }} />
            </Popover>
            {/* Create/Save button. On hover or press, displays formik errors if disabled */}
            <Grid item xs={6}>
                <Box onClick={handleSubmit}>
                    <Button
                        disabled={isSubmitDisabled}
                        fullWidth
                        startIcon={isCreate ? <CreateIcon /> : <SaveIcon />}
                    >{isCreate ? "Create" : "Save"}</Button>
                </Box>
            </Grid>
            {/* Cancel button */}
            <Grid item xs={6}>
                <Button
                    disabled={disabledCancel !== undefined ? disabledCancel : false}
                    fullWidth
                    onClick={onCancel}
                    startIcon={<CancelIcon />}
                >Cancel</Button>
            </Grid>
        </>
    )
}