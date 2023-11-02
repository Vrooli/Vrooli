import { EmailResetPasswordInput, emailResetPasswordSchema, endpointPostAuthEmailResetPassword, LINKS, Session, uuidValidate } from "@local/shared";
import { Box, Button, Grid, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { PasswordTextField } from "components/inputs/PasswordTextField/PasswordTextField";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Formik } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { formPaper, formSubmit } from "forms/styles";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { parseSearchParams, useLocation } from "route";
import { toDisplay } from "utils/display/pageTools";
import { PubSub } from "utils/pubsub";
import { ResetPasswordViewProps } from "views/types";

interface ResetPasswordFormProps {
    onClose?: () => unknown;
}

const ResetPasswordForm = ({
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
                title={t("ResetPassword")}
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
                        onCompleted: () => { helpers.setSubmitting(false); },
                    });
                }}
                validationSchema={emailResetPasswordSchema}
            >
                {(formik) => <BaseForm
                    display={"dialog"}
                    isLoading={loading}
                    style={{
                        ...formPaper,
                        paddingBottom: "unset",
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
                        variant="contained"
                        sx={{ ...formSubmit }}
                    >
                        {t("Submit")}
                    </Button>
                </BaseForm>}
            </Formik>
        </>
    );
};

export const ResetPasswordView = ({
    isOpen,
    onClose,
}: ResetPasswordViewProps) => {
    const display = toDisplay(isOpen);
    const { palette } = useTheme();

    return (
        <Box sx={{ maxHeight: "100vh", overflow: "hidden" }}>
            <TopBar
                display={display}
                onClose={onClose}
                hideTitleOnDesktop
            />
            <Box
                sx={{
                    position: "absolute",
                    top: "50%",
                    left: "50%",
                    transform: "translateX(-50%) translateY(-50%)",
                    width: "min(700px, 100%)",
                    background: palette.background.paper,
                    borderRadius: { xs: 0, sm: 2 },
                    overflow: "overlay",
                }}
            >
                <ResetPasswordForm onClose={onClose} />
            </Box>
        </Box>
    );
};
