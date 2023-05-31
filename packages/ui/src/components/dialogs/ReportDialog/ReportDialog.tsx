import { Report, reportCreate, reportCreateForm, ReportCreateInput, uuid } from "@local/shared";
import { DialogContent, Link, Typography } from "@mui/material";
import { CSSProperties } from "@mui/styles";
import { useCustomMutation } from "api";
import { mutationWrapper } from "api/utils";
import { Formik } from "formik";
import { BaseFormRef } from "forms/BaseForm/BaseForm";
import { ReportForm, reportInitialValues } from "forms/ReportForm/ReportForm";
import { formNavLink } from "forms/styles";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { clickSize } from "styles";
import { getUserLanguages } from "utils/display/translationTools";
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
    const [mutation, { loading: isLoading }] = useCustomMutation<Report, ReportCreateInput>(reportCreate);

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
                helpText={t("ReportsHelp")}
                onClose={handleCancel}
            />
            <DialogContent>
                <Link onClick={toExistingReports}>
                    <Typography sx={{
                        ...clickSize,
                        ...formNavLink,
                        justifyContent: "center",
                        marginTop: 2,
                    } as CSSProperties}>
                        {t("ViewExistingReports")}
                    </Typography>
                </Link>
                <Formik
                    enableReinitialize={true}
                    initialValues={initialValues}
                    onSubmit={(values, helpers) => {
                        mutationWrapper<Report, ReportCreateInput>({
                            mutation,
                            input: {
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
                        zIndex={zIndex}
                        {...formik}
                    />}
                </Formik>
            </DialogContent>
        </LargeDialog>
    );
};
