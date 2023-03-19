import {
    Button,
    Grid,
    Link,
    Paper,
    TextField,
    Typography
} from '@mui/material';
import { CSSProperties } from '@mui/styles';
import { EmailRequestPasswordChangeInput, LINKS, Success } from '@shared/consts';
import { useLocation } from '@shared/route';
import { emailRequestPasswordChangeSchema } from '@shared/validation';
import { authEmailRequestPasswordChange } from 'api/generated/endpoints/auth_emailRequestPasswordChange';
import { useCustomMutation } from 'api/hooks';
import { mutationWrapper } from 'api/utils';
import { useFormik } from 'formik';
import { useTranslation } from 'react-i18next';
import { clickSize } from 'styles';
import { Forms } from 'utils/consts';
import { formNavLink, formPaper, formSubmit } from '../../styles';
import { FormProps } from '../../types';

export const ForgotPasswordForm = ({
    onFormChange = () => { }
}: FormProps) => {
    const { t } = useTranslation();
    const [, setLocation] = useLocation();
    const [emailRequestPasswordChange, { loading }] = useCustomMutation<Success, EmailRequestPasswordChangeInput>(authEmailRequestPasswordChange);

    const formik = useFormik({
        initialValues: {
            email: ''
        },
        validationSchema: emailRequestPasswordChangeSchema,
        onSubmit: (values) => {
            mutationWrapper<Success, EmailRequestPasswordChangeInput>({
                mutation: emailRequestPasswordChange,
                input: { ...values },
                successCondition: (data) => data.success === true,
                onSuccess: () => setLocation(LINKS.Home),
                onError: () => { formik.setSubmitting(false) },
                successMessage: () => ({ key: 'RequestSentCheckEmail' }),
            })
        },
    });

    const toSignUp = () => onFormChange(Forms.SignUp);
    const toLogIn = () => onFormChange(Forms.LogIn);

    return (
        <Paper sx={{ ...formPaper }}>
            <form onSubmit={formik.handleSubmit}>
                <Grid container spacing={2}>
                    <Grid item xs={12}>
                        <TextField
                            fullWidth
                            autoFocus
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
                </Grid>
                <Button
                    fullWidth
                    disabled={loading}
                    type="submit"
                    color="secondary"
                    sx={{ ...formSubmit }}
                >
                    {t('Submit')}
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
                                {t('RememberLogBackIn')}
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