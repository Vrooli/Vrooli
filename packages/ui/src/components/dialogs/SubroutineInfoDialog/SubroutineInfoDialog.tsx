import { DUMMY_ID, exists, nodeRoutineListItemValidation, noopSubmit, orDefault, ResourceList, routineVersionTranslationValidation, Session, uuid } from "@local/shared";
import { Box, Grid, IconButton, Tooltip, Typography, useTheme } from "@mui/material";
import { BottomActionsButtons } from "components/buttons/BottomActionsButtons/BottomActionsButtons";
import { EditableTextCollapse } from "components/containers/EditableTextCollapse/EditableTextCollapse";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { IntegerInput } from "components/inputs/IntegerInput/IntegerInput";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { InputOutputContainer } from "components/lists/inputOutput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { ResourceListHorizontal } from "components/lists/resource";
import { TagList } from "components/lists/TagList/TagList";
import { Title } from "components/text/Title/Title";
import { VersionDisplay } from "components/text/VersionDisplay/VersionDisplay";
import { SessionContext } from "contexts/SessionContext";
import { Formik, useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { SubroutineFormProps } from "forms/types";
import { useTranslatedFields } from "hooks/useTranslatedFields";
import { CloseIcon, OpenInNewIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { FormContainer } from "styles";
import { getCurrentUser } from "utils/authentication/session";
import { combineErrorsWithTranslations, getTranslation, getUserLanguages } from "utils/display/translationTools";
import { PubSub } from "utils/pubsub";
import { NodeRoutineListItemShape, shapeNodeRoutineListItem } from "utils/shape/models/nodeRoutineListItem";
import { RoutineVersionInputShape } from "utils/shape/models/routineVersionInput";
import { RoutineVersionOutputShape } from "utils/shape/models/routineVersionOutput";
import { TagShape } from "utils/shape/models/tag";
import { validateFormValues } from "utils/validateFormValues";
import { routineInitialValues } from "views/objects/routine";
import { LargeDialog } from "../LargeDialog/LargeDialog";
import { SubroutineInfoDialogProps } from "../types";

export const subroutineInitialValues = (
    session: Session | undefined,
    existing?: NodeRoutineListItemShape | null | undefined,
): NodeRoutineListItemShape => ({
    __typename: "NodeRoutineListItem" as const,
    id: uuid(),
    index: existing?.index ?? 0,
    isOptional: existing?.isOptional ?? false,
    list: existing?.list ?? {} as any,
    routineVersion: routineInitialValues(session, existing?.routineVersion as any),
    translations: orDefault(existing?.translations, [{
        __typename: "NodeRoutineListItemTranslation" as const,
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        description: "",
        name: "",
    }]),
});

export const transformSubroutineValues = (values: NodeRoutineListItemShape, existing: NodeRoutineListItemShape, isCreate: boolean) =>
    isCreate ? shapeNodeRoutineListItem.create(values) : shapeNodeRoutineListItem.update(existing as NodeRoutineListItemShape, values);

const SubroutineForm = ({
    canUpdateRoutineVersion,
    dirty,
    handleReorder,
    handleUpdate,
    handleViewFull,
    isCreate,
    isEditing,
    isOpen,
    numSubroutines,
    onClose,
    values,
    versions,
    ...props
}: SubroutineFormProps) => {
    const session = useContext(SessionContext);
    const theme = useTheme();
    const { t } = useTranslation();

    // Handle translations
    const {
        handleAddLanguage,
        handleDeleteLanguage,
        language,
        languages,
        setLanguage,
        translationErrors,
    } = useTranslatedFields({
        defaultLanguage: getUserLanguages(session)[0],
        fields: ["description", "instructions", "name"],
        validationSchema: routineVersionTranslationValidation[isCreate ? "create" : "update"]({ env: import.meta.env.PROD ? "production" : "development" }),
    });

    const isLoading = useMemo(() => props.isSubmitting, [props.isSubmitting]);

    const [indexField] = useField<number>("index");
    const [isInternalField] = useField<boolean>("routineVersion.root.isInternal");
    const [inputsField, , inputsHelpers] = useField<RoutineVersionInputShape[]>("routineVersion.inputs");
    const [outputsField, , outputsHelpers] = useField<RoutineVersionOutputShape[]>("routineVersion.outputs");
    const [resourceListField, , resourceListHelpers] = useField<ResourceList>("routineVersion.resourceList");
    const [tagsField] = useField<TagShape[]>("routineVersion.root.tags");
    const [versionlabelField] = useField<string>("routineVersion.versionLabel");
    const [versionsField] = useField<{ versionLabel: string }[]>("routineVersion.root.versions");

    // If name or description doesn't exist, add them using the routine version
    const [nameField, , nameHelpers] = useField<string>("translations.0.name");
    const [, , descriptionHelpers] = useField<string>("translations.0.description");
    const nameDescriptionAttempted = useRef(false);
    useEffect(() => {
        if (nameDescriptionAttempted.current) return;
        const nodeTranslations = getTranslation(values, [language], true);
        const routineVersionTranslations = getTranslation(values.routineVersion, [language], true);
        const hasName = nodeTranslations.name !== undefined && nodeTranslations.name !== "";
        const hasDescription = nodeTranslations.description !== undefined && nodeTranslations.description !== "";
        const hasRoutineVersionName = routineVersionTranslations.name !== undefined && routineVersionTranslations.name !== "";
        const hasRoutineVersionDescription = routineVersionTranslations.description !== undefined && routineVersionTranslations.description !== "";
        if (!hasName && hasRoutineVersionName) {
            nameHelpers.setValue(routineVersionTranslations.name ?? "");
        }
        if (!hasDescription && hasRoutineVersionDescription) {
            descriptionHelpers.setValue(routineVersionTranslations.description ?? "");
        }
        nameDescriptionAttempted.current = true;
    }, [descriptionHelpers, language, nameHelpers, values, values.routineVersion]);

    /**
     * Navigate to the subroutine's build page
     */
    const toGraph = useCallback(() => {
        handleViewFull();
    }, [handleViewFull]);

    const onSubmit = useCallback(() => {
        if (isEditing) {
            PubSub.get().publishSnack({ messageKey: "Unauthorized", severity: "Error" });
            return;
        }
        if (!values) return;
        // Check if subroutine index has changed
        const originalIndex = values.index;
        // Update the subroutine
        handleUpdate(values);
        // If the index has changed, reorder the subroutine
        originalIndex !== values.index && handleReorder(values.id, originalIndex, values.index);
    }, [handleReorder, handleUpdate, isEditing, values]);

    return (
        <LargeDialog
            id="subroutine-dialog"
            onClose={onClose}
            isOpen={isOpen}
            titleId={""}
        >
            {/* Title bar with close icon */}
            <Box sx={{
                display: "flex",
                flexDirection: "row",
                alignItems: "center",
                background: theme.palette.primary.dark,
                color: theme.palette.primary.contrastText,
                padding: 1,
            }}>
                {/* Subroutine name */}
                <Title
                    variant="header"
                    title={nameField.value}
                />
                {/* Version */}
                <VersionDisplay
                    currentVersion={{ versionLabel: versionlabelField.value }}
                    prefix={" - "}
                    versions={versionsField.value ?? []}
                />
                {/* Position */}
                {isEditing ? <Box sx={{ margin: "auto" }}>
                    <IntegerInput
                        label={t("Order")}
                        min={0}
                        max={numSubroutines - 1}
                        name="index"
                        // Offset by 1 because the index is 0-based, 
                        // and normies don't know the real way to count
                        offset={1}
                        tooltip="The order of this subroutine in its parent routine"
                    />
                </Box> :
                    <Typography variant="h6" ml={1} mr={1}>{`(${(indexField.value ?? 0) + 1} of ${(numSubroutines)})`}</Typography>
                }
                <Box sx={{
                    justifyContent: "end",
                    marginLeft: "auto",
                }}>
                    {/* Button to open in full page */}
                    {!isInternalField.value && (
                        <Tooltip title="Open in full page">
                            <IconButton onClick={toGraph}>
                                <OpenInNewIcon fill={theme.palette.primary.contrastText} />
                            </IconButton>
                        </Tooltip>
                    )}
                    {/* Close button */}
                    <IconButton onClick={onClose} sx={{
                        color: theme.palette.primary.contrastText,
                        borderBottom: `1px solid ${theme.palette.primary.dark}`,
                    }}>
                        <CloseIcon />
                    </IconButton>
                </Box>
            </Box>
            <BaseForm
                display="dialog"
                isLoading={false}
                maxWidth={1000}
            >
                <FormContainer>
                    <RelationshipList
                        isEditing={isEditing}
                        isFormDirty={dirty}
                        objectType={"Routine"}
                    />
                    {
                        (canUpdateRoutineVersion || (exists(resourceListField.value) && Array.isArray(resourceListField.value.resources) && resourceListField.value.resources.length > 0)) && <Grid item xs={12} mb={2}>
                            <ResourceListHorizontal
                                list={resourceListField.value}
                                canUpdate={canUpdateRoutineVersion}
                                handleUpdate={(newList) => { resourceListHelpers.setValue(newList); }}
                                mutate={false}
                                parent={{ __typename: "RoutineVersion", id: values.id }}
                            />
                        </Grid>
                    }
                    <FormContainer>
                        <Grid container spacing={2} p={2}>
                            <Grid item xs={12}>
                                {canUpdateRoutineVersion ? <LanguageInput
                                    currentLanguage={language}
                                    handleAdd={handleAddLanguage}
                                    handleDelete={handleDeleteLanguage}
                                    handleCurrent={setLanguage}
                                    languages={languages}
                                /> : <SelectLanguageMenu
                                    currentLanguage={language}
                                    handleCurrent={setLanguage}
                                    languages={languages}
                                />}
                            </Grid>
                            {/* Name */}
                            <Grid item xs={12}>
                                <EditableTextCollapse
                                    component='TranslatedTextField'
                                    isEditing={isEditing}
                                    name="name"
                                    props={{
                                        fullWidth: true,
                                        language,
                                    }}
                                    title={t("Name")}
                                />
                            </Grid>
                            {/* Description */}
                            <Grid item xs={12} md={6}>
                                <EditableTextCollapse
                                    component='TranslatedMarkdown'
                                    isEditing={isEditing}
                                    name="description"
                                    props={{
                                        language,
                                        maxChars: 2048,
                                        maxRows: 4,
                                        minRows: 4,
                                        placeholder: "Description",
                                    }}
                                    title={t("Description")}
                                />
                            </Grid>
                            {/* Instructions */}
                            <Grid item xs={12} md={6}>
                                <EditableTextCollapse
                                    component='TranslatedMarkdown'
                                    isEditing={canUpdateRoutineVersion}
                                    name="instructions"
                                    props={{
                                        language,
                                        placeholder: "Instructions",
                                        maxChars: 8192,
                                        maxRows: 10,
                                        minRows: 4,
                                    }}
                                    title={t("Instructions")}
                                />
                            </Grid>
                            <Grid item xs={12} md={6}>
                                {
                                    canUpdateRoutineVersion ? <TagSelector name='routineVersion.root.tags' /> :
                                        <TagList parentId={""} tags={(tagsField.value ?? []) as any[]} />
                                }
                            </Grid>
                            {
                                canUpdateRoutineVersion && <Grid item xs={12} md={6} mb={4}>
                                    <VersionInput
                                        fullWidth
                                        name="routineVersion.versionLabel"
                                        versions={(versionsField.value ?? []).map(v => v.versionLabel)}
                                    />
                                </Grid>
                            }
                        </Grid>
                    </FormContainer>
                    {/* Inputs/Outputs */}
                    <Grid container spacing={2} p={2}>
                        {(canUpdateRoutineVersion || (inputsField.value?.length > 0)) && <Grid item xs={12} md={6}>
                            <InputOutputContainer
                                isEditing={canUpdateRoutineVersion}
                                handleUpdate={inputsHelpers.setValue as any}
                                isInput={true}
                                language={language}
                                list={inputsField.value}
                            />
                        </Grid>}
                        {/* Outputs */}
                        {(canUpdateRoutineVersion || (outputsField.value?.length > 0)) && <Grid item xs={12} md={6}>
                            <InputOutputContainer
                                isEditing={canUpdateRoutineVersion}
                                handleUpdate={outputsHelpers.setValue as any}
                                isInput={false}
                                language={language}
                                list={outputsField.value}
                            />
                        </Grid>}
                    </Grid>
                </FormContainer>
            </BaseForm>
            {canUpdateRoutineVersion && <BottomActionsButtons
                display="dialog"
                errors={combineErrorsWithTranslations(props.errors, translationErrors)}
                hideButtons={!isEditing}
                isCreate={isCreate}
                loading={isLoading}
                onCancel={onClose}
                onSetSubmitting={props.setSubmitting}
                onSubmit={onSubmit}
            />}
        </LargeDialog>
    );
};

/**
 * Drawer to display a routine list item's info on the build page. 
 * Swipes up from bottom of screen
 */
export const SubroutineInfoDialog = ({
    data,
    handleUpdate,
    handleReorder,
    handleViewFull,
    isEditing,
    onClose,
    ...props
}: SubroutineInfoDialogProps) => {
    const session = useContext(SessionContext);

    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    const { subroutine, numSubroutines, nodeId } = useMemo(() => {
        if (!data?.node || !data?.routineItemId) return { subroutine: undefined, numSubroutines: 0, nodeId: "" };
        const subroutine = data.node.routineList.items.find(r => r.id === data.routineItemId);
        return { subroutine, numSubroutines: data.node.routineList.items.length, nodeId: data.node.id };
    }, [data]);

    const initialValues = useMemo(() => subroutineInitialValues(session, subroutine), [subroutine, session]);
    const canUpdate = useMemo<boolean>(() => isEditing && (subroutine?.routineVersion?.root?.isInternal || subroutine?.routineVersion?.root?.owner?.id === userId || subroutine?.routineVersion?.you?.canUpdate === true), [isEditing, subroutine?.routineVersion?.root?.isInternal, subroutine?.routineVersion?.root?.owner?.id, subroutine?.routineVersion?.you?.canUpdate, userId]);

    return (
        <Formik
            enableReinitialize={true}
            initialValues={initialValues}
            onSubmit={noopSubmit}
            validate={async (values) => await validateFormValues(values, subroutine!, false, transformSubroutineValues, nodeRoutineListItemValidation)}
        >
            {(formik) => <SubroutineForm
                canUpdateRoutineVersion={canUpdate}
                handleReorder={(_, oldIndex, newIndex) => { handleReorder(nodeId, oldIndex, newIndex); }}
                handleUpdate={handleUpdate as any}
                handleViewFull={handleViewFull}
                isCreate={false}
                isEditing={isEditing}
                isOpen={true}
                numSubroutines={numSubroutines}
                onClose={onClose}
                versions={[]}
                {...formik}
            />}
        </Formik>
    );
};
