import { Button, Grid, TextField } from "@mui/material";
import { useMutation } from "@apollo/client";
import { organization } from "graphql/generated/organization";
import { mutationWrapper } from 'graphql/utils/wrappers';
import PubSub from 'pubsub-js';
import { organizationAdd as validationSchema, ROLES } from '@local/shared';
import { useFormik } from 'formik';
import { organizationAddMutation } from "graphql/mutation";
import { formatForAdd, Pubs } from "utils";
import { OrganizationAddProps } from "../types";
import { useCallback, useMemo, useState } from "react";
import { TagSelector } from "components";
import { Tag } from "types";
import { TagSelectorTag } from "components/inputs/types";

export const OrganizationAdd = ({
    session,
    onAdded,
    onCancel,
}: OrganizationAddProps) => {
    const canAdd = useMemo(() => Array.isArray(session?.roles) && session.roles.includes(ROLES.Actor), [session]);

    // Handle add
    const [mutation] = useMutation<organization>(organizationAddMutation);
    const formik = useFormik({
        initialValues: {
            name: '',
            bio: '',
        },
        validationSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation,
                input: formatForAdd(values),
                onSuccess: (response) => { onAdded(response.data.organizationAdd) },
                onError: (response) => {
                    PubSub.publish(Pubs.Snack, { message: 'Error occurred.', severity: 'error', data: { error: response } });
                }
            })
        },
    });

    // Handle tags
    const [tags, setTags] = useState<TagSelectorTag[]>([]);
    const addTag = useCallback((tag: TagSelectorTag) => {
        setTags(tags => [...tags, tag]);
    }, [setTags]);
    const removeTag = useCallback((tag: TagSelectorTag) => {
        console.log('removeTag', tag);
        const temp = tags.filter(t => t.tag !== tag.tag);
        console.log('temp', tags.length, temp.length);
        setTags(tags => tags.filter(t => t.tag !== tag.tag));
    }, [setTags]);
    const clearTags = useCallback(() => {
        setTags([]);
    }, [setTags]);

    console.log('formik', formik.values, formik.errors, formik.touched);

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
                <Grid item xs={12}>
                    <TagSelector
                        session={session}
                        tags={tags}
                        onTagAdd={addTag}
                        onTagRemove={removeTag}
                        onTagsClear={clearTags}
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