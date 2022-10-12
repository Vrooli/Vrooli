import { emailRequestPasswordChangeMutation } from 'graphql/mutation';
import { useMutation } from '@apollo/client';
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
import { APP_LINKS } from '@shared/consts';
import { Forms } from 'utils';
import { mutationWrapper } from 'graphql/utils/graphqlWrapper';
import { useLocation } from '@shared/route';
import { emailRequestPasswordChangeVariables, emailRequestPasswordChange_emailRequestPasswordChange } from 'graphql/generated/emailRequestPasswordChange';
import { FormProps } from './types';
import { formNavLink, formPaper, formSubmit } from './styles';
import { clickSize } from 'styles';
import { CSSProperties } from '@mui/styles';

export const ForgotPasswordForm = ({
    onFormChange = () => { }
}: FormProps) => {
    const [, setLocation] = useLocation();
    const [emailRequestPasswordChange, { loading }] = useMutation(emailRequestPasswordChangeMutation);

    const formik = useFormik({
        initialValues: {
            email: ''
        },
        validationSchema: emailRequestPasswordChangeSchema,
        onSubmit: (values) => {
            mutationWrapper<emailRequestPasswordChange_emailRequestPasswordChange, emailRequestPasswordChangeVariables>({
                mutation: emailRequestPasswordChange,
                input: { ...values },
                successCondition: (data) => data.success === true,
                onSuccess: () => setLocation(APP_LINKS.Home),
                onError: () => { formik.setSubmitting(false) },
                successMessage: () => 'Request sent. Please check email.',
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