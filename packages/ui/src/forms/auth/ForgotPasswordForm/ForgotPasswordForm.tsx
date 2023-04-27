import { EmailRequestPasswordChangeInput, emailRequestPasswordChangeSchema, LINKS, Success, useLocation } from "@local/shared";
import {
    Button,
    Grid,
    Link, TextField,
    Typography
} from "@mui/material";
import { CSSProperties } from "@mui/styles";
import { authEmailRequestPasswordChange } from "api/generated/endpoints/auth_emailRequestPasswordChange";
import { useCustomMutation } from "api/hooks";
import { mutationWrapper } from "api/utils";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { Field, Formik } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useTranslation } from "react-i18next";
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
    const [emailRequestPasswordChange, { loading }] = useCustomMutation<Success, EmailRequestPasswordChangeInput>(authEmailRequestPasswordChange);

    const toSignUp = () => onFormChange(Forms.SignUp);
    const toLogIn = () => onFormChange(Forms.LogIn);

    return (
        <>
            <TopBar
                display="dialog"
                onClose={onClose}
                titleData={{
                    titleKey: "ForgotPassword",
                }}
            />
            <Formik
                initialValues={{
                    email: "",
                }}
                onSubmit={(values, helpers) => {
                    mutationWrapper<Success, EmailRequestPasswordChangeInput>({
                        mutation: emailRequestPasswordChange,
                        input: { ...values },
                        successCondition: (data) => data.success === true,
                        onSuccess: () => setLocation(LINKS.Home),
                        onError: () => { helpers.setSubmitting(false); },
                        successMessage: () => ({ messageKey: "RequestSentCheckEmail" }),
                    });
                }}
                validationSchema={emailRequestPasswordChangeSchema}
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
                                    } as CSSProperties}
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
                                    } as CSSProperties}
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
