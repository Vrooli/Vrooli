import { Grid, TextField } from "@mui/material"
import { useRoute } from "wouter";
import { APP_LINKS } from "@local/shared";
import { useMutation, useQuery } from "@apollo/client";
import { project } from "graphql/generated/project";
import { projectQuery } from "graphql/query";
import { useMemo } from "react";
import { ProjectUpdateProps } from "../types";
import { mutationWrapper } from 'graphql/utils/wrappers';
import PubSub from 'pubsub-js';
import { projectUpdate as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { projectUpdateMutation } from "graphql/mutation";
import { formatForUpdate, Pubs } from "utils";

export const ProjectUpdate = ({
    id,
    onUpdated
}: ProjectUpdateProps) => {
    // Get URL params
    const [, params] = useRoute(`${APP_LINKS.Project}/:id/edit`);
    // Fetch existing data
    const { data, loading } = useQuery<project>(projectQuery, { variables: { input: { id: params?.id ?? id ?? '' } } });
    const project = useMemo(() => data?.project, [data]);

    // Handle update
    const [mutation] = useMutation<project>(projectUpdateMutation);
    const formik = useFormik({
        initialValues: {
            description: '',
            name: '',
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation,
                input: formatForUpdate(project, values),
                onSuccess: (response) => { onUpdated(response.data.projectUpdate) },
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