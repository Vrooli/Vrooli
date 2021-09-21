import React, { useEffect } from 'react';
import { joinWaitlistMutation, verifyWaitlistMutation } from 'graphql/mutation';
import { useMutation } from '@apollo/client';
import { DEFAULT_PRONOUNS, joinWaitlistSchema } from '@local/shared';
import { useFormik } from 'formik';
import {
    Autocomplete,
    Button,
    Grid,
    Link,
    TextField,
    Typography
} from '@material-ui/core';
import { makeStyles } from '@material-ui/styles';
import { LINKS } from 'utils';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { useHistory, useParams } from 'react-router-dom';
import { formStyles } from './styles';

const useStyles = makeStyles(formStyles);

function WaitlistForm({
    onRedirect
}) {
    const classes = useStyles();
    const history = useHistory();
    const urlParams = useParams();
    const [joinWaitlist, { loading }] = useMutation(joinWaitlistMutation);
    const [verifyWaitlist] = useMutation(verifyWaitlistMutation);

    useEffect(() => {
        if (urlParams.code) {
            mutationWrapper({
                mutation: verifyWaitlist,
                data: { variables: { confirmationCode: urlParams.code } },
                successCondition: (response) => response.data.verifyWaitlist,
                onSuccess: () => onRedirect(LINKS.Home),
                successMessage: () => 'Email verified. See you soon!',
            })
        }
    }, [urlParams])

    const formik = useFormik({
        initialValues: {
            firstName: '',
            lastName: '',
            pronouns: '',
            email: '',
        },
        validationSchema: joinWaitlistSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation: joinWaitlist,
                data: { variables: values },
                successCondition: (response) => response.data.joinWaitlist,
                onSuccess: () => onRedirect(LINKS.Home),
                successMessage: () => 'Request sent. Please check email.',
            })
        },
    });

    return (
        <form className={classes.form} onSubmit={formik.handleSubmit}>
            <Grid container spacing={2}>
                <Grid item xs={12} sm={6}>
                    <TextField
                        fullWidth
                        autoFocus
                        id="firstName"
                        name="firstName"
                        autoComplete="fname"
                        label="First Name"
                        value={formik.values.firstName}
                        onChange={formik.handleChange}
                        error={formik.touched.firstName && Boolean(formik.errors.firstName)}
                        helperText={formik.touched.firstName && formik.errors.firstName}
                    />
                </Grid>
                <Grid item xs={12} sm={6}>
                    <TextField
                        fullWidth
                        id="lastName"
                        name="lastName"
                        autoComplete="lname"
                        label="Last Name"
                        value={formik.values.lastName}
                        onChange={formik.handleChange}
                        error={formik.touched.lastName && Boolean(formik.errors.lastName)}
                        helperText={formik.touched.lastName && formik.errors.lastName}
                    />
                </Grid>
                <Grid item xs={12}>
                    <Autocomplete
                        fullWidth
                        freeSolo
                        id="pronouns"
                        name="pronouns"
                        options={DEFAULT_PRONOUNS}
                        value={formik.values.pronouns}
                        onChange={(_, value) => formik.setFieldValue('pronouns', value)}
                        renderInput={(params) => (
                            <TextField
                                {...params}
                                label="Pronouns"
                                value={formik.values.pronouns}
                                onChange={formik.handleChange}
                                error={formik.touched.pronouns && Boolean(formik.errors.pronouns)}
                                helperText={formik.touched.pronouns && formik.errors.pronouns}
                            />
                        )}
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

export { WaitlistForm };