import { endpointPostReport, Report, ReportCreateInput } from "@local/shared";
import { Link, Typography } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { ReportForm, reportInitialValues, validateReportValues } from "forms/ReportForm/ReportForm";
import { formNavLink } from "forms/styles";
import { useCallback, useContext, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { clickSize } from "styles";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { SessionContext } from "utils/SessionContext";
import { shapeReport } from "utils/shape/models/report";
import { DialogTitle } from "../DialogTitle/DialogTitle";
import { LargeDialog } from "../LargeDialog/LargeDialog";
import { ReportDialogProps } from "../types";

const titleId = "report-dialog-title";

export const ReportDialog = ({
    forId,
    onClose,
    open,
    reportFor,
    title,
    zIndex,
}: ReportDialogProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();

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
                title={title ?? t("Report", { count: 1 })}
                help={t("ReportsHelp")}
                onClose={handleCancel}
                zIndex={zIndex + 1000}
            />
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
                    console.log("form submit", values);
                    fetchLazyWrapper<ReportCreateInput, Report>({
                        fetch,
                        inputs: shapeReport.create(values),
                        successCondition: (data) => data !== null,
                        successMessage: () => ({ messageKey: "ReportSubmitted" }),
                        onSuccess: () => {
                            helpers.resetForm();
                            onClose();
                        },
                        onError: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateReportValues(values)}
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
        </LargeDialog>
    );
};
