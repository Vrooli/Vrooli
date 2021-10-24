import { useHistory, useParams } from 'react-router-dom';
import { joinWaitlistMutation, verifyWaitlistMutation } from 'graphql/mutation';
import { useMutation } from '@apollo/client';
import { joinWaitlistSchema } from '@local/shared';
import { useFormik } from 'formik';
import {
    Button,
    Grid,
    TextField,
} from '@material-ui/core';
import { makeStyles } from '@material-ui/styles';
import { LINKS } from 'utils';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { formStyles } from './styles';
import { CommonProps } from 'types';
import { useCallback, useEffect } from 'react';

const useStyles = makeStyles(formStyles);

export const WaitlistForm = ({
    onSessionUpdate
}: Pick<CommonProps, 'onSessionUpdate'>) => {
    const classes = useStyles();
    const history = useHistory();
    const urlParams = useParams<{code?: string}>();
    const [joinWaitlist, { loading }] = useMutation(joinWaitlistMutation);
    const [verifyWaitlist] = useMutation(verifyWaitlistMutation);

    const toHome = useCallback(() => history.push(LINKS.Home), [history]);

    useEffect(() => {
        if (urlParams.code) {
            mutationWrapper({
                mutation: verifyWaitlist,
                data: { variables: { confirmationCode: urlParams.code } },
                successCondition: (response) => response.data.verifyWaitlist,
                onSuccess: toHome,
                successMessage: () => 'Email verified. See you soon!',
            })
        }
    }, [toHome, urlParams, verifyWaitlist])

    const formik = useFormik({
        initialValues: {
            username: '',
            email: '',
        },
        validationSchema: joinWaitlistSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation: joinWaitlist,
                data: { variables: values },
                successCondition: (response) => response.data.joinWaitlist,
                onSuccess: toHome,
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
        </form>
    );
}