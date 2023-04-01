import { Box, Grid, IconButton, Tooltip, Typography, useTheme } from "@mui/material";
import { ResourceList, Session } from "@shared/consts";
import { CloseIcon, OpenInNewIcon } from "@shared/icons";
import { DUMMY_ID, uuid } from "@shared/uuid";
import { nodeRoutineListItemValidation, routineVersionTranslationValidation } from "@shared/validation";
import { GridSubmitButtons } from "components/buttons/GridSubmitButtons/GridSubmitButtons";
import { EditableText } from "components/containers/EditableText/EditableText";
import { EditableTextCollapse } from "components/containers/EditableTextCollapse/EditableTextCollapse";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { IntegerInput } from "components/inputs/IntegerInput/IntegerInput";
import { LanguageInput } from "components/inputs/LanguageInput/LanguageInput";
import { TagSelector } from "components/inputs/TagSelector/TagSelector";
import { TranslatedTextField } from "components/inputs/TranslatedTextField/TranslatedTextField";
import { VersionInput } from "components/inputs/VersionInput/VersionInput";
import { InputOutputContainer } from "components/lists/inputOutput";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { ResourceListHorizontal } from "components/lists/resource";
import { TagList } from "components/lists/TagList/TagList";
import { VersionDisplay } from "components/text/VersionDisplay/VersionDisplay";
import { useField } from "formik";
import { BaseForm } from "forms/BaseForm/BaseForm";
import { routineInitialValues } from "forms/RoutineForm/RoutineForm";
import { SubroutineFormProps } from "forms/types";
import { exists } from "i18next";
import { forwardRef, useCallback, useContext, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { resourceList } from "tools/api/partial/resourceList";
import { getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";
import { validateAndGetYupErrors } from "utils/shape/general";
import { NodeShape } from "utils/shape/models/node";
import { NodeLinkShape } from "utils/shape/models/nodeLink";
import { NodeRoutineListItemShape, shapeNodeRoutineListItem } from "utils/shape/models/nodeRoutineListItem";
import { RoutineVersionInputShape } from "utils/shape/models/routineVersionInput";
import { RoutineVersionOutputShape } from "utils/shape/models/routineVersionOutput";

export const subroutineInitialValues = (
    session: Session | undefined,
    existing?: NodeRoutineListItemShape | null | undefined
): NodeRoutineListItemShape => ({
    __typename: 'NodeRoutineListItem' as const,
    id: uuid(),
    index: 0,
    isOptional: false,
    list: {} as any, //TODO
    routineVersion: routineInitialValues(session, existing?.routineVersion as any),
    translations: [{
        id: DUMMY_ID,
        language: getUserLanguages(session)[0],
        description: '',
        name: '',
    }],
    ...existing
})

export const transformSubroutineValues = (o: NodeRoutineListItemShape, u?: NodeRoutineListItemShape) => {
    return u === undefined
        ? shapeNodeRoutineListItem.create(o)
        : shapeNodeRoutineListItem.update(o, u)
}

export const validateSubroutineValues = async (values: NodeRoutineListItemShape, isCreate: boolean) => {
    const transformedValues = transformSubroutineValues(values);
    const validationSchema = isCreate
        ? nodeRoutineListItemValidation.create({})
        : nodeRoutineListItemValidation.update({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
}

export const SubroutineForm = forwardRef<any, SubroutineFormProps>(({
    canUpdate,
    dirty,
    isCreate,
    isEditing,
    isOpen,
    onCancel,
    values,
    versions,
    zIndex,
    ...props
}, ref) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
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
        fields: ['description', 'instructions', 'name'],
        validationSchema: routineVersionTranslationValidation.update({}),
    });

    const [idField] = useField<string>('id');
    const [nodesField, , nodesHelpers] = useField<NodeShape[]>('routineVersion.nodes');
    const [nodeLinksField, , nodeLinksHelpers] = useField<NodeLinkShape[]>('routineVersion.nodeLinks');
    const [inputsField, , inputsHelpers] = useField<RoutineVersionInputShape[]>('routineVersion.inputs');
    const [outputsField, , outputsHelpers] = useField<RoutineVersionOutputShape[]>('routineVersion.outputs');
    const [resourceListField, , resourceListHelpers] = useField<ResourceList>('routineVersion.resourceList');

    // On index change, update the index of the subroutine
    useEffect(() => {
        if (formik.values.index !== subroutine?.index) {
            handleReorder(data?.node?.id ?? '', subroutine?.index ?? 0, formik.values.index - 1);
        }
    }, [data?.node?.id, formik.values.index, handleReorder, subroutine?.id, subroutine?.index]);

    /**
     * Navigate to the subroutine's build page
     */
    const toGraph = useCallback(() => {
        handleViewFull();
    }, [handleViewFull]);

    return (
        <>
            {/* Title bar with close icon */}
            <Box sx={{
                display: 'flex',
                flexDirection: 'row',
                alignItems: 'center',
                background: palette.primary.dark,
                color: palette.primary.contrastText,
                padding: 1,
            }}>
                {/* Subroutine name and position */}
                <EditableText
                    component="TranslatedTextField"
                    isEditing={isEditing}
                    name="name"
                    props={{ language }}
                    variant="h5"
                />
                <Typography variant="h6" ml={1} mr={1}>{`(${(subroutine?.index ?? 0) + 1} of ${(data?.node?.routineList?.items?.length ?? 1)})`}</Typography>
                {/* Version */}
                <VersionDisplay
                    currentVersion={props.values.versionLabel}
                    prefix={" - "}
                    versions={subroutine?.routineVersion?.root?.versions ?? []}
                />
                {/* Button to open in full page */}
                {!subroutine?.routineVersion?.root?.isInternal && (
                    <Tooltip title="Open in full page">
                        <IconButton onClick={toGraph}>
                            <OpenInNewIcon fill={palette.primary.contrastText} />
                        </IconButton>
                    </Tooltip>
                )}
                {/* Close button */}
                <IconButton onClick={handleClose} sx={{
                    color: palette.primary.contrastText,
                    borderBottom: `1px solid ${palette.primary.dark}`,
                    justifyContent: 'end',
                    marginLeft: 'auto',
                }}>
                    <CloseIcon />
                </IconButton>
            </Box>
            <BaseForm
                dirty={dirty}
                isLoading={isLoading}
                ref={ref}
                style={{
                    display: 'block',
                    maxWidth: '700px',
                    margin: 'auto',
                }}
            >
                <Grid container spacing={2}>
                    {/* owner, project, isPrivate, etc. */}
                    <Grid item xs={12}>
                        <RelationshipList
                            isEditing={isEditing}
                            isFormDirty={formik.dirty}
                            objectType={'Routine'}
                            zIndex={zIndex}
                        />
                    </Grid>
                    <Grid item xs={12}>
                        {canUpdate ? <LanguageInput
                            currentLanguage={language}
                            handleAdd={handleAddLanguage}
                            handleDelete={handleDeleteLanguage}
                            handleCurrent={setLanguage}
                            translations={formik.values.translationsUpdate}
                            zIndex={zIndex}
                        /> : <SelectLanguageMenu
                            currentLanguage={language}
                            handleCurrent={setLanguage}
                            translations={formik.values.translationsUpdate}
                            zIndex={zIndex}
                        />}
                    </Grid>
                    {/* Position */}
                    <Grid item xs={12}>
                        {
                            canUpdate && <Box sx={{
                                marginBottom: 2,
                            }}>
                                <Typography variant="h6">{t('Order')}</Typography>
                                <IntegerInput
                                    disabled={!canUpdate}
                                    label={t('Order')}
                                    min={1}
                                    max={data?.node?.routineList?.items?.length ?? 1}
                                    name="index"
                                    tooltip="The order of this subroutine in its parent routine"
                                />
                            </Box>
                        }
                    </Grid>
                    {/* Name */}
                    {
                        canUpdate && <Grid item xs={12}>
                            <TranslatedTextField
                                fullWidth
                                label={t('Name')}
                                language={language}
                                name="name"
                            />
                        </Grid>
                    }
                    {/* Description */}
                    <Grid item xs={12} sm={6}>
                        <EditableTextCollapse
                            component='TranslatedTextField'
                            isEditing={canUpdate}
                            name="description"
                            props={{
                                fullWidth: true,
                                InputLabelProps: { shrink: true },
                                multiline: true,
                                maxRows: 3,
                            }}
                            title={t('Description')}
                        />
                    </Grid>
                    {/* Instructions */}
                    <Grid item xs={12} sm={6}>
                        <EditableTextCollapse
                            component='TranslatedMarkdownInput'
                            isEditing={isEditing}
                            name="instructions"
                            props={{
                                placeholder: "Instructions",
                                minRows: 3,
                            }}
                            title={t('Instructions')}
                        />
                    </Grid>
                    {
                        canUpdate && <Grid item xs={12}>
                            <VersionInput
                                fullWidth
                                name="routineVersion.versionLabel"
                                versions={subroutine?.routineVersion?.root?.versions ?? []}
                            />
                        </Grid>
                    }
                    {/* Inputs */}
                    {(canUpdate || (inputsList?.length > 0)) && <Grid item xs={12} sm={6}>
                        <InputOutputContainer
                            isEditing={canUpdate}
                            handleUpdate={handleInputsUpdate as any}
                            isInput={true}
                            language={language}
                            list={inputsList}
                            zIndex={zIndex}
                        />
                    </Grid>}
                    {/* Outputs */}
                    {(canUpdate || (outputsList?.length > 0)) && <Grid item xs={12} sm={6}>
                        <InputOutputContainer
                            isEditing={canUpdate}
                            handleUpdate={handleOutputsUpdate as any}
                            isInput={false}
                            language={language}
                            list={outputsList}
                            zIndex={zIndex}
                        />
                    </Grid>}
                    {
                        (canUpdate || (exists(resourceList) && Array.isArray(resourceList.resources) && resourceList.resources.length > 0)) && <Grid item xs={12} mb={2}>
                            <ResourceListHorizontal
                                title={'Resources'}
                                list={resourceList}
                                canUpdate={canUpdate}
                                handleUpdate={handleResourcesUpdate}
                                mutate={false}
                                zIndex={zIndex}
                            />
                        </Grid>
                    }
                    <Grid item xs={12} marginBottom={4}>
                        {
                            canUpdate ? <TagSelector
                                handleTagsUpdate={handleTagsUpdate}
                                tags={tags}
                            /> :
                                <TagList parentId={''} tags={subroutine?.routineVersion?.root?.tags ?? []} />
                        }
                    </Grid>
                </Grid>
                {canSubmit && <GridSubmitButtons
                    display="dialog"
                    errors={{
                        ...props.errors,
                        ...translationErrors,
                    }}
                    isCreate={isCreate}
                    loading={props.isSubmitting}
                    onCancel={onCancel}
                    onSetSubmitting={props.setSubmitting}
                    onSubmit={props.handleSubmit}
                />}
            </BaseForm>
        </>
    )
})