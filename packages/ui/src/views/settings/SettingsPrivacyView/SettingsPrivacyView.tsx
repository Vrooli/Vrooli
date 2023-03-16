import { Grid, Stack, useTheme } from "@mui/material";
import { LINKS, ProfileUpdateInput, User } from '@shared/consts';
import { useLocation } from '@shared/route';
import { userValidation } from "@shared/validation";
import { userProfileUpdate } from "api/generated/endpoints/user_profileUpdate";
import { useCustomMutation } from "api/hooks";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { useFormik } from 'formik';
import { BaseForm } from "forms/BaseForm/BaseForm";
import { useCallback } from "react";
import { useProfileQuery } from "utils/hooks/useProfileQuery";
import { usePromptBeforeUnload } from "utils/hooks/usePromptBeforeUnload";
import { PubSub } from "utils/pubsub";
import { SettingsPrivacyViewProps } from "../types";

export const SettingsPrivacyView = ({
    display = 'page',
    session,
}: SettingsPrivacyViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery(session);

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

    const handleCancel = useCallback(() => {
        setLocation(LINKS.Profile, { replace: true })
    }, [setLocation]);

    return (
        <>
            <SettingsTopBar
                display={display}
                onClose={() => { }}
                session={session}
                titleData={{
                    titleKey: 'Privacy',
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
                    <Grid container spacing={2} sx={{
                        padding: 2,
                        marginBottom: 4,
                    }}>

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