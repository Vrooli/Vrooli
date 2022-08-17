import { useMutation } from '@apollo/client';
import { reportCreateForm as validationSchema } from '@shared/validation';
import { Box, Button, Dialog, Grid, IconButton, Stack, TextField, Typography, useTheme } from '@mui/material';
import { HelpButton } from 'components/buttons';
import { useFormik } from 'formik';
import { reportCreate, reportCreateVariables } from 'graphql/generated/reportCreate';
import { reportCreateMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { ReportDialogProps } from '../types';
import {
    Cancel as CancelIcon,
    Check as SaveIcon,
    Close as CloseIcon
} from '@mui/icons-material';
import { getUserLanguages, PubSub } from 'utils';
import { useEffect, useState } from 'react';
import { SelectLanguageDialog } from '../SelectLanguageDialog/SelectLanguageDialog';
import { Selector } from 'components';
import { v4 as uuid } from 'uuid';

const helpText =
    `Reports help us moderate content. For now, reports will be handled by moderators. 

In the future, we would like to implement a community governance system.
`

enum ReportOptions {
    Inappropriate = 'Inappropriate',
    PII = 'PII',
    Scam = 'Scam',
    Spam = 'Spam',
    Other = 'Other',
}

const ReportReasons = {
    [ReportOptions.Inappropriate]: 'Inappropriate Content',
    [ReportOptions.PII]: 'Includes Personally Identifiable Information (PII)',
    [ReportOptions.Scam]: 'Scam',
    [ReportOptions.Spam]: 'Spam',
    [ReportOptions.Other]: 'Other',
}

export const ReportDialog = ({
    forId,
    onClose,
    open,
    reportFor,
    session,
    title = 'Report',
    zIndex,
}: ReportDialogProps) => {
    const { palette } = useTheme();

    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => { setLanguage(getUserLanguages(session)[0]) }, [session]);

    const [mutation, { loading }] = useMutation<reportCreate, reportCreateVariables>(reportCreateMutation);
    const formik = useFormik({
        initialValues: {
            createdFor: reportFor,
            createdForId: forId,
            reason: '',
            otherReason: '',
            details: '',
            language,
        },
        enableReinitialize: true,
        validationSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation,
                input: {
                    id: uuid(),
                    language,
                    reason: Boolean(values.otherReason) ? values.otherReason : values.reason,
                    createdFor: reportFor,
                    createdForId: forId
                },
                successCondition: (response) => response.data.reportCreate !== null,
                onSuccess: () => {
                    PubSub.get().publishSnack({ message: 'Report submitted.' });
                    formik.resetForm();
                    onClose()
                },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });

    /**
     * On page leave, check if unsaved work. 
     * If so, prompt for confirmation.
     */
    useEffect(() => {
        const handleBeforeUnload = (e: BeforeUnloadEvent) => {
            if (formik.dirty) {
                e.preventDefault()
                e.returnValue = ''
            }
        };
        window.addEventListener('beforeunload', handleBeforeUnload);
        return () => window.removeEventListener('beforeunload', handleBeforeUnload);
    }, [formik.dirty]);

    const handleClose = () => {
        formik.resetForm();
        onClose();
    }

    return (
        <Dialog
            onClose={handleClose}
            open={open}
            sx={{
                zIndex,
                '& .MuiPaper-root': {
                    minWidth: 'min(400px, 100%)',
                    margin: '0 auto',
                },
                '& .MuiDialog-paper': {
                    textAlign: 'center',
                    overflow: 'hidden',
                }
            }}
        >
            <form onSubmit={formik.handleSubmit}>
                <Box sx={{
                    padding: 1,
                    background: palette.primary.dark,
                    color: palette.primary.contrastText,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                }}>
                    <Stack direction="row" spacing={1} sx={{ marginLeft: 'auto' }}>
                        <Typography component="h2" variant="h4" textAlign="center" sx={{ marginLeft: 'auto' }}>
                            {title}
                        </Typography>
                        <SelectLanguageDialog
                            currentLanguage={language}
                            handleCurrent={setLanguage}
                            session={session}
                            zIndex={zIndex}
                        />
                    </Stack>
                    <Box sx={{ marginLeft: 'auto' }}>
                        <HelpButton markdown={helpText} sx={{ fill: '#a0e7c4' }} />
                        <IconButton
                            edge="start"
                            onClick={handleClose}
                        >
                            <CloseIcon sx={{ fill: palette.primary.contrastText }} />
                        </IconButton>
                    </Box>
                </Box>
                <Stack direction="column" spacing={2} sx={{ padding: 2 }}>
                    {/* Text displaying what you are reporting */}
                    <Selector
                        disabled={loading}
                        options={Object.keys(ReportReasons)}
                        getOptionLabel={(r) => ReportReasons[r]}
                        selected={formik.values.reason}
                        onBlur={formik.handleBlur}
                        handleChange={(e) => formik.setFieldValue('reason', e.target.value)}
                        fullWidth
                        inputAriaLabel="select reason"
                        label="Reason"
                    />
                    {/* Textfield displayed if "Other" reason selected */}
                    {(formik.values.reason as any) === ReportOptions.Other ? <TextField
                        fullWidth
                        id="otherReason"
                        name="otherReason"
                        label="Custom Reason"
                        value={formik.values.otherReason}
                        onBlur={formik.handleBlur}
                        onChange={formik.handleChange}
                        error={formik.touched.otherReason && Boolean(formik.errors.otherReason)}
                        helperText={formik.touched.otherReason ? formik.errors.otherReason : 'Enter custom reason...'}

                    /> : null}
                    {/* Reason selector (with other option) */}
                    <TextField
                        fullWidth
                        id="details"
                        name="details"
                        label="Details (Optional)"
                        multiline
                        rows={4}
                        value={formik.values.details}
                        onBlur={formik.handleBlur}
                        onChange={formik.handleChange}
                        error={formik.touched.details && Boolean(formik.errors.details)}
                        helperText={formik.touched.details ? formik.errors.details : "Enter any details you'd like to share about this incident. DO NOT include any personal information!"}
                    />
                    {/* Details multi-line text field */}
                    {/* Action buttons */}
                    <Grid container sx={{ padding: 0 }}>
                        <Grid item xs={6} sx={{ padding: 1 }}>
                            <Button
                                type="submit"
                                fullWidth
                                disabled={loading}
                                startIcon={<SaveIcon />}
                            >
                                Save
                            </Button>
                        </Grid>
                        <Grid item xs={6} sx={{ padding: 1 }}>
                            <Button
                                type="button"
                                fullWidth
                                disabled={loading}
                                onClick={handleClose}
                                startIcon={<CancelIcon />}
                            >
                                Cancel
                            </Button>
                        </Grid>
                    </Grid>
                </Stack>
            </form>
        </Dialog>
    )
}