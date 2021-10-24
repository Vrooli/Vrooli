import { useHistory, useParams } from 'react-router-dom';
import { loginMutation } from 'graphql/mutation';
import { useMutation } from '@apollo/client';
import { CODE, logInSchema } from '@local/shared';
import { useFormik } from 'formik';
import {
    Button,
    Grid,
    Link,
    TextField,
    Typography
} from '@material-ui/core';
import { makeStyles } from '@material-ui/styles';
import { LINKS, PUBS } from 'utils';
import PubSub from 'pubsub-js';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { formStyles } from './styles';
import { CommonProps } from 'types';
import { login } from 'graphql/generated/login';
import { useCallback } from 'react';

const useStyles = makeStyles(formStyles);

export const LogInForm = ({
    onSessionUpdate
}: Pick<CommonProps, 'onSessionUpdate'>) => {
    const classes = useStyles();
    const history = useHistory();
    const urlParams = useParams<{code?: string}>();
    const [login, { loading }] = useMutation<login>(loginMutation);

    const formik = useFormik({
        initialValues: {
            email: '',
            password: ''
        },
        validationSchema: logInSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation: login,
                data: { variables: { ...values, verificationCode: urlParams.code } },
                successCondition: (response) => response.data.login !== null,
                onSuccess: (response) => { onSessionUpdate(response.data.login); history.push(LINKS.Shopping) },
                onError: (response) => {
                    if (Array.isArray(response.graphQLErrors) && response.graphQLErrors.some(e => e.extensions.code === CODE.MustResetPassword.code)) {
                        PubSub.publish(PUBS.AlertDialog, {
                            message: 'Before signing in, please follow the link sent to your email to change your password.',
                            firstButtonText: 'OK',
                            firstButtonClicked: () => history.push(LINKS.Home),
                        });
                    }
                }
            })
        },
    });

    const toForgotPassword = useCallback(() => history.push(LINKS.ForgotPassword), [history]);
    const toRegister = useCallback(() => history.push(LINKS.Register), [history]);

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
                    <Link onClick={toRegister}>
                        <Typography className={`${classes.clickSize} ${classes.linkRight}`}>
                            Don't have an account? Sign up
                        </Typography>
                    </Link>
                </Grid>
            </Grid>
        </form>
    );
}