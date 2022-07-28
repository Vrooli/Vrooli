/**
 * Prompts user to select which link the new node should be added on
 */
import { EndNodeDialogProps } from '../types';
import { Box, Dialog, IconButton, Typography, useTheme } from '@mui/material';
import {
    Close as CloseIcon,
} from '@mui/icons-material';
import { noSelect } from 'styles';
import { getTranslation } from 'utils';
import { useEffect, useState } from 'react';
import { useFormik } from 'formik';

export const EndNodeDialog = ({
    handleClose,
    isOpen,
    data,
    language,
    session,
    zIndex,
}: EndNodeDialogProps) => {
    const { palette } = useTheme();

    const [changedData, setChangedData] = useState(data);
    useEffect(() => {
        setChangedData(data);
    }, [data]);

    const formik = useFormik({
        initialValues: {
            title: getTranslation(changedData, 'title', [language], false) ?? '',
            wasSuccessful: changedData.data.wasSuccessful,
        },
        // validationSchema,
        onSubmit: (values) => {
            // TODO
        },
    });

    return (
        <Dialog
            open={isOpen}
            onClose={handleClose}
            sx={{
                zIndex,
            }}
        >
            <Box
                sx={{
                    ...noSelect,
                    display: 'flex',
                    alignItems: 'center',
                    padding: 1,
                    background: palette.primary.dark
                }}
            >
                <Typography
                    variant="h6"
                    textAlign="center"
                    sx={{ width: '-webkit-fill-available', color: palette.primary.contrastText }}
                >
                    {formik.values.title}
                </Typography>
                <IconButton
                    edge="end"
                    onClick={handleClose}
                >
                    <CloseIcon sx={{ fill: palette.primary.contrastText }} />
                </IconButton>
            </Box>
            <form onSubmit={formik.handleSubmit}>
                {/* TODO */}
            </form>
        </Dialog>
    )
}