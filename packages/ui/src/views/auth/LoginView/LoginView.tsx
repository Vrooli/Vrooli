import { emailLogInFormValidation, EmailLogInInput, endpointPostAuthEmailLogin, LINKS, Session } from "@local/shared";
import { Box, Button, InputAdornment } from "@mui/material";
import { errorToMessage, fetchLazyWrapper, hasErrorCode } from "api";
import { BreadcrumbsBase } from "components/breadcrumbs/BreadcrumbsBase/BreadcrumbsBase";
import { PasswordTextInput } from "components/inputs/PasswordTextInput/PasswordTextInput";
import { TextInput } from "components/inputs/TextInput/TextInput";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { SessionContext } from "contexts";
import { Field, Formik, FormikHelpers } from "formik";
import { InnerForm } from "forms/BaseForm/BaseForm";
import { formPaper, formSubmit } from "forms/styles";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useReactSearch } from "hooks/useReactSearch";
import { EmailIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { CenteredContentPage, CenteredContentPageWrap, CenteredContentPaper, FormContainer, FormSection } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { removeCookie } from "utils/localStorage";
import { PubSub } from "utils/pubsub";
import { LoginViewProps } from "views/types";

interface LoginFormProps {
    onClose?: () => unknown;
}
type FormValues = typeof initialValues;

const initialValues = {
    email: "",
    password: "",
};

const baseFormStyle = {
    ...formPaper,
    paddingBottom: "unset",
    margin: "0px",
} as const;
const breadcrumbsStyle = {
    margin: "auto",
} as const;

const emailStartAdornment = {
    startAdornment: (
        <InputAdornment position="start">
            <EmailIcon />
        </InputAdornment>
    ),
};

function LoginForm({
    onClose,
}: LoginFormProps) {
    const session = useContext(SessionContext);
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    const search = useReactSearch();
    const { redirect, verificationCode } = useMemo(() => ({
        redirect: typeof search.redirect === "string" ? search.redirect : undefined,
        verificationCode: typeof search.verificationCode === "string" ? search.verificationCode : undefined,
    }), [search]);

    const [emailLogIn, { loading }] = useLazyFetch<EmailLogInInput, Session>(endpointPostAuthEmailLogin);

    /**
     * If verification code supplied
     */
    useEffect(() => {
        if (verificationCode) {
            // If still logged in, call emailLogIn right away
            if (userId) {
                fetchLazyWrapper<EmailLogInInput, Session>({
                    fetch: emailLogIn,
                    inputs: { verificationCode },
                    onSuccess: (data) => {
                        PubSub.get().publish("snack", { messageKey: "EmailVerified", severity: "Success" });
                        PubSub.get().publish("session", data);
                        localStorage.setItem("isLoggedIn", "true");
                        setLocation(redirect ?? LINKS.Home);
                    },
                    onError: (response) => {
                        if (hasErrorCode(response, "MustResetPassword")) {
                            PubSub.get().publish("alertDialog", {
                                messageKey: "ChangePasswordBeforeLogin",
                                buttons: [
                                    { labelKey: "Ok", onClick: () => { setLocation(redirect ?? LINKS.Home); } },
                                ],
                            });
                        }
                    },
                });
            }
        }
    }, [emailLogIn, verificationCode, redirect, setLocation, userId]);

    const onSubmit = useCallback(function onSubmitCallback(values: FormValues, helpers: FormikHelpers<FormValues>) {
        fetchLazyWrapper<EmailLogInInput, Session>({
            fetch: emailLogIn,
            inputs: { ...values, verificationCode },
            successCondition: (data) => data !== null,
            onSuccess: (data) => {
                removeCookie("FormData"); // Clear old form data cache
                if (verificationCode) PubSub.get().publish("snack", { messageKey: "EmailVerified", severity: "Success" });
                PubSub.get().publish("session", data);
                setLocation(redirect ?? LINKS.Home);
            },
            showDefaultErrorSnack: false,
            onError: (response) => {
                // Custom dialog for changing password
                if (hasErrorCode(response, "MustResetPassword")) {
                    PubSub.get().publish("alertDialog", {
                        messageKey: "ChangePasswordBeforeLogin",
                        buttons: [
                            { labelKey: "Ok", onClick: () => { setLocation(redirect ?? LINKS.Home); } },
                        ],
                    });
                }
                // Custom snack for invalid email, that has sign up link
                else if (hasErrorCode(response, "EmailNotFound")) {
                    PubSub.get().publish("snack", {
                        messageKey: "EmailNotFound",
                        severity: "Error",
                        buttonKey: "SignUp",
                        buttonClicked: () => { setLocation(LINKS.Signup); },
                    });
                } else {
                    PubSub.get().publish("snack", { message: errorToMessage(response, ["en"]), severity: "Error", data: response });
                }
                helpers.setSubmitting(false);
            },
        });
    }, [emailLogIn, redirect, setLocation, verificationCode]);

    const breadcrumbPaths = [
        {
            text: t("ForgotPassword"),
            link: LINKS.ForgotPassword,
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
                title={t("LogIn")}
            />
            <Formik
                initialValues={initialValues}
                onSubmit={onSubmit}
                validationSchema={emailLogInFormValidation}
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
                                autoFocus
                                name="email"
                                label={t("Email", { count: 1 })}
                                placeholder={t("EmailPlaceholder")}
                                as={TextInput}
                                InputProps={emailStartAdornment}
                                helperText={formik.touched.email && formik.errors.email}
                                error={formik.touched.email && Boolean(formik.errors.email)}
                            />
                            <PasswordTextInput
                                fullWidth
                                name="password"
                                autoComplete="current-password"
                            />
                        </FormSection>
                    </FormContainer>
                    <Box width="100%" display="flex" flexDirection="column" p={2}>
                        <Button
                            fullWidth
                            disabled={loading}
                            type="submit"
                            color="secondary"
                            variant='contained'
                            sx={formSubmit}
                        >
                            {t("LogIn")}
                        </Button>
                        <BreadcrumbsBase
                            paths={breadcrumbPaths}
                            separator={"•"}
                            sx={breadcrumbsStyle}
                        />
                    </Box>
                </InnerForm>}
            </Formik>
        </Box>
    );
}

export function LoginView({
    display,
    onClose,
}: LoginViewProps) {
    return (
        <CenteredContentPage>
            <TopBar
                display={display}
                onClose={onClose}
                titleBehaviorDesktop="ShowBelow"
            />
            <CenteredContentPageWrap>
                <CenteredContentPaper maxWidth={600}>
                    <LoginForm onClose={onClose} />
                </CenteredContentPaper>
            </CenteredContentPageWrap>
        </CenteredContentPage>
    );
}
