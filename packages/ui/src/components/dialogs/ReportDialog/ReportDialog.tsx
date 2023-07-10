import { endpointPostReport, Report, reportCreateForm, ReportCreateInput, uuid } from "@local/shared";
import { DialogContent, Link, Typography } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { ReportForm, reportInitialValues } from "forms/ReportForm/ReportForm";
import { formNavLink } from "forms/styles";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { clickSize } from "styles";
import { getUserLanguages } from "utils/display/translationTools";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { SessionContext } from "utils/SessionContext";
import { DialogTitle } from "../DialogTitle/DialogTitle";
import { LargeDialog } from "../LargeDialog/LargeDialog";
import { ReportDialogProps } from "../types";

const titleId = "report-dialog-title";

export const ReportDialog = ({
    forId,
    onClose,
    open,
    reportFor,
    title = "Report",
    zIndex,
}: ReportDialogProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();

    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => { setLanguage(getUserLanguages(session)[0]); }, [session]);
    const formRef = useRef<BaseFormRef>();
    const initialValues = useMemo(() => reportInitialValues(session, reportFor, forId), [forId, reportFor, session]);
    const { handleCancel } = useUpsertActions<Report>("dialog", true, onClose, onClose);
    const [fetch, { loading: isLoading }] = useLazyFetch<ReportCreateInput, Report>(endpointPostReport);

    /**
     * Opens existing reports in a new tab
     */
    const toExistingReports = useCallback(() => {
        window.open("/reports", "_blank");// TODO change url
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
                help={t("ReportsHelp")}
                onClose={handleCancel}
                zIndex={zIndex + 1000}
            />
            <DialogContent>
                <Link onClick={toExistingReports}>
                    <Typography sx={{
                        ...clickSize,
                        ...formNavLink,
                        justifyContent: "center",
                        marginTop: 2,
                    }}>
                        {t("ViewExistingReports")}
                    </Typography>
                </Link>
                <Formik
                    enableReinitialize={true}
                    initialValues={initialValues}
                    onSubmit={(values, helpers) => {
                        fetchLazyWrapper<ReportCreateInput, Report>({
                            fetch,
                            inputs: {
                                id: uuid(),
                                createdFor: reportFor,
                                createdForConnect: forId,
                                reason: values.otherReason ?? values.reason,
                                details: "",
                                language,
                            },
                            successCondition: (data) => data !== null,
                            successMessage: () => ({ messageKey: "ReportSubmitted" }),
                            onSuccess: () => {
                                helpers.resetForm();
                                onClose();
                            },
                            onError: () => { helpers.setSubmitting(false); },
                        });
                    }}
                    validationSchema={reportCreateForm}
                >
                    {(formik) => <ReportForm
                        display="dialog"
                        isCreate={true}
                        isLoading={isLoading}
                        isOpen={true}
                        onCancel={handleCancel}
                        ref={formRef}
                        zIndex={zIndex + 1000}
                        {...formik}
                    />}
                </Formik>
            </DialogContent>
        </LargeDialog>
    );
};
