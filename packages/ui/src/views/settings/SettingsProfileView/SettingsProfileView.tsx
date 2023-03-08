import { Autocomplete, Grid, Stack, TextField, useTheme } from "@mui/material"
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { useCallback, useEffect, useState } from "react";
import { mutationWrapper } from 'api/utils';
import { APP_LINKS, FindHandlesInput, ProfileUpdateInput, User } from '@shared/consts';
import { useFormik } from 'formik';
import { getUserLanguages, shapeProfile, useProfileQuery, usePromptBeforeUnload, useTranslatedFields } from "utils";
import { SettingsProfileViewProps } from "../types";
import { useLocation } from '@shared/route';
import { LanguageInput } from "components/inputs";
import { ColorIconButton, GridSubmitButtons } from "components/buttons";
import { PubSub } from 'utils'
import { RefreshIcon } from "@shared/icons";
import { DUMMY_ID } from '@shared/uuid';
import { userTranslationValidation, userValidation } from "@shared/validation";
import { walletFindHandles } from "api/generated/endpoints/wallet_findHandles";
import { userProfileUpdate } from "api/generated/endpoints/user_profileUpdate";
import { SettingsList, SettingsTopBar } from "components";
import { BaseForm } from "forms";

export const SettingsProfileView = ({
    display = 'page',
    session,
}: SettingsProfileViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    const { isProfileLoading, onProfileUpdate, profile } = useProfileQuery(session);

    // Query for handles associated with the user
    const [findHandles, { data: handlesData, loading: handlesLoading }] = useCustomLazyQuery<string[], FindHandlesInput>(walletFindHandles);
    const [handles, setHandles] = useState<string[]>([]);
    const fetchHandles = useCallback(() => {
        const verifiedWallets = profile?.wallets?.filter(w => w.verified) ?? [];
        if (verifiedWallets.length > 0) {
            findHandles({ variables: {} }); // Intentionally empty
        } else {
            PubSub.get().publishSnack({ messageKey: 'NoVerifiedWallets', severity: 'Error' })
        }
    }, [profile, findHandles]);
    useEffect(() => {
        if (handlesData) {
            setHandles(handlesData);
        }
    }, [handlesData])

    const [selectedHandle, setSelectedHandle] = useState<string | null>(null);

    useEffect(() => {
        if (!profile) return;
        if (profile.handle) {
            setSelectedHandle(profile.handle);
        }
    }, [profile, session]);

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
                handle: selectedHandle,
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

    const {
        handleAddLanguage,
        handleDeleteLanguage,
        language,
        languages,
        onTranslationBlur,
        onTranslationChange,
        setLanguage,
        translations,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ['bio'],
        formik,
        formikField: 'translationsUpdate',
        validationSchema: userTranslationValidation.update({}),
    });
    useEffect(() => {
        if (languages.length === 0 && formik.values.translationsUpdate.length > 0) {
            setLanguage(formik.values.translationsUpdate[0].language);
        }
    }, [formik, languages.length, setLanguage])

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
                    titleKey: 'Profile',
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
                        <Grid item xs={12}>
                            <LanguageInput
                                currentLanguage={language}
                                handleAdd={handleAddLanguage}
                                handleDelete={handleDeleteLanguage}
                                handleCurrent={setLanguage}
                                session={session}
                                translations={formik.values.translationsUpdate}
                                zIndex={200}
                            />
                        </Grid>
                        <Grid item xs={12}>
                            <Stack direction="row" spacing={0}>
                                <Autocomplete
                                    disablePortal
                                    id="ada-handle-select"
                                    loading={handlesLoading}
                                    options={handles}
                                    onOpen={fetchHandles}
                                    onChange={(_, value) => { setSelectedHandle(value) }}
                                    renderInput={(params) => <TextField
                                        {...params}
                                        label="(ADA) Handle"
                                        sx={{
                                            '& .MuiInputBase-root': {
                                                borderRadius: '5px 0 0 5px',
                                            }
                                        }}
                                    />}
                                    value={selectedHandle}
                                    sx={{
                                        width: '100%',
                                    }}
                                />
                                <ColorIconButton
                                    aria-label='fetch-handles'
                                    background={palette.secondary.main}
                                    onClick={fetchHandles}
                                    sx={{ borderRadius: '0 5px 5px 0' }}
                                >
                                    <RefreshIcon />
                                </ColorIconButton>
                            </Stack>
                        </Grid>
                        <Grid item xs={12}>
                            <TextField
                                fullWidth
                                id="name"
                                name="name"
                                label="Name"
                                value={formik.values.name}
                                onBlur={formik.handleBlur}
                                onChange={formik.handleChange}
                                error={formik.touched.name && Boolean(formik.errors.name)}
                                helperText={formik.touched.name && formik.errors.name}
                            />
                        </Grid>
                        <Grid item xs={12}>
                            <TextField
                                fullWidth
                                id="bio"
                                name="bio"
                                label="Bio"
                                multiline
                                minRows={4}
                                value={translations.bio}
                                onBlur={onTranslationBlur}
                                onChange={onTranslationChange}
                                error={translations.touchedBio && Boolean(translations.errorBio)}
                                helperText={translations.touchedBio && translations.errorBio}
                            />
                        </Grid>
                    </Grid>
                    <GridSubmitButtons
                        display={display}
                        errors={translations.errorsWithTranslations}
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