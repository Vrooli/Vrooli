import React from 'react';
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

const useStyles = makeStyles(formStyles);

function ForgotPasswordForm({
    onRedirect
}) {
    const classes = useStyles();
    const history = useHistory();
    const [requestPasswordChange, {loading}] = useMutation(requestPasswordChangeMutation);

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
                onSuccess: () => onRedirect(LINKS.Home),
                successMessage: () => 'Request sent. Please check email.',
            })
        },
    });

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
                    <Link onClick={() => history.push(LINKS.LogIn)}>
                        <Typography className={classes.clickSize}>
                            Remember? Back to Log In
                        </Typography>
                    </Link>
                </Grid>
                <Grid item xs={6}>
                    <Link onClick={() => history.push(LINKS.Register)}>
                        <Typography className={`${classes.clickSize} ${classes.linkRight}`}>
                            Don't have an account? Sign up
                        </Typography>
                    </Link>
                </Grid>
            </Grid>
        </form>
    );
}

export { ForgotPasswordForm };