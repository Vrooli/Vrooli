import { useQuery } from '@apollo/client';
import { DialogContent, FormControl, Grid, InputLabel, ListItemIcon, ListItemText, MenuItem, Select, Stack, TextField, useTheme } from '@mui/material';
import { PopularInput, PopularResult, Resource, ResourceCreateInput, ResourceList, ResourceUpdateInput, ResourceUsedFor } from '@shared/consts';
import { SearchIcon } from '@shared/icons';
import { CommonKey } from '@shared/translations';
import { DUMMY_ID } from '@shared/uuid';
import { resourceTranslationValidation, resourceValidation } from '@shared/validation';
import { feedPopular } from 'api/generated/endpoints/feed_popular';
import { resourceCreate } from 'api/generated/endpoints/resource_create';
import { resourceUpdate } from 'api/generated/endpoints/resource_update';
import { useCustomMutation } from 'api/hooks';
import { mutationWrapper } from 'api/utils';
import { ColorIconButton } from 'components/buttons/ColorIconButton/ColorIconButton';
import { GridSubmitButtons } from 'components/buttons/GridSubmitButtons/GridSubmitButtons';
import { LanguageInput } from 'components/inputs/LanguageInput/LanguageInput';
import { SiteSearchBar } from 'components/inputs/search';
import { useFormik } from 'formik';
import { BaseForm } from 'forms/BaseForm/BaseForm';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { AutocompleteOption, Wrap } from 'types';
import { getResourceIcon } from 'utils/display/getResourceIcon';
import { listToAutocomplete } from 'utils/display/listTools';
import { getUserLanguages } from 'utils/display/translationTools';
import { usePromptBeforeUnload } from 'utils/hooks/usePromptBeforeUnload';
import { useTranslatedFields } from 'utils/hooks/useTranslatedFields';
import { getObjectUrl } from 'utils/navigation/openObject';
import { PubSub } from 'utils/pubsub';
import { ResourceShape, shapeResource } from 'utils/shape/models/resource';
import { DialogTitle } from '../DialogTitle/DialogTitle';
import { LargeDialog } from '../LargeDialog/LargeDialog';
import { ResourceDialogProps } from '../types';

const helpText =
    `## What are resources?\n\nResources provide context to the object they are attached to, such as a  user, organization, project, or routine.\n\n## Examples\n**For a user** - Social media links, GitHub profile, Patreon\n\n**For an organization** - Official website, tools used by your team, news article explaining the vision\n\n**For a project** - Project Catalyst proposal, Donation wallet address\n\n**For a routine** - Guide, external service`

const titleId = "resource-dialog-title";
const searchTitleId = "search-vrooli-for-link-title"

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
    const { t } = useTranslation();

    const [addMutation, { loading: addLoading }] = useCustomMutation<Resource, ResourceCreateInput>(resourceCreate);
    const [updateMutation, { loading: updateLoading }] = useCustomMutation<Resource, ResourceUpdateInput>(resourceUpdate);

    const formik = useFormik({
        initialValues: {
            __typename: 'Resource' as const,
            id: partialData?.id ?? DUMMY_ID,
            index: partialData?.index ?? Math.max(index, 0),
            link: partialData?.link ?? '',
            listConnect: listId,
            usedFor: partialData?.usedFor ?? ResourceUsedFor.Context,
            translations: partialData?.translations ?? [{
                __typename: 'ResourceTranslation' as const,
                id: DUMMY_ID,
                language: getUserLanguages(session)[0],
                description: '',
                name: '',
            }],
        },
        enableReinitialize: true,
        validationSchema: resourceValidation.update({}),
        onSubmit: (values) => {
            const input = {
                ...values,
                list: { 
                    __typename: 'ResourceList' as const,
                    id: values.listConnect, 
                } as ResourceList,
            };
            if (mutate) {
                const onSuccess = (data: Resource) => {
                    (index < 0) ? onCreated(data) : onUpdated(index ?? 0, data);
                    formik.resetForm();
                    onClose();
                }
                // If index is negative, create
                if (index < 0) {
                    mutationWrapper<Resource, ResourceCreateInput>({
                        mutation: addMutation,
                        input: shapeResource.create(input),
                        successMessage: () => ({ key: 'ResourceCreated' }),
                        successCondition: (data) => data !== null,
                        onSuccess,
                        onError: () => { formik.setSubmitting(false) },
                    })
                }
                // Otherwise, update
                else {
                    if (!partialData || !partialData.id) {
                        PubSub.get().publishSnack({ messageKey: 'ResourceNotFound', severity: 'Error' });
                        return;
                    }
                    mutationWrapper<Resource, ResourceUpdateInput>({
                        mutation: updateMutation,
                        input: shapeResource.update({ ...partialData, list: { id: listId } } as ResourceShape, input),
                        successMessage: () => ({ key: 'ResourceUpdated' }),
                        successCondition: (data) => data !== null,
                        onSuccess,
                        onError: () => { formik.setSubmitting(false) },
                    })
                }
            } else {
                onCreated({
                    ...input,
                    created_at: partialData?.created_at ?? new Date().toISOString(),
                    updated_at: partialData?.updated_at ?? new Date().toISOString(),
                });
                formik.resetForm();
                onClose();
            }
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
        fields: ['description', 'name'],
        formik,
        formikField: 'translations',
        validationSchema: resourceTranslationValidation.update({}),
    });

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
    const { data: searchData, refetch: refetchSearch, loading: searchLoading } = useQuery<Wrap<PopularResult, 'popular'>, Wrap<PopularInput, 'input'>>(feedPopular, { variables: { input: { searchString: searchString.replaceAll(/![^\s]{1,}/g, '') } }, errorPolicy: 'all' });
    useEffect(() => { open && refetchSearch() }, [open, refetchSearch, searchString]);
    const autocompleteOptions: AutocompleteOption[] = useMemo(() => {
        const firstResults: AutocompleteOption[] = [];
        // Group all query results and sort by number of bookmarks. Ignore any value that isn't an array
        const flattened = (Object.values(searchData?.popular ?? [])).filter(Array.isArray).reduce((acc, curr) => acc.concat(curr), []);
        const queryItems = listToAutocomplete(flattened, languages).sort((a: any, b: any) => {
            return b.bookmarks - a.bookmarks;
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
        const newLocation = getObjectUrl(newValue);
        // Update link
        formik.setFieldValue('link', `${window.location.origin}${newLocation}`);
    }, [closeSearch, formik]);

    const handleCancel = useCallback((_?: unknown, reason?: 'backdropClick' | 'escapeKeyDown') => {
        // Don't close if formik is dirty and clicked outside
        if (formik.dirty && reason === 'backdropClick') return;
        // Otherwise, close
        formik.resetForm();
        onClose();
    }, [formik, onClose]);

    return (
        <>
            {/* Search objects (for their URL) dialog */}
            <LargeDialog
                id="resource-find-object-dialog"
                isOpen={searchOpen}
                onClose={closeSearch}
                titleId={searchTitleId}
                zIndex={zIndex}
            >
                <DialogTitle
                    id={searchTitleId}
                    title={'Search Vrooli'}
                    onClose={closeSearch}
                />
                <DialogContent sx={{
                    overflowY: 'visible',
                    minHeight: '500px',
                }}>
                    {/* Search bar to find object */}
                    <SiteSearchBar
                        id="vrooli-object-search"
                        autoFocus={true}
                        placeholder='SearchObjectLink'
                        options={autocompleteOptions}
                        loading={searchLoading}
                        value={searchString}
                        onChange={updateSearch}
                        onInputChange={onInputSelect}
                        session={session}
                        showSecondaryLabel={true}
                        sxs={{
                            root: { width: '100%', top: 0, marginTop: 2 },
                            paper: { background: palette.background.paper },
                        }}
                    />
                    {/* If object selected (and supports versioning), version selector */}
                    {/* TODO */}
                </DialogContent>
            </LargeDialog>
            {/*  Main content */}
            <LargeDialog
                id="resource-dialog"
                onClose={handleCancel}
                isOpen={open}
                titleId={titleId}
                zIndex={zIndex}
            >
                <DialogTitle
                    id={titleId}
                    title={(index < 0) ? 'Add Resource' : 'Update Resource'}
                    helpText={helpText}
                    onClose={handleCancel}
                />
                <DialogContent>
                    <BaseForm
                        onSubmit={formik.handleSubmit}
                        style={{ display: 'block', paddingBottom: '64px' }}
                    >
                        <Stack direction="column" spacing={2} paddingTop={2}>
                            {/* Language select */}
                            <LanguageInput
                                currentLanguage={language}
                                handleAdd={handleAddLanguage}
                                handleDelete={handleDeleteLanguage}
                                handleCurrent={setLanguage}
                                session={session}
                                translations={formik.values.translations}
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
                                <ColorIconButton
                                    aria-label='find URL'
                                    onClick={openSearch}
                                    background={palette.secondary.main}
                                    sx={{
                                        borderRadius: '0 5px 5px 0',
                                        height: '56px',
                                    }}>
                                    <SearchIcon />
                                </ColorIconButton>
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
                                    {(Object.keys(ResourceUsedFor) as Array<keyof typeof ResourceUsedFor>).map((usedFor) => {
                                        const Icon = getResourceIcon(usedFor as ResourceUsedFor);
                                        return (
                                            <MenuItem key={usedFor} value={usedFor}>
                                                <ListItemIcon>
                                                    <Icon fill={palette.background.textSecondary} />
                                                </ListItemIcon>
                                                <ListItemText>{t(usedFor as CommonKey, { count: 2 })}</ListItemText>
                                            </MenuItem>
                                        )
                                    })}
                                </Select>
                            </FormControl>
                            {/* Enter name */}
                            <TextField
                                fullWidth
                                id="name"
                                name="name"
                                label="Name"
                                value={translations.name}
                                onBlur={onTranslationBlur}
                                onChange={onTranslationChange}
                                error={translations.touchedName && Boolean(translations.errorName)}
                                helperText={(translations.touchedName && translations.errorName) ?? 'Enter name (optional)'}
                            />
                            {/* Enter description */}
                            <TextField
                                fullWidth
                                id="description"
                                name="description"
                                label="Description"
                                multiline
                                maxRows={8}
                                value={translations.description}
                                onBlur={onTranslationBlur}
                                onChange={onTranslationChange}
                                error={translations.touchedDescription && Boolean(translations.errorDescription)}
                                helperText={(translations.touchedDescription && translations.errorDescription) ?? 'Enter description (optional)'}
                            />
                        </Stack>
                    </BaseForm>
                </DialogContent>
                {/* Action buttons */}
                <Grid container spacing={1}>
                    <GridSubmitButtons
                        display="dialog"
                        errors={translations.errorsWithTranslations}
                        isCreate={index < 0}
                        loading={formik.isSubmitting || addLoading || updateLoading}
                        onCancel={handleCancel}
                        onSetSubmitting={formik.setSubmitting}
                        onSubmit={formik.handleSubmit}
                    />
                </Grid>
            </LargeDialog>
        </>
    )
}