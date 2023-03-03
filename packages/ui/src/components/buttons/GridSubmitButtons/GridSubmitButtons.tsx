/**
 * Prompts user to select which link the new node should be added on
 */
import { Box, Button, CircularProgress, Grid, useTheme } from '@mui/material';
import { CancelIcon, CreateIcon, SaveIcon } from '@shared/icons';
import { exists } from '@shared/utils';
import { PopoverWithArrow } from 'components/dialogs';
import Markdown from 'markdown-to-jsx';
import { useCallback, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { GridActionButtons } from '../GridActionButtons/GridActionButtons';
import { GridSubmitButtonsProps } from '../types';

const capitalizeFirstLetter = (str: string) => { return str.charAt(0).toUpperCase() + str.slice(1); };

export const GridSubmitButtons = ({
    disabledCancel,
    disabledSubmit,
    display,
    errors,
    isCreate,
    loading = false,
    onCancel,
    onSetSubmitting,
    onSubmit,
}: GridSubmitButtonsProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    // Errors popup
    const [errorAnchorEl, setErrorAnchorEl] = useState<any | null>(null);
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
        const filteredErrors = Object.entries(errors ?? {}).filter(([key, value]) => exists(value)) as [string, string | string[]][];
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
        return errorList;
    }, [errors]);

    const hasErrors = useMemo(() => Object.values(errors ?? {}).some((value) => exists(value)), [errors]);
    const isSubmitDisabled = useMemo(() => loading || hasErrors || (disabledSubmit === true), [disabledSubmit, hasErrors, loading]);


    const handleSubmit = useCallback((ev: React.MouseEvent | React.TouchEvent) => {
        // If formik invalid, display errors in popup
        if (hasErrors) openError(ev);
        else if (!disabledSubmit && typeof onSubmit === 'function') onSubmit();
    }, [hasErrors, openError, disabledSubmit, onSubmit]);

    return (
        <GridActionButtons display={display}>
            {/* Errors popup */}
            <PopoverWithArrow
                anchorEl={errorAnchorEl}
                handleClose={closeError}
                sxs={{
                    root: {
                        // Remove horizontal spacing for list items
                        '& ul': {
                            paddingInlineStart: '20px',
                            margin: '8px',
                        }
                    }
                }}
            >
                <Markdown>{errorMessage}</Markdown>
            </PopoverWithArrow>
            {/* Create/Save button. On hover or press, displays formik errors if disabled */}
            <Grid item xs={6}>
                <Box onClick={handleSubmit}>
                    <Button
                        disabled={isSubmitDisabled}
                        fullWidth
                        startIcon={loading ? <CircularProgress size={24} sx={{ color: 'white' }} /> : (isCreate ? <CreateIcon /> : <SaveIcon />)}
                    >{t(isCreate ? 'Create' : 'Save')}</Button>
                </Box>
            </Grid>
            {/* Cancel button */}
            <Grid item xs={6}>
                <Button
                    disabled={loading || (disabledCancel !== undefined ? disabledCancel : false)}
                    fullWidth
                    onClick={onCancel}
                    startIcon={<CancelIcon />}
                >{t('Cancel')}</Button>
            </Grid>
        </GridActionButtons>
    )
}