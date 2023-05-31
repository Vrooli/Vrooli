import { DUMMY_ID, ProfileUpdateInput, User, userProfileUpdate, userValidation } from "@local/shared";
import { Stack } from "@mui/material";
import { useCustomMutation } from "api";
import { mutationWrapper } from "api/utils";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { Formik } from "formik";
import { SettingsProfileForm } from "forms/settings";
import { useContext } from "react";
import { getUserLanguages } from "utils/display/translationTools";
import { useProfileQuery } from "utils/hooks/useProfileQuery";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { shapeProfile } from "utils/shape/models/profile";
import { SettingsProfileViewProps } from "../types";

export const SettingsProfileView = ({
    display = "page",
    onClose,
    zIndex,
}: SettingsProfileViewProps) => {
    const session = useContext(SessionContext);

    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery();
    const [mutation, { loading: isUpdating }] = useCustomMutation<User, ProfileUpdateInput>(userProfileUpdate);

    return (
        <>
            <SettingsTopBar
                display={display}
                onClose={onClose}
                titleData={{
                    titleKey: "Profile",
                }}
            />
            <Stack direction="row">
                <SettingsList />
                <Formik
                    enableReinitialize={true}
                    initialValues={{
                        handle: profile?.handle ?? null,
                        name: profile?.name ?? "",
                        translations: profile?.translations?.length ? profile.translations : [{
                            id: DUMMY_ID,
                            language: getUserLanguages(session)[0],
                            bio: "",
                        }],
                    }}
                    onSubmit={(values, helpers) => {
                        if (!profile) {
                            PubSub.get().publishSnack({ messageKey: "CouldNotReadProfile", severity: "Error" });
                            return;
                        }
                        mutationWrapper<User, ProfileUpdateInput>({
                            mutation,
                            input: shapeProfile.update(profile, {
                                id: profile.id,
                                ...values,
                            }),
                            successMessage: () => ({ messageKey: "SettingsUpdated" }),
                            onError: () => { helpers.setSubmitting(false); },
                        });
                    }}
                    validationSchema={userValidation.update({})}
                >
                    {(formik) => <SettingsProfileForm
                        display={display}
                        isLoading={isProfileLoading || isUpdating}
                        numVerifiedWallets={profile?.wallets?.filter(w => w.verified)?.length ?? 0}
                        onCancel={formik.resetForm}
                        zIndex={zIndex}
                        {...formik}
                    />}
                </Formik>
            </Stack>
        </>
    );
};
