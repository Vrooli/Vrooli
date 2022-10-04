import { useLocation } from '@shared/route';
import { emailLogInMutation } from 'graphql/mutation';
import { useMutation } from '@apollo/client';
import { APP_LINKS, CODE } from '@shared/consts';
import { useFormik } from 'formik';
import {
    Button,
    Grid,
    Link,
    Paper,
    TextField,
    Typography
} from '@mui/material';
import { Forms, PubSub, useReactSearch } from 'utils';
import { emailLogInForm } from '@shared/validation';
import { mutationWrapper } from 'graphql/utils/graphqlWrapper';
import { emailLogIn, emailLogInVariables } from 'graphql/generated/emailLogIn';
import { LogInFormProps } from './types';
import { formNavLink, formPaper, formSubmit } from './styles';
import { clickSize } from 'styles';
import { PasswordTextField, SnackSeverity } from 'components';
import { useMemo } from 'react';
import { CSSProperties } from '@mui/styles';
import { errorToMessage, hasErrorCode } from 'graphql/utils';

export const LogInForm = ({
    onFormChange = () => { }
}: LogInFormProps) => {
    const [, setLocation] = useLocation();
    const search = useReactSearch();
    const { redirect, verificationCode } = useMemo(() => ({
        redirect: typeof search.redirect === 'string' ? search.redirect : undefined,
        verificationCode: typeof search.verificationCode === 'string' ? search.verificationCode : undefined,
    }), [search]);

    const [emailLogIn, { loading }] = useMutation<emailLogIn, emailLogInVariables>(emailLogInMutation);  

    const toForgotPassword = () => onFormChange(Forms.ForgotPassword);
    const toSignUp = () => onFormChange(Forms.SignUp);

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
                onSuccess: (response) => { 
                    if (verificationCode) PubSub.get().publishSnack({ message: 'Email verified!', severity: SnackSeverity.Success });
                    PubSub.get().publishSession(response.data.emailLogIn); setLocation(redirect ?? APP_LINKS.Home) 
                },
                showDefaultErrorSnack: false,
                onError: (response) => {
                    // Custom dialog for changing password
                    if (hasErrorCode(response, CODE.MustResetPassword)) {
                        PubSub.get().publishAlertDialog({
                            message: 'Before signing in, please follow the link sent to your email to change your password.',
                            buttons: [
                                { text: 'Ok', onClick: () => { setLocation(redirect ?? APP_LINKS.Home) } },
                            ]
                        });
                    }
                    // Custom snack for invalid email, that has sign up link
                    else if (hasErrorCode(response, CODE.EmailNotFound)) {
                        PubSub.get().publishSnack({ 
                            message: CODE.EmailNotFound.message, 
                            severity: SnackSeverity.Error, 
                            buttonText: 'Sign Up',
                            buttonClicked: () => { toSignUp() }
                        });
                    } else {
                        PubSub.get().publishSnack({ message: errorToMessage(response), severity: SnackSeverity.Error, data: response });
                    }
                    formik.setSubmitting(false);
                }
            })
        },
    });

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
                        <PasswordTextField
                            fullWidth
                            id="password"
                            name="password"
                            autoComplete="current-password"
                            label="Password"
                            value={formik.values.password}
                            onBlur={formik.handleBlur}
                            onChange={formik.handleChange}
                            error={formik.touched.password && Boolean(formik.errors.password)}
                            helperText={formik.touched.password ? formik.errors.password : null}
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
                                } as CSSProperties}
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
                                    flexDirection: 'row-reverse',
                                } as CSSProperties}
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