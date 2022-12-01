import { Box, CircularProgress, Grid, TextField } from "@mui/material"
import { ResourceListUsedFor } from "@shared/consts";
import { useMutation, useLazyQuery } from "@apollo/client";
import { standard, standardVariables } from "graphql/generated/standard";
import { standardQuery } from "graphql/query";
import { useCallback, useEffect, useMemo, useState } from "react";
import { StandardUpdateProps } from "../types";
import { mutationWrapper } from 'graphql/utils/graphqlWrapper';
import { standardTranslationUpdate, standardUpdate as validationSchema } from '@shared/validation';
import { useFormik } from 'formik';
import { standardUpdateMutation } from "graphql/mutation";
import { addEmptyTranslation, base36ToUuid, getFormikErrorsWithTranslations, getLastUrlPart, getTranslationData, getUserLanguages, handleTranslationBlur, handleTranslationChange, ObjectType, PubSub, removeTranslation, shapeStandardUpdate, TagShape, usePromptBeforeUnload } from "utils";
import { GridSubmitButtons, LanguageInput, PageTitle, RelationshipButtons, ResourceListHorizontal, SnackSeverity, TagSelector, userFromSession } from "components";
import { ResourceList } from "types";
import { DUMMY_ID, uuid, uuidValidate } from '@shared/uuid';
import { standardUpdateVariables, standardUpdate_standardUpdate } from "graphql/generated/standardUpdate";
import { RelationshipsObject } from "components/inputs/types";

export const StandardUpdate = ({
    onCancel,
    onUpdated,
    session,
    zIndex,
}: StandardUpdateProps) => {
    return null;//TODO

    // // Fetch existing data
    // const { id, versionGroupId } = useMemo(() => {
    //     // URL is /object/:versionGroupId/?:id
    //     const last = base36ToUuid(getLastUrlPart(0), false);
    //     const secondLast = base36ToUuid(getLastUrlPart(1), false);
    //     return {
    //         id: uuidValidate(secondLast) ? last : undefined,
    //         versionGroupId: uuidValidate(secondLast) ? secondLast : last,
    //     }
    // }, []);
    // const [getData, { data, loading }] = useLazyQuery<standard, standardVariables>(standardQuery, { errorPolicy: 'all' });
    // useEffect(() => {
    //     if (uuidValidate(id) || uuidValidate(versionGroupId)) getData({ variables: { input: { id, versionGroupId } } });
    //     else PubSub.get().publishSnack({ messageKey: 'InvalidUrlId', severity: SnackSeverity.Error });
    // }, [getData, id, versionGroupId])
    // const standard = useMemo(() => data?.standard, [data]);

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

    // // Handle resources
    // const [resourceList, setResourceList] = useState<ResourceList>({ id: uuid(), usedFor: ResourceListUsedFor.Display } as any);
    // const handleResourcesUpdate = useCallback((updatedList: ResourceList) => {
    //     setResourceList(updatedList);
    // }, [setResourceList]);

    // // Handle tags
    // const [tags, setTags] = useState<TagShape[]>([]);
    // const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

    // useEffect(() => {
    //     setRelationships({
    //         isComplete: false, //TODO
    //         isPrivate: standard?.isPrivate ?? false,
    //         owner: standard?.creator ?? null,
    //         parent: null,
    //         // parent: standard?.parent ?? null, TODO
    //         project: null // TODO
    //     });
    //     setResourceList(standard?.resourceLists?.find(list => list.usedFor === ResourceListUsedFor.Display) ?? { id: uuid(), usedFor: ResourceListUsedFor.Display } as any);
    //     setTags(standard?.tags ?? []);
    // }, [standard]);

    // // Handle update
    // const [mutation] = useMutation(standardUpdateMutation);
    // const formik = useFormik({
    //     initialValues: {
    //         translationsUpdate: standard?.translations ?? [{
    //             id: DUMMY_ID,
    //             language: getUserLanguages(session)[0],
    //             description: '',
    //             jsonVariable: null, //TODO
    //         }],
    //     },
    //     enableReinitialize: true, // Needed because existing data is obtained from async fetch
    //     validationSchema,
    //     onSubmit: (values) => {
    //         if (!standard) {
    //             PubSub.get().publishSnack({ messageKey: 'CouldNotReadStandard', severity: SnackSeverity.Error });
    //             return;
    //         }
    //         // Update
    //         mutationWrapper<standardUpdate_standardUpdate, standardUpdateVariables>({
    //             mutation,
    //             input: shapeStandardUpdate(standard, {
    //                 id: standard.id,
    //                 resourceLists: [resourceList],
    //                 tags: tags,
    //                 translations: values.translationsUpdate.map(t => ({
    //                     ...t,
    //                     id: t.id === DUMMY_ID ? uuid() : t.id,
    //                 })),
    //             }),
    //             onSuccess: (data) => { onUpdated(data) },
    //             onError: () => { formik.setSubmitting(false) },
    //         })
    //     },
    // });
    // usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    // // Handle translations
    // const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    // const { description, errorDescription, touchedDescription, errors } = useMemo(() => {
    //     const { error, touched, value } = getTranslationData(formik, 'translationsUpdate', language);
    //     return {
    //         description: value?.description ?? '',
    //         errorDescription: error?.description ?? '',
    //         touchedDescription: touched?.description ?? false,
    //         errors: getFormikErrorsWithTranslations(formik, 'translationsUpdate', standardTranslationUpdate),
    //     }
    // }, [formik, language]);
    // const languages = useMemo(() => formik.values.translationsUpdate.map(t => t.language), [formik.values.translationsUpdate]);
    // const handleAddLanguage = useCallback((newLanguage: string) => {
    //     setLanguage(newLanguage);
    //     addEmptyTranslation(formik, 'translationsUpdate', newLanguage);
    // }, [formik]);
    // const handleLanguageDelete = useCallback((language: string) => {
    //     const newLanguages = [...languages.filter(l => l !== language)]
    //     if (newLanguages.length === 0) return;
    //     setLanguage(newLanguages[0]);
    //     removeTranslation(formik, 'translationsUpdate', language);
    // }, [formik, languages]);
    // // Handles blur on translation fields
    // const onTranslationBlur = useCallback((e: { target: { name: string } }) => {
    //     handleTranslationBlur(formik, 'translationsUpdate', e, language)
    // }, [formik, language]);
    // // Handles change on translation fields
    // const onTranslationChange = useCallback((e: { target: { name: string, value: string } }) => {
    //     handleTranslationChange(formik, 'translationsUpdate', e, language)
    // }, [formik, language]);

    // const formInput = useMemo(() => (
    //     <Grid container spacing={2} sx={{ padding: 2, marginBottom: 4, maxWidth: 'min(700px, 100%)' }}>
    //         <Grid item xs={12}>
    //             <PageTitle title="Update Standard" />
    //         </Grid>
    //         <Grid item xs={12} mb={4}>
    //             <RelationshipButtons
    //                 isEditing={true}
    //                 objectType={ObjectType.Standard}
    //                 onRelationshipsChange={onRelationshipsChange}
    //                 relationships={relationships}
    //                 session={session}
    //                 zIndex={zIndex}
    //             />
    //         </Grid>
    //         <Grid item xs={12}>
    //             <LanguageInput
    //                 currentLanguage={language}
    //                 handleAdd={handleAddLanguage}
    //                 handleDelete={handleLanguageDelete}
    //                 handleCurrent={setLanguage}
    //                 session={session}
    //                 translations={formik.values.translationsUpdate}
    //                 zIndex={zIndex}
    //             />
    //         </Grid>
    //         <Grid item xs={12} mb={4}>
    //             <TextField
    //                 fullWidth
    //                 id="description"
    //                 name="description"
    //                 label="description"
    //                 multiline
    //                 minRows={4}
    //                 value={description}
    //                 onBlur={onTranslationBlur}
    //                 onChange={onTranslationChange}
    //                 error={touchedDescription && Boolean(errorDescription)}
    //                 helperText={touchedDescription && errorDescription}
    //             />
    //         </Grid>
    //         {/* TODO versioning */}
    //         <Grid item xs={12}>
    //             <ResourceListHorizontal
    //                 title={'Resources'}
    //                 list={resourceList}
    //                 canEdit={true}
    //                 handleUpdate={handleResourcesUpdate}
    //                 loading={loading}
    //                 session={session}
    //                 mutate={false}
    //                 zIndex={zIndex}
    //             />
    //         </Grid>
    //         <Grid item xs={12} marginBottom={4}>
    //             <TagSelector
    //                 handleTagsUpdate={handleTagsUpdate}
    //                 session={session}
    //                 tags={tags}
    //             />
    //         </Grid>
    //         <GridSubmitButtons
    //             errors={errors}
    //             isCreate={false}
    //             loading={formik.isSubmitting}
    //             onCancel={onCancel}
    //             onSetSubmitting={formik.setSubmitting}
    //             onSubmit={formik.handleSubmit}
    //         />
    //     </Grid>
    // ), [onRelationshipsChange, relationships, session, zIndex, language, handleAddLanguage, handleLanguageDelete, formik.values.translationsUpdate, formik.isSubmitting, formik.setSubmitting, formik.handleSubmit, description, onTranslationBlur, onTranslationChange, touchedDescription, errorDescription, resourceList, handleResourcesUpdate, loading, handleTagsUpdate, tags, errors, onCancel]);


    // return (
    //     <form onSubmit={formik.handleSubmit} style={{
    //         display: 'flex',
    //         alignItems: 'center',
    //         justifyContent: 'center',
    //     }}
    //     >
    //         {loading ? (
    //             <Box sx={{
    //                 position: 'absolute',
    //                 top: '-5vh', // Half of toolbar height
    //                 width: '100%',
    //                 height: '100%',
    //                 display: 'flex',
    //                 justifyContent: 'center',
    //                 alignItems: 'center',
    //             }}>
    //                 <CircularProgress size={100} color="secondary" />
    //             </Box>
    //         ) : formInput}
    //     </form>
    // )
}