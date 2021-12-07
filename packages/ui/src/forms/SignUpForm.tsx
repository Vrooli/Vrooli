import { signUpMutation } from 'graphql/mutation';
import { useMutation } from '@apollo/client';
import { BUSINESS_NAME, CODE, signUpSchema } from '@local/shared';
import { useFormik } from 'formik';
import {
    Button,
    Checkbox,
    FormControlLabel,
    Grid,
    Link,
    TextField,
    Typography,
    useTheme
} from '@material-ui/core';
import { makeStyles } from '@material-ui/styles';
import { combineStyles, FORMS, PUBS } from 'utils';
import { APP_LINKS } from '@local/shared';
import PubSub from 'pubsub-js';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { useHistory } from 'react-router-dom';
import { formStyles } from './styles';
import { signUp } from 'graphql/generated/signUp';
import { FormProps } from 'forms';

const componentStyles = () => ({
    phoneInput: {
        width: '100%',
    },
})

const useStyles = makeStyles(combineStyles(formStyles, componentStyles));

export const SignUpForm = ({
    onSessionUpdate,
    onFormChange = () => {},
}: FormProps) => {
    const classes = useStyles();
    const theme = useTheme();
    const history = useHistory();
    const [signUp, { loading }] = useMutation<signUp>(signUpMutation);

    const formik = useFormik({
        initialValues: {
            marketingEmails: "true",
            username: '',
            email: '',
            password: '',
            confirmPassword: ''
        },
        validationSchema: signUpSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation: signUp,
                data: { variables: { 
                    ...values, 
                    marketingEmails: Boolean(values.marketingEmails),
                    theme: theme.palette.mode ?? 'light',
                } },
                onSuccess: (response) => {
                    onSessionUpdate(response.data.signUp);
                    PubSub.publish(PUBS.AlertDialog, {
                        message: `Welcome to ${BUSINESS_NAME}. Please verify your email within 48 hours.`,
                        buttons: [ { text: 'OK', onClick: () => history.push(APP_LINKS.Profile) } ]
                    });
                },
                onError: (response) => {
                    if (Array.isArray(response.graphQLErrors) && response.graphQLErrors.some(e => e.extensions.code === CODE.EmailInUse.code)) {
                        PubSub.publish(PUBS.AlertDialog, {
                            message: `${response.message}. Press OK if you would like to be redirected to the forgot password form.`,
                            buttons: [ { text: 'OK', onClick: () => onFormChange(FORMS.ForgotPassword) } ]
                        });
                    }
                }
            })
        },
    });

    const toLogIn = () => onFormChange(FORMS.LogIn);
    const toForgotPassword = () => onFormChange(FORMS.ForgotPassword);

    return (
        <form className={classes.form} onSubmit={formik.handleSubmit}>
            <Grid container spacing={2}>
                <Grid item xs={12} sm={6}>
                <TextField
                        fullWidth
                        autoFocus
                        id="username"
                        name="username"
                        autoComplete="username"
                        label="Username"
                        value={formik.values.username}
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
                className={classes.submit}
            >
                Sign Up
            </Button>
            <Grid container spacing={2}>
                <Grid item xs={6}>
                    <Link onClick={toLogIn}>
                        <Typography className={classes.clickSize}>
                            Already have an account? Log in
                        </Typography>
                    </Link>
                </Grid>
                <Grid item xs={6}>
                    <Link onClick={toForgotPassword}>
                        <Typography className={`${classes.clickSize} ${classes.linkRight}`}>
                            Forgot Password?
                        </Typography>
                    </Link>
                </Grid>
            </Grid>
        </form>
    );
}