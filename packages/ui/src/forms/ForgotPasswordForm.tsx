import { useMutation } from 'api/hooks';
import { emailRequestPasswordChangeSchema } from '@shared/validation';
import { useFormik } from 'formik';
import {
    Button,
    Grid,
    Link,
    Paper,
    TextField,
    Typography
} from '@mui/material';
import { APP_LINKS, EmailRequestPasswordChangeInput, Success } from '@shared/consts';
import { Forms } from 'utils';
import { mutationWrapper } from 'api/utils';
import { useLocation } from '@shared/route';
import { FormProps } from './types';
import { formNavLink, formPaper, formSubmit } from './styles';
import { clickSize } from 'styles';
import { CSSProperties } from '@mui/styles';
import { authEmailRequestPasswordChange } from 'api/generated/endpoints/auth';

export const ForgotPasswordForm = ({
    onFormChange = () => { }
}: FormProps) => {
    const [, setLocation] = useLocation();
    const [emailRequestPasswordChange, { loading }] = useMutation<Success, EmailRequestPasswordChangeInput, 'emailRequestPasswordChange'>(authEmailRequestPasswordChange, 'emailRequestPasswordChange');

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
                onSuccess: () => setLocation(APP_LINKS.Home),
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
                    Submit
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
                                Remember? Back to Log In
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