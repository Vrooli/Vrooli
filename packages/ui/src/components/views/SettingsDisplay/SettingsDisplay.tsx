import { Box, Button, Grid, Stack, Typography, useTheme } from "@mui/material"
import { useMutation } from "@apollo/client";
import { useCallback, useEffect, useState } from "react";
import { mutationWrapper } from 'graphql/utils/graphqlWrapper';
import { profileUpdateSchema as validationSchema } from '@shared/validation';
import { useFormik } from 'formik';
import { profileUpdateMutation } from "graphql/mutation";
import { clearSearchHistory, PubSub, shapeProfileUpdate, TagHiddenShape, TagShape, usePromptBeforeUnload } from "utils";
import { SettingsDisplayProps } from "../types";
import { GridSubmitButtons, HelpButton, SnackSeverity, TagSelector } from "components";
import { ThemeSwitch } from "components/inputs";
import { profileUpdateVariables, profileUpdate_profileUpdate } from "graphql/generated/profileUpdate";
import { uuid } from '@shared/uuid';
import { HeartFilledIcon, InvisibleIcon, SearchIcon } from "@shared/icons";
import { getCurrentUser } from "utils/authentication";

const helpText =
    `Display preferences customize the look and feel of Vrooli. More customizations will be available in the near future.`

const interestsHelpText =
    `Specifying your interests can simplify the discovery of routines, projects, organizations, and standards, via customized feeds.

**None** of this information is available to the public, and **none** of it is sold to advertisers.
`

const hiddenHelpText =
    `Specify tags which should be hidden from your feeds.

**None** of this information is available to the public, and **none** of it is sold to advertisers.
`

export const SettingsDisplay = ({
    session,
    profile,
    onUpdated,
}: SettingsDisplayProps) => {
    const { palette } = useTheme();

    // Handle starred tags
    const [starredTags, setStarredTags] = useState<TagShape[]>([]);
    const handleStarredTagsUpdate = useCallback((updatedList: TagShape[]) => { setStarredTags(updatedList); }, [setStarredTags]);

    // Handle hidden tags
    const [hiddenTags, setHiddenTags] = useState<TagHiddenShape[]>([]);
    const handleHiddenTagsUpdate = useCallback((updatedList: TagShape[]) => { 
        // Hidden tags are wrapped in a shape that includes an isBlur flag. 
        // Because of this, we must loop through the updatedList to see which tags have been added or removed.
        const updatedHiddenTags = updatedList.map((tag) => {
            const existingTag = hiddenTags.find((hiddenTag) => hiddenTag.tag.id === tag.id);
            if (existingTag) {
                return existingTag;
            }
            return { 
                id: uuid(),
                isBlur: true,
                tag,
            };
        });
        setHiddenTags(updatedHiddenTags);
    }, [hiddenTags]);

    useEffect(() => {
        if (profile?.starredTags) {
            setStarredTags(profile.starredTags);
        }
        if (profile?.hiddenTags) {
            setHiddenTags(profile.hiddenTags);
        }
    }, [profile]);

    // Handle update
    const [mutation] = useMutation(profileUpdateMutation);
    const formik = useFormik({
        initialValues: {
            theme: getCurrentUser(session).theme ?? 'light',
        },
        validationSchema,
        onSubmit: (values) => {
            if (!profile) {
                PubSub.get().publishSnack({ message: 'Could not find existing data.', severity: SnackSeverity.Error });
                return;
            }
            if (!formik.isValid) return;
            // If any tags are in both starredTags and hiddenTags, remove them from hidden. Also give warning to user.
            const filteredHiddenTags = hiddenTags.filter(t => !starredTags.some(st => st.tag === t.tag.tag));
            if (filteredHiddenTags.length !== hiddenTags.length) {
                PubSub.get().publishSnack({ message: 'Found topics in both favorites and hidden. These have been removed from hidden.', severity: SnackSeverity.Warning });
            }
            const input = shapeProfileUpdate(profile, {
                id: profile.id,
                theme: values.theme as 'light' | 'dark',
                starredTags,
                hiddenTags: filteredHiddenTags,
            })
            if (!input || Object.keys(input).length === 0) {
                PubSub.get().publishSnack({ message: 'No changes made.', severity: SnackSeverity.Error });
                formik.setSubmitting(false);
                return;
            }
            mutationWrapper<profileUpdate_profileUpdate, profileUpdateVariables>({
                mutation,
                input,
                successMessage: () => 'Display preferences updated.',
                onSuccess: (data) => {
                    onUpdated(data);
                    formik.setSubmitting(false);
                },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    const handleSave = useCallback(() => {
        formik.submitForm();
    }, [formik]);

    const handleCancel = useCallback(() => {
        formik.resetForm();
        setStarredTags([]);
        setHiddenTags([]);
    }, [formik]);

    return (
        <form onSubmit={formik.handleSubmit} style={{ overflow: 'hidden' }}>
            {/* Title */}
            <Stack direction="row" justifyContent="center" alignItems="center" sx={{
                background: palette.primary.dark,
                color: palette.primary.contrastText,
                padding: 0.5,
                marginBottom: 2,
            }}>
                <Typography component="h1" variant="h4">Display Preferences</Typography>
                <HelpButton markdown={helpText} />
            </Stack>
            <Box sx={{ margin: 2, marginBottom: 5 }}>
                <ThemeSwitch
                    theme={formik.values.theme as 'light' | 'dark'}
                    onChange={(t) => formik.setFieldValue('theme', t)}
                />
            </Box>
            <Stack direction="row" marginRight="auto" alignItems="center" justifyContent="center">
                <HeartFilledIcon fill={palette.background.textPrimary} />
                <Typography component="h2" variant="h5" textAlign="center" ml={1}>Favorite Topics</Typography>
                <HelpButton markdown={interestsHelpText} />
            </Stack>
            <Box sx={{ margin: 2, marginBottom: 5 }}>
                <TagSelector
                    handleTagsUpdate={handleStarredTagsUpdate}
                    session={session}
                    tags={starredTags}
                    placeholder={"Enter interests, followed by commas..."}
                />
            </Box>
            <Stack direction="row" marginRight="auto" alignItems="center" justifyContent="center">
                <InvisibleIcon fill={palette.background.textPrimary} />
                <Typography component="h2" variant="h5" textAlign="center" ml={1}>Hidden Topics</Typography>
                <HelpButton markdown={hiddenHelpText} />
            </Stack>
            <Box sx={{ margin: 2, marginBottom: 5 }}>
                <TagSelector
                    handleTagsUpdate={handleHiddenTagsUpdate}
                    session={session}
                    tags={hiddenTags.map(t => t.tag)}
                    placeholder={"Enter topics you'd like to hide from view, followed by commas..."}
                />
            </Box>
            <Box sx={{ margin: 2, marginBottom: 5, display: 'flex' }}>
                <Button color="secondary" startIcon={<SearchIcon />} onClick={() => { clearSearchHistory(session) }} sx={{
                    marginLeft: 'auto',
                    marginRight: 'auto',
                }}>Clear Search History</Button>
            </Box>
            <Grid container spacing={2} p={2}>
                <GridSubmitButtons
                    errors={formik.errors}
                    isCreate={false}
                    loading={formik.isSubmitting}
                    onCancel={handleCancel}
                    onSetSubmitting={formik.setSubmitting}
                    onSubmit={handleSave}
                />
            </Grid>
        </form>
    )
}