import { Autocomplete, Box, Container, Grid, IconButton, Stack, TextField, Typography, useTheme } from "@mui/material"
import { useLazyQuery, useMutation } from "@apollo/client";
import { useCallback, useEffect, useState } from "react";
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { profileUpdateSchema as validationSchema } from '@shared/validation';
import { APP_LINKS } from '@shared/consts';
import { useFormik } from 'formik';
import { profileUpdateMutation } from "graphql/mutation";
import { getUserLanguages, ProfileTranslationShape, shapeProfileUpdate, updateArray } from "utils";
import { SettingsProfileProps } from "../types";
import { useLocation } from '@shared/route';
import { LanguageInput } from "components/inputs";
import { GridSubmitButtons, HelpButton } from "components/buttons";
import { findHandles, findHandlesVariables } from "graphql/generated/findHandles";
import { findHandlesQuery } from "graphql/query";
import { profileUpdate, profileUpdateVariables } from "graphql/generated/profileUpdate";
import { v4 as uuid } from 'uuid';
import { PubSub } from 'utils'
import { RefreshIcon } from "@shared/icons";

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
            PubSub.get().publishSnack({ message: 'No verified wallets associated with account', severity: 'error' })
        }
    }, [profile, findHandles]);
    useEffect(() => {
        if (handlesData?.findHandles) {
            setHandles(handlesData.findHandles);
        }
    }, [handlesData])

    const [selectedHandle, setSelectedHandle] = useState<string | null>(null);

    // Handle translations
    type Translation = ProfileTranslationShape;
    const [translations, setTranslations] = useState<Translation[]>([]);
    const deleteTranslation = useCallback((language: string) => {
        setTranslations([...translations.filter(t => t.language !== language)]);
    }, [translations, setTranslations]);
    const getTranslationsUpdate = useCallback((language: string, translation: Translation) => {
        // Find translation
        const index = translations.findIndex(t => language === t.language);
        // Add to array, or update if found
        return index >= 0 ? updateArray(translations, index, translation) : [...translations, translation];
    }, [translations]);
    const updateTranslation = useCallback((language: string, translation: Translation) => {
        setTranslations(getTranslationsUpdate(language, translation));
    }, [getTranslationsUpdate]);

    useEffect(() => {
        if (!profile) return;
        if (profile.handle) {
            setSelectedHandle(profile.handle);
        }
        const foundTranslations = profile.translations?.map(t => ({
            id: t.id,
            language: t.language,
            bio: t.bio ?? '',
        })) ?? []
        // If no translations found, add default
        if (foundTranslations.length === 0) {
            setTranslations([{
                id: uuid(),
                language: getUserLanguages(session)[0],
                bio: '',
            }]);
        } else {
            setTranslations(foundTranslations);
        }
    }, [profile, session]);

    // Handle update
    const [mutation] = useMutation<profileUpdate, profileUpdateVariables>(profileUpdateMutation);
    const formik = useFormik({
        initialValues: {
            bio: '',
            name: profile?.name ?? '',
        },
        enableReinitialize: true,
        validationSchema,
        onSubmit: (values) => {
            if (!profile) {
                PubSub.get().publishSnack({ message: 'Could not find existing data.', severity: 'error' });
                return;
            }
            if (!formik.isValid) {
                PubSub.get().publishSnack({ message: 'Please fix errors before submitting.', severity: 'error' });
                return;
            }
            const allTranslations = getTranslationsUpdate(language, {
                id: uuid(),
                language,
                bio: values.bio,
            })
            const input = shapeProfileUpdate(profile, {
                id: profile.id,
                name: values.name,
                handle: selectedHandle,
                translations: allTranslations,
            })
            if (!input || Object.keys(input).length === 0) {
                PubSub.get().publishSnack({ message: 'No changes made.' });
                return;
            }
            mutationWrapper({
                mutation,
                input,
                onSuccess: (response) => { onUpdated(response.data.profileUpdate); setLocation(APP_LINKS.Profile, { replace: true }) },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });

    // Handle languages
    const [language, setLanguage] = useState<string>('');
    const [languages, setLanguages] = useState<string[]>([]);
    useEffect(() => {
        if (languages.length === 0 && translations.length > 0) {
            setLanguage(translations[0].language);
            setLanguages(translations.map(t => t.language));
            formik.setFieldValue('bio', translations[0].bio ?? '');
        }
    }, [formik, languages, setLanguage, setLanguages, translations])
    const updateFormikTranslation = useCallback((language: string) => {
        const existingTranslation = translations.find(t => t.language === language);
        formik.setFieldValue('bio', existingTranslation?.bio ?? '');
    }, [formik, translations]);
    const handleLanguageSelect = useCallback((newLanguage: string) => {
        // Update old select
        updateTranslation(language, {
            id: uuid(),
            language,
            bio: formik.values.bio,
        })
        // Update formik
        if (language !== newLanguage) updateFormikTranslation(newLanguage);
        // Change language
        setLanguage(newLanguage);
    }, [updateTranslation, language, formik.values.bio, updateFormikTranslation]);
    const handleAddLanguage = useCallback((newLanguage: string) => {
        setLanguages([...languages, newLanguage]);
        handleLanguageSelect(newLanguage);
    }, [handleLanguageSelect, languages, setLanguages]);
    const handleLanguageDelete = useCallback((language: string) => {
        const newLanguages = [...languages.filter(l => l !== language)]
        if (newLanguages.length === 0) return;
        deleteTranslation(language);
        updateFormikTranslation(newLanguages[0]);
        setLanguage(newLanguages[0]);
        setLanguages(newLanguages);
    }, [deleteTranslation, languages, updateFormikTranslation]);

    const handleCancel = useCallback(() => {
        setLocation(APP_LINKS.Profile, { replace: true })
    }, [setLocation]);

    return (
        <form onSubmit={formik.handleSubmit} style={{ overflow: 'hidden' }}>
            {/* Title */}
            <Box sx={{
                background: palette.primary.dark,
                color: palette.primary.contrastText,
                padding: 0.5,
                marginBottom: 2,
                display: 'flex',
                justifyContent: 'center',
                alignItems: 'center',
            }}>
                <Typography component="h1" variant="h4" textAlign="center">Update Profile</Typography>
                <HelpButton markdown={helpText} />
            </Box>
            <Container sx={{ paddingBottom: 2 }}>
                <Grid container spacing={2}>
                    <Grid item xs={12}>
                        <LanguageInput
                            currentLanguage={language}
                            handleAdd={handleAddLanguage}
                            handleDelete={handleLanguageDelete}
                            handleCurrent={handleLanguageSelect}
                            selectedLanguages={languages}
                            session={session}
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
                                renderInput={(params) => <TextField {...params} label="(ADA) Handle" />}
                                value={selectedHandle}
                                sx={{
                                    width: '100%',
                                }}
                            />
                            <IconButton
                                aria-label='fetch-handles'
                                onClick={fetchHandles}
                                sx={{
                                    background: palette.secondary.main,
                                    borderRadius: '0 5px 5px 0',
                                }}>
                                <RefreshIcon />
                            </IconButton>
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
                            value={formik.values.bio}
                            onBlur={formik.handleBlur}
                            onChange={formik.handleChange}
                            error={formik.touched.bio && Boolean(formik.errors.bio)}
                            helperText={formik.touched.bio && formik.errors.bio}
                        />
                    </Grid>
                </Grid>
            </Container>
            <Grid container spacing={2} p={3}>
                <GridSubmitButtons
                    disabledCancel={formik.isSubmitting}
                    disabledSubmit={formik.isSubmitting || !formik.isValid}
                    errors={formik.errors}
                    isCreate={false}
                    onCancel={handleCancel}
                    onSetSubmitting={formik.setSubmitting}
                    onSubmit={formik.handleSubmit}
                />
            </Grid>
        </form>
    )
}