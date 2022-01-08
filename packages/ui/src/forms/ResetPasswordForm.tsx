import { emailResetPasswordMutation } from 'graphql/mutation';
import { useMutation } from '@apollo/client';
import { emailResetPasswordSchema } from '@local/shared';
import { useFormik } from 'formik';
import {
    Button,
    Grid,
    TextField
} from '@mui/material';
import { makeStyles } from '@mui/styles';
import { APP_LINKS } from '@local/shared';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { formStyles } from './styles';
import { useLocation } from 'wouter';
import { emailResetPassword } from 'graphql/generated/emailResetPassword';
import { ResetPasswordFormProps } from './types';

const useStyles = makeStyles(formStyles);

export const ResetPasswordForm = ({
    userId,
    code,
    onSessionUpdate
}: ResetPasswordFormProps) => {
    const classes = useStyles();
    const [, setLocation] = useLocation();
    const [emailResetPassword, {loading}] = useMutation<emailResetPassword>(emailResetPasswordMutation);

    const formik = useFormik({
        initialValues: {
            newPassword: '',
            confirmNewPassword: '',
        },
        validationSchema: emailResetPasswordSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation: emailResetPassword,
                input: { id: userId, code, newPassword: values.newPassword },
                onSuccess: (response) => { onSessionUpdate(response.data.emailResetPassword); setLocation(APP_LINKS.Home) },
                successMessage: () => 'Password reset.',
            })
        },
    });

    return (
        <form className={classes.form} onSubmit={formik.handleSubmit}>
            <Grid container spacing={2}>
                <Grid item xs={12}>
                    <TextField
                        fullWidth
                        autoFocus
                        id="newPassword"
                        name="newPassword"
                        type="password"
                        autoComplete="password"
                        label="New Password"
                        value={formik.values.newPassword}
                        onChange={formik.handleChange}
                        error={formik.touched.newPassword && Boolean(formik.errors.newPassword)}
                        helperText={formik.touched.newPassword && formik.errors.newPassword}
                    />
                </Grid>
                <Grid item xs={12}>
                    <TextField
                        fullWidth
                        autoFocus
                        id="confirmNewPassword"
                        name="confirmNewPassword"
                        type="password"
                        autoComplete="new-password"
                        label="Confirm New Password"
                        value={formik.values.confirmNewPassword}
                        onChange={formik.handleChange}
                        error={formik.touched.confirmNewPassword && Boolean(formik.errors.confirmNewPassword)}
                        helperText={formik.touched.confirmNewPassword && formik.errors.confirmNewPassword}
                    />
                </Grid>
            </Grid>
            <Button
                fullWidth
                disabled={loading}
                type="submit"
                color="secondary"
                className={classes.submit}
            >
                Submit
            </Button>
        </form>
    );
}