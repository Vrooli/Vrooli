import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { uuid } from "@local/uuid";
import { reportCreateForm } from "@local/validation";
import { DialogContent, Link, Typography } from "@mui/material";
import { Formik } from "formik";
import { useCallback, useContext, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { reportCreate } from "../../../api/generated/endpoints/report_create";
import { useCustomMutation } from "../../../api/hooks";
import { mutationWrapper } from "../../../api/utils";
import { ReportForm, reportInitialValues } from "../../../forms/ReportForm/ReportForm";
import { formNavLink } from "../../../forms/styles";
import { clickSize } from "../../../styles";
import { getUserLanguages } from "../../../utils/display/translationTools";
import { useUpsertActions } from "../../../utils/hooks/useUpsertActions";
import { SessionContext } from "../../../utils/SessionContext";
import { DialogTitle } from "../DialogTitle/DialogTitle";
import { LargeDialog } from "../LargeDialog/LargeDialog";
const titleId = "report-dialog-title";
export const ReportDialog = ({ forId, onClose, open, reportFor, title = "Report", zIndex, }) => {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const [language, setLanguage] = useState(getUserLanguages(session)[0]);
    useEffect(() => { setLanguage(getUserLanguages(session)[0]); }, [session]);
    const formRef = useRef();
    const initialValues = useMemo(() => reportInitialValues(session, reportFor, forId), [forId, reportFor, session]);
    const { handleCancel } = useUpsertActions("dialog", true, onClose, onClose);
    const [mutation, { loading: isLoading }] = useCustomMutation(reportCreate);
    const toExistingReports = useCallback(() => {
        window.open("/reports", "_blank");
    }, []);
    return (_jsxs(LargeDialog, { id: "report-dialog", isOpen: open, onClose: handleCancel, titleId: titleId, zIndex: zIndex, children: [_jsx(DialogTitle, { id: titleId, title: title, helpText: t("ReportsHelp"), onClose: handleCancel }), _jsxs(DialogContent, { children: [_jsx(Link, { onClick: toExistingReports, children: _jsx(Typography, { sx: {
                                ...clickSize,
                                ...formNavLink,
                                justifyContent: "center",
                                marginTop: 2,
                            }, children: t("ViewExistingReports") }) }), _jsx(Formik, { enableReinitialize: true, initialValues: initialValues, onSubmit: (values, helpers) => {
                            mutationWrapper({
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
                                successMessage: () => ({ key: "ReportSubmitted" }),
                                onSuccess: () => {
                                    helpers.resetForm();
                                    onClose();
                                },
                                onError: () => { helpers.setSubmitting(false); },
                            });
                        }, validationSchema: reportCreateForm, children: (formik) => _jsx(ReportForm, { display: "dialog", isCreate: true, isLoading: isLoading, isOpen: true, onCancel: handleCancel, ref: formRef, zIndex: zIndex, ...formik }) })] })] }));
};
//# sourceMappingURL=ReportDialog.js.map