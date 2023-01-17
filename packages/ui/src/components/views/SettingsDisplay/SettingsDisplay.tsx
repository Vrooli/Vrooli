import { Box, Button, Grid, Stack, Typography, useTheme } from "@mui/material"
import { useMutation } from "graphql/hooks";
import { useCallback, useEffect, useMemo, useState } from "react";
import { mutationWrapper } from 'graphql/utils';
import { useFormik } from 'formik';
import { clearSearchHistory, PubSub, shapeProfile, TagShape, usePromptBeforeUnload, UserScheduleFilterShape } from "utils";
import { SettingsDisplayProps } from "../types";
import { GridSubmitButtons, HelpButton, PageTitle, SnackSeverity, TagSelector } from "components";
import { ThemeSwitch } from "components/inputs";
import { uuid } from '@shared/uuid';
import { HeartFilledIcon, InvisibleIcon, SearchIcon } from "@shared/icons";
import { getCurrentUser } from "utils/authentication";
import { SettingsFormData } from "pages";
import { ProfileUpdateInput, User, UserScheduleFilterType } from "@shared/consts";
import { userEndpoint } from "graphql/endpoints";
import { userValidation } from "@shared/validation";
import { currentSchedules } from "utils/display/scheduleTools";

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
    return {} as any;
    // const { palette } = useTheme();

    // // Handle filters
    // const [filters, setFilters] = useState<UserScheduleFilterShape[]>([]);
    // const handleFiltersUpdate = useCallback((updatedList: UserScheduleFilterShape[], filterType: UserScheduleFilterType) => { 
    //     // Hidden tags are wrapped in a shape that includes an isBlur flag. 
    //     // Because of this, we must loop through the updatedList to see which tags have been added or removed.
    //     const updatedFilters = updatedList.map((tag) => {
    //         const existingTag = filters.find((filter) => filter.tag.id === tag.id);
    //         return existingTag ?? {
    //             id: uuid(),
    //             filterType,
    //             tag,
    //         }
    //     });
    //     setFilters(updatedFilters);
    // }, [filters]);
    // const { blurs, hides, showMores } = useMemo(() => {
    //     const blurs: TagShape[] = [];
    //     const hides: TagShape[] = [];
    //     const showMores: TagShape[] = [];
    //     filters.forEach((filter) => {
    //         if (filter.filterType === UserScheduleFilterType.Blur) {
    //             blurs.push(filter.tag);
    //         } else if (filter.filterType === UserScheduleFilterType.Hide) {
    //             hides.push(filter.tag);
    //         } else if (filter.filterType === UserScheduleFilterType.ShowMore) {
    //             showMores.push(filter.tag);
    //         }
    //     });
    //     return { blurs, hides, showMores };
    // }, [filters]);

    // useEffect(() => {
    //     const currSchedules = currentSchedules(session);
    //     // setFilters(currSchedule?.filters ?? []);
    // }, [profile]);

    // // Handle update
    // const [mutation] = useMutation<User, ProfileUpdateInput, 'profileUpdate'>(...userEndpoint.profileUpdate);
    // const formik = useFormik({
    //     initialValues: {
    //         theme: getCurrentUser(session).theme ?? 'light',
    //     },
    //     validationSchema: userValidation.update(),
    //     onSubmit: (values) => {
    //         if (!profile) {
    //             PubSub.get().publishSnack({ messageKey: 'CouldNotReadProfile', severity: SnackSeverity.Error });
    //             return;
    //         }
    //         if (!formik.isValid) return;
    //         // Filter hidden tags that are also either blurs or showMores
    //         const filteredHides = hides.filter(t => !blurs.some(b => b.tag === t.tag.tag) && !showMores.some(sm => sm.tag === t.tag.tag));
    //         // Filter blurs that are also either hidden or showMores
    //         const filteredBlurs = blurs.filter(t => !hides.some(h => h.tag === t.tag.tag) && !showMores.some(sm => sm.tag === t.tag.tag));
    //         // Filter showMores that are also either hidden or blurs
    //         const filteredShowMores = showMores.filter(t => !hides.some(h => h.tag === t.tag.tag) && !blurs.some(b => b.tag === t.tag.tag));
    //         // If any of the filtered lists are shorter than the original, give warning to user.
    //         if (filteredHides.length !== hides.length || filteredBlurs.length !== blurs.length || filteredShowMores.length !== showMores.length) {
    //             PubSub.get().publishSnack({ messageKey: 'FoundTopicsInFavAndHidden', severity: SnackSeverity.Warning });
    //         }
    //         const input = shapeProfile.update(profile, {
    //             id: profile.id,
    //             theme: values.theme as 'light' | 'dark',
    //             starredTags,
    //             filters: filteredHiddenTags,
    //         })
    //         if (!input || Object.keys(input).length === 0) {
    //             PubSub.get().publishSnack({ messageKey: 'NoChangesMade', severity: SnackSeverity.Error });
    //             formik.setSubmitting(false);
    //             return;
    //         }
    //         mutationWrapper<User, ProfileUpdateInput>({
    //             mutation,
    //             input,
    //             successMessage: () => ({ key: 'DisplayPreferencesUpdated' }),
    //             onSuccess: (data) => {
    //                 onUpdated(data);
    //                 formik.setSubmitting(false);
    //             },
    //             onError: () => { formik.setSubmitting(false) },
    //         })
    //     },
    // });
    // usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    // const handleSave = useCallback(() => {
    //     formik.submitForm();
    // }, [formik]);

    // const handleCancel = useCallback(() => {
    //     formik.resetForm();
    //     setFilters([]);
    // }, [formik]);

    // return (
    //     <form onSubmit={formik.handleSubmit} style={{ overflow: 'hidden' }}>
    //         <PageTitle titleKey='DisplayPreferences' helpKey='DisplayPreferencesHelp' session={session} />
    //         <Box id="theme-switch-box" sx={{ margin: 2, marginBottom: 5 }}>
    //             <ThemeSwitch
    //                 theme={formik.values.theme as 'light' | 'dark'}
    //                 onChange={(t) => formik.setFieldValue('theme', t)}
    //             />
    //         </Box>
    //         <Stack direction="row" marginRight="auto" alignItems="center" justifyContent="center">
    //             <HeartFilledIcon fill={palette.background.textPrimary} />
    //             <Typography component="h2" variant="h5" textAlign="center" ml={1}>Favorite Topics</Typography>
    //             <HelpButton markdown={interestsHelpText} />
    //         </Stack>
    //         <Box id="favorite-topics-box" sx={{ margin: 2, marginBottom: 5 }}>
    //             <TagSelector
    //                 handleTagsUpdate={handleStarredTagsUpdate}
    //                 session={session}
    //                 tags={starredTags}
    //                 placeholder={"Enter interests, followed by commas..."}
    //             />
    //         </Box>
    //         <Stack direction="row" marginRight="auto" alignItems="center" justifyContent="center">
    //             <InvisibleIcon fill={palette.background.textPrimary} />
    //             <Typography component="h2" variant="h5" textAlign="center" ml={1}>Hidden Topics</Typography>
    //             <HelpButton markdown={hiddenHelpText} />
    //         </Stack>
    //         <Box id="hidden-topics-box" sx={{ margin: 2, marginBottom: 5 }}>
    //             <TagSelector
    //                 handleTagsUpdate={handleFiltersUpdate}
    //                 session={session}
    //                 tags={filters.map(t => t.tag)}
    //                 placeholder={"Enter topics you'd like to hide from view, followed by commas..."}
    //             />
    //         </Box>
    //         <Box sx={{ margin: 2, marginBottom: 5, display: 'flex' }}>
    //             <Button id="clear-search-history-button" color="secondary" startIcon={<SearchIcon />} onClick={() => { clearSearchHistory(session) }} sx={{
    //                 marginLeft: 'auto',
    //                 marginRight: 'auto',
    //             }}>Clear Search History</Button>
    //         </Box>
    //         <Grid container spacing={2} p={2}>
    //             <GridSubmitButtons
    //                 errors={formik.errors}
    //                 isCreate={false}
    //                 loading={formik.isSubmitting}
    //                 onCancel={handleCancel}
    //                 onSetSubmitting={formik.setSubmitting}
    //                 onSubmit={handleSave}
    //             />
    //         </Grid>
    //     </form>
    // )
}

export const settingsDisplayFormData: SettingsFormData = {
    labels: ['Display Preferences', 'Appearance', 'Customization', 'Customize'],
    items: [
        { id: 'theme-switch-box', labels: ['Theme', 'Dark Mode', 'Light Mode', 'Color Scheme'] },
        { id: 'favorite-topics-box', labels: ['Favorite Topics', 'Favorite Interests', 'Favorite Tags', 'Favorite Categories'] },
        { id: 'hidden-topics-box', labels: ['Hidden Topics', 'Hidden Interests', 'Hidden Tags', 'Hidden Categories'] },
        { id: 'clear-search-history-button', labels: ['Clear Search History', 'Erase Search History', 'Delete Search History'] },
    ],
}