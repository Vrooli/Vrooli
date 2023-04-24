import { EmailResetPasswordInput, LINKS, Session } from ":local/consts";
import { uuidValidate } from ":local/uuid";
import { emailResetPasswordSchema } from ":local/validation";
import {
    Button,
    Grid
} from "@mui/material";
import { Formik } from "formik";
import { useMemo } from "react";
import { useTranslation } from "react-i18next";
import { authEmailResetPassword } from "../../../api/generated/endpoints/auth_emailResetPassword";
import { useCustomMutation } from "../../../api/hooks";
import { mutationWrapper } from "../../../api/utils";
import { PasswordTextField } from "../../../components/inputs/PasswordTextField/PasswordTextField";
import { TopBar } from "../../../components/navigation/TopBar/TopBar";
import { PubSub } from "../../../utils/pubsub";
import { parseSearchParams, useLocation } from "../../../utils/route";
import { BaseForm } from "../../BaseForm/BaseForm";
import { formPaper, formSubmit } from "../../styles";
import { ResetPasswordFormProps } from "../../types";

export const ResetPasswordForm = ({
    onClose,
}: ResetPasswordFormProps) => {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const [emailResetPassword, { loading }] = useCustomMutation<Session, EmailResetPasswordInput>(authEmailResetPassword);

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
                    mutationWrapper<Session, EmailResetPasswordInput>({
                        mutation: emailResetPassword,
                        input: { id: userId, code, newPassword: values.newPassword },
                        onSuccess: (data) => {
                            PubSub.get().publishSession(data);
                            setLocation(LINKS.Home);
                        },
                        successMessage: () => ({ key: "PasswordReset" }),
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
