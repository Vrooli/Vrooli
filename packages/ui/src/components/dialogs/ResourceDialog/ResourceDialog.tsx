import { DialogContent, useTheme } from '@mui/material';
import { Resource, ResourceCreateInput, ResourceList, ResourceUpdateInput, ResourceUsedFor } from '@shared/consts';
import { DUMMY_ID } from '@shared/uuid';
import { resourceValidation } from '@shared/validation';
import { resourceCreate } from 'api/generated/endpoints/resource_create';
import { resourceUpdate } from 'api/generated/endpoints/resource_update';
import { useCustomMutation } from 'api/hooks';
import { mutationWrapper } from 'api/utils';
import { Formik } from 'formik';
import { BaseFormRef } from 'forms/BaseForm/BaseForm';
import { ResourceForm } from 'forms/ResourceForm/ResourceForm';
import { useCallback, useContext, useRef } from 'react';
import { useTranslation } from 'react-i18next';
import { AutocompleteOption } from 'types';
import { getUserLanguages } from 'utils/display/translationTools';
import { getObjectUrl } from 'utils/navigation/openObject';
import { PubSub } from 'utils/pubsub';
import { SessionContext } from 'utils/SessionContext';
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
    listId,
    zIndex,
}: ResourceDialogProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();

    const formRef = useRef<BaseFormRef>();
    const [addMutation, { loading: addLoading }] = useCustomMutation<Resource, ResourceCreateInput>(resourceCreate);
    const [updateMutation, { loading: updateLoading }] = useCustomMutation<Resource, ResourceUpdateInput>(resourceUpdate);

    // // Search dialog to find routines, organizations, etc. to link to
    // const [searchString, setSearchString] = useState<string>('');
    // const updateSearch = useCallback((newValue: any) => { setSearchString(newValue) }, []);
    // const [searchOpen, setSearchOpen] = useState(false);
    // const openSearch = useCallback(() => {
    //     setSearchString('');
    //     setSearchOpen(true)
    // }, []);
    // const closeSearch = useCallback(() => {
    //     setSearchOpen(false)
    // }, []);
    // const { data: searchData, refetch: refetchSearch, loading: searchLoading } = useQuery<Wrap<PopularResult, 'popular'>, Wrap<PopularInput, 'input'>>(feedPopular, { variables: { input: { searchString: searchString.replaceAll(/![^\s]{1,}/g, '') } }, errorPolicy: 'all' });
    // useEffect(() => { open && refetchSearch() }, [open, refetchSearch, searchString]);
    // const autocompleteOptions: AutocompleteOption[] = useMemo(() => {
    //     const firstResults: AutocompleteOption[] = [];
    //     // Group all query results and sort by number of bookmarks. Ignore any value that isn't an array
    //     const flattened = (Object.values(searchData?.popular ?? [])).filter(Array.isArray).reduce((acc, curr) => acc.concat(curr), []);
    //     const queryItems = listToAutocomplete(flattened, languages).sort((a: any, b: any) => {
    //         return b.bookmarks - a.bookmarks;
    //     });
    //     return [...firstResults, ...queryItems];
    // }, [languages, searchData]);
    /**
     * When an autocomplete item is selected, set link as its URL
     */
    const onInputSelect = useCallback((newValue: AutocompleteOption) => {
        // If value is not an object, return;
        if (!newValue || newValue.__typename === 'Shortcut' || newValue.__typename === 'Action') return;
        // Clear search string and close command palette
        // closeSearch();
        // Create URL
        const newLocation = getObjectUrl(newValue);
        // Update link
        // formik.setFieldValue('link', `${window.location.origin}${newLocation}`);
    }, []);

    const handleClose = useCallback((_?: unknown, reason?: 'backdropClick' | 'escapeKeyDown') => {
        // Confirm dialog is dirty and closed by clicking outside
        console.log('ref current', formRef.current)
        formRef.current?.handleClose(onClose, reason !== 'backdropClick');
    }, [onClose]);

    return (
        <>
            {/* Search objects (for their URL) dialog */}
            {/* <LargeDialog
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
                }}> */}
            {/* Search bar to find object */}
            {/* <SiteSearchBar
                        id="vrooli-object-search"
                        autoFocus={true}
                        placeholder='SearchObjectLink'
                        options={autocompleteOptions}
                        loading={searchLoading}
                        value={searchString}
                        onChange={updateSearch}
                        onInputChange={onInputSelect}
                        showSecondaryLabel={true}
                        sxs={{
                            root: { width: '100%', top: 0, marginTop: 2 },
                            paper: { background: palette.background.paper },
                        }}
                    /> */}
            {/* If object selected (and supports versioning), version selector */}
            {/* TODO */}
            {/* </DialogContent>
            </LargeDialog> */}
            {/*  Main content */}
            <LargeDialog
                id="resource-dialog"
                onClose={handleClose}
                isOpen={open}
                titleId={titleId}
                zIndex={zIndex}
            >
                <DialogTitle
                    id={titleId}
                    title={(index < 0) ? 'Add Resource' : 'Update Resource'}
                    helpText={helpText}
                    onClose={handleClose}
                />
                <DialogContent>
                    <Formik
                        enableReinitialize={true}
                        initialValues={{
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
                        }}
                        onSubmit={(values, helpers) => {
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
                                    helpers.resetForm();
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
                                        onError: () => { helpers.setSubmitting(false) },
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
                                        onError: () => { helpers.setSubmitting(false) },
                                    })
                                }
                            } else {
                                onCreated({
                                    ...input,
                                    created_at: partialData?.created_at ?? new Date().toISOString(),
                                    updated_at: partialData?.updated_at ?? new Date().toISOString(),
                                });
                                helpers.resetForm();
                                onClose();
                            }
                        }}
                        validationSchema={resourceValidation.update({})}
                    >
                        {(formik) => <ResourceForm
                            display="dialog"
                            index={index}
                            isLoading={addLoading || updateLoading}
                            onCancel={formik.resetForm}
                            ref={formRef}
                            zIndex={zIndex}
                            {...formik}
                        />}
                    </Formik>
                </DialogContent>
            </LargeDialog>
        </>
    )
}