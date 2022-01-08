import { useLocation } from 'wouter';
import { emailLogInMutation } from 'graphql/mutation';
import { useMutation } from '@apollo/client';
import { CODE, emailLogInSchema } from '@local/shared';
import { useFormik } from 'formik';
import {
    Button,
    Grid,
    Link,
    TextField,
    Typography
} from '@mui/material';
import { makeStyles } from '@mui/styles';
import { FORMS, PUBS } from 'utils';
import { APP_LINKS } from '@local/shared';
import PubSub from 'pubsub-js';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { formStyles } from './styles';
import { emailLogIn } from 'graphql/generated/emailLogIn';
import { LogInFormProps } from './types';

const useStyles = makeStyles(formStyles);

export const LogInForm = ({
    code,
    onSessionUpdate,
    onFormChange = () => {}
}: LogInFormProps) => {
    const classes = useStyles();
    const [, setLocation] = useLocation();
    const [emailLogIn, { loading }] = useMutation<emailLogIn>(emailLogInMutation);

    const formik = useFormik({
        initialValues: {
            email: '',
            password: ''
        },
        validationSchema: emailLogInSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation: emailLogIn,
                input: { ...values, verificationCode: code },
                successCondition: (response) => response.data.emailLogIn !== null,
                onSuccess: (response) => { onSessionUpdate(response.data.emailLogIn); setLocation(APP_LINKS.Home) },
                onError: (response) => {
                    if (Array.isArray(response.graphQLErrors) && response.graphQLErrors.some(e => e.extensions.code === CODE.MustResetPassword.code)) {
                        PubSub.publish(PUBS.AlertDialog, {
                            message: 'Before signing in, please follow the link sent to your email to change your password.',
                            firstButtonText: 'OK',
                            firstButtonClicked: () => setLocation(APP_LINKS.Home),
                        });
                    }
                }
            })
        },
    });

    const toForgotPassword = () => onFormChange(FORMS.ForgotPassword);
    const toSignUp = () => onFormChange(FORMS.SignUp);

    return (
        <form className={classes.form} onSubmit={formik.handleSubmit}>
            <Grid container spacing={2}>
                <Grid item xs={12}>
                    <TextField
                        fullWidth
                        id="email"
                        name="email"
                        autoComplete="email"
                        label="Email Address"
                        value={formik.values.email}
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
                        autoComplete="current-password"
                        label="Password"
                        value={formik.values.password}
                        onChange={formik.handleChange}
                        error={formik.touched.password && Boolean(formik.errors.password)}
                        helperText={formik.touched.password && formik.errors.password}
                    />
                </Grid>
            </Grid>
            <Button
                fullWidth
                disabled={loading}
                type="submit"
                color="secondary"
                className={classes.submit}
            >
                Log In
            </Button>
            <Grid container spacing={2}>
                <Grid item xs={6}>
                    <Link onClick={toForgotPassword}>
                        <Typography className={classes.clickSize}>
                            Forgot Password?
                        </Typography>
                    </Link>
                </Grid>
                <Grid item xs={6}>
                    <Link onClick={toSignUp}>
                        <Typography className={`${classes.clickSize} ${classes.linkRight}`}>
                            Don't have an account? Sign up
                        </Typography>
                    </Link>
                </Grid>
            </Grid>
        </form>
    );
}