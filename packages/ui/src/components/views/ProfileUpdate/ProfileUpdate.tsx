import { Grid, TextField } from "@mui/material"
import { useRoute } from "wouter";
import { APP_LINKS } from "@local/shared";
import { useMutation, useQuery } from "@apollo/client";
import { user } from "graphql/generated/user";
import { userQuery } from "graphql/query";
import { useMemo } from "react";
import { ProfileUpdateProps } from "../types";
import { mutationWrapper } from 'graphql/utils/wrappers';
import PubSub from 'pubsub-js';
import { profileSchema as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { profileUpdateMutation } from "graphql/mutation";
import { formatForUpdate, Pubs } from "utils";

export const ProfileUpdate = ({
    id,
    onUpdated
}: ProfileUpdateProps) => {
    // Get URL params
    const [, params] = useRoute(`${APP_LINKS.Profile}/:id/edit`);
    // Fetch existing data
    const { data, loading } = useQuery<user>(userQuery, { variables: { input: { id: params?.id ?? id ?? '' } } });
    const profile = useMemo(() => data?.user, [data]);

    // Handle update
    const [mutation] = useMutation<user>(profileUpdateMutation);
    const formik = useFormik({
        initialValues: {
            bio: '',
            username: '',
            theme: '',
            currentPassword: '',
            newPassword: '',
            newPasswordConfirmation: '',
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation,
                input: formatForUpdate(profile, values),
                onSuccess: (response) => { onUpdated(response.data.profileUpdate) },
                onError: (response) => {
                    PubSub.publish(Pubs.Snack, { message: 'Error occurred.', severity: 'error', data: { error: response } });
                }
            })
        },
    });

    return (
        <form onSubmit={formik.handleSubmit}>
        </form>
    )
}