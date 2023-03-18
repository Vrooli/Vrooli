import { Box, Grid, Stack, Typography, useTheme } from "@mui/material";
import { LINKS, ProfileUpdateInput, User, UserScheduleFilterType } from '@shared/consts';
import { HeartFilledIcon, InvisibleIcon } from "@shared/icons";
import { useLocation } from '@shared/route';
import { userValidation } from "@shared/validation";
import { userProfileUpdate } from "api/generated/endpoints/user_profileUpdate";
import { useCustomMutation } from "api/hooks";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { HelpButton } from "components/buttons/HelpButton/HelpButton";
import { SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { useFormik } from 'formik';
import { BaseForm } from "forms/BaseForm/BaseForm";
import { t } from "i18next";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { currentSchedules } from "utils/display/scheduleTools";
import { useProfileQuery } from "utils/hooks/useProfileQuery";
import { usePromptBeforeUnload } from "utils/hooks/usePromptBeforeUnload";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { TagShape } from "utils/shape/models/tag";
import { UserScheduleFilterShape } from "utils/shape/models/userScheduleFilter";
import { SettingsSchedulesViewProps } from "../types";

const interestsHelpText =
    `Specifying your interests can simplify the discovery of routines, projects, organizations, and standards, via customized feeds.\n\n**None** of this information is available to the public, and **none** of it is sold to advertisers.`

const hiddenHelpText =
    `Specify tags which should be hidden from your feeds.\n\n**None** of this information is available to the public, and **none** of it is sold to advertisers.`

export const SettingsSchedulesView = ({
    display = 'page',
}: SettingsSchedulesViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery();

    // Handle update
    const [mutation, { loading: isUpdating }] = useCustomMutation<User, ProfileUpdateInput>(userProfileUpdate);
    const formik = useFormik({
        initialValues: {
        },
        enableReinitialize: true,
        validationSchema: userValidation.update({}),
        onSubmit: (values) => {
            if (!profile) {
                PubSub.get().publishSnack({ messageKey: 'CouldNotReadProfile', severity: 'Error' });
                return;
            }
            if (!formik.isValid) {
                PubSub.get().publishSnack({ messageKey: 'FixErrorsBeforeSubmitting', severity: 'Error' });
                return;
            }
            // const input = shapeProfile.update(profile, {
            //     id: profile.id,
            // })
            // if (!input || Object.keys(input).length === 0) {
            //     PubSub.get().publishSnack({ messageKey: 'NoChangesMade', severity: 'Info' });
            //     return;
            // }
            // mutationWrapper<User, ProfileUpdateInput>({
            //     mutation,
            //     input,
            //     successMessage: () => ({ key: 'SettingsUpdated' }),
            //     onError: () => { formik.setSubmitting(false) },
            // })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

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
    }, [profile, session]);

    const handleCancel = useCallback(() => {
        setLocation(LINKS.Profile, { replace: true })
    }, [setLocation]);

    return (
        <>
            <SettingsTopBar
                display={display}
                onClose={() => { }}
                titleData={{
                    titleKey: 'Schedule',
                    titleVariables: { count: 2 },
                }}
            />
            <Stack direction="row">
                <BaseForm
                    isLoading={isProfileLoading || isUpdating}
                    onSubmit={formik.handleSubmit}
                    style={{
                        width: { xs: '100%', md: 'min(100%, 700px)' },
                        marginRight: 'auto',
                        display: 'block',
                    }}
                >
                    <Grid container spacing={2}>
                        <Stack direction="row" marginRight="auto" alignItems="center" justifyContent="center">
                            <HeartFilledIcon fill={palette.background.textPrimary} />
                            <Typography component="h2" variant="h5" textAlign="center" ml={1}>{t('TopicsFavorite')}</Typography>
                            <HelpButton markdown={interestsHelpText} />
                        </Stack>
                        <Box id="favorite-topics-box" sx={{ margin: 2, marginBottom: 5 }}>
                            {/* <TagSelector
                    handleTagsUpdate={handleBookmarkedTagsUpdate}
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
                    tags={filters.map(t => t.tag)}
                    placeholder={"Enter topics you'd like to hide from view, followed by commas..."}
                /> */}
                        </Box>
                    </Grid>
                    <GridSubmitButtons
                        display={display}
                        errors={formik.errors}
                        isCreate={false}
                        loading={formik.isSubmitting}
                        onCancel={handleCancel}
                        onSetSubmitting={formik.setSubmitting}
                        onSubmit={formik.handleSubmit}
                    />
                </BaseForm>
            </Stack>
        </>
    )
}