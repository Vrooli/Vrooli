import { Box, Checkbox, CircularProgress, FormControlLabel, Grid, TextField, Tooltip } from "@mui/material"
import { useLazyQuery, useMutation } from "api/hooks";
import { useCallback, useEffect, useMemo, useState } from "react";
import { SmartContractUpdateProps } from "../types";
import { mutationWrapper } from 'api/utils';
import { smartContractVersionValidation, smartContractVersionTranslationValidation } from '@shared/validation';
import { useFormik } from 'formik';
import { addEmptyTranslation, defaultRelationships, defaultResourceList, getPreferredLanguage, getUserLanguages, handleTranslationBlur, handleTranslationChange, parseSingleItemUrl, PubSub, removeTranslation, shapeSmartContractVersion, TagShape, usePromptBeforeUnload, useTranslatedFields } from "utils";
import { GridSubmitButtons, LanguageInput, PageTitle, RelationshipButtons, ResourceListHorizontal, SnackSeverity, TagSelector } from "components";
import { DUMMY_ID, uuid } from '@shared/uuid';
import { RelationshipsObject } from "components/inputs/types";
import { FindByIdInput, SmartContractVersion, SmartContractVersionUpdateInput, ResourceList } from "@shared/consts";
import { smartContractVersionFindOne, smartContractVersionUpdate } from "api/generated/endpoints/smartContractVersion";

export const SmartContractUpdate = ({
    onCancel,
    onUpdated,
session,
    zIndex,
}: SmartContractUpdateProps) => {
    // Fetch existing data
    const { id } = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data, loading }] = useLazyQuery<SmartContractVersion, FindByIdInput, 'smartContractVersion'>(smartContractVersionFindOne, 'smartContractVersion');
    useEffect(() => { id && getData({ variables: { id } }) }, [getData, id])
    const smartContractVersion = useMemo(() => data?.smartContractVersion, [data]);

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
    const [mutation] = useMutation<SmartContractVersion, SmartContractVersionUpdateInput, 'smartContractVersionUpdate'>(smartContractVersionUpdate, 'smartContractVersionUpdate');
    const formik = useFormik({
        initialValues: {
            id: smartContractVersion?.id ?? uuid(),
            translationsUpdate: smartContractVersion?.translations ?? [{
                id: DUMMY_ID,
                language: getUserLanguages(session)[0],
                description: '',
                jsonVariable: '',
                name: '',
            }],
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema: smartContractVersionValidation.update({}),
        onSubmit: (values) => {
            if (!smartContractVersion) {
                PubSub.get().publishSnack({ messageKey: 'CouldNotReadSmartContract', severity: SnackSeverity.Error });
                return;
            }
            // mutationWrapper<SmartContractVersion, SmartContractVersionUpdateInput>({
            //     mutation,
            //     input: shapeSmartContractVersion.update(smartContractVersion, {
            //         id: smartContractVersion.id,
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
        fields: ['description', 'jsonVariable', 'name'],
        formik, 
        formikField: 'translationsUpdate', 
        language, 
        validationSchema: smartContractVersionTranslationValidation.update({}),
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
            isPrivate: smartContractVersion?.isPrivate ?? false,
            owner: null,
            parent: null,
            project: null,
        });
        setResourceList(smartContractVersion?.resourceList ?? { id: uuid() } as any);
        setTags(smartContractVersion?.root?.tags ?? []);
        if (smartContractVersion?.translations?.length) {
            setLanguage(getPreferredLanguage(smartContractVersion.translations.map(t => t.language), getUserLanguages(session)));
        }
    }, [smartContractVersion, session]);

    const formInput = useMemo(() => (
        <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
            <Grid item xs={12}>
                <PageTitle titleKey='UpdateSmartContract' session={session} />
            </Grid>
            <Grid item xs={12} mb={4}>
                <RelationshipButtons
                    isEditing={true}
                    objectType={'SmartContract'}
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
            {/* TODO */}
            <GridSubmitButtons
                errors={translations.errorsWithTranslations}
                isCreate={false}
                loading={formik.isSubmitting}
                onCancel={onCancel}
                onSetSubmitting={formik.setSubmitting}
                onSubmit={formik.handleSubmit}
            />
        </Grid>
    ), [onRelationshipsChange, relationships, session, zIndex, language, handleAddLanguage, handleLanguageDelete, formik.values.translationsUpdate, formik.isSubmitting, formik.setSubmitting, formik.handleSubmit, translations, onCancel]);

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