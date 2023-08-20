import { EmailRequestPasswordChangeInput, emailRequestPasswordChangeSchema, endpointPostAuthEmailRequestPasswordChange, LINKS, Success } from "@local/shared";
import { Button, Grid, Link, TextField, Typography } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Field, Formik } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useTranslation } from "react-i18next";
import { useLocation } from "route";
import { clickSize } from "styles";
import { Forms } from "utils/consts";
import { formNavLink, formPaper, formSubmit } from "../../styles";
import { ForgotPasswordFormProps } from "../../types";

export const ForgotPasswordForm = ({
    onClose,
    onFormChange = () => { },
}: ForgotPasswordFormProps) => {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const [emailRequestPasswordChange, { loading }] = useLazyFetch<EmailRequestPasswordChangeInput, Success>(endpointPostAuthEmailRequestPasswordChange);

    const toSignUp = () => onFormChange(Forms.SignUp);
    const toLogIn = () => onFormChange(Forms.LogIn);

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
                    dirty={formik.dirty}
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
                                name="email"
                                label={t("Email", { count: 1 })}
                                as={TextField}
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
                    <Grid container spacing={2}>
                        <Grid item xs={6}>
                            <Link onClick={toLogIn}>
                                <Typography
                                    sx={{
                                        ...clickSize,
                                        ...formNavLink,
                                    }}
                                >
                                    {t("RememberLogBackIn")}
                                </Typography>
                            </Link>
                        </Grid>
                        <Grid item xs={6}>
                            <Link onClick={toSignUp}>
                                <Typography
                                    sx={{
                                        ...clickSize,
                                        ...formNavLink,
                                        flexDirection: "row-reverse",
                                    }}
                                >
                                    {t("DontHaveAccountSignUp")}
                                </Typography>
                            </Link>
                        </Grid>
                    </Grid>
                </BaseForm>}
            </Formik>
        </>
    );
};
