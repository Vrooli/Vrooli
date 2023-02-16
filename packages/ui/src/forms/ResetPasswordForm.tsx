import { useMutation } from 'api/hooks';
import { emailResetPasswordSchema } from '@shared/validation';
import { useFormik } from 'formik';
import {
    Button,
    Grid,
    Paper,
} from '@mui/material';
import { APP_LINKS, EmailResetPasswordInput, Session } from '@shared/consts';
import { mutationWrapper } from 'api/utils';
import { useLocation } from '@shared/route';
import { ResetPasswordFormProps } from './types';
import { formPaper, formSubmit } from './styles';
import { PasswordTextField } from 'components';
import { PubSub } from 'utils';
import { authEmailResetPassword } from 'api/generated/endpoints/auth';

export const ResetPasswordForm = ({
    userId,
    code,
}: ResetPasswordFormProps) => {
    const [, setLocation] = useLocation();
    const [emailResetPassword, { loading }] = useMutation<Session, EmailResetPasswordInput, 'emailResetPassword'>(authEmailResetPassword, 'emailResetPassword');

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
                    setLocation(APP_LINKS.Home)
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
                    Submit
                </Button>
            </form>
        </Paper>
    );
}