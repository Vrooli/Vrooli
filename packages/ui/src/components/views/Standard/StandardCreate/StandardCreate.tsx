import { Box, Grid, TextField } from "@mui/material";
import { useCustomMutation } from "api/hooks";
import { mutationWrapper } from 'api/utils';
import { useFormik } from 'formik';
import { addEmptyTranslation, defaultRelationships, defaultResourceList, getUserLanguages, handleTranslationBlur, handleTranslationChange, InputTypeOption, InputTypeOptions, removeTranslation, shapeStandardVersion, TagShape, usePromptBeforeUnload, useTranslatedFields } from "utils";
import { StandardCreateProps } from "../types";
import { useCallback, useEffect, useMemo, useState } from "react";
import { GridSubmitButtons, LanguageInput, PageTitle, ResourceListHorizontal, Selector, TagSelector } from "components";
import { uuid } from '@shared/uuid';
import { FieldData } from "forms/types";
import { BaseStandardInput, GeneratedInputComponent, PreviewSwitch, RelationshipButtons } from "components/inputs";
import { generateYupSchema } from "forms/generators";
import { RelationshipsObject } from "components/inputs/types";
import { getCurrentUser } from "utils/authentication";
import { ResourceList, StandardVersion, StandardVersionCreateInput } from "@shared/consts";
import { standardVersionTranslationValidation, standardVersionValidation } from "@shared/validation";
import { standardVersionCreate } from "api/generated/endpoints/standardVersion";
import { parseSearchParams } from "@shared/route";

export const StandardCreate = ({
    onCreated,
    onCancel,
    session,
    zIndex,
}: StandardCreateProps) => {

    // Handle relationships
    const [relationships, setRelationships] = useState<RelationshipsObject>(defaultRelationships(true, session));
    const onRelationshipsChange = useCallback((change: Partial<RelationshipsObject>) => setRelationships({ ...relationships, ...change }), [relationships]);

    // Handle input type selector
    const [inputType, setInputType] = useState<InputTypeOption>(InputTypeOptions[1]);
    const handleInputTypeSelect = useCallback((selected: InputTypeOption) => {
        setInputType(selected);
    }, []);

    // Handle standard schema
    const [schema, setSchema] = useState<FieldData | null>(null);
    const handleSchemaUpdate = useCallback((schema: FieldData) => {
        setSchema(schema);
    }, []);
    const [schemaKey] = useState(`standard-create-schema-preview-${Math.random().toString(36).substring(2, 15)}`);

    const previewFormik = useFormik({
        initialValues: {
            preview: schema?.props?.defaultValue,
        },
        enableReinitialize: true,
        validationSchema: schema ? generateYupSchema({
            fields: [schema],
        }) : undefined,
        onSubmit: () => { },
    });

    // Handle resources
    const [resourceList, setResourceList] = useState<ResourceList>(defaultResourceList);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => setResourceList(updatedList), [setResourceList]);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>([]);
    const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

    useEffect(() => {
        const params = parseSearchParams();
        if (typeof params.tag === 'string') setTags([{ tag: params.tag }]);
        else if (Array.isArray(params.tags)) setTags(params.tags.map((t: any) => ({ tag: t })));
    }, []);

    // Handle create
    const [mutation] = useCustomMutation<StandardVersion, StandardVersionCreateInput>(standardVersionCreate);
    const formik = useFormik({
        initialValues: {
            id: uuid(),
            default: '',
            name: '',
            translationsCreate: [{
                id: uuid(),
                language: getUserLanguages(session)[0],
                description: '',
                jsonVariable: null, //TODO
            }],
            versionInfo: {
                versionIndex: 0,
                versionLabel: '1.0.0',
                versionNotes: '',
            }
        },
        validationSchema: standardVersionValidation.create({}),
        onSubmit: (values) => {
            mutationWrapper<StandardVersion, StandardVersionCreateInput>({
                mutation,
                input: shapeStandardVersion.create({
                    id: values.id,
                    default: values.default,
                    isComplete: relationships.isComplete,
                    isLatest: true,
                    isPrivate: relationships.isPrivate,
                    props: JSON.stringify(schema?.props),
                    yup: JSON.stringify(schema?.yup),
                    resourceList: resourceList,
                    root: {
                        id: uuid(),
                        isInternal: false,
                        isPrivate: relationships.isPrivate,
                        name: values.name,
                        owner: relationships.owner,
                        parent: relationships.parent,
                        permissions: JSON.stringify({}),
                        tags: tags,
                    },
                    standardType: inputType.value,
                    translations: values.translationsCreate,
                    ...values.versionInfo,
                }),
                onSuccess: (data) => {
                    onCreated(data)
                },
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
        formikField: 'translationsCreate',
        language,
        validationSchema: standardVersionTranslationValidation.create({}),
    });
    const languages = useMemo(() => formik.values.translationsCreate.map(t => t.language), [formik.values.translationsCreate]);
    const handleAddLanguage = useCallback((newLanguage: string) => {
        setLanguage(newLanguage);
        addEmptyTranslation(formik, 'translationsCreate', newLanguage);
    }, [formik]);
    const handleLanguageDelete = useCallback((language: string) => {
        const newLanguages = [...languages.filter(l => l !== language)]
        if (newLanguages.length === 0) return;
        setLanguage(newLanguages[0]);
        removeTranslation(formik, 'translationsCreate', language);
    }, [formik, languages]);
    // Handles blur on translation fields
    const onTranslationBlur = useCallback((e: { target: { name: string } }) => {
        handleTranslationBlur(formik, 'translationsCreate', e, language)
    }, [formik, language]);
    // Handles change on translation fields
    const onTranslationChange = useCallback((e: { target: { name: string, value: string } }) => {
        handleTranslationChange(formik, 'translationsCreate', e, language)
    }, [formik, language]);

    const isLoggedIn = useMemo(() => Boolean(getCurrentUser(session).id), [session]);

    const [isPreviewOn, setIsPreviewOn] = useState<boolean>(false);
    const onPreviewChange = useCallback((isOn: boolean) => { setIsPreviewOn(isOn); }, []);

    return (
        <form onSubmit={formik.handleSubmit} style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
        }}
        >
            <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
                <Grid item xs={12}>
                    <PageTitle titleKey='CreateStandard' session={session} />
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
                        translations={formik.values.translationsCreate}
                        zIndex={zIndex}
                    />
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
                {/* <Grid item xs={12}>
                    <TextField
                        fullWidth
                        id="version"
                        name="version"
                        label="version"
                        value={formik.values.version}
                        onBlur={formik.handleBlur}
                        onChange={formik.handleChange}
                        error={formik.touched.version && Boolean(formik.errors.version)}
                        helperText={formik.touched.version && formik.errors.version}
                    />
                </Grid> */}
                {/* Standard build/preview */}
                <Grid item xs={12}>
                    <PreviewSwitch
                        isPreviewOn={isPreviewOn}
                        onChange={onPreviewChange}
                        sx={{
                            marginBottom: 2
                        }}
                    />
                    {
                        isPreviewOn ?
                            (schema && <GeneratedInputComponent
                                disabled={true}
                                fieldData={schema}
                                formik={previewFormik}
                                session={session}
                                onUpload={() => { }}
                                zIndex={zIndex}
                            />) :
                            <Box>
                                <Selector
                                    fullWidth
                                    options={InputTypeOptions}
                                    selected={inputType}
                                    handleChange={handleInputTypeSelect}
                                    getOptionLabel={(option: InputTypeOption) => option.label}
                                    inputAriaLabel='input-type-selector'
                                    label="Type"
                                    sx={{ marginBottom: 2 }}
                                />
                                <BaseStandardInput
                                    fieldName="preview"
                                    inputType={inputType.value}
                                    isEditing={true}
                                    schema={schema}
                                    onChange={handleSchemaUpdate}
                                    storageKey={schemaKey}
                                />
                            </Box>
                    }
                </Grid>
                <Grid item xs={12}>
                    <ResourceListHorizontal
                        title={'Resources'}
                        list={resourceList}
                        canUpdate={true}
                        handleUpdate={handleResourcesUpdate}
                        loading={false}
                        session={session}
                        mutate={false}
                        zIndex={zIndex}
                    />
                </Grid>
                <Grid item xs={12} mb={4}>
                    <TagSelector
                        handleTagsUpdate={handleTagsUpdate}
                        session={session}
                        tags={tags}
                    />
                </Grid>
                <GridSubmitButtons
                    disabledSubmit={!isLoggedIn}
                    errors={translations.errorsWithTranslations}
                    isCreate={true}
                    loading={formik.isSubmitting}
                    onCancel={onCancel}
                    onSetSubmitting={formik.setSubmitting}
                    onSubmit={formik.handleSubmit}
                />
            </Grid>
        </form>
    )
}