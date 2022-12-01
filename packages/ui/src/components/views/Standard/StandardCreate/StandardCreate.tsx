import { Box, Grid, TextField } from "@mui/material";
import { useMutation } from "@apollo/client";
import { mutationWrapper } from 'graphql/utils/graphqlWrapper';
import { standardCreate as validationSchema, standardTranslationCreate } from '@shared/validation';
import { useFormik } from 'formik';
import { standardCreateMutation } from "graphql/mutation";
import { addEmptyTranslation, getFormikErrorsWithTranslations, getTranslationData, getUserLanguages, handleTranslationBlur, handleTranslationChange, InputTypeOption, InputTypeOptions, ObjectType, parseSearchParams, removeTranslation, shapeStandardCreate, TagShape, usePromptBeforeUnload } from "utils";
import { StandardCreateProps } from "../types";
import { useCallback, useEffect, useMemo, useState } from "react";
import { GridSubmitButtons, LanguageInput, PageTitle, ResourceListHorizontal, Selector, TagSelector } from "components";
import { ResourceList } from "types";
import { ResourceListUsedFor } from "graphql/generated/globalTypes";
import { uuid } from '@shared/uuid';
import { FieldData } from "forms/types";
import { BaseStandardInput, PreviewSwitch, RelationshipButtons, userFromSession } from "components/inputs";
import { generateInputComponent, generateYupSchema } from "forms/generators";
import { standardCreateVariables, standardCreate_standardCreate } from "graphql/generated/standardCreate";
import { RelationshipsObject } from "components/inputs/types";
import { getCurrentUser } from "utils/authentication";

export const StandardCreate = ({
    onCreated,
    onCancel,
    session,
    zIndex,
}: StandardCreateProps) => {
    return null;//TODO

    // const [relationships, setRelationships] = useState<RelationshipsObject>({
    //     isComplete: false,
    //     isPrivate: false,
    //     owner: userFromSession(session),
    //     parent: null,
    //     project: null,
    // });
    // const onRelationshipsChange = useCallback((newRelationshipsObject: Partial<RelationshipsObject>) => {
    //     setRelationships({
    //         ...relationships,
    //         ...newRelationshipsObject,
    //     });
    // }, [relationships]);

    // // Handle input type selector
    // const [inputType, setInputType] = useState<InputTypeOption>(InputTypeOptions[1]);
    // const handleInputTypeSelect = useCallback((selected: InputTypeOption) => {
    //     setInputType(selected);
    // }, []);

    // // Handle standard schema
    // const [schema, setSchema] = useState<FieldData | null>(null);
    // const handleSchemaUpdate = useCallback((schema: FieldData) => {
    //     setSchema(schema);
    // }, []);
    // const [schemaKey] = useState(`standard-create-schema-preview-${Math.random().toString(36).substring(2, 15)}`);

    // const previewFormik = useFormik({
    //     initialValues: {
    //         preview: schema?.props?.defaultValue,
    //     },
    //     enableReinitialize: true,
    //     validationSchema: schema ? generateYupSchema({
    //         fields: [schema],
    //     }) : undefined,
    //     onSubmit: () => { },
    // });

    // // Handle resources
    // const [resourceList, setResourceList] = useState<ResourceList>({ id: uuid(), usedFor: ResourceListUsedFor.Display } as any);
    // const handleResourcesUpdate = useCallback((updatedList: ResourceList) => {
    //     setResourceList(updatedList);
    // }, [setResourceList]);

    // // Handle tags
    // const [tags, setTags] = useState<TagShape[]>([]);
    // const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

    // useEffect(() => {
    //     const params = parseSearchParams();
    //     if (typeof params.tag === 'string') setTags([{ tag: params.tag }]);
    //     else if (Array.isArray(params.tags)) setTags(params.tags.map((t: any) => ({ tag: t })));
    // }, []);

    // // Handle create
    // const [mutation] = useMutation(standardCreateMutation);
    // const formik = useFormik({
    //     initialValues: {
    //         id: uuid(),
    //         default: '',
    //         name: '',
    //         translationsCreate: [{
    //             id: uuid(),
    //             language: getUserLanguages(session)[0],
    //             description: '',
    //             jsonVariable: null, //TODO
    //         }],
    //         version: '1.0',
    //     },
    //     validationSchema,
    //     onSubmit: (values) => {
    //         mutationWrapper<standardCreate_standardCreate, standardCreateVariables>({
    //             mutation,
    //             input: shapeStandardCreate({
    //                 id: values.id,
    //                 default: values.default,
    //                 isInternal: false,
    //                 name: values.name,
    //                 props: JSON.stringify(schema?.props),
    //                 yup: JSON.stringify(schema?.yup),
    //                 translations: values.translationsCreate,
    //                 resourceLists: [resourceList],
    //                 tags: tags as any,
    //                 type: inputType.value,
    //                 version: values.version,
    //             }),
    //             onSuccess: (data) => {
    //                 onCreated(data)
    //             },
    //             onError: () => { formik.setSubmitting(false) },
    //         })
    //     },
    // });
    // usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    // // Handle translations
    // const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    // const { description, errorDescription, touchedDescription, errors } = useMemo(() => {
    //     const { error, touched, value } = getTranslationData(formik, 'translationsCreate', language);
    //     return {
    //         description: value?.description ?? '',
    //         errorDescription: error?.description ?? '',
    //         touchedDescription: touched?.description ?? false,
    //         errors: getFormikErrorsWithTranslations(formik, 'translationsCreate', standardTranslationCreate),
    //     }
    // }, [formik, language]);
    // const languages = useMemo(() => formik.values.translationsCreate.map(t => t.language), [formik.values.translationsCreate]);
    // const handleAddLanguage = useCallback((newLanguage: string) => {
    //     setLanguage(newLanguage);
    //     addEmptyTranslation(formik, 'translationsCreate', newLanguage);
    // }, [formik]);
    // const handleLanguageDelete = useCallback((language: string) => {
    //     const newLanguages = [...languages.filter(l => l !== language)]
    //     if (newLanguages.length === 0) return;
    //     setLanguage(newLanguages[0]);
    //     removeTranslation(formik, 'translationsCreate', language);
    // }, [formik, languages]);
    // // Handles blur on translation fields
    // const onTranslationBlur = useCallback((e: { target: { name: string } }) => {
    //     handleTranslationBlur(formik, 'translationsCreate', e, language)
    // }, [formik, language]);
    // // Handles change on translation fields
    // const onTranslationChange = useCallback((e: { target: { name: string, value: string } }) => {
    //     handleTranslationChange(formik, 'translationsCreate', e, language)
    // }, [formik, language]);

    // const isLoggedIn = useMemo(() => Boolean(getCurrentUser(session).id), [session]);

    // const [isPreviewOn, setIsPreviewOn] = useState<boolean>(false);
    // const onPreviewChange = useCallback((isOn: boolean) => { setIsPreviewOn(isOn); }, []);

    // return (
    //     <form onSubmit={formik.handleSubmit} style={{
    //         display: 'flex',
    //         alignItems: 'center',
    //         justifyContent: 'center',
    //     }}
    //     >
    //         <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
    //             <Grid item xs={12}>
    //                 <PageTitle title="Create Standard" />
    //             </Grid>
    //             <Grid item xs={12} mb={4}>
    //                 <RelationshipButtons
    //                     isEditing={true}
    //                     objectType={ObjectType.Standard}
    //                     onRelationshipsChange={onRelationshipsChange}
    //                     relationships={relationships}
    //                     session={session}
    //                     zIndex={zIndex}
    //                 />
    //             </Grid>
    //             <Grid item xs={12}>
    //                 <LanguageInput
    //                     currentLanguage={language}
    //                     handleAdd={handleAddLanguage}
    //                     handleDelete={handleLanguageDelete}
    //                     handleCurrent={setLanguage}
    //                     session={session}
    //                     translations={formik.values.translationsCreate}
    //                     zIndex={zIndex}
    //                 />
    //             </Grid>
    //             <Grid item xs={12}>
    //                 <TextField
    //                     fullWidth
    //                     id="name"
    //                     name="name"
    //                     label="Name"
    //                     value={formik.values.name}
    //                     onBlur={formik.handleBlur}
    //                     onChange={formik.handleChange}
    //                     error={formik.touched.name && Boolean(formik.errors.name)}
    //                     helperText={formik.touched.name && formik.errors.name}
    //                 />
    //             </Grid>
    //             <Grid item xs={12} mb={4}>
    //                 <TextField
    //                     fullWidth
    //                     id="description"
    //                     name="description"
    //                     label="description"
    //                     multiline
    //                     minRows={4}
    //                     value={description}
    //                     onBlur={onTranslationBlur}
    //                     onChange={onTranslationChange}
    //                     error={touchedDescription && Boolean(errorDescription)}
    //                     helperText={touchedDescription && errorDescription}
    //                 />
    //             </Grid>
    //             {/* <Grid item xs={12}>
    //                 <TextField
    //                     fullWidth
    //                     id="version"
    //                     name="version"
    //                     label="version"
    //                     value={formik.values.version}
    //                     onBlur={formik.handleBlur}
    //                     onChange={formik.handleChange}
    //                     error={formik.touched.version && Boolean(formik.errors.version)}
    //                     helperText={formik.touched.version && formik.errors.version}
    //                 />
    //             </Grid> */}
    //             {/* Standard build/preview */}
    //             <Grid item xs={12}>
    //                 <PreviewSwitch
    //                     isPreviewOn={isPreviewOn}
    //                     onChange={onPreviewChange}
    //                     sx={{
    //                         marginBottom: 2
    //                     }}
    //                 />
    //                 {
    //                     isPreviewOn ?
    //                         (schema && generateInputComponent({
    //                             disabled: true,
    //                             fieldData: schema,
    //                             formik: previewFormik,
    //                             session,
    //                             onUpload: () => { },
    //                             zIndex,
    //                         })) :
    //                         <Box>
    //                             <Selector
    //                                 fullWidth
    //                                 options={InputTypeOptions}
    //                                 selected={inputType}
    //                                 handleChange={handleInputTypeSelect}
    //                                 getOptionLabel={(option: InputTypeOption) => option.label}
    //                                 inputAriaLabel='input-type-selector'
    //                                 label="Type"
    //                                 sx={{ marginBottom: 2 }}
    //                             />
    //                             <BaseStandardInput
    //                                 fieldName="preview"
    //                                 inputType={inputType.value}
    //                                 isEditing={true}
    //                                 schema={schema}
    //                                 onChange={handleSchemaUpdate}
    //                                 storageKey={schemaKey}
    //                             />
    //                         </Box>
    //                 }
    //             </Grid>
    //             <Grid item xs={12}>
    //                 <ResourceListHorizontal
    //                     title={'Resources'}
    //                     list={resourceList}
    //                     canEdit={true}
    //                     handleUpdate={handleResourcesUpdate}
    //                     loading={false}
    //                     session={session}
    //                     mutate={false}
    //                     zIndex={zIndex}
    //                 />
    //             </Grid>
    //             <Grid item xs={12} mb={4}>
    //                 <TagSelector
    //                     handleTagsUpdate={handleTagsUpdate}
    //                     session={session}
    //                     tags={tags}
    //                 />
    //             </Grid>
    //             <GridSubmitButtons
    //                 disabledSubmit={!isLoggedIn}
    //                 errors={errors}
    //                 isCreate={true}
    //                 loading={formik.isSubmitting}
    //                 onCancel={onCancel}
    //                 onSetSubmitting={formik.setSubmitting}
    //                 onSubmit={formik.handleSubmit}
    //             />
    //         </Grid>
    //     </form>
    // )
}