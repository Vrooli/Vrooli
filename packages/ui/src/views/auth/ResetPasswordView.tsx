import { EmailResetPasswordInput, LINKS, Session, UrlTools, emailResetPasswordFormSchema, endpointsAuth, uuidValidate } from "@local/shared";
import { Box, Button } from "@mui/material";
import { Formik, FormikHelpers } from "formik";
import { useCallback, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route/router.js";
import { fetchLazyWrapper } from "../../api/fetchWrapper.js";
import { PasswordTextInput } from "../../components/inputs/PasswordTextInput/PasswordTextInput.js";
import { TopBar } from "../../components/navigation/TopBar/TopBar.js";
import { InnerForm } from "../../forms/BaseForm/BaseForm.js";
import { formPaper, formSubmit } from "../../forms/styles.js";
import { useLazyFetch } from "../../hooks/useLazyFetch.js";
import { CenteredContentPage, CenteredContentPageWrap, CenteredContentPaper, FormContainer, FormSection } from "../../styles.js";
import { PubSub } from "../../utils/pubsub.js";
import { ResetPasswordViewProps } from "../types.js";

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
        <>
            <TopBar
                display="dialog"
                onClose={onClose}
                title={t("ResetPassword")}
            />
            <Formik
                initialValues={initialValues}
                onSubmit={onSubmit}
                validationSchema={emailResetPasswordFormSchema}
            >
                {(formik) => <InnerForm
                    display={"dialog"}
                    isLoading={loading}
                    style={baseFormStyle}
                >
                    <FormContainer width="unset" maxWidth="unset">
                        <FormSection variant="transparent">
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
                        </FormSection>
                        <Box width="100%" display="flex" flexDirection="column" p={2}>
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
                        </Box>
                    </FormContainer>
                </InnerForm>}
            </Formik>
        </>
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
                titleBehaviorDesktop="ShowBelow"
            />
            <CenteredContentPageWrap>
                <CenteredContentPaper maxWidth={600}>
                    <ResetPasswordForm onClose={onClose} />
                </CenteredContentPaper>
            </CenteredContentPageWrap>
        </CenteredContentPage>
    );
}
