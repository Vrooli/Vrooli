import { Grid, TextField } from "@mui/material";
import { useMutation } from "@apollo/client";
import { standard } from "graphql/generated/standard";
import { mutationWrapper } from 'graphql/utils/wrappers';
import { ROLES, standardCreateForm as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { standardCreateMutation } from "graphql/mutation";
import { formatForCreate } from "utils";
import { StandardCreateProps } from "../types";
import { useCallback, useMemo, useState } from "react";
import { DialogActionItem } from "components/containers/types";
import {
    Add as CreateIcon,
    Restore as CancelIcon,
} from '@mui/icons-material';
import { TagSelectorTag } from "components/inputs/types";
import { ResourceListHorizontal, TagSelector } from "components";
import { DialogActionsContainer } from "components/containers/DialogActionsContainer/DialogActionsContainer";
import { ResourceList } from "types";
import { ResourceListUsedFor } from "graphql/generated/globalTypes";
import { v4 as uuidv4 } from 'uuid';

export const StandardCreate = ({
    session,
    onCreated,
    onCancel,
}: StandardCreateProps) => {

    // Handle resources
    const [resourceList, setResourceList] = useState<ResourceList>({ id: uuidv4(), usedFor: ResourceListUsedFor.Display } as any);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => {
        setResourceList(updatedList);
    }, [setResourceList]);

    // Handle tags
    const [tags, setTags] = useState<TagSelectorTag[]>([]);
    const addTag = useCallback((tag: TagSelectorTag) => {
        setTags(t => [...t, tag]);
    }, [setTags]);
    const removeTag = useCallback((tag: TagSelectorTag) => {
        setTags(tags => tags.filter(t => t.tag !== tag.tag));
    }, [setTags]);
    const clearTags = useCallback(() => {
        setTags([]);
    }, [setTags]);

    // Handle create
    const [mutation] = useMutation<standard>(standardCreateMutation);
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
            const resourceListAdd = resourceList ? formatForCreate(resourceList) : {};
            const tagsAdd = tags.length > 0 ? {
                tagsCreate: tags.filter(t => !t.id).map(t => ({ tag: t.tag })),
                tagsConnect: tags.filter(t => t.id).map(t => (t.id)),
            } : {};
            mutationWrapper({
                mutation,
                input: formatForCreate({
                    default: values.default,
                    description: values.description,
                    name: values.name,
                    schema: values.schema,
                    translations: [{
                        language: 'en',
                        description: values.description,
                    }],
                    resourceListsCreate: [resourceListAdd],
                    ...tagsAdd,
                    type: values.type,
                    version: values.version,
                }) as any,
                onSuccess: (response) => { onCreated(response.data.standardCreate) },
            })
        },
    });

    const actions: DialogActionItem[] = useMemo(() => {
        const correctRole = Array.isArray(session?.roles) && session.roles.includes(ROLES.Actor);
        return [
            ['Create', CreateIcon, Boolean(!correctRole || formik.isSubmitting), true, () => { }],
            ['Cancel', CancelIcon, formik.isSubmitting, false, onCancel],
        ] as DialogActionItem[]
    }, [formik, onCancel, session]);
    const [formBottom, setFormBottom] = useState<number>(0);
    const handleResize = useCallback(({ height }: any) => {
        setFormBottom(height);
    }, [setFormBottom]);

    return (
        <form onSubmit={formik.handleSubmit} style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            paddingBottom: `${formBottom}px`,
        }}
        >
            <Grid container spacing={2} sx={{ padding: 2, maxWidth: 'min(700px, 100%)' }}>
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
                <Grid item xs={12}>
                    <ResourceListHorizontal
                        title={'Resources'}
                        list={resourceList}
                        canEdit={true}
                        handleUpdate={handleResourcesUpdate}
                        session={session}
                        mutate={false}
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