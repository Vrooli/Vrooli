import { Grid, TextField } from "@mui/material"
import { useRoute } from "wouter";
import { APP_LINKS } from "@local/shared";
import { useMutation, useQuery } from "@apollo/client";
import { standard } from "graphql/generated/standard";
import { standardQuery } from "graphql/query";
import { useMemo } from "react";
import { StandardUpdateProps } from "../types";
import { mutationWrapper } from 'graphql/utils/wrappers';
import PubSub from 'pubsub-js';
import { standardUpdate as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { standardUpdateMutation } from "graphql/mutation";
import { formatForUpdate, Pubs } from "utils";

export const StandardUpdate = ({
    onUpdated
}: StandardUpdateProps) => {
    // Get URL params
    const [, params] = useRoute(`${APP_LINKS.Standard}/:id`);
    const [, params2] = useRoute(`${APP_LINKS.SearchStandards}/edit/:id`);
    const id: string = params?.id ?? params2?.id ?? '';
    // Fetch existing data
    const { data, loading } = useQuery<standard>(standardQuery, { variables: { input: { id } } });
    const standard = useMemo(() => data?.standard, [data]);

    // Handle update
    const [mutation] = useMutation<standard>(standardUpdateMutation);
    const formik = useFormik({
        initialValues: {
            description: '',
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation,
                input: formatForUpdate(standard, values),
                onSuccess: (response) => { onUpdated(response.data.standardUpdate) },
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