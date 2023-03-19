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
import { CSSProperties } from '@mui/styles';
import { BUSINESS_NAME, EmailSignUpInput, LINKS, Session } from '@shared/consts';
import { useLocation } from '@shared/route';
import { emailSignUpFormValidation } from '@shared/validation';
import { authEmailSignUp } from 'api/generated/endpoints/auth_emailSignUp';
import { useCustomMutation } from 'api/hooks';
import { hasErrorCode, mutationWrapper } from 'api/utils';
import { PasswordTextField } from 'components/inputs/PasswordTextField/PasswordTextField';
import { useFormik } from 'formik';
import { useTranslation } from 'react-i18next';
import { subscribeUserToPush } from 'serviceWorkerRegistration';
import { clickSize } from 'styles';
import { Forms } from 'utils/consts';
import { PubSub } from 'utils/pubsub';
import { formNavLink, formPaper, formSubmit } from '../../styles';
import { FormProps } from '../../types';

export const SignUpForm = ({
    onFormChange = () => { },
}: FormProps) => {
    const theme = useTheme();
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const [emailSignUp, { loading }] = useCustomMutation<Session, EmailSignUpInput>(authEmailSignUp);

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
                                setLocation(LINKS.Welcome);
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
                            name="password"
                            autoComplete="new-password"
                            label={t('Password')}
                        />
                    </Grid>
                    <Grid item xs={12}>
                        <PasswordTextField
                            fullWidth
                            name="confirmPassword"
                            autoComplete="new-password"
                            label={t('PasswordConfirm')}
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
                    {t('SignUp')}
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
                                {t('AlreadyHaveAccountLogIn')}
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
                                {t('ForgotPassword')}
                            </Typography>
                        </Link>
                    </Grid>
                </Grid>
            </form>
        </Paper>
    );
}