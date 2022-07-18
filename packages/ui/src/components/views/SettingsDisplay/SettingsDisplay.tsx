import { Box, Stack, Typography, useTheme } from "@mui/material"
import { useMutation } from "@apollo/client";
import { useCallback, useEffect, useMemo, useState } from "react";
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { profileUpdateSchema as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { profileUpdateMutation } from "graphql/mutation";
import { PubSub, shapeProfileUpdate, TagHiddenShape, TagShape, TERTIARY_COLOR } from "utils";
import {
    Restore as RevertIcon,
    Save as SaveIcon,
    ThumbUp as InterestsIcon,
    VisibilityOff as HiddenIcon,
} from '@mui/icons-material';
import { DialogActionItem } from "components/containers/types";
import { DialogActionsContainer } from "components/containers/DialogActionsContainer/DialogActionsContainer";
import { SettingsDisplayProps } from "../types";
import { HelpButton, TagSelector } from "components";
import { ThemeSwitch } from "components/inputs";
import { profileUpdate, profileUpdateVariables } from "graphql/generated/profileUpdate";
import { v4 as uuid } from 'uuid';

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
    const addStarredTag = useCallback((tag: TagShape) => {
        setStarredTags(t => [...t, tag]);
    }, [setStarredTags]);
    const removeStarredTag = useCallback((tag: TagShape) => {
        setStarredTags(tags => tags.filter(t => t.tag !== tag.tag));
    }, [setStarredTags]);
    const clearStarredTags = useCallback(() => {
        setStarredTags([]);
    }, [setStarredTags]);

    // Handle hidden tags
    const [hiddenTags, setHiddenTags] = useState<TagHiddenShape[]>([]);
    const addHiddenTag = useCallback((tag: TagShape) => {
        setHiddenTags(t => [...t, { 
            id: uuid(),
            isBlur: true, 
            tag 
        }]);
    }, [setHiddenTags]);
    const removeHiddenTag = useCallback((tag: TagShape) => {
        setHiddenTags(ht => ht.filter(t => t.tag.tag !== tag.tag));
    }, [setHiddenTags]);
    const clearHiddenTags = useCallback(() => {
        setHiddenTags([]);
    }, [setHiddenTags]);

    // Handle theme
    const [theme, setTheme] = useState<string>('light');
    useEffect(() => {
        setTheme(palette.mode);
    }, [palette.mode]);

    useEffect(() => {
        if (profile?.starredTags) {
            setStarredTags(profile.starredTags);
        }
        if (profile?.hiddenTags) {
            setHiddenTags(profile.hiddenTags);
        }
    }, [profile]);

    // Handle update
    const [mutation] = useMutation<profileUpdate, profileUpdateVariables>(profileUpdateMutation);
    const formik = useFormik({
        initialValues: {
            theme,
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            if (!profile) {
                PubSub.get().publishSnack({ message: 'Could not find existing data.', severity: 'error' });
                return;
            }
            if (!formik.isValid) return;
            // If any tags are in both starredTags and hiddenTags, remove them from hidden. Also give warning to user.
            const filteredHiddenTags = hiddenTags.filter(t => !starredTags.some(st => st.tag === t.tag.tag));
            if (filteredHiddenTags.length !== hiddenTags.length) {
                PubSub.get().publishSnack({ message: 'Found topics in both favorites and hidden. These have been removed from hidden.', severity: 'warning' });
            }
            const input = shapeProfileUpdate(profile, {
                id: profile.id,
                theme: values.theme,
                starredTags,
                hiddenTags: filteredHiddenTags,
            })
            if (!input || Object.keys(input).length === 0) {
                PubSub.get().publishSnack({ message: 'No changes made.', severity: 'error' });
                return;
            }
            mutationWrapper({
                mutation,
                input,
                onSuccess: (response) => {
                    PubSub.get().publishSnack({ message: 'Display preferences updated.' });
                    onUpdated(response.data.profileUpdate);
                    formik.setSubmitting(false);
                },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });

    /**
     * On page leave, check if unsaved work. 
     * If so, prompt for confirmation.
     */
    useEffect(() => {
        const handleBeforeUnload = (e: BeforeUnloadEvent) => {
            if (formik.dirty) {
                e.preventDefault()
                e.returnValue = ''
            }
        };
        window.addEventListener('beforeunload', handleBeforeUnload);
        return () => window.removeEventListener('beforeunload', handleBeforeUnload);
    }, [formik.dirty]);

    const actions: DialogActionItem[] = useMemo(() => [
        ['Save', SaveIcon, !formik.touched || formik.isSubmitting, false, () => {
            formik.submitForm();
        }],
        ['Revert', RevertIcon, !formik.touched || formik.isSubmitting, false, () => {
            formik.resetForm();
            setStarredTags([]);
            setHiddenTags([]);
        }],
    ], [formik]);

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
                <HelpButton markdown={helpText} sx={{ fill: TERTIARY_COLOR }} />
            </Stack>
            <Box sx={{ margin: 2, marginBottom: 5 }}>
                <ThemeSwitch
                    theme={formik.values.theme as 'light' | 'dark'}
                    onChange={(t) => formik.setFieldValue('theme', t)}
                />
            </Box>
            <Stack direction="row" marginRight="auto" alignItems="center" justifyContent="center">
                <InterestsIcon sx={{ marginRight: 1 }} />
                <Typography component="h2" variant="h5" textAlign="center">Favorite Topics</Typography>
                <HelpButton markdown={interestsHelpText} />
            </Stack>
            <Box sx={{ margin: 2, marginBottom: 5 }}>
                <TagSelector
                    session={session}
                    tags={starredTags}
                    placeholder={"Enter interests, followed by commas..."}
                    onTagAdd={addStarredTag}
                    onTagRemove={removeStarredTag}
                    onTagsClear={clearStarredTags}
                />
            </Box>
            <Stack direction="row" marginRight="auto" alignItems="center" justifyContent="center">
                <HiddenIcon sx={{ marginRight: 1 }} />
                <Typography component="h2" variant="h5" textAlign="center">Hidden Topics</Typography>
                <HelpButton markdown={hiddenHelpText} />
            </Stack>
            <Box sx={{ margin: 2, marginBottom: 5 }}>
                <TagSelector
                    session={session}
                    tags={hiddenTags.map(t => t.tag)}
                    placeholder={"Enter topics you'd like to hide from view, followed by commas..."}
                    onTagAdd={addHiddenTag}
                    onTagRemove={removeHiddenTag}
                    onTagsClear={clearHiddenTags}
                />
            </Box>
            <DialogActionsContainer fixed={false} actions={actions} />
        </form>
    )
}