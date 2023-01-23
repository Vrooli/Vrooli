import { useMutation } from 'api/hooks';
import { reportCreateForm as validationSchema } from '@shared/validation';
import { Dialog, DialogContent, Grid, Stack, TextField } from '@mui/material';
import { useFormik } from 'formik';
import { ReportDialogProps } from '../types';
import { getUserLanguages, usePromptBeforeUnload } from 'utils';
import { useCallback, useEffect, useState } from 'react';
import { SelectLanguageMenu } from '../SelectLanguageMenu/SelectLanguageMenu';
import { DialogTitle, GridSubmitButtons, Selector } from 'components';
import { uuid } from '@shared/uuid';
import { Report, ReportCreateInput } from '@shared/consts';
import { mutationWrapper } from 'api/utils';
import { endpoints } from 'api';

const helpText =
    `Reports help us moderate content. For now, reports will be handled by moderators.\n\nIn the future, we would like to implement a community governance system.`

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

const titleAria = "report-dialog-title";

export const ReportDialog = ({
    forId,
    onClose,
    open,
    reportFor,
    session,
    title = 'Report',
    zIndex,
}: ReportDialogProps) => {
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => { setLanguage(getUserLanguages(session)[0]) }, [session]);

    const [mutation, { loading }] = useMutation<Report, ReportCreateInput, 'reportCreate'>(...endpoints.report().create);
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
            mutationWrapper<Report, ReportCreateInput>({
                mutation,
                input: {
                    createdFor: reportFor,
                    createdForConnect: forId,
                    details: values.details,
                    id: uuid(),
                    language,
                    reason: Boolean(values.otherReason) ? values.otherReason : values.reason,
                },
                successCondition: (data) => data !== null,
                successMessage: () => ({ key: 'ReportSubmitted' }),
                onSuccess: () => {
                    formik.resetForm();
                    onClose()
                },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    const handleCancel = useCallback((_?: unknown, reason?: 'backdropClick' | 'escapeKeyDown') => {
        // Don't close if formik is dirty and clicked outside
        if (formik.dirty && reason === 'backdropClick') return;
        // Otherwise, close
        formik.resetForm();
        onClose();
    }, [formik, onClose]);

    return (
        <Dialog
            onClose={handleCancel}
            open={open}
            aria-labelledby={titleAria}
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
            <DialogTitle
                ariaLabel={titleAria}
                title={title}
                helpText={helpText}
                onClose={handleCancel}
            />
            <DialogContent>
                <form onSubmit={formik.handleSubmit}>
                    <Stack direction="column" spacing={2} paddingTop={2}>
                        {/* Language select */}
                        <SelectLanguageMenu
                            currentLanguage={language}
                            handleCurrent={setLanguage}
                            session={session}
                            translations={[{ language }]}
                            zIndex={zIndex}
                        />
                        {/* Text displaying what you are reporting */}
                        <Selector
                            id="reasonSelector"
                            name="reasonSelector"
                            disabled={loading}
                            options={Object.keys(ReportReasons)}
                            getOptionLabel={(r) => ReportReasons[r]}
                            selected={formik.values.reason}
                            onBlur={formik.handleBlur}
                            handleChange={(c) => formik.setFieldValue('reason', c)}
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
                        <Grid container spacing={1}>
                            <GridSubmitButtons
                                errors={formik.errors}
                                isCreate={true}
                                loading={formik.isSubmitting}
                                onCancel={handleCancel}
                                onSetSubmitting={formik.setSubmitting}
                                onSubmit={formik.handleSubmit}
                            />
                        </Grid>
                    </Stack>
                </form>
            </DialogContent>
        </Dialog>
    )
}