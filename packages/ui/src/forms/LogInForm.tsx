import {
    Button,
    Grid,
    Link,
    Paper,
    TextField,
    Typography
} from '@mui/material';
import { CSSProperties } from '@mui/styles';
import { EmailLogInInput, LINKS, Session } from '@shared/consts';
import { parseSearchParams, useLocation } from '@shared/route';
import { emailLogInFormValidation } from '@shared/validation';
import { authEmailLogIn } from 'api/generated/endpoints/auth_emailLogIn';
import { useCustomMutation } from 'api/hooks';
import { errorToCode, hasErrorCode, mutationWrapper } from 'api/utils';
import { PasswordTextField } from 'components/inputs/PasswordTextField/PasswordTextField';
import { useFormik } from 'formik';
import { useMemo } from 'react';
import { useTranslation } from 'react-i18next';
import { clickSize } from 'styles';
import { Forms } from 'utils/consts';
import { PubSub } from 'utils/pubsub';
import { formNavLink, formPaper, formSubmit } from './styles';
import { LogInFormProps } from './types';

export const LogInForm = ({
    onFormChange = () => { }
}: LogInFormProps) => {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();

    const { redirect, verificationCode } = useMemo(() => {
        const params = parseSearchParams();
        return {
            redirect: typeof params.redirect === 'string' ? params.redirect : undefined,
            verificationCode: typeof params.code === 'string' ? params.code : undefined,
        }
    }, []);

    const [emailLogIn, { loading }] = useCustomMutation<Session, EmailLogInInput>(authEmailLogIn);

    const toForgotPassword = () => onFormChange(Forms.ForgotPassword);
    const toSignUp = () => onFormChange(Forms.SignUp);

    const formik = useFormik({
        initialValues: {
            email: '',
            password: ''
        },
        validationSchema: emailLogInFormValidation,
        onSubmit: (values) => {
            mutationWrapper<Session, EmailLogInInput>({
                mutation: emailLogIn,
                input: { ...values, verificationCode },
                successCondition: (data) => data !== null,
                onSuccess: (data) => {
                    if (verificationCode) PubSub.get().publishSnack({ messageKey: 'EmailVerified', severity: 'Success' });
                    PubSub.get().publishSession(data);
                    setLocation(redirect ?? LINKS.Home);
                },
                showDefaultErrorSnack: false,
                onError: (response) => {
                    // Custom dialog for changing password
                    if (hasErrorCode(response, 'MustResetPassword')) {
                        PubSub.get().publishAlertDialog({
                            messageKey: 'ChangePasswordBeforeLogin',
                            buttons: [
                                { labelKey: 'Ok', onClick: () => { setLocation(redirect ?? LINKS.Home) } },
                            ]
                        });
                    }
                    // Custom snack for invalid email, that has sign up link
                    else if (hasErrorCode(response, 'EmailNotFound')) {
                        PubSub.get().publishSnack({
                            messageKey: 'EmailNotFound',
                            severity: 'Error',
                            buttonKey: 'SignUp',
                            buttonClicked: () => { toSignUp() }
                        });
                    } else {
                        PubSub.get().publishSnack({ messageKey: errorToCode(response), severity: 'Error', data: response });
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
                            name="password"
                            autoComplete="current-password"
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
                    {t('LogIn')}
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
                                {t('ForgotPassword')}
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
                                {t('DontHaveAccountSignUp')}
                            </Typography>
                        </Link>
                    </Grid>
                </Grid>
            </form>
        </Paper>
    );
}