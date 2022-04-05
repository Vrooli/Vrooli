import { Box, Container, Grid, TextField, Typography } from "@mui/material"
import { useMutation } from "@apollo/client";
import { user } from "graphql/generated/user";
import { useCallback, useEffect, useMemo, useState } from "react";
import { mutationWrapper } from 'graphql/utils/wrappers';
import { APP_LINKS, profileUpdateSchema as validationSchema } from '@local/shared';
import { useFormik } from 'formik';
import { profileUpdateMutation } from "graphql/mutation";
import { formatForUpdate, getUserLanguages } from "utils";
import {
    Restore as CancelIcon,
    Save as SaveIcon,
} from '@mui/icons-material';
import { DialogActionItem } from "components/containers/types";
import { DialogActionsContainer } from "components/containers/DialogActionsContainer/DialogActionsContainer";
import { SettingsProfileProps } from "../types";
import { useLocation } from "wouter";
import { containerShadow } from "styles";
import { LanguageInput } from "components/inputs";

export const SettingsProfile = ({
    onUpdated,
    profile,
    session,
}: SettingsProfileProps) => {
    const [, setLocation] = useLocation();

    // Handle translations
    type Translation = {
        language: string;
        bio: string;
    };
    const [translations, setTranslations] = useState<Translation[]>([]);
    const deleteTranslation = useCallback((language: string) => {
        setTranslations([...translations.filter(t => t.language !== language)]);
    }, [translations, setTranslations]);
    const getTranslationsUpdate = useCallback((language: string, translation: Translation) => {
        // Find translation
        const index = translations.findIndex(t => language === t.language);
        // If language exists, update
        if (index >= 0) {
            const newTranslations = [...translations];
            newTranslations[index] = { ...translation };
            return newTranslations;
        }
        // Otherwise, add new
        else {
            return [...translations, translation];
        }
    }, [translations]);
    const updateTranslation = useCallback((language: string, translation: Translation) => {
        setTranslations(getTranslationsUpdate(language, translation));
    }, [translations, setTranslations]);

    useEffect(() => {
        setTranslations(profile?.translations?.map(t => ({
            id: t.id,
            language: t.language,
            bio: t.bio ?? '',
        })) ?? [{ language: getUserLanguages(session)[0], bio: '' }]);
    }, [profile, session]);

    // Handle update
    const [mutation] = useMutation<user>(profileUpdateMutation);
    const formik = useFormik({
        initialValues: {
            bio: '',
            username: profile?.username ?? '',
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            if (!formik.isValid) return;
            const allTranslations = getTranslationsUpdate(language, {
                language,
                bio: values.bio,
            })
            mutationWrapper({
                mutation,
                input: formatForUpdate(profile, {
                    ...values,
                    translations: allTranslations,
                }),
                onSuccess: (response) => { onUpdated(response.data.profileUpdate) },
            })
        },
    });

    // Handle languages
    const [language, setLanguage] = useState<string>('');
    const [languages, setLanguages] = useState<string[]>([]);
    useEffect(() => {
        if (languages.length === 0) {
            if (translations.length > 0) {
                setLanguage(translations[0].language);
                setLanguages(translations.map(t => t.language));
                formik.setValues({
                    ...formik.values,
                    bio: translations[0].bio,
                })
            } else {
                setLanguage(getUserLanguages(session)[0]);
                setLanguages([getUserLanguages(session)[0]]);
            }
        }
    }, [languages, setLanguage, setLanguages, translations])
    const handleLanguageChange = useCallback((oldLanguage: string, newLanguage: string) => {
        // Update translation
        updateTranslation(oldLanguage, {
            language: newLanguage,
            bio: formik.values.bio,
        });
        // Change selection
        setLanguage(newLanguage);
        // Update languages
        const newLanguages = [...languages];
        const index = newLanguages.findIndex(l => l === oldLanguage);
        if (index >= 0) {
            newLanguages[index] = newLanguage;
            setLanguages(newLanguages);
        }
    }, [formik.values, languages, translations, setLanguage, setLanguages, updateTranslation]);
    const updateFormikTranslation = useCallback((language: string) => {
        const existingTranslation = translations.find(t => t.language === language);
        formik.setValues({
            ...formik.values,
            bio: existingTranslation?.bio ?? '',
        });
    }, [formik.setValues, translations]);
    const handleLanguageSelect = useCallback((newLanguage: string) => {
        // Update old select
        updateTranslation(language, {
            language,
            bio: formik.values.bio,
        })
        // Update formik
        updateFormikTranslation(newLanguage);
        // Change language
        setLanguage(newLanguage);
    }, [formik.values, formik.setValues, language, translations, setLanguage, updateTranslation]);
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
    }, [deleteTranslation, handleLanguageSelect, languages, setLanguages]);

    const actions: DialogActionItem[] = useMemo(() => [
        ['Save', SaveIcon, !formik.touched || formik.isSubmitting, true, () => { }],
        ['Cancel', CancelIcon, !formik.touched || formik.isSubmitting, false, () => { setLocation(`${APP_LINKS.Settings}/?page=profile`, { replace: true }) }],
    ], [formik, setLocation]);

    return (
        <Box sx={{ minHeight: '100vh', display: 'flex', justifyContent: 'center', alignItems: 'center' }}>
            <Box sx={{
                ...containerShadow,
                borderRadius: 2,
                overflow: 'overlay',
                marginTop: '-5vh',
                background: (t) => t.palette.background.default,
                width: 'min(100%, 700px)',
            }}>
                <form onSubmit={formik.handleSubmit} style={{ overflow: 'hidden' }}>
                    {/* Title */}
                    <Box sx={{
                        background: (t) => t.palette.primary.dark,
                        color: (t) => t.palette.primary.contrastText,
                        padding: 0.5,
                        marginBottom: 2,
                    }}>
                        <Typography component="h1" variant="h3" textAlign="center">Update Profile</Typography>
                    </Box>
                    <Container sx={{ paddingBottom: 2 }}>
                        <Grid container spacing={2}>
                            <Grid item xs={12}>
                                <LanguageInput
                                    currentLanguage={language}
                                    handleAdd={handleAddLanguage}
                                    handleChange={handleLanguageChange}
                                    handleDelete={handleLanguageDelete}
                                    handleSelect={handleLanguageSelect}
                                    languages={languages}
                                    session={session}
                                />
                            </Grid>
                            <Grid item xs={12}>
                                <TextField
                                    fullWidth
                                    id="username"
                                    name="username"
                                    label="Username"
                                    value={formik.values.username}
                                    onBlur={formik.handleBlur}
                                    onChange={formik.handleChange}
                                    error={formik.touched.username && Boolean(formik.errors.username)}
                                    helperText={formik.touched.username && formik.errors.username}
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
                    <DialogActionsContainer fixed={false} actions={actions} />
                </form>
            </Box>
        </Box>
    )
}