import { Grid, Grid, TextField } from "@mui/material"
import { useLocation, useRoute } from "wouter";
import { APP_LINKS } from "@local/shared";
import { useMutation, useQuery } from "@apollo/client";
import { organization } from "graphql/generated/organization";
import { organizationQuery } from "graphql/query";
import { useMemo } from "react";
import { OrganizationModifyProps } from "../types";
import { mutationWrapper } from 'graphql/utils/wrappers';
import PubSub from 'pubsub-js';
import { CODE, updateOrganizationSchema } from '@local/shared';
import { useFormik } from 'formik';

export const OrganizationModify = ({
    partialData,
}: OrganizationModifyProps) => {
    const [, setLocation] = useLocation();
    // Get URL params
    const [, params] = useRoute(`${APP_LINKS.Organization}/:id/edit`);
    // Fetch existing data
    const { data, loading } = useQuery<organization>(organizationQuery, { variables: { input: { id: params?.id ?? '' } } });
    const organization = useMemo(() => data?.organization, [data]);

    // Update organization
    const [updateOrganization] = useMutation<organization>(updateOrganizationMutation, { variables: { input: { id: params?.id ?? '' } } });
    const formik = useFormik({
        initialValues: {
            name: '',
            bio: '',
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
                        PubSub.publish(Pubs.AlertDialog, {
                            message: 'Before signing in, please follow the link sent to your email to change your password.',
                            firstButtonText: 'OK',
                            firstButtonClicked: () => setLocation(APP_LINKS.Home),
                        });
                    }
                }
            })
        },
    });

    return (
        <form onSubmit={formik.handleSubmit}>
            <Grid container spacing={2}>
                <Grid item xs={12}>
                    <TextField
                        fullWidth
                        id="name"
                        name="name"
                        autoComplete="organization-name"
                        label="Name"
                        value={formik.values.name}
                        onChange={formik.handleChange}
                        error={formik.touched.name && Boolean(formik.errors.name)}
                        helperText={formik.touched.name && formik.errors.name}
                    />
                </Grid>
                <Grid item xs={12}>
                    <TextField
                        fullWidth
                        id="bio"
                        name="bio"
                        label="Bio"
                        multiline
                        minRows={4}
                        value={formik.values.bio}
                        onChange={formik.handleChange}
                        error={formik.touched.bio && Boolean(formik.errors.bio)}
                        helperText={formik.touched.bio && formik.errors.bio}
                    />
                </Grid>
            </Grid>
        </form>
    )
}