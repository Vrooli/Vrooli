import { requestPasswordChangeMutation } from 'graphql/mutation';
import { useMutation } from '@apollo/client';
import { requestPasswordChangeSchema } from '@local/shared';
import { useFormik } from 'formik';
import {
    Button,
    Grid,
    Link,
    TextField,
    Typography
} from '@material-ui/core';
import { makeStyles } from '@material-ui/styles';
import { LINKS } from 'utils';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { useHistory } from 'react-router-dom';
import { formStyles } from './styles';
import { requestPasswordChange } from 'graphql/generated/requestPasswordChange';
import { useCallback } from 'react';

const useStyles = makeStyles(formStyles);

export const ForgotPasswordForm = () => {
    const classes = useStyles();
    const history = useHistory();
    const [requestPasswordChange, {loading}] = useMutation<requestPasswordChange>(requestPasswordChangeMutation);

    const formik = useFormik({
        initialValues: {
            email: ''
        },
        validationSchema: requestPasswordChangeSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation: requestPasswordChange,
                data: { variables: values },
                successCondition: (response) => response.data.requestPasswordChange,
                onSuccess: () => history.push(LINKS.Home),
                successMessage: () => 'Request sent. Please check email.',
            })
        },
    });

    const toRegister = useCallback(() => history.push(LINKS.Register), [history]);
    const toLogIn = useCallback(() => history.push(LINKS.LogIn), [history]);

    return (
        <form className={classes.form} onSubmit={formik.handleSubmit}>
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
                className={classes.submit}
            >
                Submit
            </Button>
            <Grid container spacing={2}>
                <Grid item xs={6}>
                    <Link onClick={toLogIn}>
                        <Typography className={classes.clickSize}>
                            Remember? Back to Log In
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