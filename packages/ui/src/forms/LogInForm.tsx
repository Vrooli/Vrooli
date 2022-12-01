import { useLocation } from '@shared/route';
import { emailLogInMutation } from 'graphql/mutation';
import { useMutation } from '@apollo/client';
import { APP_LINKS } from '@shared/consts';
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
import { emailLogInVariables, emailLogIn_emailLogIn } from 'graphql/generated/emailLogIn';
import { LogInFormProps } from './types';
import { formNavLink, formPaper, formSubmit } from './styles';
import { clickSize } from 'styles';
import { PasswordTextField, SnackSeverity } from 'components';
import { useMemo } from 'react';
import { CSSProperties } from '@mui/styles';
import { errorToCode, hasErrorCode } from 'graphql/utils';

export const LogInForm = ({
    onFormChange = () => { }
}: LogInFormProps) => {
    const [, setLocation] = useLocation();
    const search = useReactSearch();
    const { redirect, verificationCode } = useMemo(() => ({
        redirect: typeof search.redirect === 'string' ? search.redirect : undefined,
        verificationCode: typeof search.verificationCode === 'string' ? search.verificationCode : undefined,
    }), [search]);

    const [emailLogIn, { loading }] = useMutation(emailLogInMutation);  

    const toForgotPassword = () => onFormChange(Forms.ForgotPassword);
    const toSignUp = () => onFormChange(Forms.SignUp);

    const formik = useFormik({
        initialValues: {
            email: '',
            password: ''
        },
        validationSchema: emailLogInForm,
        onSubmit: (values) => {
            mutationWrapper<emailLogIn_emailLogIn, emailLogInVariables>({
                mutation: emailLogIn,
                input: { ...values, verificationCode },
                successCondition: (data) => data !== null,
                onSuccess: (data) => { 
                    if (verificationCode) PubSub.get().publishSnack({ messageKey: 'EmailVerified', severity: SnackSeverity.Success });
                    PubSub.get().publishSession(data); setLocation(redirect ?? APP_LINKS.Home) 
                },
                showDefaultErrorSnack: false,
                onError: (response) => {
                    // Custom dialog for changing password
                    if (hasErrorCode(response, 'MustResetPassword')) {
                        PubSub.get().publishAlertDialog({
                            messageKey: 'ChangePasswordBeforeLogin',
                            buttons: [
                                { labelKey: 'Ok', onClick: () => { setLocation(redirect ?? APP_LINKS.Home) } },
                            ]
                        });
                    }
                    // Custom snack for invalid email, that has sign up link
                    else if (hasErrorCode(response, 'EmailNotFound')) {
                        PubSub.get().publishSnack({ 
                            messageKey: 'EmailNotFound', 
                            severity: SnackSeverity.Error, 
                            buttonKey: 'SignUp',
                            buttonClicked: () => { toSignUp() }
                        });
                    } else {
                        PubSub.get().publishSnack({ messageKey: errorToCode(response), severity: SnackSeverity.Error, data: response });
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