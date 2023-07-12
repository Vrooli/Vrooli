import { DUMMY_ID, endpointPutProfile, ProfileUpdateInput, User, userValidation } from "@local/shared";
import { Box, Stack } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { SettingsList } from "components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "components/navigation/SettingsTopBar/SettingsTopBar";
import { Formik } from "formik";
import { SettingsProfileForm } from "forms/settings";
import { useContext } from "react";
import { useTranslation } from "react-i18next";
import { getUserLanguages } from "utils/display/translationTools";
import { useLazyFetch } from "utils/hooks/useLazyFetch";
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
    const { t } = useTranslation();
    const session = useContext(SessionContext);

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
                            fetchLazyWrapper<ProfileUpdateInput, User>({
                                fetch,
                                inputs: shapeProfile.update(profile, {
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
                </Box>
            </Stack>
        </>
    );
};
