import { useMutation, useQuery } from '@apollo/client';
import { resourceCreateForm as validationSchema } from '@shared/validation';
import { Dialog, DialogContent, FormControl, Grid, IconButton, InputLabel, ListItemIcon, ListItemText, MenuItem, Select, Stack, TextField, useTheme } from '@mui/material';
import { useFormik } from 'formik';
import { resourceCreateMutation, resourceUpdateMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { ResourceDialogProps } from '../types';
import { getObjectSlug, getObjectUrlBase, getTranslation, getUserLanguages, listToAutocomplete, PubSub, ResourceShape, ResourceTranslationShape, shapeResourceCreate, shapeResourceUpdate, updateArray } from 'utils';
import { resourceCreate, resourceCreateVariables } from 'graphql/generated/resourceCreate';
import { ResourceUsedFor } from 'graphql/generated/globalTypes';
import { resourceUpdate, resourceUpdateVariables } from 'graphql/generated/resourceUpdate';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { AutocompleteSearchBar, LanguageInput } from 'components/inputs';
import { AutocompleteOption, Resource } from 'types';
import { v4 as uuid } from 'uuid';
import { DialogTitle, getResourceIcon, GridSubmitButtons } from 'components';
import { SearchIcon } from '@shared/icons';
import { homePage, homePageVariables } from 'graphql/generated/homePage';
import { homePageQuery } from 'graphql/query';

const helpText =
    `## What are resources?

Resources provide context to the object they are attached to, such as a  user, organization, project, or routine.

## Examples
**For a user** - Social media links, GitHub profile, Patreon

**For an organization** - Official website, tools used by your team, news article explaining the vision

**For a project** - Project Catalyst proposal, Donation wallet address

**For a routine** - Guide, external service
`

export const UsedForDisplay: { [key in ResourceUsedFor]: string } = {
    [ResourceUsedFor.Community]: 'Community',
    [ResourceUsedFor.Context]: 'Context',
    [ResourceUsedFor.Developer]: 'Developer',
    [ResourceUsedFor.Donation]: 'Donation',
    [ResourceUsedFor.ExternalService]: 'External Service',
    [ResourceUsedFor.Feed]: 'Feed',
    [ResourceUsedFor.Install]: 'Install',
    [ResourceUsedFor.Learning]: 'Learning',
    [ResourceUsedFor.Notes]: 'Notes',
    [ResourceUsedFor.OfficialWebsite]: 'Official Webiste',
    [ResourceUsedFor.Proposal]: 'Proposal',
    [ResourceUsedFor.Related]: 'Related',
    [ResourceUsedFor.Researching]: 'Researching',
    [ResourceUsedFor.Scheduling]: 'Scheduling',
    [ResourceUsedFor.Social]: 'Social',
    [ResourceUsedFor.Tutorial]: 'Tutorial',
}

const titleAria = "resource-dialog-title";
const searchTitleAria = "search-vrooli-for-link-title"

export const ResourceDialog = ({
    mutate,
    open,
    onClose,
    onCreated,
    onUpdated,
    index,
    partialData,
    session,
    listId,
    zIndex,
}: ResourceDialogProps) => {
    const { palette } = useTheme();

    const [addMutation, { loading: addLoading }] = useMutation<resourceCreate, resourceCreateVariables>(resourceCreateMutation);
    const [updateMutation, { loading: updateLoading }] = useMutation<resourceUpdate, resourceUpdateVariables>(resourceUpdateMutation);

    // Handle translations
    type Translation = ResourceTranslationShape;
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
        setTranslations(partialData?.translations?.map(t => ({
            id: t.id,
            language: t.language,
            description: t.description ?? '',
            title: t.title ?? '',
        })) ?? []);
    }, [partialData]);

    const [language, setLanguage] = useState<string>(index < 0 ? getUserLanguages(session)[0] : '');
    const [languages, setLanguages] = useState<string[]>(index < 0 ? getUserLanguages(session) : []);

    const formik = useFormik({
        initialValues: {
            link: partialData?.link ?? '',
            usedFor: partialData?.usedFor ?? ResourceUsedFor.Context,
            title: getTranslation(partialData, 'title', [language], false) ?? '',
            description: getTranslation(partialData, 'description', [language], false) ?? '',
        },
        enableReinitialize: true,
        validationSchema,
        onSubmit: (values) => {
            const allTranslations = getTranslationsUpdate(language, {
                id: uuid(),
                language,
                description: values.description,
                title: values.title,
            })
            const input: ResourceShape = {
                id: partialData?.id ?? uuid(),
                index: Math.max(index, 0),
                listId,
                link: values.link,
                usedFor: values.usedFor,
                translations: allTranslations,
            };
            if (mutate) {
                const onSuccess = (response) => {
                    PubSub.get().publishSnack({ message: (index < 0) ? 'Resource created.' : 'Resource updated.' });
                    (index < 0) ? onCreated(response.data.resourceCreate) : onUpdated(index ?? 0, response.data.resourceUpdate);
                    formik.resetForm();
                    onClose();
                }
                // If index is negative, create
                if (index < 0) {
                    mutationWrapper({
                        mutation: addMutation,
                        input: shapeResourceCreate(input),
                        successCondition: (response) => response.data.resourceCreate !== null,
                        onSuccess,
                        onError: () => { formik.setSubmitting(false) },
                    })
                }
                // Otherwise, update
                else {
                    if (!partialData || !partialData.id || !listId) {
                        PubSub.get().publishSnack({ message: 'Could not find resource to update.', severity: 'error' });
                        return;
                    }
                    mutationWrapper({
                        mutation: updateMutation,
                        input: shapeResourceUpdate({ ...partialData, listId } as ResourceShape, input),
                        successCondition: (response) => response.data.resourceUpdate !== null,
                        onSuccess,
                        onError: () => { formik.setSubmitting(false) },
                    })
                }
            } else {
                onCreated(input as Resource);
                formik.resetForm();
                onClose();
            }
        },
    });

    // Handle languages
    useEffect(() => {
        if (languages.length === 0 && translations.length > 0) {
            setLanguage(translations[0].language);
            setLanguages(translations.map(t => t.language));
            formik.setValues({
                ...formik.values,
                description: translations[0].description ?? '',
                title: translations[0].title ?? '',
            })
        }
    }, [formik, languages, setLanguage, setLanguages, translations])
    const updateFormikTranslation = useCallback((language: string) => {
        const existingTranslation = translations.find(t => t.language === language);
        formik.setValues({
            ...formik.values,
            description: existingTranslation?.description ?? '',
            title: existingTranslation?.title ?? '',
        });
    }, [formik, translations]);
    const handleLanguageSelect = useCallback((newLanguage: string) => {
        // Update old select
        updateTranslation(language, {
            id: uuid(),
            language,
            description: formik.values.description,
            title: formik.values.title,
        })
        // Update formik
        if (language !== newLanguage) updateFormikTranslation(newLanguage);
        // Change language
        setLanguage(newLanguage);
    }, [updateTranslation, language, formik.values.description, formik.values.title, updateFormikTranslation]);
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

    const handleClose = () => {
        formik.resetForm();
        onClose();
    }

    // Search dialog to find routines, organizations, etc. to link to
    const [searchString, setSearchString] = useState<string>('');
    const updateSearch = useCallback((newValue: any) => { setSearchString(newValue) }, []);
    const [searchOpen, setSearchOpen] = useState(false);
    const openSearch = useCallback(() => {
        setSearchString('');
        setSearchOpen(true)
    }, []);
    const closeSearch = useCallback(() => {
        setSearchOpen(false)
    }, []);
    const { data: searchData, refetch: refetchSearch, loading: searchLoading } = useQuery<homePage, homePageVariables>(homePageQuery, { variables: { input: { searchString: searchString.replaceAll(/![^\s]{1,}/g, '') } }, errorPolicy: 'all' });
    useEffect(() => { open && refetchSearch() }, [open, refetchSearch, searchString]);
    const autocompleteOptions: AutocompleteOption[] = useMemo(() => {
        const firstResults: AutocompleteOption[] = [];
        // Group all query results and sort by number of stars
        const flattened = (Object.values(searchData?.homePage ?? [])).reduce((acc, curr) => acc.concat(curr), []);
        const queryItems = listToAutocomplete(flattened, languages).sort((a: any, b: any) => {
            return b.stars - a.stars;
        });
        return [...firstResults, ...queryItems];
    }, [languages, searchData]);
    /**
     * When an autocomplete item is selected, set link as its URL
     */
    const onInputSelect = useCallback((newValue: AutocompleteOption) => {
        if (!newValue) return;
        // Clear search string and close command palette
        closeSearch();
        // Replace current state with search string, so that search is not lost. 
        // Only do this if the selected item is not a shortcut
        let newLocation: string;
        // If selected item is a shortcut, newLocation is in the id field
        if (newValue.__typename === 'Shortcut') {
            newLocation = newValue.id
        }
        // Otherwise, object url must be constructed
        else {
            newLocation = `${getObjectUrlBase(newValue)}/${getObjectSlug(newValue)}`
        }
        // Update link
        formik.setFieldValue('link', `${window.location.origin}${newLocation}`);
    }, [closeSearch, formik]);


    return (
        <>
            {/* Search objects (for their URL) dialog */}
            <Dialog
                open={searchOpen}
                onClose={closeSearch}
                aria-labelledby={searchTitleAria}
                sx={{
                    '& .MuiDialog-paper': {
                        border: palette.mode === 'dark' ? `1px solid white` : 'unset',
                        minWidth: 'min(100%, 600px)',
                        minHeight: 'min(100%, 200px)',
                        position: 'absolute',
                        top: '5%',
                        overflowY: 'visible',
                    }
                }}
            >
                <DialogTitle
                    ariaLabel={searchTitleAria}
                    title={'Search Vrooli'}
                    onClose={closeSearch}
                />
                <DialogContent sx={{
                    background: palette.background.default,
                    position: 'relative',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    overflowY: 'visible',
                }}>
                    {/* Search bar to find object */}
                    <AutocompleteSearchBar
                        id="vrooli-object-search"
                        autoFocus={true}
                        placeholder='Search Vrooli for objects to link'
                        options={autocompleteOptions}
                        loading={searchLoading}
                        value={searchString}
                        onChange={updateSearch}
                        onInputChange={onInputSelect}
                        session={session}
                        showSecondaryLabel={true}
                        sxs={{
                            root: { width: '100%' },
                            paper: { background: palette.background.paper },
                        }}
                    />
                    {/* If object selected (and supports versioning), version selector */}
                    {/* TODO */}
                </DialogContent>
            </Dialog >
            {/*  Main content */}
            <Dialog
                onClose={handleClose}
                open={open}
                aria-labelledby={titleAria}
                sx={{
                    zIndex,
                    '& .MuiDialog-paper': {
                        width: 'min(500px, 100vw)',
                        textAlign: 'center',
                        overflow: 'hidden',
                    }
                }}
            >
                <DialogTitle
                    ariaLabel={titleAria}
                    title={(index < 0) ? 'Add Resource' : 'Update Resource'}
                    helpText={helpText}
                    onClose={handleClose}
                />
                <DialogContent>
                    <form>
                        <Stack direction="column" spacing={2} paddingTop={2}>
                            {/* Language select */}
                            <LanguageInput
                                currentLanguage={language}
                                handleAdd={handleAddLanguage}
                                handleDelete={handleLanguageDelete}
                                handleCurrent={handleLanguageSelect}
                                selectedLanguages={languages}
                                session={session}
                                zIndex={zIndex + 1}
                            />
                            {/* Enter link or search for object */}
                            <Stack direction="row" spacing={0}>
                                <TextField
                                    fullWidth
                                    id="link"
                                    name="link"
                                    label="Link"
                                    value={formik.values.link}
                                    onBlur={formik.handleBlur}
                                    onChange={formik.handleChange}
                                    error={formik.touched.link && Boolean(formik.errors.link)}
                                    helperText={formik.touched.link && formik.errors.link}
                                    sx={{
                                        '& .MuiInputBase-root': {
                                            borderRadius: '5px 0 0 5px',
                                        }
                                    }}
                                />
                                <IconButton
                                    aria-label='find URL'
                                    onClick={openSearch}
                                    sx={{
                                        background: palette.secondary.main,
                                        borderRadius: '0 5px 5px 0',
                                        height: '56px',
                                    }}>
                                    <SearchIcon />
                                </IconButton>
                            </Stack>
                            {/* Select resource type */}
                            <FormControl fullWidth>
                                <InputLabel id="resource-type-label">Reason</InputLabel>
                                <Select
                                    labelId="resource-type-label"
                                    id="usedFor"
                                    value={formik.values.usedFor}
                                    label="Resource type"
                                    onChange={(e) => formik.setFieldValue('usedFor', e.target.value)}
                                    sx={{
                                        '& .MuiSelect-select': {
                                            display: 'flex',
                                            alignItems: 'center',
                                            justifyContent: 'center',
                                            textAlign: 'left',
                                        },
                                    }}
                                >
                                    {Object.entries(UsedForDisplay).map(([key, value]) => {
                                        const Icon = getResourceIcon(key as ResourceUsedFor);
                                        return (
                                            <MenuItem key={key} value={key}>
                                                <ListItemIcon>
                                                    <Icon fill={palette.background.textSecondary} />
                                                </ListItemIcon>
                                                <ListItemText>{value}</ListItemText>
                                            </MenuItem>
                                        )
                                    })}
                                </Select>
                            </FormControl>
                            {/* Enter title */}
                            <TextField
                                fullWidth
                                id="title"
                                name="title"
                                label="Title"
                                value={formik.values.title}
                                onBlur={formik.handleBlur}
                                onChange={formik.handleChange}
                                error={formik.touched.title && Boolean(formik.errors.title)}
                                helperText={(formik.touched.title && formik.errors.title) ?? "Enter title (optional)"}
                            />
                            {/* Enter description */}
                            <TextField
                                fullWidth
                                id="description"
                                name="description"
                                label="Description"
                                multiline
                                maxRows={8}
                                value={formik.values.description}
                                onBlur={formik.handleBlur}
                                onChange={formik.handleChange}
                                error={formik.touched.description && Boolean(formik.errors.description)}
                                helperText={(formik.touched.description && formik.errors.description) ?? "Enter description (optional)"}
                            />
                            {/* Action buttons */}
                            <Grid container spacing={1}>
                                <GridSubmitButtons
                                    disabledCancel={formik.isSubmitting || addLoading || updateLoading}
                                    disabledSubmit={formik.isSubmitting || !formik.isValid || addLoading || updateLoading}
                                    errors={formik.errors}
                                    isCreate={index < 0}
                                    onCancel={handleClose}
                                    onSetSubmitting={formik.setSubmitting}
                                    onSubmit={formik.handleSubmit}
                                />
                            </Grid>
                        </Stack>
                    </form>
                </DialogContent>
            </Dialog>
        </>
    )
}