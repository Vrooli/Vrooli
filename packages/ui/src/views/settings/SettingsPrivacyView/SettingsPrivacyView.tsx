import { Grid, Stack, useTheme } from "@mui/material"
import { useCustomMutation } from "api/hooks";
import { useCallback } from "react";
import { mutationWrapper } from 'api/utils';
import { APP_LINKS, ProfileUpdateInput, User } from '@shared/consts';
import { useFormik } from 'formik';
import { getUserLanguages, shapeProfile, useProfileQuery, usePromptBeforeUnload } from "utils";
import { SettingsPrivacyViewProps } from "../types";
import { useLocation } from '@shared/route';
import { GridSubmitButtons } from "components/buttons";
import { PubSub } from 'utils'
import { DUMMY_ID } from '@shared/uuid';
import { userValidation } from "@shared/validation";
import { userProfileUpdate } from "api/generated/endpoints/user_profileUpdate";
import { SettingsList, SettingsTopBar, TopBar } from "components";
import { BaseForm } from "forms";

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
            name: profile?.name ?? '',
            translationsUpdate: profile?.translations ?? [{
                id: DUMMY_ID,
                language: getUserLanguages(session)[0],
                bio: '',
            }],
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
            const input = shapeProfile.update(profile, {
                id: profile.id,
                name: values.name,
                translations: values.translationsUpdate,
            })
            if (!input || Object.keys(input).length === 0) {
                PubSub.get().publishSnack({ messageKey: 'NoChangesMade', severity: 'Info' });
                return;
            }
            mutationWrapper<User, ProfileUpdateInput>({
                mutation,
                input,
                successMessage: () => ({ key: 'SettingsUpdated' }),
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    const handleCancel = useCallback(() => {
        setLocation(APP_LINKS.Profile, { replace: true })
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