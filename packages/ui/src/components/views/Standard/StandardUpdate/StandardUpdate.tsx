import { Box, CircularProgress, Grid, TextField } from "@mui/material"
import { useMutation, useLazyQuery } from "graphql/hooks";
import { useCallback, useEffect, useMemo, useState } from "react";
import { StandardUpdateProps } from "../types";
import { mutationWrapper } from 'graphql/utils';
import { useFormik } from 'formik';
import { addEmptyTranslation, getUserLanguages, handleTranslationBlur, handleTranslationChange, parseSingleItemUrl, PubSub, removeTranslation, shapeStandardVersion, TagShape, usePromptBeforeUnload, useTranslatedFields } from "utils";
import { GridSubmitButtons, LanguageInput, PageTitle, RelationshipButtons, ResourceListHorizontal, SnackSeverity, TagSelector, userFromSession } from "components";
import { DUMMY_ID, uuid } from '@shared/uuid';
import { RelationshipsObject } from "components/inputs/types";
import { FindVersionInput, ResourceList, Standard, StandardUpdateInput, StandardVersion, StandardVersionUpdateInput } from "@shared/consts";
import { standardVersionEndpoint } from "graphql/endpoints";
import { standardVersionValidation } from "@shared/validation";

export const StandardUpdate = ({
    onCancel,
    onUpdated,
    session,
    zIndex,
}: StandardUpdateProps) => {
    // Fetch existing data
    const urlData = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data, loading }] = useLazyQuery<StandardVersion, FindVersionInput, 'standardVersion'>(...standardVersionEndpoint.findOne, { errorPolicy: 'all' });
    useEffect(() => {
        if (urlData.id || urlData.idRoot) getData({ variables: urlData });
        else PubSub.get().publishSnack({ messageKey: 'InvalidUrlId', severity: SnackSeverity.Error });
    }, [getData, urlData])
    const standardVersion = useMemo(() => data?.standardVersion, [data]);

    const [relationships, setRelationships] = useState<RelationshipsObject>({
        isComplete: false,
        isPrivate: false,
        owner: userFromSession(session),
        parent: null,
        project: null,
    });
    const onRelationshipsChange = useCallback((newRelationshipsObject: Partial<RelationshipsObject>) => {
        setRelationships({
            ...relationships,
            ...newRelationshipsObject,
        });
    }, [relationships]);

    // Handle resources
    const [resourceList, setResourceList] = useState<ResourceList>({ id: uuid() } as any);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => {
        setResourceList(updatedList);
    }, [setResourceList]);

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
        setTags(standardVersion?.tags ?? []);
    }, [standardVersion]);

    // Handle update
    const [mutation] = useMutation<StandardVersion, StandardVersionUpdateInput, 'standardVersionUpdate'>(...standardVersionEndpoint.update);
    const formik = useFormik({
        initialValues: {
            translationsUpdate: standardVersion?.translations ?? [{
                id: DUMMY_ID,
                language: getUserLanguages(session)[0],
                description: '',
                jsonVariable: null, //TODO
            }],
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema: standardVersionValidation.update(),
        onSubmit: (values) => {
            if (!standardVersion) {
                PubSub.get().publishSnack({ messageKey: 'CouldNotReadStandard', severity: SnackSeverity.Error });
                return;
            }
            // Update
            mutationWrapper<StandardVersion, StandardVersionUpdateInput>({
                mutation,
                input: shapeStandardVersion.update(standardVersion, {
                    id: standardVersion.id,
                    resourceList: resourceList,
                    tags: tags,
                    translations: values.translationsUpdate.map(t => ({
                        ...t,
                        id: t.id === DUMMY_ID ? uuid() : t.id,
                    })),
                }),
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
        validationSchema: standardVersionTranslationValidation.update(),
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
                    canEdit={true}
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