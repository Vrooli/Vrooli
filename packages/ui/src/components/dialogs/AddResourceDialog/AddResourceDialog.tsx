import { useMutation } from '@apollo/client';
import { reportCreate as validationSchema } from '@local/shared';
import { Box, Button, Dialog, FormControl, Grid, IconButton, InputLabel, MenuItem, Select, Stack, TextField, Typography } from '@mui/material';
import { HelpButton } from 'components/buttons';
import { useFormik } from 'formik';
import { resourceCreateMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { AddResourceDialogProps } from '../types';
import {
    Close as CloseIcon
} from '@mui/icons-material';
import { Pubs } from 'utils';
import { resourceCreate } from 'graphql/generated/resourceCreate';
import { ResourceUsedFor } from 'graphql/generated/globalTypes';

const helpText =
`# Test
this is a test`

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

export const AddResourceDialog = ({
    open,
    onClose,
    onCreated,
    title = 'Add Resource',
    listId,
}: AddResourceDialogProps) => {

    const [mutation, { loading }] = useMutation<resourceCreate>(resourceCreateMutation);
    const formik = useFormik({
        initialValues: {
            link: '',
            usedFor: ResourceUsedFor.Context,
            title: '',
            description: '',
        },
        validationSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation,
                input: { 
                    listId,
                    link: values.link,
                    translations: [{
                        language: 'en',
                        title: values.title,
                        description: values.description,
                    }],
                },
                successCondition: (response) => response.data.resourceCreate !== null,
                onSuccess: (response) => {
                    PubSub.publish(Pubs.Snack, { message: 'Resource created.' });
                    onCreated(response.data.resourceCreate);
                    onClose();
                },
            })
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
                    </Typography>
                    <Box sx={{ marginLeft: 'auto' }}>
                        <HelpButton markdown={helpText} sx={{ fill: '#a0e7c4' }} />
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
                        helperText="Enter URL of resource"
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
                        helperText="Enter title (optional)"
                    />
                    {/* Enter description */}
                    <TextField
                        fullWidth
                        id="description"
                        name="description"
                        label="Description"
                        value={formik.values.description}
                        onChange={formik.handleChange}
                        helperText="Enter description (optional)"
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