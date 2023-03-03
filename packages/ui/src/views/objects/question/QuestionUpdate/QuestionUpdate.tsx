import { Box, CircularProgress, Grid, TextField } from "@mui/material"
import { useCustomLazyQuery, useCustomMutation } from "api/hooks";
import { useCallback, useEffect, useMemo, useState } from "react";
import { QuestionUpdateProps } from "../types";
import { questionValidation, questionTranslationValidation } from '@shared/validation';
import { useFormik } from 'formik';
import { addEmptyTranslation, defaultRelationships, defaultResourceList, getPreferredLanguage, getUserLanguages, handleTranslationBlur, handleTranslationChange, parseSingleItemUrl, PubSub, removeTranslation, shapeQuestion, TagShape, usePromptBeforeUnload, useTopBar, useTranslatedFields, useUpdateActions } from "utils";
import { GridSubmitButtons, LanguageInput, PageTitle, RelationshipButtons, TagSelector } from "components";
import { DUMMY_ID, uuid } from '@shared/uuid';
import { RelationshipsObject } from "components/inputs/types";
import { FindByIdInput, Question, QuestionUpdateInput, ResourceList } from "@shared/consts";
import { questionFindOne } from "api/generated/endpoints/question_findOne";
import { questionUpdate } from "api/generated/endpoints/question_update";

export const QuestionUpdate = ({
    display = 'page',
    session,
    zIndex = 200,
}: QuestionUpdateProps) => {
    const { onCancel, onUpdated } = useUpdateActions<Question>();

    // Fetch existing data
    const { id } = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data: question, loading }] = useCustomLazyQuery<Question, FindByIdInput>(questionFindOne);
    useEffect(() => { id && getData({ variables: { id } }) }, [getData, id])

    // Handle relationships
    const [relationships, setRelationships] = useState<RelationshipsObject>(defaultRelationships(true, session));
    const onRelationshipsChange = useCallback((change: Partial<RelationshipsObject>) => setRelationships({ ...relationships, ...change }), [relationships]);

    // Handle resources
    const [resourceList, setResourceList] = useState<ResourceList>(defaultResourceList);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => setResourceList(updatedList), [setResourceList]);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>([]);
    const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

    // Handle update
    const [mutation] = useCustomMutation<Question, QuestionUpdateInput>(questionUpdate);
    const formik = useFormik({
        initialValues: {
            id: question?.id ?? uuid(),
            translationsUpdate: question?.translations ?? [{
                id: DUMMY_ID,
                language: getUserLanguages(session)[0],
                description: '',
                name: '',
            }],
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema: questionValidation.update({}),
        onSubmit: (values) => {
            if (!question) {
                PubSub.get().publishSnack({ messageKey: 'CouldNotReadQuestion', severity: 'Error' });
                return;
            }
            // mutationWrapper<Question, QuestionUpdateInput>({
            //     mutation,
            //     input: shapeQuestion.update(question, {
            //         id: question.id,
            //         isOpenToNewMembers: values.isOpenToNewMembers,
            //         isPrivate: relationships.isPrivate,
            //         resourceList: resourceList,
            //         tags: tags,
            //         translations: values.translationsUpdate,
            //     }),
            //     onSuccess: (data) => { onUpdated(data) },
            //     onError: () => { formik.setSubmitting(false) },
            // })
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    // Handle translations
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    const translations = useTranslatedFields({
        fields: ['description', 'name'],
        formik,
        formikField: 'translationsUpdate',
        language,
        validationSchema: questionTranslationValidation.update({}),
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
    const onTranslationBlur = useCallback((e: { target: { name: string } }) => handleTranslationBlur(formik, 'translationsUpdate', e, language), [formik, language]);
    // Handles change on translation fields
    const onTranslationChange = useCallback((e: { target: { name: string, value: string } }) => handleTranslationChange(formik, 'translationsUpdate', e, language), [formik, language]);

    useEffect(() => {
        setRelationships({
            isComplete: false,
            isPrivate: false,
            owner: null,
            parent: null,
            project: null,
        });
        setTags(question?.tags ?? []);
        if (question?.translations?.length) {
            setLanguage(getPreferredLanguage(question.translations.map(t => t.language), getUserLanguages(session)));
        }
    }, [question, session]);

    const TopBar = useTopBar({
        display,
        session,
        titleData: {
            titleKey: 'UpdateQuestion',
        },
    })

    const formInput = useMemo(() => (
        <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
            <Grid item xs={12} mb={4}>
                <RelationshipButtons
                    isEditing={true}
                    objectType={'Question'}
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
            <Grid item xs={12}>
                <TextField
                    fullWidth
                    id="name"
                    name="name"
                    label="Name"
                    value={translations.name}
                    onBlur={onTranslationBlur}
                    onChange={onTranslationChange}
                    error={translations.touchedName && Boolean(translations.errorName)}
                    helperText={translations.touchedName && translations.errorName}
                />
            </Grid>
            <Grid item xs={12} mb={4}>
                <TextField
                    fullWidth
                    id="description"
                    name="description"
                    label="Description"
                    multiline
                    minRows={4}
                    value={translations.description}
                    onBlur={onTranslationBlur}
                    onChange={onTranslationChange}
                    error={translations.touchedDescription && Boolean(translations.errorDescription)}
                    helperText={translations.touchedDescription && translations.errorDescription}
                />
            </Grid>
            <Grid item xs={12}>
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
    ), [onRelationshipsChange, relationships, session, zIndex, language, handleAddLanguage, handleLanguageDelete, formik.values.translationsUpdate, formik.isSubmitting, formik.setSubmitting, formik.handleSubmit, onTranslationBlur, onTranslationChange, translations, handleTagsUpdate, tags, onCancel]);

    return (
        <>
            {TopBar}
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
        </>
    )
}