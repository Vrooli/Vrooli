import { Button, Grid, TextField } from "@mui/material";
import { useMutation } from "@apollo/client";
import { project } from "graphql/generated/project";
import { mutationWrapper } from 'graphql/utils/wrappers';
import PubSub from 'pubsub-js';
import { projectAdd as validationSchema, ROLES } from '@local/shared';
import { useFormik } from 'formik';
import { projectAddMutation } from "graphql/mutation";
import { formatForAdd, Pubs } from "utils";
import { ProjectAddProps } from "../types";
import { useMemo } from "react";

export const ProjectAdd = ({
    session,
    onAdded,
    onCancel,
}: ProjectAddProps) => {
    const canAdd = useMemo(() => Array.isArray(session?.roles) && !session.roles.includes(ROLES.Actor), [session]);

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
            <Grid container spacing={2} sx={{ padding: 2 }}>
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