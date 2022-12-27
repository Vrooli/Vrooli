import { Box, Button, Palette, Stack, useTheme } from "@mui/material";
import { CommentContainer, ContentCollapse, DateDisplay, ObjectActionsRow, ObjectTitle, RelationshipButtons, ResourceListHorizontal, SnackSeverity, StatsCompact, TagList, TextCollapse, VersionDisplay } from "components";
import { useCallback, useEffect, useMemo, useState } from "react";
import { formikToRunInputs, getTranslation, getUserLanguages, ObjectAction, ObjectActionComplete, openObject, PubSub, runInputsToFormik, standardToFieldData, TagShape, uuidToBase36 } from "utils";
import { useLocation } from '@shared/route';
import { SubroutineViewProps } from "../types";
import { FieldData } from "forms/types";
import { generateInputWithLabel } from 'forms/generators';
import { useFormik } from "formik";
import { APP_LINKS } from "@shared/consts";
import { ResourceList, Routine } from "types";
import { RelationshipsObject } from "components/inputs/types";
import { smallHorizontalScrollbar } from "components/lists/styles";
import { uuid } from "@shared/uuid";
import { SuccessIcon } from "@shared/icons";

const containerProps = (palette: Palette) => ({
    boxShadow: 1,
    background: palette.background.paper,
    borderRadius: 1,
    overflow: 'overlay',
    marginTop: 4,
    marginBottom: 4,
    padding: 2,
})

export const SubroutineView = ({
    loading,
    handleUserInputsUpdate,
    handleSaveProgress,
    owner,
    routine,
    run,
    session,
    zIndex,
}: SubroutineViewProps) => {
    return null;//TODO
    // const { palette } = useTheme();
    // const [, setLocation] = useLocation();
    // const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);

    // const [internalRoutine, setInternalRoutine] = useState(routine);
    // useEffect(() => {
    //     setInternalRoutine(routine);
    // }, [routine]);
    // const updateRoutine = useCallback((routine: Routine) => { setInternalRoutine(routine); }, [setInternalRoutine]);

    // const { description, instructions, title } = useMemo(() => {
    //     const languages = getUserLanguages(session);
    //     const { description, instructions, title } = getTranslation(internalRoutine, languages, true);
    //     return {
    //         description,
    //         instructions,
    //         title,
    //     }
    // }, [internalRoutine, session]);

    // const confirmLeave = useCallback((callback: () => any) => {
    //     // Confirmation dialog for leaving routine
    //     PubSub.get().publishAlertDialog({
    //         messageKey: 'RunStopConfirm',
    //         buttons: [
    //             {
    //                 labelKey: 'Yes',
    //                 onClick: () => {
    //                     // Save progress
    //                     handleSaveProgress();
    //                     // Trigger callback
    //                     callback();
    //                 }
    //             },
    //             { labelKey: 'Cancel' },
    //         ]
    //     });
    // }, [handleSaveProgress]);

    // // The schema and formik keys for the form
    // const formValueMap = useMemo<{ [fieldName: string]: FieldData }>(() => {
    //     if (!internalRoutine) return {};
    //     const schemas: { [fieldName: string]: FieldData } = {};
    //     for (let i = 0; i < internalRoutine.inputs?.length; i++) {
    //         const currInput = internalRoutine.inputs[i];
    //         if (!currInput.standard) continue;
    //         const currSchema = standardToFieldData({
    //             description: getTranslation(currInput, getUserLanguages(session), false).description ?? getTranslation(currInput.standard, getUserLanguages(session), false).description,
    //             fieldName: `inputs-${currInput.id}`,
    //             helpText: getTranslation(currInput, getUserLanguages(session), false).helpText,
    //             props: currInput.standard.props,
    //             name: currInput.name ?? currInput.standard.name,
    //             type: currInput.standard.type,
    //             yup: currInput.standard.yup,
    //         });
    //         if (currSchema) {
    //             schemas[currSchema.fieldName] = currSchema;
    //         }
    //     }
    //     return schemas;
    // }, [internalRoutine, session]);
    // const formik = useFormik({
    //     initialValues: Object.entries(formValueMap).reduce((acc, [key, value]) => {
    //         acc[key] = value.props.defaultValue ?? '';
    //         return acc;
    //     }, {}),
    //     enableReinitialize: true,
    //     onSubmit: () => { },
    // });

    // /**
    //  * Update formik values with the current user inputs, if any
    //  */
    // useEffect(() => {
    //     console.log('useeffect1 calculating preview formik values', run)
    //     if (!run?.inputs || !Array.isArray(run?.inputs) || run.inputs.length === 0) return;
    //     console.log('useeffect 1calling runInputsToFormik', run.inputs)
    //     const updatedValues = runInputsToFormik(run.inputs);
    //     console.log('useeffect1 updating formik, values', updatedValues)
    //     formik.setValues(updatedValues);
    // },
    //     // eslint-disable-next-line react-hooks/exhaustive-deps
    //     [formik.setValues, run?.inputs]);

    // /**
    //  * Update run with updated user inputs
    //  */
    // useEffect(() => {
    //     if (!formik.values) return;
    //     const updatedValues = formikToRunInputs(formik.values);
    //     handleUserInputsUpdate(updatedValues);
    // }, [handleUserInputsUpdate, formik.values, run?.inputs]);

    // /**
    //  * Copy current value of input to clipboard
    //  * @param fieldName Name of input
    //  */
    // const copyInput = useCallback((fieldName: string) => {
    //     const input = formik.values[fieldName];
    //     if (input) {
    //         navigator.clipboard.writeText(input);
    //         PubSub.get().publishSnack({ messageKey: 'CopiedToClipboard', severity: SnackSeverity.Success });
    //     } else {
    //         PubSub.get().publishSnack({ messageKey: 'InputEmpty', severity: SnackSeverity.Error });
    //     }
    // }, [formik.values]);

    // const inputComponents = useMemo(() => {
    //     if (!internalRoutine?.inputs || !Array.isArray(internalRoutine?.inputs) || internalRoutine.inputs.length === 0) return null;
    //     return (
    //         <Box>
    //             {Object.values(formValueMap).map((fieldData: FieldData, index: number) => (
    //                 generateInputWithLabel({
    //                     copyInput,
    //                     disabled: false,
    //                     fieldData,
    //                     formik: formik,
    //                     index,
    //                     session,
    //                     textPrimary: palette.background.textPrimary,
    //                     onUpload: () => { },
    //                     zIndex,
    //                 })
    //             ))}
    //         </Box>
    //     )
    // }, [copyInput, formValueMap, palette.background.textPrimary, formik, internalRoutine?.inputs, session, zIndex]);

    // const onEdit = useCallback(() => {
    //     setLocation(`${APP_LINKS.Routine}/edit/${uuidToBase36(internalRoutine?.id ?? '')}`);
    // }, [internalRoutine?.id, setLocation]);

    // const [isAddCommentOpen, setIsAddCommentOpen] = useState(false);
    // const openAddCommentDialog = useCallback(() => { setIsAddCommentOpen(true); }, []);
    // const closeAddCommentDialog = useCallback(() => { setIsAddCommentOpen(false); }, []);

    // const onActionStart = useCallback((action: ObjectAction) => {
    //     switch (action) {
    //         case ObjectAction.Comment:
    //             openAddCommentDialog();
    //             break;
    //         case ObjectAction.Edit:
    //             onEdit();
    //             break;
    //         case ObjectAction.Stats:
    //             //TODO
    //             break;
    //     }
    // }, [onEdit, openAddCommentDialog]);

    // const onActionComplete = useCallback((action: ObjectActionComplete, data: any) => {
    //     switch (action) {
    //         case ObjectActionComplete.VoteDown:
    //         case ObjectActionComplete.VoteUp:
    //             if (data.success) {
    //                 setInternalRoutine({
    //                     ...internalRoutine,
    //                     isUpvoted: action === ObjectActionComplete.VoteUp,
    //                 } as any)
    //             }
    //             break;
    //         case ObjectActionComplete.Star:
    //         case ObjectActionComplete.StarUndo:
    //             if (data.success) {
    //                 setInternalRoutine({
    //                     ...internalRoutine,
    //                     isStarred: action === ObjectActionComplete.Star,
    //                 } as any)
    //             }
    //             break;
    //         case ObjectActionComplete.Fork:
    //             openObject(data.routine, setLocation);
    //             window.location.reload();
    //             break;
    //     }
    // }, [internalRoutine, setLocation]);

    // // Handle relationships
    // const [relationships, setRelationships] = useState<RelationshipsObject>({
    //     isComplete: true,
    //     isPrivate: false,
    //     owner: null,
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
    // const [resourceList, setResourceList] = useState<ResourceList>({ id: uuid() } as any);

    // // Handle tags
    // const [tags, setTags] = useState<TagShape[]>([]);

    // useEffect(() => {
    //     setRelationships({
    //         isComplete: internalRoutine?.isComplete ?? false,
    //         isPrivate: internalRoutine?.isPrivate ?? false,
    //         owner: internalRoutine?.owner ?? null,
    //         parent: null,
    //         // parent: internalRoutine?.parent ?? null, TODO
    //         project: null, //TODO
    //     });
    //     setResourceList(internalRoutine?.resourceList ?? { id: uuid() } as any);
    //     setTags(internalRoutine?.tags ?? []);
    // }, [internalRoutine]);

    // return (
    //     <Box sx={{
    //         marginLeft: 'auto',
    //         marginRight: 'auto',
    //         width: 'min(100%, 700px)',
    //         padding: 2,
    //     }}>
    //         <ObjectTitle
    //             language={language}
    //             loading={loading}
    //             title={title}
    //             session={session}
    //             setLanguage={setLanguage}
    //             translations={internalRoutine?.translations ?? []}
    //             zIndex={zIndex}
    //         />
    //         {/* Resources */}
    //         {Array.isArray(resourceList.resources) && resourceList.resources.length > 0 && <ResourceListHorizontal
    //             title={'Resources'}
    //             list={resourceList}
    //             canEdit={false}
    //             handleUpdate={() => { }} // Intentionally blank
    //             loading={loading}
    //             session={session}
    //             zIndex={zIndex}
    //         />}
    //         {/* Box with description and instructions */}
    //         <Stack direction="column" spacing={4} sx={containerProps(palette)}>
    //             {/* Description */}
    //             <TextCollapse title="Description" text={description} loading={loading} loadingLines={2} />
    //             {/* Instructions */}
    //             <TextCollapse title="Instructions" text={instructions} loading={loading} loadingLines={4} />
    //         </Stack>
    //         <Box sx={containerProps(palette)}>
    //             <ContentCollapse title="Inputs">
    //                 {inputComponents}
    //                 <Button startIcon={<SuccessIcon />} fullWidth onClick={() => { }} color="secondary" sx={{ marginTop: 2 }}>Submit</Button>
    //             </ContentCollapse>
    //         </Box>
    //         {/* Action buttons */}
    //         <ObjectActionsRow
    //             exclude={[ObjectAction.Edit, ObjectAction.VoteDown, ObjectAction.VoteUp]} // Handled elsewhere
    //             onActionStart={onActionStart}
    //             onActionComplete={onActionComplete}
    //             object={internalRoutine}
    //             session={session}
    //             zIndex={zIndex}
    //         />
    //         <Box sx={containerProps(palette)}>
    //             <ContentCollapse isOpen={false} title="Additional Information">
    //                 {/* Relationships */}
    //                 <RelationshipButtons
    //                     isEditing={false}
    //                     objectType={'Routine'}
    //                     onRelationshipsChange={onRelationshipsChange}
    //                     relationships={relationships}
    //                     session={session}
    //                     zIndex={zIndex}
    //                 />
    //                 {/* Tags */}
    //                 {tags.length > 0 && <TagList
    //                     maxCharacters={30}
    //                     parentId={routine?.id ?? ''}
    //                     session={session}
    //                     tags={tags as any[]}
    //                     sx={{ ...smallHorizontalScrollbar(palette), marginTop: 4 }}
    //                 />}
    //                 {/* Date and version labels */}
    //                 <Stack direction="row" spacing={1} mt={2} mb={1}>
    //                     {/* Date created */}
    //                     <DateDisplay
    //                         loading={loading}
    //                         showIcon={true}
    //                         timestamp={internalRoutine?.created_at}
    //                     />
    //                     <VersionDisplay
    //                         confirmVersionChange={confirmLeave}
    //                         currentVersion={internalRoutine?.version}
    //                         prefix={" - "}
    //                         versions={internalRoutine?.versions}
    //                     />
    //                 </Stack>
    //                 {/* Votes, reports, and other basic stats */}
    //                 <StatsCompact
    //                     handleObjectUpdate={updateRoutine}
    //                     loading={loading}
    //                     object={internalRoutine ?? null}
    //                     session={session}
    //                 />
    //             </ContentCollapse>
    //         </Box>
    //         {/* Comments */}
    //         <Box sx={containerProps(palette)}>
    //             <CommentContainer
    //                 forceAddCommentOpen={isAddCommentOpen}
    //                 isOpen={false}
    //                 language={language}
    //                 objectId={internalRoutine?.id ?? ''}
    //                 objectType={CommentFor.Routine}
    //                 onAddCommentClose={closeAddCommentDialog}
    //                 session={session}
    //                 zIndex={zIndex}
    //             />
    //         </Box>
    //     </Box>
    // )
}