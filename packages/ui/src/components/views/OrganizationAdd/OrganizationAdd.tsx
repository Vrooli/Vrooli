import { Grid, TextField } from "@mui/material";
import { useMutation } from "@apollo/client";
import { organization } from "graphql/generated/organization";
import { mutationWrapper } from 'graphql/utils/wrappers';
import { organizationAdd as validationSchema, ROLES } from '@local/shared';
import { useFormik } from 'formik';
import { organizationAddMutation } from "graphql/mutation";
import { formatForAdd } from "utils";
import { OrganizationAddProps } from "../types";
import { useCallback, useMemo, useState } from "react";
import { TagSelector } from "components";
import { TagSelectorTag } from "components/inputs/types";
import {
    Add as AddIcon,
    Restore as CancelIcon,
} from '@mui/icons-material';
import { DialogActionItem } from "components/containers/types";
import { DialogActionsContainer } from "components/containers/DialogActionsContainer/DialogActionsContainer";

export const OrganizationAdd = ({
    session,
    onAdded,
    onCancel,
}: OrganizationAddProps) => {
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

    // Handle add
    const [mutation] = useMutation<organization>(organizationAddMutation);
    const formik = useFormik({
        initialValues: {
            name: '',
            bio: '',
        },
        validationSchema,
        onSubmit: (values) => {
            console.log('submitting', values);
            mutationWrapper({
                mutation,
                input: formatForAdd({ ...values, tags }),
                onSuccess: (response) => { onAdded(response.data.organizationAdd) },
            })
        },
    });

    const actions: DialogActionItem[] = useMemo(() => {
        const correctRole = Array.isArray(session?.roles) && session.roles.includes(ROLES.Actor);
        return [
            ['Add', AddIcon, Boolean(!correctRole || formik.isSubmitting || !formik.isValid), true, () => {}],
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