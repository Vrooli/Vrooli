import { EmailRequestPasswordChangeInput, emailRequestPasswordChangeSchema, endpointPostAuthEmailRequestPasswordChange, LINKS, Success } from "@local/shared";
import { Box, Button, Grid, InputAdornment, Link, Typography, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { TextInput } from "components/inputs/TextInput/TextInput";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Field, Formik } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { formNavLink, formPaper, formSubmit } from "forms/styles";
import { useLazyFetch } from "hooks/useLazyFetch";
import { EmailIcon } from "icons";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { clickSize } from "styles";
import { ForgotPasswordViewProps } from "views/types";

interface ForgotPasswordFormProps {
    onClose?: () => unknown;
}

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
    const [emailRequestPasswordChange, { loading }] = useLazyFetch<EmailRequestPasswordChangeInput, Success>(endpointPostAuthEmailRequestPasswordChange);

    return (
        <>
            <TopBar
                display="dialog"
                onClose={onClose}
                title={t("ForgotPassword")}
            />
            <Formik
                initialValues={{
                    email: "",
                }}
                onSubmit={(values, helpers) => {
                    fetchLazyWrapper<EmailRequestPasswordChangeInput, Success>({
                        fetch: emailRequestPasswordChange,
                        inputs: { ...values },
                        successCondition: (data) => data.success === true,
                        onSuccess: () => setLocation(LINKS.Home),
                        onCompleted: () => { helpers.setSubmitting(false); },
                        successMessage: () => ({ messageKey: "RequestSentCheckEmail" }),
                    });
                }}
                validationSchema={emailRequestPasswordChangeSchema}
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
                            <Field
                                fullWidth
                                autoComplete="email"
                                autoFocus
                                name="email"
                                label={t("Email", { count: 1 })}
                                placeholder={t("EmailPlaceholder")}
                                as={TextInput}
                                InputProps={emailInputProps}
                                helperText={formik.touched.email && formik.errors.email}
                                error={formik.touched.email && Boolean(formik.errors.email)}
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
                    <Box sx={{
                        display: "flex",
                        flexDirection: "row",
                        justifyContent: "space-between",
                        alignItems: "center",
                    }}>
                        <Link href={LINKS.Login}>
                            <Typography
                                sx={{
                                    ...clickSize,
                                    ...formNavLink,
                                }}
                            >
                                {t("LogIn")}
                            </Typography>
                        </Link>
                        <Link href={LINKS.Signup}>
                            <Typography
                                sx={{
                                    ...clickSize,
                                    ...formNavLink,
                                    flexDirection: "row-reverse",
                                }}
                            >
                                {t("SignUp")}
                            </Typography>
                        </Link>
                    </Box>
                </BaseForm>}
            </Formik>
        </>
    );
}

export function ForgotPasswordView({
    display,
    onClose,
}: ForgotPasswordViewProps) {
    const { palette } = useTheme();

    return (
        <Box sx={{ maxHeight: "100vh", overflow: "hidden" }}>
            <TopBar
                display={display}
                onClose={onClose}
                titleBehaviorDesktop="ShowBelow"
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
                <ForgotPasswordForm onClose={onClose} />
            </Box>
        </Box>
    );
}
