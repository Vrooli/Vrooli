import { useMutation } from '@apollo/client';
import { resourceCreateForm as validationSchema } from '@local/shared';
import { Box, Button, Dialog, FormControl, Grid, IconButton, InputLabel, MenuItem, Select, Stack, TextField, Typography } from '@mui/material';
import { HelpButton } from 'components/buttons';
import { useFormik } from 'formik';
import { resourceCreateMutation, resourceUpdateMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { ResourceDialogProps } from '../types';
import {
    Close as CloseIcon
} from '@mui/icons-material';
import { formatForCreate, getTranslation, Pubs } from 'utils';
import { resourceCreate } from 'graphql/generated/resourceCreate';
import { ResourceUsedFor } from 'graphql/generated/globalTypes';
import { resourceUpdate } from 'graphql/generated/resourceUpdate';

const helpText =
`## What are resources?

Resources provide context to the object they are attached to, such as a  user, organization, project, or routine.

## Examples
**For a user** - Social media links, GitHub profile, Patreon

**For an organization** - Official website, tools used by your team, news article explaining the vision

**For a project** - Project Catalyst proposal, Donation wallet address

**For a routine** - Guide, external service
`

const UsedForDisplay = {
    [ResourceUsedFor.Community]: 'Community',
    [ResourceUsedFor.Context]: 'Context',
    [ResourceUsedFor.Developer]: 'Developer',
    [ResourceUsedFor.Donation]: 'Donation',
    [ResourceUsedFor.ExternalService]: 'External Service',
    [ResourceUsedFor.Feed]: 'Feed',
    [ResourceUsedFor.Install]: 'Install',
    [ResourceUsedFor.Learning]: 'Learning',
    [ResourceUsedFor.Notes]: 'Notes',
    [ResourceUsedFor.OfficialWebsite]: 'Official Webiste',
    [ResourceUsedFor.Proposal]: 'Proposal',
    [ResourceUsedFor.Related]: 'Related',
    [ResourceUsedFor.Researching]: 'Researching',
    [ResourceUsedFor.Scheduling]: 'Scheduling',
    [ResourceUsedFor.Social]: 'Social',
    [ResourceUsedFor.Tutorial]: 'Tutorial',
}

export const ResourceDialog = ({
    isAdd,
    mutate,
    open,
    onClose,
    onCreated,
    onUpdated,
    index,
    partialData,
    title = 'Add Resource',
    listId,
}: ResourceDialogProps) => {

    const [addMutation, { loading: addLoading }] = useMutation<resourceCreate>(resourceCreateMutation);
    const [updateMutation, { loading: updateLoading }] = useMutation<resourceUpdate>(resourceUpdateMutation);

    const formik = useFormik({
        initialValues: {
            link: partialData?.link ?? '',
            usedFor: partialData?.usedFor ?? ResourceUsedFor.Context,
            title: getTranslation(partialData, 'title', ['en'], false) ?? '',
            description: getTranslation(partialData, 'description', ['en'], false) ?? '',
        },
        validationSchema,
        onSubmit: (values) => {
            const input = formatForCreate({
                listId,
                link: values.link,
                usedFor: values.usedFor,
                translations: [{
                    language: 'en',
                    title: values.title,
                    description: values.description,
                }],
            })
            if (mutate) {
                mutationWrapper({
                    mutation: isAdd ? addMutation : updateMutation,
                    input,
                    successCondition: (response) => response.data.resourceCreate !== null,
                    onSuccess: (response) => {
                        PubSub.publish(Pubs.Snack, { message: 'Resource created.' });
                        isAdd ? onCreated(response.data.resourceCreate) : onUpdated(index ?? 0, response.data.resourceUpdate);
                        onClose();
                    },
                })
            } else {
                onCreated(input as any);
                onClose();
            }
        },
    });

    return (
        <Dialog
            onClose={onClose}
            open={open}
            sx={{
                '& .MuiDialog-paper': {
                    width: 'min(500px, 100vw)',
                    textAlign: 'center',
                    overflow: 'hidden',
                }
            }}
        >
            <form onSubmit={formik.handleSubmit}>
                <Box sx={{
                    padding: 1,
                    background: (t) => t.palette.primary.dark,
                    color: (t) => t.palette.primary.contrastText,
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'space-between',
                }}>
                    <Typography component="h2" variant="h4" textAlign="center" sx={{ marginLeft: 'auto' }}>
                        {title}
                        <HelpButton markdown={helpText} sx={{ fill: '#a0e7c4' }} />
                    </Typography>
                    <Box sx={{ marginLeft: 'auto' }}>
                        <IconButton
                            edge="start"
                            onClick={onClose}
                        >
                            <CloseIcon sx={{ fill: (t) => t.palette.primary.contrastText }} />
                        </IconButton>
                    </Box>
                </Box>
                <Stack direction="column" spacing={2} sx={{ padding: 2 }}>
                    {/* Enter link */}
                    <TextField
                        fullWidth
                        id="link"
                        name="link"
                        label="Link"
                        value={formik.values.link}
                        onChange={formik.handleChange}
                        error={formik.touched.link && Boolean(formik.errors.link)}
                        helperText={(formik.touched.link && formik.errors.link) ?? "Enter URL of resource"}
                    />
                    {/* Select resource type */}
                    <FormControl fullWidth>
                        <InputLabel id="rsource-type-label">Reason</InputLabel>
                        <Select
                            labelId="resource-type-label"
                            id="usedFor"
                            value={formik.values.usedFor}
                            label="Resource type"
                            onChange={(e) => formik.setFieldValue('usedFor', e.target.value)}
                        >
                            {Object.entries(UsedForDisplay).map(([key, value]) => (
                                <MenuItem key={key} value={key}>{value}</MenuItem>
                            ))}
                        </Select>
                    </FormControl>
                    {/* Enter title */}
                    <TextField
                        fullWidth
                        id="title"
                        name="title"
                        label="Title"
                        value={formik.values.title}
                        onChange={formik.handleChange}
                        error={formik.touched.title && Boolean(formik.errors.title)}
                        helperText={(formik.touched.title && formik.errors.title) ?? "Enter title (optional)"}
                    />
                    {/* Enter description */}
                    <TextField
                        fullWidth
                        id="description"
                        name="description"
                        label="Description"
                        value={formik.values.description}
                        onChange={formik.handleChange}
                        error={formik.touched.description && Boolean(formik.errors.description)}
                        helperText={(formik.touched.description && formik.errors.description) ?? "Enter description (optional)"}
                    />
                    {/* Action buttons */}
                    <Grid container sx={{ padding: 0 }}>
                        <Grid item xs={12} sm={6} sx={{ paddingRight: 1 }}>
                            <Button fullWidth type="submit">Submit</Button>
                        </Grid>
                        <Grid item xs={12} sm={6}>
                            <Button fullWidth onClick={onClose} sx={{ paddingLeft: 1 }}>Cancel</Button>
                        </Grid>
                    </Grid>
                </Stack>
            </form>
        </Dialog>
    )
}