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
        // Filter out null and undefined errors
        const filteredErrors = Object.entries(errors ?? {}).filter(([key, value]) => value !== null && value !== undefined) as [string, string | string[]][];
        // Helper to convert string to markdown list item
        const toListItem = (str: string, level: number) => { return `${'  '.repeat(level)}* ${str}`; };
        // Convert errors to markdown list
        const errorList = filteredErrors.map(([key, value]) => {
            if (Array.isArray(value)) {
                return toListItem(capitalizeFirstLetter(key), 0) + ': \n' + value.map((str) => toListItem(str, 1)).join('\n');
            }
            else {
                return toListItem(capitalizeFirstLetter(key + ': ' + value), 0);
            }
        }).join('\n');
        console.log('error list', errorList)
        return errorList;
    }, [errors]);

    const hasErrors = useMemo(() => Object.values(errors ?? {}).some((value) => value !== null && value !== undefined), [errors]);
    const isSubmitDisabled = useMemo(() => hasErrors || (disabledSubmit === true), [hasErrors, disabledSubmit]);


    const handleSubmit = useCallback((ev: React.MouseEvent | React.TouchEvent) => {
        console.log('handle submit', hasErrors, isSubmitDisabled)
        // If formik invalid, display errors in popup
        if (hasErrors) openError(ev);
        else if (!disabledSubmit && typeof onSubmit === 'function') onSubmit();
    }, [hasErrors, isSubmitDisabled, openError, disabledSubmit, onSubmit]);

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