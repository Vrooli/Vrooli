import { Grid, TextField } from "@mui/material";
import { useMutation } from "@apollo/client";
import { routine } from "graphql/generated/routine";
import { mutationWrapper } from 'graphql/utils/wrappers';
import PubSub from 'pubsub-js';
import { routineAdd as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { routineAddMutation } from "graphql/mutation";
import { formatForAdd, Pubs } from "utils";
import { RoutineAddProps } from "../types";

export const RoutineAdd = ({
    onAdded,
}: RoutineAddProps) => {

    // Handle add
    const [mutation] = useMutation<routine>(routineAddMutation);
    const formik = useFormik({
        initialValues: {
            description: '',
            instructions: '',
            title: '',
            version: ''
        },
        validationSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation,
                input: formatForAdd(values),
                onSuccess: (response) => { onAdded(response.data.routineAdd) },
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