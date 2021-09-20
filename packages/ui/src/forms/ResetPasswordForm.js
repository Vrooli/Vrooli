import React from 'react';
import { resetPasswordMutation } from 'graphql/mutation';
import { useMutation } from '@apollo/client';
import { resetPasswordSchema } from '@local/shared';
import { useFormik } from 'formik';
import {
    Button,
    Grid,
    TextField
} from '@material-ui/core';
import { makeStyles } from '@material-ui/styles';
import { LINKS } from 'utils';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { useParams } from 'react-router-dom';
import { formStyles } from './styles';

const useStyles = makeStyles(formStyles);

function ResetPasswordForm({
    onSessionUpdate,
    onRedirect
}) {
    const classes = useStyles();
    const urlParams = useParams();
    const [resetPassword, {loading}] = useMutation(resetPasswordMutation);

    const formik = useFormik({
        initialValues: {
            newPassword: '',
            confirmNewPassword: '',
        },
        validationSchema: resetPasswordSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation: resetPassword,
                data: { variables: { id: urlParams.id, code: urlParams.code, newPassword: values.newPassword } },
                onSuccess: (response) => { onSessionUpdate(response.data.resetPassword); onRedirect(LINKS.Portal) },
                successMessage: () => 'Password reset.',
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
                        id="newPassword"
                        name="newPassword"
                        type="password"
                        autoComplete="password"
                        label="New Password"
                        value={formik.values.newPassword}
                        onChange={formik.handleChange}
                        error={formik.touched.newPassword && Boolean(formik.errors.newPassword)}
                        helperText={formik.touched.newPassword && formik.errors.newPassword}
                    />
                </Grid>
                <Grid item xs={12}>
                    <TextField
                        fullWidth
                        autoFocus
                        id="confirmNewPassword"
                        name="confirmNewPassword"
                        type="password"
                        autoComplete="new-password"
                        label="Confirm New Password"
                        value={formik.values.confirmNewPassword}
                        onChange={formik.handleChange}
                        error={formik.touched.confirmNewPassword && Boolean(formik.errors.confirmNewPassword)}
                        helperText={formik.touched.confirmNewPassword && formik.errors.confirmNewPassword}
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

export { ResetPasswordForm };