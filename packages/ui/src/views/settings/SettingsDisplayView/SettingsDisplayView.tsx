import { Box, Button, Grid, Stack, Typography, useTheme } from "@mui/material"
import { useCallback, useEffect, useMemo, useState } from "react";
import { useFormik } from 'formik';
import { clearSearchHistory, TagShape, useProfileQuery, usePromptBeforeUnload, UserScheduleFilterShape } from "utils";
import { GridSubmitButtons, HelpButton, SettingsList, SettingsTopBar } from "components";
import { ThemeSwitch } from "components/inputs";
import { HeartFilledIcon, InvisibleIcon, SearchIcon } from "@shared/icons";
import { getCurrentUser } from "utils/authentication";
import { ProfileUpdateInput, User, UserScheduleFilterType } from "@shared/consts";
import { userValidation } from "@shared/validation";
import { currentSchedules } from "utils/display/scheduleTools";
import { SettingsDisplayViewProps } from "../types";
import { useTranslation } from "react-i18next";
import { BaseForm } from "forms";
import { userProfileUpdate } from "api/generated/endpoints/user_profileUpdate";
import { useCustomMutation } from "api";

const interestsHelpText =
    `Specifying your interests can simplify the discovery of routines, projects, organizations, and standards, via customized feeds.\n\n**None** of this information is available to the public, and **none** of it is sold to advertisers.`

const hiddenHelpText =
    `Specify tags which should be hidden from your feeds.\n\n**None** of this information is available to the public, and **none** of it is sold to advertisers.`

export const SettingsDisplayView = ({
    display = 'page',
    session,
}: SettingsDisplayViewProps) => {
    const { palette } = useTheme();
    const { t } = useTranslation();

    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery(session);

    // Handle filters
    const [filters, setFilters] = useState<UserScheduleFilterShape[]>([]);
    const handleFiltersUpdate = useCallback((updatedList: UserScheduleFilterShape[], filterType: UserScheduleFilterType) => {
        // Hidden tags are wrapped in a shape that includes an isBlur flag. 
        // Because of this, we must loop through the updatedList to see which tags have been added or removed.
        // const updatedFilters = updatedList.map((tag) => {
        //     const existingTag = filters.find((filter) => filter.tag.id === tag.id);
        //     return existingTag ?? {
        //         id: uuid(),
        //         filterType,
        //         tag,
        //     }
        // });
        // setFilters(updatedFilters);
    }, [filters]);
    const { blurs, hides, showMores } = useMemo(() => {
        const blurs: TagShape[] = [];
        const hides: TagShape[] = [];
        const showMores: TagShape[] = [];
        filters.forEach((filter) => {
            if (filter.filterType === UserScheduleFilterType.Blur) {
                blurs.push(filter.tag);
            } else if (filter.filterType === UserScheduleFilterType.Hide) {
                hides.push(filter.tag);
            } else if (filter.filterType === UserScheduleFilterType.ShowMore) {
                showMores.push(filter.tag);
            }
        });
        return { blurs, hides, showMores };
    }, [filters]);

    useEffect(() => {
        const currSchedules = currentSchedules(session);
        // setFilters(currSchedule?.filters ?? []);
    }, [profile]);

    // Handle update
    const [mutation, { loading: isUpdating }] = useCustomMutation<User, ProfileUpdateInput>(userProfileUpdate);
    const formik = useFormik({
        initialValues: {
            theme: getCurrentUser(session).theme ?? 'light',
        },
        validationSchema: userValidation.update({}),
        onSubmit: (values) => {
            // if (!profile) {
            //     PubSub.get().publishSnack({ messageKey: 'CouldNotReadProfile', severity: 'Error' });
            //     return;
            // }
            // if (!formik.isValid) return;
            // // Filter hidden tags that are also either blurs or showMores
            // const filteredHides = hides.filter(t => !blurs.some(b => b.tag === t.tag.tag) && !showMores.some(sm => sm.tag === t.tag.tag));
            // // Filter blurs that are also either hidden or showMores
            // const filteredBlurs = blurs.filter(t => !hides.some(h => h.tag === t.tag.tag) && !showMores.some(sm => sm.tag === t.tag.tag));
            // // Filter showMores that are also either hidden or blurs
            // const filteredShowMores = showMores.filter(t => !hides.some(h => h.tag === t.tag.tag) && !blurs.some(b => b.tag === t.tag.tag));
            // // If any of the filtered lists are shorter than the original, give warning to user.
            // if (filteredHides.length !== hides.length || filteredBlurs.length !== blurs.length || filteredShowMores.length !== showMores.length) {
            //     PubSub.get().publishSnack({ messageKey: 'FoundTopicsInFavAndHidden', severity: 'Warning' });
            // }
            // const input = shapeProfile.update(profile, {
            //     id: profile.id,
            //     theme: values.theme as 'light' | 'dark',
            //     bookmarkedTags,
            //     filters: filteredHiddenTags,
            // })
            // if (!input || Object.keys(input).length === 0) {
            //     PubSub.get().publishSnack({ messageKey: 'NoChangesMade', severity: 'Error' });
            //     formik.setSubmitting(false);
            //     return;
            // }
            // mutationWrapper<User, ProfileUpdateInput>({
            //     mutation,
            //     input,
            //     successMessage: () => ({ key: 'DisplayPreferencesUpdated' }),
            //     onSuccess: (data) => {
            //         onUpdated(data);
            //         formik.setSubmitting(false);
            //     },
            //     onError: () => { formik.setSubmitting(false) },
            // })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    const handleSave = useCallback(() => {
        formik.submitForm();
    }, [formik]);

    const handleCancel = useCallback(() => {
        formik.resetForm();
        setFilters([]);
    }, [formik]);

    return (
        <>
            <SettingsTopBar
                display={display}
                onClose={() => { }}
                session={session}
                titleData={{
                    titleKey: 'Display',
                }}
            />
            <Stack direction="row">
                <SettingsList />
                <BaseForm
                    isLoading={isProfileLoading || isUpdating}
                    onSubmit={formik.handleSubmit}
                    style={{
                        width: { xs: '100%', md: 'min(100%, 700px)' },
                        marginRight: 'auto',
                        display: 'block',
                    }}
                >
                    {/* <Header titleKey='DisplayPreferences' helpKey='DisplayPreferencesHelp' /> */}
                    <Box id="theme-switch-box" sx={{ margin: 2, marginBottom: 5 }}>
                        <ThemeSwitch
                            theme={formik.values.theme as 'light' | 'dark'}
                            onChange={(t) => formik.setFieldValue('theme', t)}
                        />
                    </Box>
                    <Stack direction="row" marginRight="auto" alignItems="center" justifyContent="center">
                        <HeartFilledIcon fill={palette.background.textPrimary} />
                        <Typography component="h2" variant="h5" textAlign="center" ml={1}>{t('TopicsFavorite')}</Typography>
                        <HelpButton markdown={interestsHelpText} />
                    </Stack>
                    <Box id="favorite-topics-box" sx={{ margin: 2, marginBottom: 5 }}>
                        {/* <TagSelector
                    handleTagsUpdate={handleBookmarkedTagsUpdate}
                    session={session}
                    tags={bookmarkedTags}
                    placeholder={"Enter interests, followed by commas..."}
                /> */}
                    </Box>
                    <Stack direction="row" marginRight="auto" alignItems="center" justifyContent="center">
                        <InvisibleIcon fill={palette.background.textPrimary} />
                        <Typography component="h2" variant="h5" textAlign="center" ml={1}>{t('TopicsHidden')}</Typography>
                        <HelpButton markdown={hiddenHelpText} />
                    </Stack>
                    <Box id="hidden-topics-box" sx={{ margin: 2, marginBottom: 5 }}>
                        {/* <TagSelector
                    handleTagsUpdate={handleFiltersUpdate}
                    session={session}
                    tags={filters.map(t => t.tag)}
                    placeholder={"Enter topics you'd like to hide from view, followed by commas..."}
                /> */}
                    </Box>
                    <Box sx={{ margin: 2, marginBottom: 5, display: 'flex' }}>
                        <Button id="clear-search-history-button" color="secondary" startIcon={<SearchIcon />} onClick={() => { session && clearSearchHistory(session) }} sx={{
                            marginLeft: 'auto',
                            marginRight: 'auto',
                        }}>{t('ClearSearchHistory')}</Button>
                    </Box>
                    <Grid container spacing={2} p={2}>
                        <GridSubmitButtons
                            display={display}
                            errors={formik.errors}
                            isCreate={false}
                            loading={formik.isSubmitting}
                            onCancel={handleCancel}
                            onSetSubmitting={formik.setSubmitting}
                            onSubmit={handleSave}
                        />
                    </Grid>
                </BaseForm>
            </Stack>
        </>
    )
}