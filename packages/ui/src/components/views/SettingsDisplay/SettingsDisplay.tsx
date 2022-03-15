import { Box, Container, Grid, Stack, Typography } from "@mui/material"
import { useMutation } from "@apollo/client";
import { user } from "graphql/generated/user";
import { useCallback, useEffect, useMemo, useState } from "react";
import { mutationWrapper } from 'graphql/utils/wrappers';
import { APP_LINKS, profileUpdateSchema as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { profileUpdateMutation } from "graphql/mutation";
import { formatForUpdate } from "utils";
import {
    Restore as RevertIcon,
    Save as SaveIcon,
} from '@mui/icons-material';
import { DialogActionItem } from "components/containers/types";
import { DialogActionsContainer } from "components/containers/DialogActionsContainer/DialogActionsContainer";
import { SettingsDisplayProps } from "../types";
import helpMarkdown from './SettingsDisplayHelp.md';
import { useLocation } from "wouter";
import { HelpButton, TagSelector } from "components";
import { TagSelectorTag } from "components/inputs/types";
import { containerShadow } from "styles";

const helpText = 
`Display preferences customize the look and feel of Vrooli. More customizations will be available in the near future.  

Specifying your interests can simplify the discovery of routines, projects, organizations, and standards, via customized feeds.

You can also specify tags which should be hidden from your feeds.

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

    // Handle update
    const [mutation] = useMutation<user>(profileUpdateMutation);
    const formik = useFormik({
        initialValues: {
            theme: profile?.theme ?? 'light',
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            console.log('onSubmit', formik.isValid);
            if (!formik.isValid) return;
            mutationWrapper({
                mutation,
                input: formatForUpdate(profile, { ...values, starredTags, hiddenTags }),
                onSuccess: (response) => { onUpdated(response.data.profileUpdate) },
            })
        },
    });

    const actions: DialogActionItem[] = useMemo(() => [
        ['Save', SaveIcon, !formik.touched || formik.isSubmitting, false, () => { 
            console.log('submit');
            formik.submitForm();
        }],
        ['Revert', RevertIcon, !formik.touched || formik.isSubmitting, false, () => { 
            formik.resetForm();
            setStarredTags([]);
            setHiddenTags([]);
        }],
    ], [formik, setLocation]);

    return (
        <Box sx={{ minHeight: '100vh', display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
            <Box sx={{
                ...containerShadow,
                borderRadius: 2,
                overflow: 'overlay',
                marginTop: '-5vh',
                background: (t) => t.palette.background.default,
                width: 'min(100%, 700px)',
            }}>
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
                    <Container sx={{ paddingBottom: 2 }}>
                        <Grid container spacing={2}>
                            <Grid item xs={12} sx={{marginBottom: 2}}>
                                <TagSelector
                                    session={session}
                                    tags={starredTags}
                                    placeholder={"Enter interests, followed by commas..."}
                                    onTagAdd={addStarredTag}
                                    onTagRemove={removeStarredTag}
                                    onTagsClear={clearStarredTags}
                                />
                            </Grid>
                            <Grid item xs={12}>
                                <TagSelector
                                    session={session}
                                    tags={hiddenTags}
                                    placeholder={"Enter topics you'd like to hide from view, followed by commas..."}
                                    onTagAdd={addHiddenTag}
                                    onTagRemove={removeHiddenTag}
                                    onTagsClear={clearHiddenTags}
                                />
                            </Grid>
                        </Grid>
                    </Container>
                    <DialogActionsContainer fixed={false} actions={actions} />
                </form>
            </Box>
        </Box>
    )
}