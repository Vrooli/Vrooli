import { Grid, TextField } from "@mui/material";
import { useMutation } from "@apollo/client";
import { project } from "graphql/generated/project";
import { mutationWrapper } from 'graphql/utils/wrappers';
import PubSub from 'pubsub-js';
import { projectAdd as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { projectAddMutation } from "graphql/mutation";
import { formatForAdd, Pubs } from "utils";
import { ProjectAddProps } from "../types";

export const ProjectAdd = ({
    onAdded,
}: ProjectAddProps) => {

    // Handle add
    const [mutation] = useMutation<project>(projectAddMutation);
    const formik = useFormik({
        initialValues: {
            description: '',
            name: '',
        },
        validationSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation,
                input: formatForAdd(values),
                onSuccess: (response) => { onAdded(response.data.projectAdd) },
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