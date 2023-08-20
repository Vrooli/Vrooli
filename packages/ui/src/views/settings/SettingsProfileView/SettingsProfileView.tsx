import { DUMMY_ID, endpointPutProfile, ProfileUpdateInput, User, userValidation } from "@local/shared";
import { Box, Stack } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { SessionContext } from "contexts/SessionContext";
import { Formik } from "formik";
import { SettingsProfileForm } from "forms/settings";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useProfileQuery } from "hooks/useProfileQuery";
import { useContext } from "react";
import { useTranslation } from "react-i18next";
import { toDisplay } from "utils/display/pageTools";
import { getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { shapeProfile } from "utils/shape/models/profile";
import { createPrims } from "utils/shape/models/tools";
import { SettingsProfileViewProps } from "../types";

export const SettingsProfileView = ({
    isOpen,
    onClose,
    zIndex,
}: SettingsProfileViewProps) => {
    const { t } = useTranslation();
    const session = useContext(SessionContext);
    const display = toDisplay(isOpen);

    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery();
    const [fetch, { loading: isUpdating }] = useLazyFetch<ProfileUpdateInput, User>(endpointPutProfile);

    return (
        <>
            <SettingsTopBar
                display={display}
                onClose={onClose}
                title={t("Profile")}
                zIndex={zIndex}
            />
            <Stack direction="row">
                <SettingsList />
                <Box m="auto" mt={2}>
                    <Formik
                        enableReinitialize={true}
                        initialValues={{
                            bannerImage: profile?.bannerImage ?? null,
                            handle: profile?.handle ?? null,
                            name: profile?.name ?? "",
                            profileImage: profile?.profileImage ?? null,
                            translations: profile?.translations?.length ? profile.translations : [{
                                id: DUMMY_ID,
                                language: getUserLanguages(session)[0],
                                bio: "",
                            }],
                            updated_at: profile?.updated_at ?? null, // Used for cache busting on profile image
                        }}
                        onSubmit={(values, helpers) => {
                            if (!profile) {
                                PubSub.get().publishSnack({ messageKey: "CouldNotReadProfile", severity: "Error" });
                                return;
                            }
                            console.log("submitting profile update: values", values);
                            console.log("submitting profile update: profile", profile);
                            console.log("submitting profile update: shapeProfile.update(profile, values)", shapeProfile.update(profile, {
                                id: profile.id,
                                ...values,
                            }));
                            console.log("test1", createPrims(values, "profileImage"));
                            fetchLazyWrapper<ProfileUpdateInput, User>({
                                fetch,
                                inputs: shapeProfile.update(profile, {
                                    id: profile.id,
                                    ...values,
                                }),
                                successMessage: () => ({ messageKey: "SettingsUpdated" }),
                                onSuccess: (updated) => { onProfileUpdate(updated); },
                                onCompleted: () => { helpers.setSubmitting(false); },
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
                </Box>
            </Stack>
        </>
    );
};
