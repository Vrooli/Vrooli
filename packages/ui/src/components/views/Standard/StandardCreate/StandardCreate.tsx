import { Box, Button, Grid, TextField, Typography } from "@mui/material";
import { useMutation } from "@apollo/client";
import { mutationWrapper } from 'graphql/utils/mutationWrapper';
import { standardCreateForm as validationSchema } from '@shared/validation';
import { useFormik } from 'formik';
import { standardCreateMutation } from "graphql/mutation";
import { getUserLanguages, InputTypeOption, InputTypeOptions, ObjectType, shapeStandardCreate, StandardTranslationShape, TagShape, updateArray, useReactSearch } from "utils";
import { StandardCreateProps } from "../types";
import { useCallback, useEffect, useMemo, useState } from "react";
import { LanguageInput, ResourceListHorizontal, Selector, TagSelector } from "components";
import { ResourceList } from "types";
import { ResourceListUsedFor } from "graphql/generated/globalTypes";
import { v4 as uuid, validate as uuidValidate } from 'uuid';
import { FieldData } from "forms/types";
import { BaseStandardInput, PreviewSwitch, RelationshipButtons, userFromSession } from "components/inputs";
import { generateInputComponent, generateYupSchema } from "forms/generators";
import { standardCreate, standardCreateVariables } from "graphql/generated/standardCreate";
import { RelationshipsObject } from "components/inputs/types";
import { CancelIcon, CreateIcon } from "@shared/icons";

export const StandardCreate = ({
    onCreated,
    onCancel,
    session,
    zIndex,
}: StandardCreateProps) => {
    const params = useReactSearch(null);

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

    // Handle input type selector
    const [inputType, setInputType] = useState<InputTypeOption>(InputTypeOptions[1]);
    const handleInputTypeSelect = useCallback((event: any) => {
        setInputType(event.target.value)
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
    const [resourceList, setResourceList] = useState<ResourceList>({ id: uuid(), usedFor: ResourceListUsedFor.Display } as any);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => {
        setResourceList(updatedList);
    }, [setResourceList]);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>([]);
    const addTag = useCallback((tag: TagShape) => {
        setTags(t => [...t, tag]);
    }, [setTags]);
    const removeTag = useCallback((tag: TagShape) => {
        setTags(tags => tags.filter(t => t.tag !== tag.tag));
    }, [setTags]);
    const clearTags = useCallback(() => {
        setTags([]);
    }, [setTags]);

    // Handle translations
    type Translation = StandardTranslationShape;
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
        if (typeof params.tag === 'string') setTags([{ tag: params.tag }]);
        else if (Array.isArray(params.tags)) setTags(params.tags.map((t: any) => ({ tag: t })));
    }, [params]);

    // Handle create
    const [mutation] = useMutation<standardCreate, standardCreateVariables>(standardCreateMutation);
    const formik = useFormik({
        initialValues: {
            default: '',
            description: '',
            name: '',
            version: '1.0',
        },
        validationSchema,
        onSubmit: (values) => {
            // Update translations with final values
            const allTranslations = getTranslationsUpdate(language, {
                id: uuid(),
                language,
                description: values.description,
                jsonVariable: null, //TODO
            })
            // Mutate
            mutationWrapper({
                mutation,
                input: shapeStandardCreate({
                    id: uuid(),
                    default: values.default,
                    isInternal: false,
                    name: values.name,
                    props: JSON.stringify(schema?.props),
                    yup: JSON.stringify(schema?.yup),
                    translations: allTranslations,
                    resourceLists: [resourceList],
                    tags: tags as any,
                    type: inputType.value,
                    version: values.version,
                }),
                onSuccess: (response) => {
                    onCreated(response.data.standardCreate)
                },
                onError: () => { formik.setSubmitting(false) },
            })
        },
    });

    /**
     * On page leave, check if unsaved work. 
     * If so, prompt for confirmation.
     */
    useEffect(() => {
        const handleBeforeUnload = (e: BeforeUnloadEvent) => {
            if (formik.dirty) {
                e.preventDefault()
                e.returnValue = ''
            }
        };
        window.addEventListener('beforeunload', handleBeforeUnload);
        return () => window.removeEventListener('beforeunload', handleBeforeUnload);
    }, [formik.dirty]);

    // Handle languages
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    const [languages, setLanguages] = useState<string[]>([getUserLanguages(session)[0]]);
    useEffect(() => {
        if (languages.length === 0) {
            const userLanguage = getUserLanguages(session)[0]
            setLanguage(userLanguage)
            setLanguages([userLanguage])
        }
    }, [languages, session, setLanguage, setLanguages])
    const updateFormikTranslation = useCallback((language: string) => {
        const existingTranslation = translations.find(t => t.language === language);
        formik.setValues({
            ...formik.values,
            description: existingTranslation?.description ?? '',
        });
    }, [formik, translations]);
    const handleLanguageSelect = useCallback((newLanguage: string) => {
        // Update old select
        updateTranslation(language, {
            id: uuid(),
            language,
            description: formik.values.description,
            jsonVariable: null, //TODO
        })
        // Update formik
        if (language !== newLanguage) updateFormikTranslation(newLanguage);
        // Change language
        setLanguage(newLanguage);
    }, [updateTranslation, language, formik.values.description, updateFormikTranslation]);
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

    const isLoggedIn = useMemo(() => session?.isLoggedIn === true && uuidValidate(session?.id ?? ''), [session]);

    const [isPreviewOn, setIsPreviewOn] = useState<boolean>(false);
    const onPreviewChange = useCallback((isOn: boolean) => { setIsPreviewOn(isOn); }, []);

    return (
        <form onSubmit={formik.handleSubmit} style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
        }}
        >
            <Grid container spacing={2} sx={{ padding: 2, maxWidth: 'min(700px, 100%)' }}>
                <Grid item xs={12}>
                    <Typography
                        component="h1"
                        variant="h3"
                        sx={{
                            textAlign: 'center',
                            sx: { marginTop: 2, marginBottom: 2 },
                        }}
                    >Create Standard</Typography>
                </Grid>
                <Grid item xs={12}>
                    <RelationshipButtons
                        objectType={ObjectType.Standard}
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
                        handleCurrent={handleLanguageSelect}
                        selectedLanguages={languages}
                        session={session}
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
                <Grid item xs={12}>
                    <TextField
                        fullWidth
                        id="description"
                        name="description"
                        label="description"
                        multiline
                        minRows={4}
                        value={formik.values.description}
                        onBlur={formik.handleBlur}
                        onChange={formik.handleChange}
                        error={formik.touched.description && Boolean(formik.errors.description)}
                        helperText={formik.touched.description && formik.errors.description}
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
                            (schema && generateInputComponent({
                                disabled: true,
                                fieldData: schema,
                                formik: previewFormik,
                                session,
                                onUpload: () => { },
                                zIndex,
                            })) :
                            <Box>
                                <Selector
                                    fullWidth
                                    options={InputTypeOptions}
                                    selected={inputType}
                                    handleChange={handleInputTypeSelect}
                                    getOptionLabel={(option: InputTypeOption) => option.label}
                                    inputAriaLabel='input-type-selector'
                                    label="Type"
                                    style={{
                                        marginBottom: 2
                                    }}
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
                        canEdit={true}
                        handleUpdate={handleResourcesUpdate}
                        loading={false}
                        session={session}
                        mutate={false}
                        zIndex={zIndex}
                    />
                </Grid>
                <Grid item xs={12}>
                    <TagSelector
                        session={session}
                        tags={tags}
                        onTagAdd={addTag}
                        onTagRemove={removeTag}
                        onTagsClear={clearTags}
                    />
                </Grid>
                <Grid item xs={6} marginBottom={4}>
                    <Button
                        disabled={Boolean(!isLoggedIn || formik.isSubmitting)}
                        fullWidth
                        type="submit"
                        startIcon={<CreateIcon />}
                    >Create</Button>
                </Grid>
                <Grid item xs={6} marginBottom={4}>
                    <Button
                        fullWidth
                        onClick={onCancel}
                        startIcon={<CancelIcon />}
                    >Cancel</Button>
                </Grid>
            </Grid>
        </form>
    )
}