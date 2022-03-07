import { Box, CircularProgress, Grid, TextField } from "@mui/material"
import { useRoute } from "wouter";
import { APP_LINKS } from "@local/shared";
import { useMutation, useQuery } from "@apollo/client";
import { project } from "graphql/generated/project";
import { projectQuery } from "graphql/query";
import { useCallback, useMemo, useState } from "react";
import { ProjectUpdateProps } from "../types";
import { mutationWrapper } from 'graphql/utils/wrappers';
import { projectUpdate as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { projectUpdateMutation } from "graphql/mutation";
import { formatForUpdate, getTranslation } from "utils";
import {
    Restore as CancelIcon,
    Save as SaveIcon,
} from '@mui/icons-material';
import { DialogActionItem } from "components/containers/types";
import { TagSelectorTag } from "components/inputs/types";
import { TagSelector, UserOrganizationSwitch } from "components";
import { DialogActionsContainer } from "components/containers/DialogActionsContainer/DialogActionsContainer";
import { Organization } from "types";

export const ProjectUpdate = ({
    session,
    onUpdated,
    onCancel,
}: ProjectUpdateProps) => {
    // Get URL params
    const [, params] = useRoute(`${APP_LINKS.Project}/:id`);
    const [, params2] = useRoute(`${APP_LINKS.SearchProjects}/edit/:id`);
    const id: string = params?.id ?? params2?.id ?? '';
    // Fetch existing data
    const { data, loading } = useQuery<project>(projectQuery, { variables: { input: { id } } });
    const project = useMemo(() => data?.project, [data]);

    const { description, name } = useMemo(() => {
        const languages = session?.languages ?? navigator.languages;
        return {
            description: getTranslation(project, 'description', languages, false) ?? '',
            name: getTranslation(project, 'name', languages, false) ?? '',
        }
    }, [project, session]);

    // Handle user/organization switch
    const [organizationFor, setOrganizationFor] = useState<Organization | null>(null);
    const onSwitchChange = useCallback((organization: Organization | null) => { setOrganizationFor(organization) }, [setOrganizationFor]);

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
    const [mutation] = useMutation<project>(projectUpdateMutation);
    const formik = useFormik({
        initialValues: {
            description,
            name,
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation,
                input: formatForUpdate(project, { id, ...values, tags }), //TODO handle translations
                onSuccess: (response) => { onUpdated(response.data.projectUpdate) },
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
            <UserOrganizationSwitch session={session} selected={organizationFor} onChange={onSwitchChange} />
        </Grid>
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