import { useLocation } from 'wouter';
import { emailLogInMutation } from 'graphql/mutation';
import { useMutation } from '@apollo/client';
import { CODE, emailLogInForm } from '@local/shared';
import { useFormik } from 'formik';
import {
    Button,
    Grid,
    Link,
    Paper,
    TextField,
    Typography
} from '@mui/material';
import { Forms, Pubs } from 'utils';
import { APP_LINKS } from '@local/shared';
import PubSub from 'pubsub-js';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { emailLogIn } from 'graphql/generated/emailLogIn';
import { LogInFormProps } from './types';
import { formNavLink, formPaper, formSubmit } from './styles';
import { clickSize } from 'styles';
import { useMemo } from 'react';
import { parseSearchParams } from 'utils/urlTools';

export const LogInForm = ({
    onFormChange = () => { }
}: LogInFormProps) => {
    const [, setLocation] = useLocation();
    const redirect = useMemo(() => parseSearchParams(window.location.search).redirect?.replaceAll('%2F', '/'), [window.location.search]);
    const verificationCode = useMemo(() => parseSearchParams(window.location.search).code, [window.location.search]);

    const [emailLogIn, { loading }] = useMutation<emailLogIn>(emailLogInMutation);

    const formik = useFormik({
        initialValues: {
            email: '',
            password: ''
        },
        validationSchema: emailLogInForm,
        onSubmit: (values) => {
            mutationWrapper({
                mutation: emailLogIn,
                input: { ...values, verificationCode },
                successCondition: (response) => response.data.emailLogIn !== null,
                onSuccess: (response) => { PubSub.publish(Pubs.Session, response.data.emailLogIn); setLocation(redirect ?? APP_LINKS.Home) },
                onError: (response) => {
                    if (Array.isArray(response.graphQLErrors) && response.graphQLErrors.some(e => e.extensions.code === CODE.MustResetPassword.code)) {
                        PubSub.publish(Pubs.AlertDialog, {
                            message: 'Before signing in, please follow the link sent to your email to change your password.',
                            buttons: [
                                { text: 'Ok', onClick: () => { setLocation(redirect ?? APP_LINKS.Home) } },
                            ]
                        });
                    }
                    formik.setSubmitting(false);
                }
            })
        },
    });

    const toForgotPassword = () => onFormChange(Forms.ForgotPassword);
    const toSignUp = () => onFormChange(Forms.SignUp);

    return (
        <Paper sx={{ ...formPaper }}>
            <form onSubmit={formik.handleSubmit}>
                <Grid container spacing={2}>
                    <Grid item xs={12}>
                        <TextField
                            fullWidth
                            id="email"
                            name="email"
                            autoComplete="email"
                            label="Email Address"
                            value={formik.values.email}
                            onBlur={formik.handleBlur}
                            onChange={formik.handleChange}
                            error={formik.touched.email && Boolean(formik.errors.email)}
                            helperText={formik.touched.email && formik.errors.email}
                        />
                    </Grid>
                    <Grid item xs={12}>
                        <TextField
                            fullWidth
                            id="password"
                            name="password"
                            type="password"
                            autoComplete="current-password"
                            label="Password"
                            value={formik.values.password}
                            onBlur={formik.handleBlur}
                            onChange={formik.handleChange}
                            error={formik.touched.password && Boolean(formik.errors.password)}
                            helperText={formik.touched.password && formik.errors.password}
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
                    Log In
                </Button>
                <Grid container spacing={2}>
                    <Grid item xs={6}>
                        <Link onClick={toForgotPassword}>
                            <Typography
                                sx={{
                                    ...clickSize,
                                    ...formNavLink,
                                }}
                            >
                                Forgot Password?
                            </Typography>
                        </Link>
                    </Grid>
                    <Grid item xs={6}>
                        <Link onClick={toSignUp}>
                            <Typography
                                sx={{
                                    ...clickSize,
                                    ...formNavLink,
                                    flexDirection: 'row-reverse' as any,
                                }}
                            >
                                Don't have an account? Sign up
                            </Typography>
                        </Link>
                    </Grid>
                </Grid>
            </form>
        </Paper>
    );
}