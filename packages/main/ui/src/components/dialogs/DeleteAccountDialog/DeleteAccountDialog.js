import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { LINKS } from "@local/consts";
import { DeleteIcon } from "@local/icons";
import { userDeleteOneSchema as validationSchema } from "@local/validation";
import { Button, Checkbox, DialogContent, FormControlLabel, Stack, Tooltip, Typography, useTheme } from "@mui/material";
import { Formik } from "formik";
import { useContext, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useCustomMutation } from "../../../api";
import { userDeleteOne } from "../../../api/generated/endpoints/user_deleteOne";
import { mutationWrapper } from "../../../api/utils";
import { getCurrentUser } from "../../../utils/authentication/session";
import { PubSub } from "../../../utils/pubsub";
import { useLocation } from "../../../utils/route";
import { SessionContext } from "../../../utils/SessionContext";
import { PasswordTextField } from "../../inputs/PasswordTextField/PasswordTextField";
import { DialogTitle } from "../DialogTitle/DialogTitle";
import { LargeDialog } from "../LargeDialog/LargeDialog";
const titleId = "delete-object-dialog-title";
export const DeleteAccountDialog = ({ handleClose, isOpen, zIndex, }) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const { id, name } = useMemo(() => getCurrentUser(session), [session]);
    const [deleteAccount] = useCustomMutation(userDeleteOne);
    return (_jsxs(LargeDialog, { id: "delete-account-dialog", isOpen: isOpen, onClose: () => { handleClose(false); }, titleId: titleId, zIndex: zIndex, children: [_jsx(DialogTitle, { id: titleId, title: `Delete "${name}"`, onClose: () => { handleClose(false); } }), _jsx(Formik, { enableReinitialize: true, initialValues: {
                    password: "",
                    deletePublicData: false,
                }, onSubmit: (values, helpers) => {
                    if (!id) {
                        PubSub.get().publishSnack({ messageKey: "NoUserIdFound", severity: "Error" });
                        return;
                    }
                    mutationWrapper({
                        mutation: deleteAccount,
                        input: values,
                        successCondition: (data) => data.success,
                        successMessage: () => ({ key: "AccountDeleteSuccess" }),
                        onSuccess: () => {
                            setLocation(LINKS.Home);
                            handleClose(true);
                        },
                        errorMessage: () => ({ key: "AccountDeleteFail" }),
                        onError: () => {
                            handleClose(false);
                        },
                    });
                }, validationSchema: validationSchema, children: (formik) => _jsx(DialogContent, { children: _jsxs(Stack, { direction: "column", spacing: 2, mt: 2, children: [_jsxs(Typography, { variant: "h6", children: ["Are you absolutely certain you want to delete the account of \"", name, "\"?"] }), _jsx(Typography, { variant: "h6", sx: { color: palette.error.main, paddingBottom: 3 }, children: _jsx("b", { children: "This action cannot be undone." }) }), _jsx(Typography, { variant: "h6", children: "Enter your password to confirm." }), _jsx(PasswordTextField, { fullWidth: true, name: "password", autoComplete: "current-password" }), _jsx(Tooltip, { placement: "top", title: "If checked, all public data owned by your account will be deleted. Please consider transferring and/or exporting anything important if you decide to choose this.", children: _jsx(FormControlLabel, { label: 'Delete public data', control: _jsx(Checkbox, { id: 'delete-public-data', size: "small", name: 'deletePublicData', color: 'secondary', checked: formik.values.deletePublicData, onChange: formik.handleChange }) }) }), _jsx(Button, { disabled: formik.isSubmitting || !formik.isValid, startIcon: _jsx(DeleteIcon, {}), color: "secondary", onClick: () => { formik.submitForm(); }, children: t("Delete") })] }) }) })] }));
};
//# sourceMappingURL=DeleteAccountDialog.js.map