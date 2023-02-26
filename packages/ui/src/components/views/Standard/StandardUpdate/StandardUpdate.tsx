import { Box, CircularProgress, Grid, TextField } from "@mui/material"
import { useMutation, useCustomLazyQuery } from "api/hooks";
import { useCallback, useEffect, useMemo, useState } from "react";
import { StandardUpdateProps } from "../types";
import { mutationWrapper } from 'api/utils';
import { useFormik } from 'formik';
import { addEmptyTranslation, defaultRelationships, defaultResourceList, getUserLanguages, handleTranslationBlur, handleTranslationChange, parseSingleItemUrl, PubSub, removeTranslation, shapeStandardVersion, TagShape, usePromptBeforeUnload, useTranslatedFields } from "utils";
import { GridSubmitButtons, LanguageInput, PageTitle, RelationshipButtons, ResourceListHorizontal, TagSelector } from "components";
import { DUMMY_ID, uuid } from '@shared/uuid';
import { RelationshipsObject } from "components/inputs/types";
import { FindVersionInput, ResourceList, Standard, StandardUpdateInput, StandardVersion, StandardVersionUpdateInput } from "@shared/consts";
import { standardVersionValidation } from "@shared/validation";
import { standardVersionFindOne, standardVersionUpdate } from "api/generated/endpoints/standardVersion";

export const StandardUpdate = ({
    onCancel,
    onUpdated,
    session,
    zIndex,
}: StandardUpdateProps) => {
    // Fetch existing data
    const urlData = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data: standardVersion, loading }] = useCustomLazyQuery<StandardVersion, FindVersionInput>(standardVersionFindOne, { errorPolicy: 'all' });
    useEffect(() => {
        if (urlData.id || urlData.idRoot) getData({ variables: urlData });
        else PubSub.get().publishSnack({ messageKey: 'InvalidUrlId', severity: 'Error' });
    }, [getData, urlData])

    // Handle relationships
    const [relationships, setRelationships] = useState<RelationshipsObject>(defaultRelationships(true, session));
    const onRelationshipsChange = useCallback((change: Partial<RelationshipsObject>) => setRelationships({ ...relationships, ...change }), [relationships]);

   // Handle resources
   const [resourceList, setResourceList] = useState<ResourceList>(defaultResourceList);
   const handleResourcesUpdate = useCallback((updatedList: ResourceList) => setResourceList(updatedList), [setResourceList]);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>([]);
    const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

    useEffect(() => {
        setRelationships({
            isComplete: false, //TODO
            isPrivate: standardVersion?.isPrivate ?? false,
            owner: standardVersion?.root?.owner ?? null,
            parent: null,
            // parent: standard?.parent ?? null, TODO
            project: null // TODO
        });
        setResourceList(standardVersion?.resourceList ?? { id: uuid() } as any);
        setTags(standardVersion?.root?.tags ?? []);
    }, [standardVersion]);

    // Handle update
    const [mutation] = useMutation<StandardVersion, StandardVersionUpdateInput, 'standardVersionUpdate'>(standardVersionUpdate, 'standardVersionUpdate');
    const formik = useFormik({
        initialValues: {
            translationsUpdate: standardVersion?.translations ?? [{
                id: DUMMY_ID,
                language: getUserLanguages(session)[0],
                description: '',
                jsonVariable: null, //TODO
            }],
            versionInfo: {
                versionIndex: standardVersion?.root?.versions?.length ?? 0,
                versionLabel: standardVersion?.versionLabel ?? '1.0.0',
                versionNotes: '',
            }
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema: standardVersionValidation.update({}),
        onSubmit: (values) => {
            if (!standardVersion) {
                PubSub.get().publishSnack({ messageKey: 'CouldNotReadStandard', severity: 'Error' });
                return;
            }
            // Update
            mutationWrapper<StandardVersion, StandardVersionUpdateInput>({
                mutation,
                input: shapeStandardVersion.update(standardVersion, {
                    id: standardVersion.id,
                    isComplete: relationships.isComplete,
                    isLatest: true,
                    isPrivate: relationships.isPrivate,
                    resourceList: resourceList,
                    root: {
                        id: standardVersion.root.id,
                        isInternal: false,
                        isPrivate: relationships.isPrivate,
                        name: standardVersion.root.name,
                        owner: relationships.owner,
                        permissions: JSON.stringify({}),
                        tags: tags,
                    },
                    standardType: standardVersion.standardType,
                    translations: values.translationsUpdate,
                    ...values.versionInfo,
                } as any), //TODO
                onSuccess: (data) => { onUpdated(data) },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    // Handle translations
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    const translations = useTranslatedFields({
        fields: ['description'],
        formik,
        formikField: 'translationsUpdate',
        language,
        validationSchema: {} as any,//standardVersionTranslationValidation.update({}), TODO
    });
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
        handleTranslationBlur(formik, 'translationsUpdate', e, language)
    }, [formik, language]);
    // Handles change on translation fields
    const onTranslationChange = useCallback((e: { target: { name: string, value: string } }) => {
        handleTranslationChange(formik, 'translationsUpdate', e, language)
    }, [formik, language]);

    const formInput = useMemo(() => (
        <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
            <Grid item xs={12}>
                <PageTitle titleKey='UpdateStandard' session={session} />
            </Grid>
            <Grid item xs={12} mb={4}>
                <RelationshipButtons
                    isEditing={true}
                    objectType={'Standard'}
                    onRelationshipsChange={onRelationshipsChange}
                    relationships={relationships}
                    session={session}
                    zIndex={zIndex}
                />
            </Grid>
            <Grid item xs={12}>
                <LanguageInput
                    currentLanguage={language}
                    handleAdd={handleAddLanguage}
                    handleDelete={handleLanguageDelete}
                    handleCurrent={setLanguage}
                    session={session}
                    translations={formik.values.translationsUpdate}
                    zIndex={zIndex}
                />
            </Grid>
            <Grid item xs={12} mb={4}>
                <TextField
                    fullWidth
                    id="description"
                    name="description"
                    label="description"
                    multiline
                    minRows={4}
                    value={translations.description}
                    onBlur={onTranslationBlur}
                    onChange={onTranslationChange}
                    error={translations.touchedDescription && Boolean(translations.errorDescription)}
                    helperText={translations.touchedDescription && translations.errorDescription}
                />
            </Grid>
            {/* TODO versioning */}
            <Grid item xs={12}>
                <ResourceListHorizontal
                    title={'Resources'}
                    list={resourceList}
                    canUpdate={true}
                    handleUpdate={handleResourcesUpdate}
                    loading={loading}
                    session={session}
                    mutate={false}
                    zIndex={zIndex}
                />
            </Grid>
            <Grid item xs={12} marginBottom={4}>
                <TagSelector
                    handleTagsUpdate={handleTagsUpdate}
                    session={session}
                    tags={tags}
                />
            </Grid>
            <GridSubmitButtons
                errors={translations.errorsWithTranslations}
                isCreate={false}
                loading={formik.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={formik.setSubmitting}
                onSubmit={formik.handleSubmit}
            />
        </Grid>
    ), [onRelationshipsChange, relationships, session, zIndex, language, handleAddLanguage, handleLanguageDelete, formik.values.translationsUpdate, formik.isSubmitting, formik.setSubmitting, formik.handleSubmit, onTranslationBlur, onTranslationChange, translations, resourceList, handleResourcesUpdate, loading, handleTagsUpdate, tags, onCancel]);


    return (
        <form onSubmit={formik.handleSubmit} style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
        }}
        >
            {loading ? (
                <Box sx={{
                    position: 'absolute',
                    top: '-5vh', // Half of toolbar height
                    width: '100%',
                    height: '100%',
                    display: 'flex',
                    justifyContent: 'center',
                    alignItems: 'center',
                }}>
                    <CircularProgress size={100} color="secondary" />
                </Box>
            ) : formInput}
        </form>
    )
}