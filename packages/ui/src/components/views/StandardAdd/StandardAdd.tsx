import { Grid, TextField } from "@mui/material";
import { useMutation } from "@apollo/client";
import { standard } from "graphql/generated/standard";
import { mutationWrapper } from 'graphql/utils/wrappers';
import PubSub from 'pubsub-js';
import { standardAdd as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { standardAddMutation } from "graphql/mutation";
import { formatForAdd, Pubs } from "utils";
import { StandardAddProps } from "../types";

export const StandardAdd = ({
    onAdded,
}: StandardAddProps) => {

    // Handle add
    const [mutation] = useMutation<standard>(standardAddMutation);
    const formik = useFormik({
        initialValues: {
            default: '',
            description: '',
            name: '',
            schema: '',
            type: '',
            version: '',
        },
        validationSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation,
                input: formatForAdd(values),
                onSuccess: (response) => { onAdded(response.data.standardAdd) },
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