import { Grid, TextField } from "@mui/material"
import { useRoute } from "wouter";
import { APP_LINKS } from "@local/shared";
import { useMutation, useQuery } from "@apollo/client";
import { organization } from "graphql/generated/organization";
import { organizationQuery } from "graphql/query";
import { useCallback, useMemo, useState } from "react";
import { OrganizationUpdateProps } from "../types";
import { mutationWrapper } from 'graphql/utils/wrappers';
import { organizationUpdate as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { organizationUpdateMutation } from "graphql/mutation";
import { formatForUpdate } from "utils";
import {
    Restore as CancelIcon,
    Save as SaveIcon,
} from '@mui/icons-material';
import { DialogActionItem } from "components/containers/types";
import { TagSelectorTag } from "components/inputs/types";
import { TagSelector } from "components";
import { DialogActionsContainer } from "components/containers/DialogActionsContainer/DialogActionsContainer";

export const OrganizationUpdate = ({
    session,
    onUpdated,
    onCancel,
}: OrganizationUpdateProps) => {
    // Get URL params
    const [, params] = useRoute(`${APP_LINKS.Organization}/:id`);
    const [, params2] = useRoute(`${APP_LINKS.SearchOrganizations}/edit/:id`);
    const id: string = params?.id ?? params2?.id ?? '';
    // Fetch existing data
    const { data, loading } = useQuery<organization>(organizationQuery, { variables: { input: { id } } });
    const organization = useMemo(() => data?.organization, [data]);

    // Handle tags
    const [tags, setTags] = useState<TagSelectorTag[]>([]);
    const addTag = useCallback((tag: TagSelectorTag) => {
        setTags(t => [...t, tag]);
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

    // Handle update
    const [mutation] = useMutation<organization>(organizationUpdateMutation);
    const formik = useFormik({
        initialValues: {
            name: organization?.name ?? '',
            bio: organization?.bio ?? '',
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation,
                input: formatForUpdate(organization, { ...values, tags }),
                onSuccess: (response) => { onUpdated(response.data.organizationUpdate) },
            })
        },
    });

    const actions: DialogActionItem[] = useMemo(() => [
        ['Save', SaveIcon, Boolean(formik.isSubmitting || !formik.isValid), true, () => { }],
        ['Cancel', CancelIcon, formik.isSubmitting, false, onCancel],
    ], [formik, onCancel, session]);
    const [formBottom, setFormBottom] = useState<number>(0);
    const handleResize = useCallback(({ height }: any) => {
        setFormBottom(height);
    }, [setFormBottom]);

    return (
        <form onSubmit={formik.handleSubmit} style={{ paddingBottom: `${formBottom}px` }}>
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
                <Grid item xs={12} marginBottom={4}>
                    <TagSelector
                        session={session}
                        tags={tags}
                        onTagAdd={addTag}
                        onTagRemove={removeTag}
                        onTagsClear={clearTags}
                    />
                </Grid>
            </Grid>
            <DialogActionsContainer actions={actions} onResize={handleResize} />
        </form>
    )
}