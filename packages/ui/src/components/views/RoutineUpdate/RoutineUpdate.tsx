import { Box, CircularProgress, Grid, TextField } from "@mui/material"
import { useRoute } from "wouter";
import { APP_LINKS } from "@local/shared";
import { useMutation, useQuery } from "@apollo/client";
import { routine } from "graphql/generated/routine";
import { routineQuery } from "graphql/query";
import { useCallback, useMemo, useState } from "react";
import { RoutineUpdateProps } from "../types";
import { mutationWrapper } from 'graphql/utils/wrappers';
import PubSub from 'pubsub-js';
import { routineUpdateForm as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { routineUpdateMutation } from "graphql/mutation";
import { formatForUpdate, getTranslation, Pubs, updateTranslation } from "utils";
import {
    Restore as CancelIcon,
    Save as SaveIcon,
} from '@mui/icons-material';
import { TagSelectorTag } from "components/inputs/types";
import { DialogActionItem } from "components/containers/types";
import { MarkdownInput, TagSelector } from "components";
import { DialogActionsContainer } from "components/containers/DialogActionsContainer/DialogActionsContainer";

export const RoutineUpdate = ({
    session,
    onUpdated,
    onCancel,
}: RoutineUpdateProps) => {
    // Get URL params
    const [, params] = useRoute(`${APP_LINKS.Run}/:id`);
    const [, params2] = useRoute(`${APP_LINKS.SearchRoutines}/edit/:id`);
    const id: string = params?.id ?? params2?.id ?? '';
    // Fetch existing data
    const { data, loading } = useQuery<routine>(routineQuery, { variables: { input: { id } } });
    const routine = useMemo(() => data?.routine, [data]);

    const { title, description, instructions } = useMemo(() => {
        const languages = session?.languages ?? navigator.languages;
        return {
            title: getTranslation(routine, 'title', languages) ?? '',
            description: getTranslation(routine, 'description', languages) ?? '',
            instructions: getTranslation(routine, 'instructions', languages) ?? '',
        };
    }, [routine, session]);

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
    const [mutation] = useMutation<routine>(routineUpdateMutation);
    const formik = useFormik({
        initialValues: {
            description,
            instructions,
            title,
            version: routine?.version ?? 0,
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation,
                input: formatForUpdate(routine, { 
                    id, 
                    version: values.version,
                    //TODO tags
                    translations: updateTranslation(routine as any, { language: 'en', description: values.description, instructions: values.instructions, title: values.title }),
                }),
                onSuccess: (response) => { onUpdated(response.data.routineUpdate) },
                onError: (response) => {
                    PubSub.publish(Pubs.Snack, { message: 'Error occurred.', severity: 'error', data: { error: response } });
                }
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
                    value={formik.values.description}
                    rows={3}
                    onBlur={formik.handleBlur}
                    onChange={formik.handleChange}
                    error={formik.touched.description && Boolean(formik.errors.description)}
                    helperText={formik.touched.description && formik.errors.description}
                />
            </Grid>
            <Grid item xs={12}>
                <MarkdownInput
                    id="instructions"
                    placeholder="Instructions"
                    value={formik.values.instructions}
                    minRows={4}
                    onChange={(newText: string) => formik.setFieldValue('instructions', newText)}
                    error={formik.touched.instructions && Boolean(formik.errors.instructions)}
                    helperText={formik.touched.instructions ? formik.errors.instructions as string : null}
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