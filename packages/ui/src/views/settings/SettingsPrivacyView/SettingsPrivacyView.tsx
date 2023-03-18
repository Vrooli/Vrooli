import { Stack } from "@mui/material";
import { ProfileUpdateInput, User } from '@shared/consts';
import { userValidation } from "@shared/validation";
import { userProfileUpdate } from "api/generated/endpoints/user_profileUpdate";
import { useCustomMutation } from "api/hooks";
import { mutationWrapper } from 'api/utils';
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { Formik } from 'formik';
import { SettingsPrivacyForm } from "forms/settings";
import { useProfileQuery } from "utils/hooks/useProfileQuery";
import { PubSub } from "utils/pubsub";
import { SettingsPrivacyViewProps } from "../types";

export const SettingsPrivacyView = ({
    display = 'page',
}: SettingsPrivacyViewProps) => {

    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery();
    const [mutation, { loading: isUpdating }] = useCustomMutation<User, ProfileUpdateInput>(userProfileUpdate);

    return (
        <>
            <SettingsTopBar
                display={display}
                onClose={() => { }}
                titleData={{
                    titleKey: 'Authentication',
                }}
            />
            <Stack direction="row">
                <SettingsList />
                <Formik
                    enableReinitialize={true}
                    initialValues={{
                        isPrivate: profile?.isPrivate ?? false,
                        isPrivateApis: profile?.isPrivateApis ?? false,
                        isPrivateBookmarks: profile?.isPrivateBookmarks ?? false,
                        isPrivateProjects: profile?.isPrivateProjects ?? false,
                        isPrivateRoutines: profile?.isPrivateRoutines ?? false,
                        isPrivateSmartContracts: profile?.isPrivateSmartContracts ?? false,
                        isPrivateStandards: profile?.isPrivateStandards ?? false,
                    } as ProfileUpdateInput}
                    onSubmit={(values, helpers) => {
                        if (!profile) {
                            PubSub.get().publishSnack({ messageKey: 'CouldNotReadProfile', severity: 'Error' });
                            return;
                        }
                        mutationWrapper<User, ProfileUpdateInput>({
                            mutation,
                            input: values,
                            onSuccess: (data) => { onProfileUpdate(data) },
                            onError: () => { helpers.setSubmitting(false) },
                        })
                    }}
                    validationSchema={userValidation.update({})}
                >
                    {(formik) => <SettingsPrivacyForm
                        display={display}
                        isLoading={isProfileLoading || isUpdating}
                        onCancel={formik.resetForm}
                        {...formik}
                    />}
                </Formik>
            </Stack>
        </>
    )
}