import { Button, Grid, TextField } from "@mui/material";
import { useMutation } from "@apollo/client";
import { routine } from "graphql/generated/routine";
import { mutationWrapper } from 'graphql/utils/wrappers';
import PubSub from 'pubsub-js';
import { ROLES, routineAdd as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { routineAddMutation } from "graphql/mutation";
import { formatForAdd, Pubs } from "utils";
import { RoutineAddProps } from "../types";
import { useMemo } from "react";

export const RoutineAdd = ({
    session,
    onAdded,
    onCancel,
}: RoutineAddProps) => {
    const canAdd = useMemo(() => Array.isArray(session?.roles) && !session.roles.includes(ROLES.Actor), [session]);

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
            <Grid container spacing={2} sx={{ padding: 2 }}>
                <Grid item xs={12}>
                    <TextField
                        fullWidth
                        id="title"
                        name="title"
                        label="title"
                        value={formik.values.title}
                        onBlur={formik.handleBlur}
                        onChange={formik.handleChange}
                        error={formik.touched.title && Boolean(formik.errors.title)}
                        helperText={formik.touched.title && formik.errors.title}
                    />
                </Grid>
                <Grid item xs={12}>
                    <TextField
                        fullWidth
                        id="description"
                        name="description"
                        label="description"
                        multiline
                        minRows={4}
                        value={formik.values.description}
                        onBlur={formik.handleBlur}
                        onChange={formik.handleChange}
                        error={formik.touched.description && Boolean(formik.errors.description)}
                        helperText={formik.touched.description && formik.errors.description}
                    />
                </Grid>
                <Grid item xs={12} sm={6}>
                    <Button
                        fullWidth
                        color="secondary"
                        type="submit"
                        disabled={!canAdd}
                    >Add</Button>
                </Grid>
                <Grid item xs={12} sm={6}>
                    <Button
                        fullWidth
                        color="secondary"
                        disabled={!canAdd}
                        onClick={onCancel}
                    >Cancel</Button>
                </Grid>
            </Grid>
        </form>
    )
}