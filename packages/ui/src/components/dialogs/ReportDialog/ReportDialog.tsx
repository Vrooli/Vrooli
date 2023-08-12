import { endpointPostReport, endpointPutReport, Report, ReportCreateInput, ReportUpdateInput } from "@local/shared";
import { Link, Typography } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { ReportForm, reportInitialValues, validateReportValues } from "forms/ReportForm/ReportForm";
import { formNavLink } from "forms/styles";
import { useCallback, useContext, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { clickSize } from "styles";
import { toDisplay } from "utils/display/pageTools";
import { useUpsertActions } from "utils/hooks/useUpsertActions";
import { SessionContext } from "utils/SessionContext";
import { shapeReport } from "utils/shape/models/report";
import { MaybeLargeDialog } from "../LargeDialog/LargeDialog";
import { ReportDialogProps } from "../types";

export const ReportDialog = ({
    forId,
    isOpen,
    onClose,
    reportFor,
    title,
    zIndex,
}: ReportDialogProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const display = toDisplay(isOpen);

    const initialValues = useMemo(() => reportInitialValues(session, reportFor, forId), [forId, reportFor, session]);
    const formRef = useRef<BaseFormRef>();
    const {
        fetch,
        handleCancel,
        handleCompleted,
        isCreateLoading,
        isUpdateLoading,
    } = useUpsertActions<Report, ReportCreateInput, ReportUpdateInput>({
        display,
        endpointCreate: endpointPostReport,
        endpointUpdate: endpointPutReport,
        isCreate: true,
        onCancel: onClose,
        onCompleted: onClose,
    });

    /**
     * Opens existing reports in a new tab
     */
    const toExistingReports = useCallback(() => {
        window.open("/reports", "_blank");// TODO change url
    }, []);

    return (
        <MaybeLargeDialog
            display={display}
            id="report-dialog"
            isOpen={isOpen}
            onClose={handleCancel}
            zIndex={zIndex}
        >
            <TopBar
                display={display}
                onClose={handleCancel}
                title={title ?? t("Report")}
                help={t("ReportsHelp")}
                zIndex={zIndex}
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
                    isLoading={isCreateLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    ref={formRef}
                    zIndex={zIndex}
                    {...formik}
                />}
            </Formik>
        </MaybeLargeDialog>
    );
};
