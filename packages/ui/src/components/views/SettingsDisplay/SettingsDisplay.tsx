import { Box, Container, Grid, Stack, Typography } from "@mui/material"
import { useMutation } from "@apollo/client";
import { user } from "graphql/generated/user";
import { useCallback, useEffect, useMemo, useState } from "react";
import { mutationWrapper } from 'graphql/utils/wrappers';
import { profileUpdateSchema as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { profileUpdateMutation } from "graphql/mutation";
import { formatForUpdate, Pubs } from "utils";
import {
    Restore as RevertIcon,
    Save as SaveIcon,
    ThumbUp as InterestsIcon,
    VisibilityOff as HiddenIcon,
} from '@mui/icons-material';
import { DialogActionItem } from "components/containers/types";
import { DialogActionsContainer } from "components/containers/DialogActionsContainer/DialogActionsContainer";
import { SettingsDisplayProps } from "../types";
import { useLocation } from "wouter";
import { HelpButton, TagSelector } from "components";
import { TagSelectorTag } from "components/inputs/types";

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

const TERTIARY_COLOR = '#95f3cd';

export const SettingsDisplay = ({
    session,
    profile,
    onUpdated,
}: SettingsDisplayProps) => {
    const [, setLocation] = useLocation();

    // Handle starred tags
    const [starredTags, setStarredTags] = useState<TagSelectorTag[]>([]);
    const addStarredTag = useCallback((tag: TagSelectorTag) => {
        setStarredTags(t => [...t, tag]);
    }, [setStarredTags]);
    const removeStarredTag = useCallback((tag: TagSelectorTag) => {
        setStarredTags(tags => tags.filter(t => t.tag !== tag.tag));
    }, [setStarredTags]);
    const clearStarredTags = useCallback(() => {
        setStarredTags([]);
    }, [setStarredTags]);

    // Handle hidden tags
    const [hiddenTags, setHiddenTags] = useState<TagSelectorTag[]>([]);
    const addHiddenTag = useCallback((tag: TagSelectorTag) => {
        setHiddenTags(t => [...t, tag]);
    }, [setHiddenTags]);
    const removeHiddenTag = useCallback((tag: TagSelectorTag) => {
        setHiddenTags(tags => tags.filter(t => t.tag !== tag.tag));
    }, [setHiddenTags]);
    const clearHiddenTags = useCallback(() => {
        setHiddenTags([]);
    }, [setHiddenTags]);

    useEffect(() => {
        if (profile?.starredTags) {
            setStarredTags(profile.starredTags);
        }
        if (profile?.hiddenTags) {
            setHiddenTags(profile.hiddenTags.map(t => t.tag as any));
        }
    }, [profile]);

    // Handle update
    const [mutation] = useMutation<user>(profileUpdateMutation);
    const formik = useFormik({
        initialValues: {
            theme: profile?.theme ?? 'light',
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            if (!formik.isValid) return;
            // If any tags are in both starredTags and hiddenTags, remove them from hidden. Also give warning to user.
            const filteredHiddenTags = hiddenTags.filter(t => !starredTags.some(st => st.tag === t.tag));
            if (filteredHiddenTags.length !== hiddenTags.length) {
                PubSub.publish(Pubs.Snack, { message: 'Detected topics in both favorites and hidden. These have been removed from hidden.', severity: 'warning' });
            }
            const starredTagsUpdate = starredTags.length > 0 ? {
                starredTagsCreate: starredTags.filter(t => !t.id && !profile?.starredTags?.some(tag => tag.tag === t.tag)).map(t => ({ tag: t.tag })),
                starredTagsConnect: starredTags.filter(t => t.id && !profile?.starredTags?.some(tag => tag.tag === t.tag)).map(t => (t.id)),
                starredTagsDisconnect: profile?.starredTags?.filter(t => !starredTags.some(st => st.tag === t.tag)).map(t => (t.id)),
            } : {};
            const hiddenTagsUpdate = filteredHiddenTags.length > 0 ? {
                hiddenTagsCreate: filteredHiddenTags.filter(t => !t.id && !profile?.hiddenTags?.some(tag => tag.tag.tag === t.tag)).map(t => ({ tag: t.tag })),
                hiddenTagsConnect: filteredHiddenTags.filter(t => t.id && !profile?.hiddenTags?.some(tag => tag.tag.tag === t.tag)).map(t => (t.id)),
                hiddenTagsDisconnect: profile?.hiddenTags?.filter(t => !filteredHiddenTags.some(ht => ht.tag === t.tag.tag)).map((t: any) => (t.id)),
            } : {};
            //TODO temp
            PubSub.publish(Pubs.Snack, { message: 'Available next update. Please be patient with us😬', severity: 'error' });
            return;
            mutationWrapper({
                mutation,
                input: formatForUpdate(profile, {
                    ...values,
                    ...starredTagsUpdate,
                    ...hiddenTagsUpdate,
                }),
                onSuccess: (response) => {
                    PubSub.publish(Pubs.Snack, { message: 'Display preferences updated.' });
                    onUpdated(response.data.profileUpdate);
                    formik.setSubmitting(false);
                },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });

    const actions: DialogActionItem[] = useMemo(() => [
        ['Save', SaveIcon, !formik.touched || formik.isSubmitting, false, () => {
            formik.submitForm();
        }],
        ['Revert', RevertIcon, !formik.touched || formik.isSubmitting, false, () => {
            formik.resetForm();
            setStarredTags([]);
            setHiddenTags([]);
        }],
    ], [formik, setLocation]);

    return (
        <form onSubmit={formik.handleSubmit} style={{ overflow: 'hidden' }}>
            {/* Title */}
            <Stack direction="row" justifyContent="center" alignItems="center" sx={{
                background: (t) => t.palette.primary.dark,
                color: (t) => t.palette.primary.contrastText,
                padding: 0.5,
                marginBottom: 2,
            }}>
                <Typography component="h1" variant="h3">Display Preferences</Typography>
                <HelpButton markdown={helpText} sx={{ fill: TERTIARY_COLOR }} />
            </Stack>
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
                    tags={hiddenTags}
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