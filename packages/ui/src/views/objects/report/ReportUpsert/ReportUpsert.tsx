import { endpointGetReport, endpointPostReport, endpointPutReport, Report, ReportCreateInput, ReportUpdateInput } from "@local/shared";
import { Link, Typography } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { MaybeLargeDialog } from "components/dialogs/LargeDialog/LargeDialog";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { NewReportShape, ReportForm, reportInitialValues, transformReportValues, validateReportValues } from "forms/ReportForm/ReportForm";
import { formNavLink } from "forms/styles";
import { useFormDialog } from "hooks/useFormDialog";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { useUpsertActions } from "hooks/useUpsertActions";
import { useCallback, useContext } from "react";
import { useTranslation } from "react-i18next";
import { clickSize } from "styles";
import { toDisplay } from "utils/display/pageTools";
import { PubSub } from "utils/pubsub";
import { ReportShape } from "utils/shape/models/report";
import { ReportUpsertProps } from "../types";

export const ReportUpsert = ({
    isCreate,
    isOpen,
    onCancel,
    onCompleted,
    overrideObject,
}: ReportUpsertProps) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const display = toDisplay(isOpen);

    const { isLoading: isReadLoading, object: existing } = useObjectFromUrl<Report, ReportShape>({
        ...endpointGetReport,
        objectType: "Report",
        overrideObject,
        transform: (existing) => reportInitialValues(session, existing as NewReportShape),
    });

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
        isCreate,
        onCancel,
        onCompleted,
    });
    const { formRef, handleClose } = useFormDialog({ handleCancel });

    /**
     * Opens existing reports in a new tab
     */
    const toExistingReports = useCallback(() => {
        window.open("/reports", "_blank");// TODO change url
    }, []);

    return (
        <MaybeLargeDialog
            display={display}
            id="report-upsert-dialog"
            isOpen={isOpen ?? false}
            onClose={handleClose}
        >
            <TopBar
                display={display}
                onClose={handleClose}
                title={t("Report", { count: 1 })}
                help={t("ReportsHelp")}
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
                initialValues={existing}
                onSubmit={(values, helpers) => {
                    if (!isCreate && !existing) {
                        PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
                        return;
                    }
                    fetchLazyWrapper<ReportCreateInput | ReportUpdateInput, Report>({
                        fetch,
                        inputs: transformReportValues(values, existing, isCreate),
                        onSuccess: (data) => { handleCompleted(data); },
                        onCompleted: () => { helpers.setSubmitting(false); },
                    });
                }}
                validate={async (values) => await validateReportValues(values, existing, isCreate)}
            >
                {(formik) => <ReportForm
                    display={display}
                    isCreate={isCreate}
                    isLoading={isCreateLoading || isReadLoading || isUpdateLoading}
                    isOpen={true}
                    onCancel={handleCancel}
                    onClose={handleClose}
                    ref={formRef}
                    {...formik}
                />}
            </Formik>
        </MaybeLargeDialog>
    );
};
