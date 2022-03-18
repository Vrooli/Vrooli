import { Box, CircularProgress, Grid, TextField } from "@mui/material"
import { useRoute } from "wouter";
import { APP_LINKS } from "@local/shared";
import { useMutation, useQuery } from "@apollo/client";
import { organization } from "graphql/generated/organization";
import { organizationQuery } from "graphql/query";
import { useCallback, useEffect, useMemo, useState } from "react";
import { OrganizationUpdateProps } from "../types";
import { mutationWrapper } from 'graphql/utils/wrappers';
import { organizationUpdateForm as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { organizationUpdateMutation } from "graphql/mutation";
import { formatForUpdate, getTranslation, updateTranslation } from "utils";
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

    const { bio, name } = useMemo(() => {
        const languages = session?.languages ?? navigator.languages;
        return {
            bio: getTranslation(organization, 'bio', languages, false) ?? '',
            name: getTranslation(organization, 'name', languages, false) ?? '',
        }
    }, [organization, session]);

    // Handle tags
    const [tags, setTags] = useState<TagSelectorTag[]>([]);
    useEffect(() => {
        setTags(organization?.tags ?? []);
    }, [organization]);
    const addTag = useCallback((tag: TagSelectorTag) => {
        setTags(t => [...t, tag]);
    }, [setTags]);
    const removeTag = useCallback((tag: TagSelectorTag) => {
        const temp = tags.filter(t => t.tag !== tag.tag);
        setTags(tags => tags.filter(t => t.tag !== tag.tag));
    }, [setTags]);
    const clearTags = useCallback(() => {
        setTags([]);
    }, [setTags]);

    // Handle update
    const [mutation] = useMutation<organization>(organizationUpdateMutation);
    const formik = useFormik({
        initialValues: {
            name,
            bio,
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            const tagsAdd = tags.length > 0 ? {
                // Create/connect new tags
                tagsCreate: tags.filter(t => !t.id && !organization?.tags?.some(tag => tag.tag === t.tag)).map(t => ({ tag: t.tag })),
                tagsConnect: tags.filter(t => t.id && !organization?.tags?.some(tag => tag.tag === t.tag)).map(t => (t.id)),
            } : {};
            mutationWrapper({
                mutation,
                input: formatForUpdate(organization, { 
                    id, 
                    ...tagsAdd,
                    translations: updateTranslation(organization as any, { language: 'en', name: values.name, bio: values.bio })
                }, ['tags'], ['translations']),
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

    const formInput = useMemo(() => (
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
    ), [formik, actions, handleResize, formBottom, session, tags, addTag, removeTag, clearTags]);

    return (
        <form onSubmit={formik.handleSubmit} style={{ paddingBottom: `${formBottom}px` }}>
            {loading ? (
                <Box sx={{
                    position: 'absolute',
                    top: '-5vh', // Half of toolbar height
                    width: '100%',
                    height: '100%',
                    display: 'flex',
                    justifyContent: 'center',
                    alignItems: 'center',
                }}>
                    <CircularProgress size={100} color="secondary" />
                </Box>
            ) : formInput}
            <DialogActionsContainer actions={actions} onResize={handleResize} />
        </form>
    )
}