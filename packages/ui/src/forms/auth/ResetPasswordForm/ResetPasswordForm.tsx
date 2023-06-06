import { EmailResetPasswordInput, emailResetPasswordSchema, endpointPostAuthEmailResetPassword, LINKS, parseSearchParams, Session, useLocation, uuidValidate } from "@local/shared";
import { Button, Grid } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { PasswordTextField } from "components/inputs/PasswordTextField/PasswordTextField";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { ResetPasswordFormProps } from "forms/types";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
import { PubSub } from "utils/pubsub";
import { formPaper, formSubmit } from "../../styles";

export const ResetPasswordForm = ({
    onClose,
}: ResetPasswordFormProps) => {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const [emailResetPassword, { loading }] = useLazyFetch<EmailResetPasswordInput, Session>(endpointPostAuthEmailResetPassword);

    // Get userId and code from url. Should be set if coming from email link
    const { userId, code } = useMemo(() => {
        const params = parseSearchParams();
        if (typeof params.code !== "string" || !params.code.includes(":")) return { userId: undefined, code: undefined };
        const [userId, code] = params.code.split(":");
        if (!uuidValidate(userId)) return { userId: undefined, code: undefined };
        return { userId, code };
    }, []);

    return (
        <>
            <TopBar
                display="dialog"
                onClose={onClose}
                titleData={{
                    titleKey: "ResetPassword",
                }}
            />
            <Formik
                initialValues={{
                    newPassword: "",
                    confirmNewPassword: "",
                }}
                onSubmit={(values, helpers) => {
                    // Check for valid userId and code
                    if (!userId || !code) {
                        PubSub.get().publishSnack({ messageKey: "InvalidResetPasswordUrl", severity: "Error" });
                        return;
                    }
                    fetchLazyWrapper<EmailResetPasswordInput, Session>({
                        fetch: emailResetPassword,
                        inputs: { id: userId, code, newPassword: values.newPassword },
                        onSuccess: (data) => {
                            PubSub.get().publishSession(data);
                            setLocation(LINKS.Home);
                        },
                        successMessage: () => ({ messageKey: "PasswordReset" }),
                        onError: () => { helpers.setSubmitting(false); },
                    });
                }}
                validationSchema={emailResetPasswordSchema}
            >
                {(formik) => <BaseForm
                    dirty={formik.dirty}
                    isLoading={loading}
                    style={{
                        display: "block",
                        ...formPaper,
                    }}
                >
                    <Grid container spacing={2}>
                        <Grid item xs={12}>
                            <PasswordTextField
                                fullWidth
                                autoFocus
                                name="newPassword"
                                autoComplete="new-password"
                                label={t("PasswordNew")}
                            />
                        </Grid>
                        <Grid item xs={12}>
                            <PasswordTextField
                                fullWidth
                                name="confirmNewPassword"
                                autoComplete="new-password"
                                label={t("PasswordNewConfirm")}
                            />
                        </Grid>
                    </Grid>
                    <Button
                        fullWidth
                        disabled={loading}
                        type="submit"
                        color="secondary"
                        sx={{ ...formSubmit }}
                    >
                        {t("Submit")}
                    </Button>
                </BaseForm>}
            </Formik>
        </>
    );
};
