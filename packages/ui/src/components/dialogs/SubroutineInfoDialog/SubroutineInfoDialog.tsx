import { Box, IconButton, Typography, styled } from "@mui/material";
import { SubroutineFormProps, SubroutineInfoDialogProps } from "../types.js";

const emptyArray = [] as const;

// export function subroutineInitialValues(
//     session: Session | undefined,
//     existing?: NodeRoutineListItemShape | null | undefined,
// ): NodeRoutineListItemShape {
//     return {
//         __typename: "NodeRoutineListItem" as const,
//         id: uuid(),
//         index: existing?.index ?? 0,
//         isOptional: existing?.isOptional ?? false,
//         list: existing?.list ?? {} as any,
//         routineVersion: routineSingleStepInitialValues(session, existing?.routineVersion as any),
//         translations: orDefault(existing?.translations, [{
//             __typename: "NodeRoutineListItemTranslation" as const,
//             id: DUMMY_ID,
//             language: getUserLanguages(session)[0],
//             description: "",
//             name: "",
//         }]),
//     };
// }

// export function transformSubroutineValues(values: NodeRoutineListItemShape, existing: NodeRoutineListItemShape, isCreate: boolean) {
//     return isCreate ? shapeNodeRoutineListItem.create(values) : shapeNodeRoutineListItem.update(existing as NodeRoutineListItemShape, values);
// }

const TitleBar = styled(Box)(({ theme }) => ({
    display: "flex",
    flexDirection: "row",
    alignItems: "center",
    background: theme.palette.primary.dark,
    color: theme.palette.primary.contrastText,
    padding: theme.spacing(1),
}));
const CloseButton = styled(IconButton)(({ theme }) => ({
    color: theme.palette.primary.contrastText,
}));
const WarningLabel = styled(Typography)(({ theme }) => ({
    display: "block",
    fontStyle: "italic",
    color: theme.palette.background.textSecondary,
    paddingLeft: 0,
    marginLeft: 0,
}));

const formSectionStyle = { overflowX: "hidden", marginBottom: 2 } as const;

function SubroutineForm({
    // canUpdateRoutineVersion,
    // dirty,
    // handleReorder,
    // handleUpdate,
    // handleViewFull,
    // isCreate,
    // isEditing,
    // isOpen,
    // numSubroutines,
    // onClose,
    // values,
    // ...props
}: SubroutineFormProps) {
    // const session = useContext(SessionContext);
    // const theme = useTheme();
    // const { t } = useTranslation();

    // // Handle translations
    // const {
    //     handleAddLanguage,
    //     handleDeleteLanguage,
    //     language,
    //     languages,
    //     setLanguage,
    //     translationErrors,
    // } = useTranslatedFields({
    //     defaultLanguage: getUserLanguages(session)[0],
    //     validationSchema: routineVersionTranslationValidation.create({ env: process.env.NODE_ENV }),
    // });

    // const isLoading = useMemo(() => props.isSubmitting, [props.isSubmitting]);

    // const [indexField] = useField<number>("index");
    // const [isInternalField] = useField<boolean>("routineVersion.root.isInternal");
    // const [resourceListField] = useField<ResourceListType>("routineVersion.resourceList");
    // const [tagsField] = useField<TagShape[]>("routineVersion.root.tags");
    // const [versionlabelField] = useField<string>("routineVersion.versionLabel");
    // const [versionsField] = useField<{ versionLabel: string }[]>("routineVersion.root.versions");

    // /**
    //  * Navigate to the subroutine's build page
    //  */
    // const toGraph = useCallback(() => {
    //     handleViewFull();
    // }, [handleViewFull]);

    // const onSubmit = useCallback(() => {
    //     if (isEditing) {
    //         PubSub.get().publish("snack", { messageKey: "Unauthorized", severity: "Error" });
    //         return;
    //     }
    //     if (!values) return;
    //     // Check if subroutine index has changed
    //     const originalIndex = values.index;
    //     // Update the subroutine
    //     handleUpdate(values);
    //     // If the index has changed, reorder the subroutine
    //     originalIndex !== values.index && handleReorder(values.id, originalIndex, values.index);
    // }, [handleReorder, handleUpdate, isEditing, values]);

    // const nameProps = useMemo(function namePropsMemo() {
    //     return {
    //         fullWidth: true,
    //         language,
    //     } as const;
    // }, [language]);

    // const descriptionProps = useMemo(function descriptionPropsMemo() {
    //     return {
    //         language,
    //         maxChars: 2048,
    //         maxRows: 4,
    //         minRows: 4,
    //         placeholder: "Description",
    //     } as const;
    // }, [language]);

    // const instructionsProps = useMemo(function instructionsPropsMemo() {
    //     return {
    //         language,
    //         placeholder: "Instructions",
    //         maxChars: 8192,
    //         maxRows: 10,
    //         minRows: 4,
    //     } as const;
    // }, [language]);

    // const configCallData = useMemo(function configCallDataMemo() {
    //     return parseConfigCallData(values.routineVersion?.configCallData, values.routineVersion?.routineType, console);
    // }, [values.routineVersion]);
    // const schemaInput = useMemo(function schemaInputMemo() {
    //     return parseSchemaInput(values.routineVersion?.configFormInput, values.routineVersion?.routineType, console);
    // }, [values.routineVersion]);
    // const schemaOutput = useMemo(function schemaOutputMemo() {
    //     return parseSchemaOutput(values.routineVersion?.configFormOutput, values.routineVersion?.routineType, console);
    // }, [values.routineVersion]);

    // const routineTypeComponents = useMemo(function routineTypeComponentsMemo() {
    //     if (!values.routineVersion) return null;

    //     const routineTypeBaseProps: RoutineFormPropsBase = {
    //         configCallData,
    //         disabled: false,
    //         display: "view",
    //         handleClearRun: noop, // Currently disabled in this context
    //         handleCompleteStep: noop, // Currently disabled in this context
    //         handleRunStep: noop, // Currently disabled in this context
    //         hasErrors: false, // Currently disabled in this context
    //         isCompleteStepDisabled: true, // Currently disabled in this context
    //         isPartOfMultiStepRoutine: true,
    //         isRunStepDisabled: true, // Currently disabled in this context
    //         isRunningStep: false,
    //         onConfigCallDataChange: noop, // Only used in edit mode
    //         onRunChange: noop, // Currently disabled in this context
    //         onSchemaInputChange: noop, // Only used in edit mode
    //         onSchemaOutputChange: noop, // Only used in edit mode
    //         routineId: values.id,
    //         routineName: getDisplay(values).title,
    //         schemaInput,
    //         schemaOutput,
    //         run: null, // Currently disabled in this context
    //     };

    //     switch (values.routineVersion.routineType) {
    //         case RoutineType.Api:
    //             return <RoutineApiForm {...routineTypeBaseProps} />;
    //         case RoutineType.Code:
    //             return <RoutineDataConverterForm {...routineTypeBaseProps} />;
    //         case RoutineType.Data:
    //             return <RoutineDataForm {...routineTypeBaseProps} />;
    //         case RoutineType.Generate:
    //             return <RoutineGenerateForm {...routineTypeBaseProps} />;
    //         case RoutineType.Informational:
    //             return <RoutineInformationalForm {...routineTypeBaseProps} />;
    //         //TODO
    //         // case RoutineType.MultiStep:
    //         //     return <RoutineMultiStepForm
    //         //         {...routineTypeBaseProps}
    //         //         isGraphOpen={isGraphOpen}
    //         //         handleGraphClose={closeGraph}
    //         //         handleGraphOpen={openGraph}
    //         //         handleGraphSubmit={noop}
    //         //         nodeLinks={existing.nodeLinks}
    //         //         nodes={existing.nodes}
    //         //         routineId={existing.id}
    //         //         translations={existing.translations}
    //         //         translationData={translationData}
    //         //     />;
    //         case RoutineType.SmartContract:
    //             return <RoutineSmartContractForm {...routineTypeBaseProps} />;
    //         default:
    //             return null;
    //     }
    // }, [configCallData, schemaInput, schemaOutput, values.routineVersion]);

    // const routineTypeOption = useMemo(function routineTypeMemo() {
    //     return routineTypes.find((option) => option.type === (values.routineVersion?.routineType ?? RoutineType.Informational));
    // }, [values.routineVersion]);

    // return (
    //     <LargeDialog
    //         id="subroutine-dialog"
    //         onClose={onClose}
    //         isOpen={isOpen}
    //         titleId={""}
    //     >
    //         <TitleBar>
    //             {/* Subroutine name */}
    //             <Title
    //                 variant="subheader"
    //                 title={firstString(getTranslation(values, [language]).name, t("Routine", { count: 1 }))}
    //             />
    //             {/* Version */}
    //             <VersionDisplay
    //                 currentVersion={{ versionLabel: versionlabelField.value }}
    //                 prefix={" - v"}
    //                 versions={versionsField.value ?? emptyArray}
    //             />
    //             <Box justifyContent="end" marginLeft="auto">
    //                 {!isInternalField.value && (
    //                     <Tooltip title="Open in full page">
    //                         <IconButton onClick={toGraph}>
    //                             <OpenInNewIcon fill={theme.palette.primary.contrastText} />
    //                         </IconButton>
    //                     </Tooltip>
    //                 )}
    //                 <CloseButton>
    //                     <CloseIcon />
    //                 </CloseButton>
    //             </Box>
    //         </TitleBar>
    //         <BaseForm
    //             display="Dialog"
    //             isLoading={false}
    //             maxWidth={1000}
    //         >
    //             <FormContainer>
    //                 {
    //                     (canUpdateRoutineVersion || (exists(resourceListField.value) && Array.isArray(resourceListField.value.resources) && resourceListField.value.resources.length > 0)) && <Grid item xs={12} mb={2}>
    //                         <ResourceList
    //                             horizontal
    //                             list={resourceListField.value}
    //                             canUpdate={false}
    //                             handleUpdate={noop}
    //                             mutate={false}
    //                             parent={{ __typename: "RoutineVersion", id: values.id }}
    //                         />
    //                     </Grid>
    //                 }
    //                 {/* Position */}
    //                 {isEditing && numSubroutines > 1 ? <Box>
    //                     <IntegerInput
    //                         fullWidth
    //                         label={"Order in routine list"}
    //                         min={0}
    //                         max={numSubroutines - 1}
    //                         name="index"
    //                         // Offset by 1 because the index is 0-based, and normies don't know the real way to count
    //                         offset={1}
    //                         tooltip="The order of this subroutine in its parent routine"
    //                     />
    //                 </Box> :
    //                     <Typography variant="h6" ml={1} mr={1}>
    //                         Order in routine list: {(indexField.value ?? 0) + 1} of {numSubroutines}
    //                     </Typography>
    //                 }
    //                 <FormSection sx={formSectionStyle}>
    //                     {isEditing && <WarningLabel variant="caption">
    //                         Changing the name or description only changes how it displays in the routine list, not the underlying subroutine
    //                     </WarningLabel>}
    //                     <EditableTextCollapse
    //                         component='TranslatedTextInput'
    //                         isEditing={isEditing}
    //                         name="name"
    //                         props={nameProps}
    //                         title={t("Name")}
    //                     />
    //                     <EditableTextCollapse
    //                         component='TranslatedMarkdown'
    //                         isEditing={isEditing}
    //                         name="description"
    //                         props={descriptionProps}
    //                         title={t("Description")}
    //                     />
    //                     <EditableTextCollapse
    //                         component='TranslatedMarkdown'
    //                         isEditing={false}
    //                         name="instructions"
    //                         props={instructionsProps}
    //                         title={t("Instructions")}
    //                     />
    //                     {isEditing ? <LanguageInput
    //                         currentLanguage={language}
    //                         handleAdd={handleAddLanguage}
    //                         handleDelete={handleDeleteLanguage}
    //                         handleCurrent={setLanguage}
    //                         languages={languages}
    //                     /> : <SelectLanguageMenu
    //                         currentLanguage={language}
    //                         handleCurrent={setLanguage}
    //                         languages={languages}
    //                     />}
    //                 </FormSection>
    //                 {Boolean(routineTypeComponents) && (
    //                     <FormSection>
    //                         <ContentCollapse
    //                             title={`Type: ${routineTypeOption?.label ?? "Unknown"}`}
    //                             helpText={routineTypeOption?.description ?? ""}
    //                         >
    //                             {/* warning label when routine is marked as incomplete */}
    //                             {values.routineVersion?.isComplete !== true && <WarningLabel variant="caption">
    //                                 This routine is marked as incomplete. It may change in the future.
    //                             </WarningLabel>}
    //                             <br />
    //                             <br />
    //                             {routineTypeComponents}
    //                         </ContentCollapse>
    //                     </FormSection>
    //                 )}
    //             </FormContainer>
    //         </BaseForm>
    //         {canUpdateRoutineVersion && <BottomActionsButtons
    //             display="Dialog"
    //             errors={combineErrorsWithTranslations(props.errors, translationErrors)}
    //             hideButtons={!isEditing}
    //             isCreate={isCreate}
    //             loading={isLoading}
    //             onCancel={onClose}
    //             onSetSubmitting={props.setSubmitting}
    //             onSubmit={onSubmit}
    //         />}
    //     </LargeDialog>
    // );
    return null;
}

/**
 * Drawer to display a routine list item's info on the build page. 
 * Swipes up from bottom of screen
 */
export function SubroutineInfoDialog({
    // data,
    // handleUpdate,
    // handleReorder,
    // handleViewFull,
    // isEditing,
    // ...props
}: SubroutineInfoDialogProps) {
    // const session = useContext(SessionContext);

    // const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    // const { subroutine, numSubroutines, nodeId } = useMemo(() => {
    //     if (!data?.node || !data?.routineItemId) return { subroutine: undefined, numSubroutines: 0, nodeId: "" };
    //     const subroutine = data.node.routineList.items.find(r => r.id === data.routineItemId);
    //     return { subroutine, numSubroutines: data.node.routineList.items.length, nodeId: data.node.id };
    // }, [data]);

    // const initialValues = useMemo(() => subroutineInitialValues(session, subroutine), [subroutine, session]);
    // const canUpdate = useMemo<boolean>(() => isEditing && (subroutine?.routineVersion?.root?.isInternal || subroutine?.routineVersion?.root?.owner?.id === userId || subroutine?.routineVersion?.you?.canUpdate === true), [isEditing, subroutine?.routineVersion?.root?.isInternal, subroutine?.routineVersion?.root?.owner?.id, subroutine?.routineVersion?.you?.canUpdate, userId]);

    // async function validateValues(values: NodeRoutineListItemShape) {
    //     if (!subroutine) return;
    //     return await validateFormValues(values, subroutine, false, transformSubroutineValues, nodeRoutineListItemValidation);
    // }

    // const doReorder = useCallback(function doReorderCallback(_: unknown, oldIndex: number, newIndex: number) {
    //     handleReorder(nodeId, oldIndex, newIndex);
    // }, [handleReorder, nodeId]);

    // return (
    //     <Formik
    //         enableReinitialize={true}
    //         initialValues={initialValues}
    //         onSubmit={noopSubmit}
    //         validate={validateValues}
    //     >
    //         {(formik) => <SubroutineForm
    //             canUpdateRoutineVersion={canUpdate}
    //             handleReorder={doReorder}
    //             handleUpdate={handleUpdate as any}
    //             handleViewFull={handleViewFull}
    //             isCreate={false}
    //             isEditing={isEditing}
    //             isOpen={!!data}
    //             numSubroutines={numSubroutines}
    //             {...formik}
    //             {...props}
    //         />}
    //     </Formik>
    // );
    return null;
}
