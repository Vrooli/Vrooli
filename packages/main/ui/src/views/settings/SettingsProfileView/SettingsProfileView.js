import { jsx as _jsx, jsxs as _jsxs, Fragment as _Fragment } from "react/jsx-runtime";
import { DUMMY_ID } from "@local/uuid";
import { userValidation } from "@local/validation";
import { Stack } from "@mui/material";
import { Formik } from "formik";
import { useContext } from "react";
import { userProfileUpdate } from "../../../api/generated/endpoints/user_profileUpdate";
import { useCustomMutation } from "../../../api/hooks";
import { mutationWrapper } from "../../../api/utils";
import { SettingsList } from "../../../components/lists/SettingsList/SettingsList";
import { SettingsTopBar } from "../../../components/navigation/SettingsTopBar/SettingsTopBar";
import { SettingsProfileForm } from "../../../forms/settings";
import { getUserLanguages } from "../../../utils/display/translationTools";
import { useProfileQuery } from "../../../utils/hooks/useProfileQuery";
import { PubSub } from "../../../utils/pubsub";
import { SessionContext } from "../../../utils/SessionContext";
import { shapeProfile } from "../../../utils/shape/models/profile";
export const SettingsProfileView = ({ display = "page", }) => {
    const session = useContext(SessionContext);
    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery();
    const [mutation, { loading: isUpdating }] = useCustomMutation(userProfileUpdate);
    return (_jsxs(_Fragment, { children: [_jsx(SettingsTopBar, { display: display, onClose: () => { }, titleData: {
                    titleKey: "Profile",
                } }), _jsxs(Stack, { direction: "row", children: [_jsx(SettingsList, {}), _jsx(Formik, { enableReinitialize: true, initialValues: {
                            handle: profile?.handle ?? null,
                            name: profile?.name ?? "",
                            translations: profile?.translations?.length ? profile.translations : [{
                                    id: DUMMY_ID,
                                    language: getUserLanguages(session)[0],
                                    bio: "",
                                }],
                        }, onSubmit: (values, helpers) => {
                            if (!profile) {
                                PubSub.get().publishSnack({ messageKey: "CouldNotReadProfile", severity: "Error" });
                                return;
                            }
                            mutationWrapper({
                                mutation,
                                input: shapeProfile.update(profile, {
                                    id: profile.id,
                                    ...values,
                                }),
                                successMessage: () => ({ key: "SettingsUpdated" }),
                                onError: () => { helpers.setSubmitting(false); },
                            });
                        }, validationSchema: userValidation.update({}), children: (formik) => _jsx(SettingsProfileForm, { display: display, isLoading: isProfileLoading || isUpdating, numVerifiedWallets: profile?.wallets?.filter(w => w.verified)?.length ?? 0, onCancel: formik.resetForm, ...formik }) })] })] }));
};
//# sourceMappingURL=SettingsProfileView.js.map