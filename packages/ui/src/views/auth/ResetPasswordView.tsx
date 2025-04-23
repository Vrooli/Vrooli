import { EmailResetPasswordInput, LINKS, Session, UrlTools, emailResetPasswordFormSchema, endpointsAuth, uuidValidate } from "@local/shared";
import { Box, Button } from "@mui/material";
import { Formik, FormikHelpers } from "formik";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../api/fetchWrapper.js";
import { PasswordTextInput } from "../../components/inputs/PasswordTextInput/PasswordTextInput.js";
import { TopBar } from "../../components/navigation/TopBar.js";
import { InnerForm } from "../../forms/BaseForm/BaseForm.js";
import { formPaper, formSubmit } from "../../forms/styles.js";
import { useLazyFetch } from "../../hooks/useLazyFetch.js";
import { useLocation } from "../../route/router.js";
import { CenteredContentPage, FormContainer, pagePaddingBottom } from "../../styles.js";
import { PubSub } from "../../utils/pubsub.js";
import { ResetPasswordViewProps } from "../types.js";
import { FormSection } from "./authStyles.js";

interface ResetPasswordFormProps {
    onClose?: () => unknown;
}
type FormValues = typeof initialValues;

const initialValues = {
    newPassword: "",
    confirmNewPassword: "",
};

const baseFormStyle = {
    ...formPaper,
    paddingBottom: "unset",
    margin: "0px",
} as const;


function ResetPasswordForm({
    onClose,
}: ResetPasswordFormProps) {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const [emailResetPassword, { loading }] = useLazyFetch<EmailResetPasswordInput, Session>(endpointsAuth.emailResetPassword);

    // Get userId and code from url. Should be set if coming from email link
    const { userId, code } = useMemo(() => {
        const params = UrlTools.parseSearchParams(LINKS.ResetPassword);
        if (typeof params.code !== "string" || !params.code.includes(":")) return { userId: undefined, code: undefined };
        const [userId, code] = params.code.split(":");
        if (!uuidValidate(userId)) return { userId: undefined, code: undefined };
        return { userId, code };
    }, []);

    const onSubmit = useCallback(function onSubmitCallback(values: FormValues, helpers: FormikHelpers<FormValues>) {
        // Check for valid userId and code
        if (!userId || !code) {
            PubSub.get().publish("snack", { messageKey: "InvalidResetPasswordUrl", severity: "Error" });
            return;
        }
        fetchLazyWrapper<EmailResetPasswordInput, Session>({
            fetch: emailResetPassword,
            inputs: { id: userId, code, newPassword: values.newPassword },
            onSuccess: (data) => {
                PubSub.get().publish("session", data);
                setLocation(LINKS.Home);
            },
            successMessage: () => ({ messageKey: "PasswordReset" }),
            onCompleted: () => { helpers.setSubmitting(false); },
        });
    }, [emailResetPassword, setLocation, userId, code]);

    return (
        <Formik
            initialValues={initialValues}
            onSubmit={onSubmit}
            validationSchema={emailResetPasswordFormSchema}
        >
            {() => <InnerForm
                display={"Dialog"}
                isLoading={loading}
                style={baseFormStyle}
            >
                <FormContainer width="unset" maxWidth="unset">
                    <FormSection>
                        <PasswordTextInput
                            fullWidth
                            autoFocus
                            name="newPassword"
                            autoComplete="new-password"
                            label={t("PasswordNew")}
                        />
                        <PasswordTextInput
                            fullWidth
                            name="confirmNewPassword"
                            autoComplete="new-password"
                            label={t("PasswordNewConfirm")}
                        />
                        <Button
                            fullWidth
                            disabled={loading}
                            type="submit"
                            color="secondary"
                            variant="contained"
                            sx={formSubmit}
                        >
                            {t("ResetPassword")}
                        </Button>
                    </FormSection>
                </FormContainer>
            </InnerForm>}
        </Formik>
    );
}

export function ResetPasswordView({
    display,
    onClose,
}: ResetPasswordViewProps) {
    return (
        <CenteredContentPage>
            <TopBar
                display={display}
                onClose={onClose}
            />
            <Box
                display="flex"
                flexDirection="column"
                gap={1}
                margin="auto"
                overflow="auto"
                padding={2}
                paddingBottom={pagePaddingBottom}
            >
                <Box
                    bgcolor="background.paper"
                    borderRadius={2}
                    maxWidth={600}
                    width="100%"
                >
                    <ResetPasswordForm onClose={onClose} />
                </Box>
            </Box>
        </CenteredContentPage>
    );
}
