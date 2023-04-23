import { jsx as _jsx, jsxs as _jsxs } from "react/jsx-runtime";
import { DeleteType } from "@local/consts";
import { AddIcon } from "@local/icons";
import { emailValidation } from "@local/validation";
import { Stack, TextField, useTheme } from "@mui/material";
import { useFormik } from "formik";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { deleteOneOrManyDeleteOne } from "../../../../../api/generated/endpoints/deleteOneOrMany_deleteOne";
import { emailCreate } from "../../../../../api/generated/endpoints/email_create";
import { emailVerify } from "../../../../../api/generated/endpoints/email_verify";
import { useCustomMutation } from "../../../../../api/hooks";
import { mutationWrapper } from "../../../../../api/utils";
import { PubSub } from "../../../../../utils/pubsub";
import { ColorIconButton } from "../../../../buttons/ColorIconButton/ColorIconButton";
import { ListContainer } from "../../../../containers/ListContainer/ListContainer";
import { EmailListItem } from "../EmailListItem/EmailListItem";
export const EmailList = ({ handleUpdate, numVerifiedWallets, list, }) => {
    const { palette } = useTheme();
    const { t } = useTranslation();
    const [addMutation, { loading: loadingAdd }] = useCustomMutation(emailCreate);
    const formik = useFormik({
        initialValues: {
            emailAddress: "",
        },
        enableReinitialize: true,
        validationSchema: emailValidation.create({}),
        onSubmit: (values) => {
            if (!formik.isValid || loadingAdd)
                return;
            mutationWrapper({
                mutation: addMutation,
                input: {
                    emailAddress: values.emailAddress,
                },
                onSuccess: (data) => {
                    PubSub.get().publishSnack({ messageKey: "CompleteVerificationInEmail", severity: "Info" });
                    handleUpdate([...list, data]);
                    formik.resetForm();
                },
                onError: () => { formik.setSubmitting(false); },
            });
        },
    });
    const [deleteMutation, { loading: loadingDelete }] = useCustomMutation(deleteOneOrManyDeleteOne);
    const onDelete = useCallback((email) => {
        if (loadingDelete)
            return;
        if (list.length <= 1 && numVerifiedWallets === 0) {
            PubSub.get().publishSnack({ messageKey: "MustLeaveVerificationMethod", severity: "Error" });
            return;
        }
        PubSub.get().publishAlertDialog({
            messageKey: "EmailDeleteConfirm",
            messageVariables: { emailAddress: email.emailAddress },
            buttons: [
                {
                    labelKey: "Yes",
                    onClick: () => {
                        mutationWrapper({
                            mutation: deleteMutation,
                            input: { id: email.id, objectType: DeleteType.Email },
                            onSuccess: () => {
                                handleUpdate([...list.filter(w => w.id !== email.id)]);
                            },
                        });
                    },
                },
                { labelKey: "Cancel", onClick: () => { } },
            ],
        });
    }, [deleteMutation, handleUpdate, list, loadingDelete, numVerifiedWallets]);
    const [verifyMutation, { loading: loadingVerifyEmail }] = useCustomMutation(emailVerify);
    const sendVerificationEmail = useCallback((email) => {
        if (loadingVerifyEmail)
            return;
        mutationWrapper({
            mutation: verifyMutation,
            input: { emailAddress: email.emailAddress },
            onSuccess: () => {
                PubSub.get().publishSnack({ messageKey: "CompleteVerificationInEmail", severity: "Info" });
            },
        });
    }, [loadingVerifyEmail, verifyMutation]);
    return (_jsxs("form", { onSubmit: formik.handleSubmit, children: [_jsx(ListContainer, { emptyText: t("NoEmails", { ns: "error" }), isEmpty: list.length === 0, sx: { maxWidth: "500px" }, children: list.map((email, index) => (_jsx(EmailListItem, { data: email, index: index, handleDelete: onDelete, handleVerify: sendVerificationEmail }, `email-${index}`))) }), _jsxs(Stack, { direction: "row", sx: {
                    display: "flex",
                    justifyContent: "center",
                    paddingTop: 2,
                    paddingBottom: 6,
                }, children: [_jsx(TextField, { autoComplete: 'email', fullWidth: true, id: "emailAddress", name: "emailAddress", label: "New Email Address", value: formik.values.emailAddress, onBlur: formik.handleBlur, onChange: formik.handleChange, error: formik.touched.emailAddress && Boolean(formik.errors.emailAddress), helperText: formik.touched.emailAddress && formik.errors.emailAddress, sx: {
                            height: "56px",
                            maxWidth: "400px",
                            "& .MuiInputBase-root": {
                                borderRadius: "5px 0 0 5px",
                            },
                        } }), _jsx(ColorIconButton, { "aria-label": 'add-new-email-button', background: palette.secondary.main, type: 'submit', sx: {
                            borderRadius: "0 5px 5px 0",
                            height: "56px",
                        }, children: _jsx(AddIcon, {}) })] })] }));
};
//# sourceMappingURL=EmailList.js.map