import { parseSearchParams, useLocation } from '@shared/route';
import { useCustomMutation } from 'api/hooks';
import { APP_LINKS, EmailLogInInput, Session } from '@shared/consts';
import { useFormik } from 'formik';
import {
    Button,
    Grid,
    Link,
    Paper,
    TextField,
    Typography
} from '@mui/material';
import { Forms, PubSub } from 'utils';
import { LogInFormProps } from './types';
import { formNavLink, formPaper, formSubmit } from './styles';
import { clickSize } from 'styles';
import { PasswordTextField } from 'components';
import { useMemo } from 'react';
import { CSSProperties } from '@mui/styles';
import { errorToCode, hasErrorCode, mutationWrapper } from 'api/utils';
import { emailLogInFormValidation } from '@shared/validation';
import { authEmailLogIn } from 'api/generated/endpoints/auth_emailLogIn';

export const LogInForm = ({
    onFormChange = () => { }
}: LogInFormProps) => {
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