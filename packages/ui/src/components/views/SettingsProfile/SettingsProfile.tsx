import { Autocomplete, Container, Grid, Stack, TextField, useTheme } from "@mui/material"
import { useLazyQuery, useMutation } from "@apollo/client";
import { useCallback, useEffect, useMemo, useState } from "react";
import { mutationWrapper } from 'graphql/utils/graphqlWrapper';
import { profileUpdateSchema as validationSchema, userTranslationUpdate } from '@shared/validation';
import { APP_LINKS } from '@shared/consts';
import { useFormik } from 'formik';
import { profileUpdateMutation } from "graphql/mutation";
import { addEmptyTranslation, getFormikErrorsWithTranslations, getTranslationData, getUserLanguages, handleTranslationBlur, handleTranslationChange, removeTranslation, shapeProfileUpdate, usePromptBeforeUnload } from "utils";
import { SettingsProfileProps } from "../types";
import { useLocation } from '@shared/route';
import { LanguageInput } from "components/inputs";
import { ColorIconButton, GridSubmitButtons } from "components/buttons";
import { findHandles, findHandlesVariables } from "graphql/generated/findHandles";
import { findHandlesQuery } from "graphql/query";
import { profileUpdateVariables, profileUpdate_profileUpdate } from "graphql/generated/profileUpdate";
import { PubSub } from 'utils'
import { RefreshIcon } from "@shared/icons";
import { DUMMY_ID, uuid } from '@shared/uuid';
import { PageTitle, SnackSeverity } from "components";
import { SettingsFormData } from "pages";

const helpText =
    `This page allows you to update your profile, including your name, handle, and bio.
    
Handles are unique, and handled (pun intended) by the [ADA Handle Protocol](https://adahandle.com/). This allows for handle to be 
used across many Cardano applications, and exchanged with others on an open market.

To use an ADA Handle, make sure it is in a wallet which is authenticated with your account.

If you set a handle, your Vrooli profile will be accessible via https://app.vrooli.com/profile/{handle}.

Your bio is displayed on your profile page. You may add multiple translations if you'd like.`;

export const SettingsProfile = ({
    onUpdated,
    profile,
    session,
    zIndex,
}: SettingsProfileProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();

    // Query for handles associated with the user
    const [findHandles, { data: handlesData, loading: handlesLoading }] = useLazyQuery<findHandles, findHandlesVariables>(findHandlesQuery);
    const [handles, setHandles] = useState<string[]>([]);
    const fetchHandles = useCallback(() => {
        const verifiedWallets = profile?.wallets?.filter(w => w.verified) ?? [];
        if (verifiedWallets.length > 0) {
            findHandles({ variables: { input: {} } }); // Intentionally empty
        } else {
            PubSub.get().publishSnack({ messageKey: 'NoVerifiedWallets', severity: SnackSeverity.Error })
        }
    }, [profile, findHandles]);
    useEffect(() => {
        if (handlesData?.findHandles) {
            setHandles(handlesData.findHandles);
        }
    }, [handlesData])

    const [selectedHandle, setSelectedHandle] = useState<string | null>(null);

    // Handle languages
    const [language, setLanguage] = useState<string>('');
    const [languages, setLanguages] = useState<string[]>([]);

    useEffect(() => {
        if (!profile) return;
        if (profile.handle) {
            setSelectedHandle(profile.handle);
        }
    }, [profile, session]);

    // Handle update
    const [mutation] = useMutation(profileUpdateMutation);
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
        validationSchema,
        onSubmit: (values) => {
            if (!profile) {
                PubSub.get().publishSnack({ messageKey: 'CouldNotReadProfile', severity: SnackSeverity.Error });
                return;
            }
            if (!formik.isValid) {
                PubSub.get().publishSnack({ messageKey: 'FixErrorsBeforeSubmitting', severity: SnackSeverity.Error });
                return;
            }
            const input = shapeProfileUpdate(profile, {
                id: profile.id,
                name: values.name,
                handle: selectedHandle,
                translations: values.translationsUpdate.map(t => ({
                    ...t,
                    id: t.id === DUMMY_ID ? uuid() : t.id,
                })),
            })
            if (!input || Object.keys(input).length === 0) {
                PubSub.get().publishSnack({ messageKey: 'NoChangesMade', severity: SnackSeverity.Info });
                return;
            }
            mutationWrapper<profileUpdate_profileUpdate, profileUpdateVariables>({
                mutation,
                input,
                onSuccess: (data) => { onUpdated(data); setLocation(APP_LINKS.Profile, { replace: true }) },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    // Current bio info, as well as errors
    const { bio, errorBio, touchedBio, errors } = useMemo(() => {
        const { error, touched, value } = getTranslationData(formik, 'translationsUpdate', language);
        return {
            bio: value?.bio ?? '',
            errorBio: error?.bio ?? '',
            touchedBio: touched?.bio ?? false,
            errors: getFormikErrorsWithTranslations(formik, 'translationsUpdate', userTranslationUpdate),
        }
    }, [formik, language]);
    // Handles blur on translation fields
    const onTranslationBlur = useCallback((e: { target: { name: string } }) => {
        handleTranslationBlur(formik, 'translationsUpdate', e, language)
    }, [formik, language]);
    // Handles change on translation fields
    const onTranslationChange = useCallback((e: { target: { name: string, value: string } }) => {
        handleTranslationChange(formik, 'translationsUpdate', e, language)
    }, [formik, language]);

    // Handle languages
    useEffect(() => {
        if (languages.length === 0 && formik.values.translationsUpdate.length > 0) {
            setLanguage(formik.values.translationsUpdate[0].language);
            setLanguages(formik.values.translationsUpdate.map(t => t.language));
        }
    }, [formik, languages, setLanguage, setLanguages])
    const handleLanguageSelect = useCallback((newLanguage: string) => { setLanguage(newLanguage) }, []);
    const handleAddLanguage = useCallback((newLanguage: string) => {
        setLanguages([...languages, newLanguage]);
        handleLanguageSelect(newLanguage);
        addEmptyTranslation(formik, 'translationsUpdate', newLanguage);
    }, [formik, handleLanguageSelect, languages]);
    const handleLanguageDelete = useCallback((language: string) => {
        const newLanguages = [...languages.filter(l => l !== language)]
        if (newLanguages.length === 0) return;
        setLanguage(newLanguages[0]);
        setLanguages(newLanguages);
        removeTranslation(formik, 'translationsUpdate', language);
    }, [formik, languages]);

    const handleCancel = useCallback(() => {
        setLocation(APP_LINKS.Profile, { replace: true })
    }, [setLocation]);

    return (
        <form onSubmit={formik.handleSubmit} style={{ overflow: 'hidden' }}>
            <PageTitle title="Update Profile" helpText={helpText} />
            <Container sx={{ paddingBottom: 2 }}>
                <Grid container spacing={2}>
                    <Grid item xs={12}>
                        <LanguageInput
                            currentLanguage={language}
                            handleAdd={handleAddLanguage}
                            handleDelete={handleLanguageDelete}
                            handleCurrent={handleLanguageSelect}
                            session={session}
                            translations={formik.values.translationsUpdate}
                            zIndex={zIndex}
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
                            value={bio}
                            onBlur={onTranslationBlur}
                            onChange={onTranslationChange}
                            error={touchedBio && Boolean(errorBio)}
                            helperText={touchedBio && errorBio}
                        />
                    </Grid>
                </Grid>
            </Container>
            <Grid container spacing={2} p={3}>
                <GridSubmitButtons
                    errors={errors}
                    isCreate={false}
                    loading={formik.isSubmitting}
                    onCancel={handleCancel}
                    onSetSubmitting={formik.setSubmitting}
                    onSubmit={formik.handleSubmit}
                />
            </Grid>
        </form>
    )
}

export const settingsProfileFormData: SettingsFormData = {
    labels: ['Update Profile', 'Profile Update', 'Update Account', 'Account Update'],
    items: [
        { id: 'ada-handle-select', labels: ['(ADA) Handle', 'ADA Handle', 'Cardano Handle', 'Account Handle', 'Profile Handle'] },
        { id: 'name', labels: ['Name', 'Full Name', 'Display Name'] },
        { id: 'bio', labels: ['Bio', 'Biography', 'About Me', 'About You', 'About'] },
    ],
}