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
import { mutationWrapper } from 'graphql/utils/wrappers';
import { useLocation } from 'wouter';
import { emailSignUp } from 'graphql/generated/emailSignUp';
import { FormProps } from './types';
import { formNavLink, formPaper, formSubmit } from './styles';
import { clickSize } from 'styles';

export const SignUpForm = ({
    onSessionUpdate,
    onFormChange = () => { },
}: FormProps) => {
    const theme = useTheme();
    const [, setLocation] = useLocation();
    const [emailSignUp, { loading }] = useMutation<emailSignUp>(emailSignUpMutation);

    const formik = useFormik({
        initialValues: {
            marketingEmails: "true",
            username: '',
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
                    onSessionUpdate(response.data.emailSignUp);
                    PubSub.publish(Pubs.AlertDialog, {
                        message: `Welcome to ${BUSINESS_NAME}. Please verify your email within 48 hours.`,
                        buttons: [{ text: 'OK', onClick: () => setLocation(APP_LINKS.Profile) }]
                    });
                },
                onError: (response) => {
                    if (Array.isArray(response.graphQLErrors) && response.graphQLErrors.some(e => e.extensions.code === CODE.EmailInUse.code)) {
                        PubSub.publish(Pubs.AlertDialog, {
                            message: `${response.message}. Press OK if you would like to be redirected to the forgot password form.`,
                            buttons: [{ text: 'OK', onClick: () => onFormChange(Forms.ForgotPassword) }]
                        });
                    }
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
                            id="username"
                            name="username"
                            autoComplete="username"
                            label="Username"
                            value={formik.values.username}
                            onBlur={formik.handleBlur}
                            onChange={formik.handleChange}
                            error={formik.touched.username && Boolean(formik.errors.username)}
                            helperText={formik.touched.username && formik.errors.username}
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
                        <TextField
                            fullWidth
                            id="password"
                            name="password"
                            type="password"
                            autoComplete="new-password"
                            label="Password"
                            value={formik.values.password}
                            onBlur={formik.handleBlur}
                            onChange={formik.handleChange}
                            error={formik.touched.password && Boolean(formik.errors.password)}
                            helperText={formik.touched.password && formik.errors.password}
                        />
                    </Grid>
                    <Grid item xs={12}>
                        <TextField
                            fullWidth
                            id="confirmPassword"
                            name="confirmPassword"
                            type="password"
                            autoComplete="new-password"
                            label="Confirm Password"
                            value={formik.values.confirmPassword}
                            onBlur={formik.handleBlur}
                            onChange={formik.handleChange}
                            error={formik.touched.confirmPassword && Boolean(formik.errors.confirmPassword)}
                            helperText={formik.touched.confirmPassword && formik.errors.confirmPassword}
                        />
                    </Grid>
                    <Grid item xs={12}>
                        <FormControlLabel
                            control={
                                <Checkbox
                                    id="marketingEmails"
                                    name="marketingEmails"
                                    value="marketingEmails"
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
                                }}
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
                                    flexDirection: 'row-reverse' as const,
                                }}
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