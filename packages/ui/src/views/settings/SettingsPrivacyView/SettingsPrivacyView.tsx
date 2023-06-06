import { endpointPutProfile, ProfileUpdateInput, User, userValidation } from "@local/shared";
import { Stack } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { Formik } from "formik";
import { SettingsPrivacyForm } from "forms/settings";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
import { useProfileQuery } from "utils/hooks/useProfileQuery";
import { PubSub } from "utils/pubsub";
import { SettingsPrivacyViewProps } from "../types";

export const SettingsPrivacyView = ({
    display = "page",
    onClose,
    zIndex,
}: SettingsPrivacyViewProps) => {

    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery();
    const [fetch, { loading: isUpdating }] = useLazyFetch<ProfileUpdateInput, User>(endpointPutProfile);

    return (
        <>
            <SettingsTopBar
                display={display}
                onClose={onClose}
                titleData={{
                    titleKey: "Authentication",
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
                            PubSub.get().publishSnack({ messageKey: "CouldNotReadProfile", severity: "Error" });
                            return;
                        }
                        fetchLazyWrapper<ProfileUpdateInput, User>({
                            fetch,
                            inputs: values,
                            successMessage: () => ({ messageKey: "SettingsUpdated" }),
                            onSuccess: (data) => { onProfileUpdate(data); },
                            onError: () => { helpers.setSubmitting(false); },
                        });
                    }}
                    validationSchema={userValidation.update({})}
                >
                    {(formik) => <SettingsPrivacyForm
                        display={display}
                        isLoading={isProfileLoading || isUpdating}
                        onCancel={formik.resetForm}
                        zIndex={zIndex}
                        {...formik}
                    />}
                </Formik>
            </Stack>
        </>
    );
};
