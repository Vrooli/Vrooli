import { useCallback, useState } from 'react'
import { profileSchema } from '@local/shared';
import { useMutation, useQuery } from '@apollo/client';
import { updateCustomerMutation } from 'graphql/mutation';
import { profileQuery } from 'graphql/query';
import { useFormik } from 'formik';
import { combineStyles, PUBS } from 'utils';
import PubSub from 'pubsub-js';
import { Button, Container, FormHelperText, Grid, TextField, Checkbox, FormControlLabel, Theme } from '@material-ui/core';
import FormControl from '@material-ui/core/FormControl';
import Radio from '@material-ui/core/Radio';
import RadioGroup from '@material-ui/core/RadioGroup';
import { makeStyles } from '@material-ui/styles';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { formStyles } from './styles';
import { profile } from 'graphql/generated/profile';
import { updateCustomer } from 'graphql/generated/updateCustomer';

const componentStyles = (theme: Theme) => ({
    buttons: {
        paddingTop: theme.spacing(2),
        paddingBottom: theme.spacing(2),
    },
    phoneInput: {
        width: '100%',
    }
})

const useStyles = makeStyles(combineStyles(formStyles, componentStyles));

export const ProfileForm = () => {
    const classes = useStyles()
    const [editing, setEditing] = useState(false);
    const { data: profile } = useQuery<profile>(profileQuery);
    const [updateCustomer, { loading }] = useMutation<updateCustomer>(updateCustomerMutation);

    const formik = useFormik({
        enableReinitialize: true,
        initialValues: {
            username: profile?.profile?.username ?? '',
            email: profile?.profile?.emails?.length > 0 ? profile.profile.emails[0].emailAddress : '',
            theme: profile?.profile?.theme ?? 'light',
            marketingEmails: profile?.profile?.emails?.length > 0 ? profile.profile.emails[0].receivesDeliveryUpdates : false,
            currentPassword: '',
            newPassword: '',
            newPasswordConfirmation: ''
        },
        validationSchema: profileSchema,
        onSubmit: (values) => {
            let input = ({
                id: profile.profile.id,
                username: values.username,
                emails: [
                    {
                        id: profile?.profile?.emails?.length > 0 ? profile.profile.emails[0].id : '',
                        emailAddress: values.email,
                        receivesDeliveryUpdates: values.marketingEmails
                    }
                ],
                theme: values.theme,
            });
            // Only add email ids if they previously existed
            if (profile?.profile?.emails?.length > 0) input.emails[0].id = profile.profile.emails[0].id;
            mutationWrapper({
                mutation: updateCustomer,
                data: { variables: {
                    input: input,
                    currentPassword: values.currentPassword,
                    newPassword: values.newPassword
                } },
                successMessage: () => 'Profile updated.',
            })
        },
    });

    const toggleEdit = (event) => {
        event.preventDefault();
        setEditing(edit => !edit);
    }

    const setTheme = useCallback((e) => { formik.handleChange(e); PubSub.publish(PUBS.Theme, e.target.value) }, [formik])

    return (
        <form className={classes.form} onSubmit={formik.handleSubmit}>
            <fieldset disabled={!editing}>
                <Container>
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
                            <FormControl component="fieldset">
                                <RadioGroup
                                    id="theme"
                                    name="theme"
                                    aria-label="theme-check"
                                    value={formik.values.theme}
                                    onChange={setTheme}
                                >
                                    <FormControlLabel value="light" control={<Radio />} label="Light ☀️" />
                                    <FormControlLabel value="dark" control={<Radio />} label="Dark 🌙" />
                                </RadioGroup>
                                <FormHelperText>{formik.touched.theme && formik.errors.theme}</FormHelperText>
                            </FormControl>
                        </Grid>
                        <Grid item xs={12}>
                            <FormControlLabel
                                control={
                                    <Checkbox
                                        id="marketingEmails"
                                        name="marketingEmails"
                                        value="marketingEmails"
                                        color="secondary"
                                        checked={formik.values.marketingEmails}
                                        onChange={formik.handleChange}
                                    />
                                }
                                label="I want to receive marketing promotions and updates via email."
                            />
                        </Grid>
                    </Grid>
                </Container>
                <Container>
                    <Grid container spacing={2}>
                        <Grid item xs={12}>
                            <TextField
                                fullWidth
                                id="currentPassword"
                                name="currentPassword"
                                type="password"
                                autoComplete="password"
                                label="Current Password"
                                value={formik.values.currentPassword}
                                onChange={formik.handleChange}
                                error={formik.touched.currentPassword && Boolean(formik.errors.currentPassword)}
                                helperText={formik.touched.currentPassword && formik.errors.currentPassword}
                            />
                        </Grid>
                        <Grid item xs={12}>
                            <TextField
                                fullWidth
                                id="newPassword"
                                name="newPassword"
                                type="password"
                                autoComplete="new-password"
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
                                id="newPasswordConfirmation"
                                name="newPasswordConfirmation"
                                type="password"
                                autoComplete="new-password"
                                label="Confirm New Password"
                                value={formik.values.newPasswordConfirmation}
                                onChange={formik.handleChange}
                                error={formik.touched.newPasswordConfirmation && Boolean(formik.errors.newPasswordConfirmation)}
                                helperText={formik.touched.newPasswordConfirmation && formik.errors.newPasswordConfirmation}
                            />
                        </Grid>
                    </Grid>
                </Container>
            </fieldset>
            <Grid className={classes.buttons} container spacing={2}>
                <Grid item xs={12} sm={6}>
                    <Button fullWidth onClick={toggleEdit}>
                        {editing ? "Cancel" : "Edit"}
                    </Button>
                </Grid>
                <Grid item xs={12} sm={6}>
                    <Button
                        fullWidth
                        disabled={loading}
                        type="submit"
                        color="secondary"
                        className={classes.submit}
                    >
                        Save Changes
                    </Button>
                </Grid>
            </Grid>
        </form>
    );
}