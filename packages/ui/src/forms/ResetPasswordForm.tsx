import {
    Button,
    Grid,
    Paper
} from '@mui/material';
import { EmailResetPasswordInput, LINKS, Session } from '@shared/consts';
import { parseSearchParams, useLocation } from '@shared/route';
import { uuidValidate } from '@shared/uuid';
import { emailResetPasswordSchema } from '@shared/validation';
import { authEmailResetPassword } from 'api/generated/endpoints/auth_emailResetPassword';
import { useCustomMutation } from 'api/hooks';
import { mutationWrapper } from 'api/utils';
import { PasswordTextField } from 'components/inputs/PasswordTextField/PasswordTextField';
import { useFormik } from 'formik';
import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { PubSub } from 'utils/pubsub';
import { formPaper, formSubmit } from './styles';

export const ResetPasswordForm = () => {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const [emailResetPassword, { loading }] = useCustomMutation<Session, EmailResetPasswordInput>(authEmailResetPassword);

    // Get userId and code from url. Should be set if coming from email link
    const { userId, code } = useMemo(() => {
        const params = parseSearchParams();
        if (typeof params.code !== 'string' || !params.code.includes(':')) return { userId: undefined, code: undefined };
        const [userId, code] = params.code.split(':');
        if (!uuidValidate(userId)) return { userId: undefined, code: undefined };
        return { userId, code };
    }, []);

    const formik = useFormik({
        initialValues: {
            newPassword: '',
            confirmNewPassword: '',
        },
        validationSchema: emailResetPasswordSchema,
        onSubmit: (values) => {
            // Check for valid userId and code
            if (!userId || !code) {
                PubSub.get().publishSnack({ messageKey: 'InvalidResetPasswordUrl', severity: 'Error' });
                return;
            }
            mutationWrapper<Session, EmailResetPasswordInput>({
                mutation: emailResetPassword,
                input: { id: userId, code, newPassword: values.newPassword },
                onSuccess: (data) => {
                    PubSub.get().publishSession(data);
                    setLocation(LINKS.Home)
                },
                successMessage: () => ({ key: 'PasswordReset' }),
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });

    return (
        <Paper sx={{ ...formPaper }}>
            <form onSubmit={formik.handleSubmit}>
                <Grid container spacing={2}>
                    <Grid item xs={12}>
                        <PasswordTextField
                            fullWidth
                            autoFocus
                            id="newPassword"
                            name="newPassword"
                            autoComplete="new-password"
                            label="New Password"
                            value={formik.values.newPassword}
                            onBlur={formik.handleBlur}
                            onChange={formik.handleChange}
                            error={formik.touched.newPassword && Boolean(formik.errors.newPassword)}
                            helperText={formik.touched.newPassword ? formik.errors.newPassword : null}
                        />
                    </Grid>
                    <Grid item xs={12}>
                        <PasswordTextField
                            fullWidth
                            id="confirmNewPassword"
                            name="confirmNewPassword"
                            autoComplete="new-password"
                            label="Confirm New Password"
                            value={formik.values.confirmNewPassword}
                            onBlur={formik.handleBlur}
                            onChange={formik.handleChange}
                            error={formik.touched.confirmNewPassword && Boolean(formik.errors.confirmNewPassword)}
                            helperText={formik.touched.confirmNewPassword ? formik.errors.confirmNewPassword : null}
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
                    {t('Submit')}
                </Button>
            </form>
        </Paper>
    );
}