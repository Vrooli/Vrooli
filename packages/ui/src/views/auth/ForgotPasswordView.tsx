import { EmailRequestPasswordChangeInput, emailRequestPasswordChangeSchema, endpointsAuth, LINKS, Success } from "@local/shared";
import { Box, Button, InputAdornment } from "@mui/material";
import { Field, Formik, FormikHelpers } from "formik";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { fetchLazyWrapper } from "../../api/fetchWrapper.js";
import { BreadcrumbsBase } from "../../components/breadcrumbs/BreadcrumbsBase.js";
import { TextInput } from "../../components/inputs/TextInput/TextInput.js";
import { TopBar } from "../../components/navigation/TopBar.js";
import { InnerForm } from "../../forms/BaseForm/BaseForm.js";
import { formPaper, formSubmit } from "../../forms/styles.js";
import { useLazyFetch } from "../../hooks/useLazyFetch.js";
import { IconCommon } from "../../icons/Icons.js";
import { useLocation } from "../../route/router.js";
import { CenteredContentPage, FormContainer, pagePaddingBottom } from "../../styles.js";
import { ForgotPasswordViewProps } from "../types.js";
import { FormSection } from "./authStyles.js";

interface ForgotPasswordFormProps {
    onClose?: () => unknown;
}
type FormValues = typeof initialValues;

const initialValues = {
    email: "",
};

const baseFormStyle = {
    ...formPaper,
    paddingBottom: "unset",
    margin: "0px",
} as const;
const breadcrumbsStyle = {
    margin: "auto",
} as const;

const emailInputProps = {
    startAdornment: (
        <InputAdornment position="start">
            <IconCommon
                decorative
                name="Email"
            />
        </InputAdornment>
    ),
} as const;

function ForgotPasswordForm({
    onClose,
}: ForgotPasswordFormProps) {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const [emailRequestPasswordChange, { loading }] = useLazyFetch<EmailRequestPasswordChangeInput, Success>(endpointsAuth.emailRequestPasswordChange);

    const onSubmit = useCallback(function onSubmitCallback(values: FormValues, helpers: FormikHelpers<FormValues>) {
        fetchLazyWrapper<EmailRequestPasswordChangeInput, Success>({
            fetch: emailRequestPasswordChange,
            inputs: { ...values },
            successCondition: (data) => data.success === true,
            onSuccess: () => setLocation(LINKS.Home),
            onCompleted: () => { helpers.setSubmitting(false); },
            successMessage: () => ({ messageKey: "RequestSentCheckEmail" }),
        });
    }, [emailRequestPasswordChange, setLocation]);

    const breadcrumbPaths = [
        {
            text: t("LogIn"),
            link: LINKS.Login,
        },
        {
            text: t("SignUp"),
            link: LINKS.Signup,
        },
    ] as const;

    return (
        <Formik
            initialValues={initialValues}
            onSubmit={onSubmit}
            validationSchema={emailRequestPasswordChangeSchema}
        >
            {(formik) => <InnerForm
                display={"dialog"}
                isLoading={loading}
                style={baseFormStyle}
            >
                <FormContainer width="unset" maxWidth="unset">
                    <FormSection>
                        <Field
                            fullWidth
                            autoComplete="email"
                            name="email"
                            label={t("Email", { count: 1 })}
                            placeholder={t("EmailPlaceholder")}
                            as={TextInput}
                            InputProps={emailInputProps}
                            helperText={formik.touched.email && formik.errors.email}
                            error={formik.touched.email && Boolean(formik.errors.email)}
                        />
                        <Box display="flex" flexDirection="column">
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
                            <BreadcrumbsBase
                                paths={breadcrumbPaths}
                                separator={"•"}
                                sx={breadcrumbsStyle}
                            />
                        </Box>
                    </FormSection>
                </FormContainer>
            </InnerForm>}
        </Formik>
    );
}

export function ForgotPasswordView({
    display,
    onClose,
}: ForgotPasswordViewProps) {
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
                    <ForgotPasswordForm onClose={onClose} />
                </Box>
            </Box>
        </CenteredContentPage>
    );
}
