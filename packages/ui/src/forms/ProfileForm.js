import React, { useState } from 'react'
import { DEFAULT_PRONOUNS, profileSchema } from '@local/shared';
import { useMutation, useQuery } from '@apollo/client';
import { updateCustomerMutation } from 'graphql/mutation';
import { profileQuery } from 'graphql/query';
import { useFormik } from 'formik';
import { Autocomplete } from '@material-ui/lab';
import { combineStyles, PUBS, PubSub } from 'utils';
import { Button, Container, FormHelperText, Grid, TextField, Checkbox, FormControlLabel } from '@material-ui/core';
import FormControl from '@material-ui/core/FormControl';
import Radio from '@material-ui/core/Radio';
import RadioGroup from '@material-ui/core/RadioGroup';
import { makeStyles } from '@material-ui/styles';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { formStyles } from './styles';

const componentStyles = (theme) => ({
    buttons: {
        paddingTop: theme.spacing(2),
        paddingBottom: theme.spacing(2),
    },
    phoneInput: {
        width: '100%',
    }
})

const useStyles = makeStyles(combineStyles(formStyles, componentStyles));

function ProfileForm() {
    const classes = useStyles()
    const [editing, setEditing] = useState(false);
    const { data: profile } = useQuery(profileQuery);
    const [updateCustomer, { loading }] = useMutation(updateCustomerMutation);

    const formik = useFormik({
        enableReinitialize: true,
        initialValues: {
            firstName: profile?.profile?.firstName ?? '',
            lastName: profile?.profile?.lastName ?? '',
            business: profile?.profile?.business?.name ?? '',
            pronouns: profile?.profile?.pronouns ?? '',
            email: profile?.profile?.emails?.length > 0 ? profile.profile.emails[0].emailAddress : '',
            phone: profile?.profile?.phones?.length > 0 ? profile.profile.phones[0].number : '1',
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
                firstName: values.firstName,
                lastName: values.lastName,
                business: {
                    id: profile.profile.business.id,
                    name: values.business
                },
                pronouns: values.pronouns,
                emails: [
                    {
                        id: profile?.profile?.emails?.length > 0 ? profile.profile.emails[0].id : '',
                        emailAddress: values.email,
                        receivesDeliveryUpdates: values.marketingEmails
                    }
                ],
                phones: [
                    {
                        id: profile?.profile?.phones?.length > 0 ? profile.profile.phones[0].id : '',
                        number: values.phone
                    }
                ],
                theme: values.theme,
            });
            // Only add email and phone ids if they previously existed
            if (profile?.profile?.emails?.length > 0) input.emails[0].id = profile.profile.emails[0].id;
            if (profile?.profile?.phones?.length > 0) input.phones[0].id = profile.profile.phones[0].id;
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

    return (
        <form className={classes.form} onSubmit={formik.handleSubmit} disabled={!editing}>
            <Container>
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
                            id="business"
                            name="business"
                            autoComplete="business"
                            label="Business"
                            value={formik.values.business}
                            onChange={formik.handleChange}
                            error={formik.touched.business && Boolean(formik.errors.business)}
                            helperText={formik.touched.business && formik.errors.business}
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
                            id="phone"
                            name="phone"
                            autoComplete="tel"
                            label="Phone Number"
                            value={formik.values.phone}
                            onChange={formik.handleChange}
                            error={formik.touched.phone && Boolean(formik.errors.phone)}
                            helperText={formik.touched.phone && formik.errors.phone}
                        />
                    </Grid>
                    <Grid item xs={12}>
                        <FormControl component="fieldset">
                            <RadioGroup
                                id="theme"
                                name="theme"
                                aria-label="theme-check"
                                value={formik.values.theme}
                                onChange={(e) => { formik.handleChange(e); PubSub.publish(PUBS.Theme, e.target.value) }}
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

ProfileForm.propTypes = {

}

export { ProfileForm };