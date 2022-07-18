import { emailResetPasswordMutation } from 'graphql/mutation';
import { useMutation } from '@apollo/client';
import { emailResetPasswordSchema } from '@local/shared';
import { useFormik } from 'formik';
import {
    Button,
    Grid,
    Paper,
} from '@mui/material';
import { APP_LINKS } from '@local/shared';
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { useLocation } from 'wouter';
import { emailResetPassword, emailResetPasswordVariables } from 'graphql/generated/emailResetPassword';
import { ResetPasswordFormProps } from './types';
import { formPaper, formSubmit } from './styles';
import { PasswordTextField } from 'components';
import { PubSub } from 'utils';

export const ResetPasswordForm = ({
    userId,
    code,
}: ResetPasswordFormProps) => {
    const [, setLocation] = useLocation();
    const [emailResetPassword, { loading }] = useMutation<emailResetPassword, emailResetPasswordVariables>(emailResetPasswordMutation);

    const formik = useFormik({
        initialValues: {
            newPassword: '',
            confirmNewPassword: '',
        },
        validationSchema: emailResetPasswordSchema,
        onSubmit: (values) => {
            // Check for valid userId and code
            if (!userId || !code) {
                PubSub.get().publishSnack({ message: 'Invalid reset password URL.', severity: 'error' });
                return;
            }
            mutationWrapper({
                mutation: emailResetPassword,
                input: { id: userId, code, newPassword: values.newPassword },
                onSuccess: (response) => { 
                    PubSub.get().publishSession(response.data.emailResetPassword); 
                    setLocation(APP_LINKS.Home) 
                },
                successMessage: () => 'Password reset.',
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