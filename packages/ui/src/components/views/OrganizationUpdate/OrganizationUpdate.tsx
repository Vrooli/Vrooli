import { Grid, TextField } from "@mui/material"
import { useRoute } from "wouter";
import { APP_LINKS } from "@local/shared";
import { useMutation, useQuery } from "@apollo/client";
import { organization } from "graphql/generated/organization";
import { organizationQuery } from "graphql/query";
import { useMemo } from "react";
import { OrganizationUpdateProps } from "../types";
import { mutationWrapper } from 'graphql/utils/wrappers';
import PubSub from 'pubsub-js';
import { organizationUpdate as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { organizationUpdateMutation } from "graphql/mutation";
import { formatForUpdate, Pubs } from "utils";

export const OrganizationUpdate = ({
    onUpdated
}: OrganizationUpdateProps) => {
    // Get URL params
    const [, params] = useRoute(`${APP_LINKS.Organization}/:id`);
    const [, params2] = useRoute(`${APP_LINKS.SearchOrganizations}/edit/:id`);
    const id: string = params?.id ?? params2?.id ?? '';
    // Fetch existing data
    const { data, loading } = useQuery<organization>(organizationQuery, { variables: { input: { id } } });
    const organization = useMemo(() => data?.organization, [data]);

    // Handle update
    const [mutation] = useMutation<organization>(organizationUpdateMutation);
    const formik = useFormik({
        initialValues: {
            name: '',
            bio: '',
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation,
                input: formatForUpdate(organization, values),
                onSuccess: (response) => { onUpdated(response.data.organizationUpdate) },
                onError: (response) => {
                    PubSub.publish(Pubs.Snack, { message: 'Error occurred.', severity: 'error', data: { error: response } });
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
                        label="Name"
                        value={formik.values.name}
                        onBlur={formik.handleBlur}
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
                        onBlur={formik.handleBlur}
                        onChange={formik.handleChange}
                        error={formik.touched.bio && Boolean(formik.errors.bio)}
                        helperText={formik.touched.bio && formik.errors.bio}
                    />
                </Grid>
            </Grid>
        </form>
    )
}