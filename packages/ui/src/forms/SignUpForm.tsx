import { useMutation } from 'api/hooks';
import { APP_LINKS, BUSINESS_NAME, EmailSignUpInput, Session } from '@shared/consts';
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
import { Forms, PubSub } from 'utils';
import { emailSignUpFormValidation } from '@shared/validation';
import { hasErrorCode, mutationWrapper } from 'api/utils';
import { useLocation } from '@shared/route';
import { FormProps } from './types';
import { formNavLink, formPaper, formSubmit } from './styles';
import { clickSize } from 'styles';
import { PasswordTextField } from 'components';
import { CSSProperties } from '@mui/styles';
import { subscribeUserToPush } from 'serviceWorkerRegistration';
import { authEmailSignUp } from 'api/generated/endpoints/auth';

export const SignUpForm = ({
    onFormChange = () => { },
}: FormProps) => {
    const theme = useTheme();
    const [, setLocation] = useLocation();
    const [emailSignUp, { loading }] = useMutation<Session, EmailSignUpInput, 'emailSignUp'>(authEmailSignUp, 'emailSignUp');

    const formik = useFormik({
        initialValues: {
            marketingEmails: true,
            name: '',
            email: '',
            password: '',
            confirmPassword: ''
        },
        validationSchema: emailSignUpFormValidation,
        onSubmit: (values) => {
            mutationWrapper<Session, EmailSignUpInput>({
                mutation: emailSignUp,
                input: {
                    ...values,
                    marketingEmails: Boolean(values.marketingEmails),
                    theme: theme.palette.mode ?? 'light',
                },
                onSuccess: (data) => {
                    PubSub.get().publishSession(data)
                    PubSub.get().publishAlertDialog({
                        messageKey: 'WelcomeVerifyEmail',
                        messageVariables: { appName: BUSINESS_NAME },
                        buttons: [{
                            labelKey: 'Ok', onClick: () => {
                                setLocation(APP_LINKS.Welcome);
                                // Request user to enable notifications
                                subscribeUserToPush();
                            }
                        }]
                    });
                },
                onError: (response) => {
                    if (hasErrorCode(response, 'EmailInUse')) {
                        PubSub.get().publishAlertDialog({
                            messageKey: 'EmailInUseWrongPassword',
                            buttons: [
                                { labelKey: 'Yes', onClick: () => onFormChange(Forms.ForgotPassword) },
                                { labelKey: 'No' }
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