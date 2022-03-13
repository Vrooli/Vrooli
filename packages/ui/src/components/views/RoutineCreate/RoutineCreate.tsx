import { Grid, TextField } from "@mui/material";
import { useMutation } from "@apollo/client";
import { routine } from "graphql/generated/routine";
import { mutationWrapper } from 'graphql/utils/wrappers';
import PubSub from 'pubsub-js';
import { ROLES, routineCreate as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { routineCreateMutation } from "graphql/mutation";
import { formatForCreate, Pubs } from "utils";
import { RoutineCreateProps } from "../types";
import { useCallback, useMemo, useState } from "react";
import { DialogActionItem } from "components/containers/types";
import {
    Add as CreateIcon,
    Restore as CancelIcon,
} from '@mui/icons-material';
import { TagSelectorTag } from "components/inputs/types";
import { MarkdownInput, TagSelector } from "components";
import { DialogActionsContainer } from "components/containers/DialogActionsContainer/DialogActionsContainer";

export const RoutineCreate = ({
    session,
    onCreated,
    onCancel,
}: RoutineCreateProps) => {
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

    // Handle create
    const [mutation] = useMutation<routine>(routineCreateMutation);
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
                input: formatForCreate({ ...values, tags }),
                onSuccess: (response) => { onCreated(response.data.routineCreate) },
                onError: (response) => {
                    PubSub.publish(Pubs.Snack, { message: 'Error occurred.', severity: 'error', data: { error: response } });
                }
            })
        },
    });

    const actions: DialogActionItem[] = useMemo(() => {
        const correctRole = Array.isArray(session?.roles) && session.roles.includes(ROLES.Actor);
        return [
            ['Create', CreateIcon, Boolean(!correctRole || formik.isSubmitting || !formik.isValid), true, () => { }],
            ['Cancel', CancelIcon, formik.isSubmitting, false, onCancel],
        ] as DialogActionItem[]
    }, [formik, onCancel, session]);
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
                        helperText={formik.touched.instructions ? formik.errors.instructions : null}
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