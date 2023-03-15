import { useCustomMutation } from 'api/hooks';
import { reportCreateForm as validationSchema } from '@shared/validation';
import { DialogContent, Link, Stack, TextField, Typography } from '@mui/material';
import { useFormik } from 'formik';
import { ReportDialogProps } from '../types';
import { getUserLanguages, usePromptBeforeUnload } from 'utils';
import { useCallback, useEffect, useState } from 'react';
import { SelectLanguageMenu } from '../SelectLanguageMenu/SelectLanguageMenu';
import { DialogTitle, GridSubmitButtons, LargeDialog, Selector } from 'components';
import { uuid } from '@shared/uuid';
import { Report, ReportCreateInput } from '@shared/consts';
import { mutationWrapper } from 'api/utils';
import { reportCreate } from 'api/generated/endpoints/report_create';
import { clickSize } from 'styles';
import { formNavLink } from 'forms/styles';
import { useTranslation } from 'react-i18next';
import { CSSProperties } from '@mui/styles';

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

const titleId = "report-dialog-title";

export const ReportDialog = ({
    forId,
    onClose,
    open,
    reportFor,
    session,
    title = 'Report',
    zIndex,
}: ReportDialogProps) => {
    const { t } = useTranslation();

    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => { setLanguage(getUserLanguages(session)[0]) }, [session]);

    const [mutation, { loading }] = useCustomMutation<Report, ReportCreateInput>(reportCreate);
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

    /**
     * Opens existing reports in a new tab
     */
    const toExistingReports = useCallback(() => {
        window.open('/reports', '_blank');// TODO change url
    }, []);

    return (
        <LargeDialog
            id="report-dialog"
            isOpen={open}
            onClose={handleCancel}
            titleId={titleId}
            zIndex={zIndex}
        >
            <DialogTitle
                id={titleId}
                title={title}
                helpText={t('ReportsHelp')}
                onClose={handleCancel}
            />
            <DialogContent>
                <Link onClick={toExistingReports}>
                    <Typography sx={{
                        ...clickSize,
                        ...formNavLink,
                        justifyContent: 'center',
                        marginTop: 2,
                    } as CSSProperties}>
                        {t('ViewExistingReports')}
                    </Typography>
                </Link>
                <form onSubmit={formik.handleSubmit} style={{ paddingBottom: '64px' }}>
                    <Stack direction="column" spacing={2} paddingTop={2}>
                        <SelectLanguageMenu
                            currentLanguage={language}
                            handleCurrent={setLanguage}
                            session={session}
                            translations={[{ language }]}
                            zIndex={zIndex}
                        />
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
                            label={t('Reason')}
                        />
                        {(formik.values.reason as any) === ReportOptions.Other ? <TextField
                            fullWidth
                            id="otherReason"
                            name="otherReason"
                            label={t('ReasonCustom')}
                            value={formik.values.otherReason}
                            onBlur={formik.handleBlur}
                            onChange={formik.handleChange}
                            error={formik.touched.otherReason && Boolean(formik.errors.otherReason)}
                            helperText={formik.touched.otherReason ? formik.errors.otherReason : 'Enter custom reason...'}

                        /> : null}
                        <TextField
                            fullWidth
                            id="details"
                            name="details"
                            label={t('DetailsOptional')}
                            multiline
                            rows={4}
                            value={formik.values.details}
                            onBlur={formik.handleBlur}
                            onChange={formik.handleChange}
                            error={formik.touched.details && Boolean(formik.errors.details)}
                            helperText={formik.touched.details ? formik.errors.details : "Enter any details you'd like to share about this incident. DO NOT include any personal information!"}
                        />
                    </Stack>
                </form>
            </DialogContent>
            <GridSubmitButtons
                display="dialog"
                errors={formik.errors}
                isCreate={true}
                loading={formik.isSubmitting}
                onCancel={handleCancel}
                onSetSubmitting={formik.setSubmitting}
                onSubmit={formik.handleSubmit}
            />
        </LargeDialog>
    )
}