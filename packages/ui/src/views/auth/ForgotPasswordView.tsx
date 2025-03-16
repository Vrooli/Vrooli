import { EmailRequestPasswordChangeInput, emailRequestPasswordChangeSchema, endpointsAuth, LINKS, Success } from "@local/shared";
import { Box, Button, InputAdornment } from "@mui/material";
import { Field, Formik, FormikHelpers } from "formik";
import { useCallback } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route/router.js";
import { ForgotPasswordViewProps } from "views/types.js";
import { fetchLazyWrapper } from "../../api/fetchWrapper.js";
import { BreadcrumbsBase } from "../../components/breadcrumbs/BreadcrumbsBase/BreadcrumbsBase.js";
import { TextInput } from "../../components/inputs/TextInput/TextInput.js";
import { TopBar } from "../../components/navigation/TopBar/TopBar.js";
import { InnerForm } from "../../forms/BaseForm/BaseForm.js";
import { formPaper, formSubmit } from "../../forms/styles.js";
import { useLazyFetch } from "../../hooks/useLazyFetch.js";
import { EmailIcon } from "../../icons/common.js";
import { CenteredContentPage, CenteredContentPageWrap, CenteredContentPaper, FormContainer, FormSection } from "../../styles.js";

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
            <EmailIcon />
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
        <Box>
            <TopBar
                display="dialog"
                onClose={onClose}
                title={t("ForgotPassword")}
            />
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
                        <FormSection variant="transparent">
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
                        </FormSection>
                        <Box width="100%" display="flex" flexDirection="column" p={2}>
                            <Button
                                fullWidth
                                disabled={loading}
                                type="submit"
                                color="secondary"
                                variant="contained"
                                sx={formSubmit}
                            >
                                {t("Submit")}
                            </Button>
                            <BreadcrumbsBase
                                paths={breadcrumbPaths}
                                separator={"•"}
                                sx={breadcrumbsStyle}
                            />
                        </Box>
                    </FormContainer>
                </InnerForm>}
            </Formik>
        </Box>
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
                titleBehaviorDesktop="ShowBelow"
            />
            <CenteredContentPageWrap>
                <CenteredContentPaper maxWidth={600}>
                    <ForgotPasswordForm onClose={onClose} />
                </CenteredContentPaper>
            </CenteredContentPageWrap>
        </CenteredContentPage>
    );
}
