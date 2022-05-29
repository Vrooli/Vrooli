import { emailSignUpMutation } from 'graphql/mutation';
import { useMutation } from '@apollo/client';
import { BUSINESS_NAME, CODE, emailSignUpSchema } from '@local/shared';
import { useFormik } from 'formik';
import {
    Button,
    Checkbox,
    FormControlLabel,
    Grid,
    Link,
    Paper,
    TextField,
    Typography,
    useTheme
} from '@mui/material';
import { Forms, Pubs } from 'utils';
import { APP_LINKS } from '@local/shared';
import PubSub from 'pubsub-js';
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { useLocation } from 'wouter';
import { emailSignUp } from 'graphql/generated/emailSignUp';
import { FormProps } from './types';
import { formNavLink, formPaper, formSubmit } from './styles';
import { clickSize } from 'styles';
import { PasswordTextField } from 'components';
import { CSSProperties } from '@mui/styles';
import { errorToMessage, hasErrorCode } from 'graphql/utils';

export const SignUpForm = ({
    onFormChange = () => { },
}: FormProps) => {
    const theme = useTheme();
    const [, setLocation] = useLocation();
    const [emailSignUp, { loading }] = useMutation<emailSignUp>(emailSignUpMutation);

    const formik = useFormik({
        initialValues: {
            marketingEmails: true,
            name: '',
            email: '',
            password: '',
            confirmPassword: ''
        },
        validationSchema: emailSignUpSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation: emailSignUp,
                input: {
                    ...values,
                    marketingEmails: Boolean(values.marketingEmails),
                    theme: theme.palette.mode ?? 'light',
                },
                onSuccess: (response) => {
                    PubSub.publish(Pubs.Session, response.data.emailSignUp)
                    PubSub.publish(Pubs.AlertDialog, {
                        message: `Welcome to ${BUSINESS_NAME}. Please verify your email within 48 hours.`,
                        buttons: [{ text: 'OK', onClick: () => setLocation(APP_LINKS.Welcome) }]
                    });
                },
                onError: (response) => {
                    if (hasErrorCode(response, CODE.EmailInUse)) {
                        PubSub.publish(Pubs.AlertDialog, {
                            message: `${errorToMessage(response)}. Did you forget your password?`,
                            buttons: [
                                { text: 'Yes', onClick: () => onFormChange(Forms.ForgotPassword) },
                                { text: 'No' }
                            ]
                        });
                    }
                    formik.setSubmitting(false);
                }
            })
        },
    });

    const toLogIn = () => onFormChange(Forms.LogIn);
    const toForgotPassword = () => onFormChange(Forms.ForgotPassword);

    return (
        <Paper sx={{ ...formPaper }}>
            <form onSubmit={formik.handleSubmit}>
                <Grid container spacing={2}>
                    <Grid item xs={12}>
                        <TextField
                            fullWidth
                            autoFocus
                            id="name"
                            name="name"
                            autoComplete="name"
                            label="Name"
                            value={formik.values.name}
                            onBlur={formik.handleBlur}
                            onChange={formik.handleChange}
                            error={formik.touched.name && Boolean(formik.errors.name)}
                            helperText={formik.touched.name && formik.errors.name}
                        />
                    </Grid>
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
                            autoComplete="new-password"
                            label="Password"
                            value={formik.values.password}
                            onBlur={formik.handleBlur}
                            onChange={formik.handleChange}
                            error={formik.touched.password && Boolean(formik.errors.password)}
                            helperText={formik.touched.password ? formik.errors.password : null}
                        />
                    </Grid>
                    <Grid item xs={12}>
                        <PasswordTextField
                            fullWidth
                            id="confirmPassword"
                            name="confirmPassword"
                            autoComplete="new-password"
                            label="Confirm Password"
                            value={formik.values.confirmPassword}
                            onBlur={formik.handleBlur}
                            onChange={formik.handleChange}
                            error={formik.touched.confirmPassword && Boolean(formik.errors.confirmPassword)}
                            helperText={formik.touched.confirmPassword ? formik.errors.confirmPassword : null}
                        />
                    </Grid>
                    <Grid item xs={12}>
                        <FormControlLabel
                            control={
                                <Checkbox
                                    id="marketingEmails"
                                    name="marketingEmails"
                                    color="secondary"
                                    checked={Boolean(formik.values.marketingEmails)}
                                    onBlur={formik.handleBlur}
                                    onChange={formik.handleChange}
                                />
                            }
                            label="I want to receive marketing promotions and updates via email."
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
                    Sign Up
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
                                Already have an account? Log in
                            </Typography>
                        </Link>
                    </Grid>
                    <Grid item xs={6}>
                        <Link onClick={toForgotPassword}>
                            <Typography
                                sx={{
                                    ...clickSize,
                                    ...formNavLink,
                                    flexDirection: 'row-reverse',
                                } as CSSProperties}
                            >
                                Forgot Password?
                            </Typography>
                        </Link>
                    </Grid>
                </Grid>
            </form>
        </Paper>
    );
}