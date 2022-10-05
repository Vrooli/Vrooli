import { useMutation, useQuery } from '@apollo/client';
import { resourceCreate as validationSchema, resourceTranslationUpdate } from '@shared/validation';
import { Dialog, DialogContent, FormControl, Grid, IconButton, InputLabel, ListItemIcon, ListItemText, MenuItem, Select, Stack, TextField, useTheme } from '@mui/material';
import { useFormik } from 'formik';
import { resourceCreateMutation, resourceUpdateMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/graphqlWrapper';
import { ResourceDialogProps } from '../types';
import { addEmptyTranslation, getFormikErrorsWithTranslations, getObjectSlug, getObjectUrlBase, getTranslationData, getUserLanguages, handleTranslationBlur, handleTranslationChange, listToAutocomplete, PubSub, removeTranslation, ResourceShape, shapeResourceCreate, shapeResourceUpdate, usePromptBeforeUnload } from 'utils';
import { resourceCreate, resourceCreateVariables } from 'graphql/generated/resourceCreate';
import { ResourceUsedFor } from 'graphql/generated/globalTypes';
import { resourceUpdate, resourceUpdateVariables } from 'graphql/generated/resourceUpdate';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { AutocompleteSearchBar, LanguageInput } from 'components/inputs';
import { AutocompleteOption, Resource } from 'types';
import { DUMMY_ID, uuid } from '@shared/uuid';
import { DialogTitle, getResourceIcon, GridSubmitButtons, SnackSeverity } from 'components';
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
    console.log('rendering resource dialog')
    const { palette } = useTheme();

    const [addMutation, { loading: addLoading }] = useMutation<resourceCreate, resourceCreateVariables>(resourceCreateMutation);
    const [updateMutation, { loading: updateLoading }] = useMutation<resourceUpdate, resourceUpdateVariables>(resourceUpdateMutation);

    const formik = useFormik({
        initialValues: {
            id: DUMMY_ID,
            link: partialData?.link ?? '',
            listId,
            usedFor: partialData?.usedFor ?? ResourceUsedFor.Context,
            translationsUpdate: partialData?.translations ?? [{
                id: DUMMY_ID,
                language: getUserLanguages(session)[0],
                description: '',
                title: '',
            }],
        },
        enableReinitialize: true,
        validationSchema,
        onSubmit: (values) => {
            console.log('resourcedialog onsubmit')
            const input: ResourceShape = {
                id: partialData?.id ?? uuid(),
                index: Math.max(index, 0),
                listId,
                link: values.link,
                usedFor: values.usedFor,
                translations: values.translationsUpdate.map(t => ({
                    ...t,
                    id: t.id === DUMMY_ID ? uuid() : t.id,
                })),
            };
            if (mutate) {
                const onSuccess = (response) => {
                    (index < 0) ? onCreated(response.data.resourceCreate) : onUpdated(index ?? 0, response.data.resourceUpdate);
                    formik.resetForm();
                    onClose();
                }
                // If index is negative, create
                if (index < 0) {
                    mutationWrapper({
                        mutation: addMutation,
                        input: shapeResourceCreate(input),
                        successMessage: () => 'Resource created.',
                        successCondition: (data) => data.resourceCreate !== null,
                        onSuccess,
                        onError: () => { formik.setSubmitting(false) },
                    })
                }
                // Otherwise, update
                else {
                    if (!partialData || !partialData.id || !listId) {
                        PubSub.get().publishSnack({ message: 'Could not find resource to update.', severity: SnackSeverity.Error });
                        return;
                    }
                    mutationWrapper({
                        mutation: updateMutation,
                        input: shapeResourceUpdate({ ...partialData, listId } as ResourceShape, input),
                        successMessage: () => 'Resource updated.',
                        successCondition: (data) => data.resourceUpdate !== null,
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
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    // Handle translations
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    const { description, title, errorDescription, errorTitle, touchedDescription, touchedTitle, errors } = useMemo(() => {
        const { error, touched, value } = getTranslationData(formik, 'translationsUpdate', language);
        return {
            description: value?.description ?? '',
            title: value?.title ?? '',
            errorDescription: error?.description ?? '',
            errorTitle: error?.title ?? '',
            touchedDescription: touched?.description ?? false,
            touchedTitle: touched?.title ?? false,
            errors: getFormikErrorsWithTranslations(formik, 'translationsUpdate', resourceTranslationUpdate),
        }
    }, [formik, language]);
    const languages = useMemo(() => formik.values.translationsUpdate.map(t => t.language), [formik.values.translationsUpdate]);
    const handleAddLanguage = useCallback((newLanguage: string) => {
        setLanguage(newLanguage);
        addEmptyTranslation(formik, 'translationsUpdate', newLanguage);
    }, [formik]);
    const handleLanguageDelete = useCallback((language: string) => {
        const newLanguages = [...languages.filter(l => l !== language)]
        if (newLanguages.length === 0) return;
        setLanguage(newLanguages[0]);
        removeTranslation(formik, 'translationsUpdate', language);
    }, [formik, languages]);
    // Handles blur on translation fields
    const onTranslationBlur = useCallback((e: { target: { name: string } }) => {
        console.log('resource dialog trans blur', e.target)
        handleTranslationBlur(formik, 'translationsUpdate', e, language)
    }, [formik, language]);
    // Handles change on translation fields
    const onTranslationChange = useCallback((e: { target: { name: string, value: string } }) => {
        console.log('resource dialog trans change', e.target)
        handleTranslationChange(formik, 'translationsUpdate', e, language)
    }, [formik, language]);

    const handleClose = () => {
        console.log('resourcedialog resetting form')
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
        // If value is not an object, return;
        if (!newValue || newValue.__typename === 'Shortcut' || newValue.__typename === 'Action') return;
        // Clear search string and close command palette
        closeSearch();
        // Create URL
        const newLocation = `${getObjectUrlBase(newValue)}/${getObjectSlug(newValue)}`
        // Update link
        console.log('resourcedialog oninputselect')
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
                                handleCurrent={setLanguage}
                                session={session}
                                translations={formik.values.translationsUpdate}
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
                                <InputLabel id="resource-type-label">Type</InputLabel>
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
                                value={title}
                                onBlur={onTranslationBlur}
                                onChange={onTranslationChange}
                                error={touchedTitle && Boolean(errorTitle)}
                                helperText={(touchedTitle && errorTitle) ?? 'Enter title (optional)'}
                            />
                            {/* Enter description */}
                            <TextField
                                fullWidth
                                id="description"
                                name="description"
                                label="Description"
                                multiline
                                maxRows={8}
                                value={description}
                                onBlur={onTranslationBlur}
                                onChange={onTranslationChange}
                                error={touchedDescription && Boolean(errorDescription)}
                                helperText={(touchedDescription && errorDescription) ?? 'Enter description (optional)'}
                            />
                            {/* Action buttons */}
                            <Grid container spacing={1}>
                                <GridSubmitButtons
                                    errors={errors}
                                    isCreate={index < 0}
                                    loading={formik.isSubmitting || addLoading || updateLoading}
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