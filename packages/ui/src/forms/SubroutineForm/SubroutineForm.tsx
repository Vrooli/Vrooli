import { Box, Grid, IconButton, Tooltip, Typography, useTheme } from "@mui/material";
import { ResourceList, Session } from "@shared/consts";
import { CloseIcon, OpenInNewIcon } from "@shared/icons";
import { exists } from "@shared/utils";
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
import { forwardRef, useCallback, useContext } from "react";
import { useTranslation } from "react-i18next";
import { getUserLanguages } from "utils/display/translationTools";
import { useTranslatedFields } from "utils/hooks/useTranslatedFields";
import { SessionContext } from "utils/SessionContext";
import { validateAndGetYupErrors } from "utils/shape/general";
import { NodeRoutineListShape } from "utils/shape/models/nodeRoutineList";
import { NodeRoutineListItemShape, shapeNodeRoutineListItem } from "utils/shape/models/nodeRoutineListItem";
import { RoutineVersionInputShape } from "utils/shape/models/routineVersionInput";
import { RoutineVersionOutputShape } from "utils/shape/models/routineVersionOutput";
import { TagShape } from "utils/shape/models/tag";

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

export const transformSubroutineValues = (values: NodeRoutineListItemShape, existing?: NodeRoutineListItemShape) => {
    return existing === undefined
        ? shapeNodeRoutineListItem.create(values)
        : shapeNodeRoutineListItem.update(existing, values)
}

export const validateSubroutineValues = async (values: NodeRoutineListItemShape, existing?: NodeRoutineListItemShape) => {
    const transformedValues = transformSubroutineValues(values, existing);
    const validationSchema = existing === undefined
        ? nodeRoutineListItemValidation.create({})
        : nodeRoutineListItemValidation.update({});
    const result = await validateAndGetYupErrors(validationSchema, transformedValues);
    return result;
}

export const SubroutineForm = forwardRef<any, SubroutineFormProps>(({
    canUpdate,
    dirty,
    handleViewFull,
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

    const [indexField] = useField<number>('index');
    const [isInternalField] = useField<boolean>('routineVersion.root.isInternal');
    const [listField] = useField<NodeRoutineListShape>('list');
    const [inputsField, , inputsHelpers] = useField<RoutineVersionInputShape[]>('routineVersion.inputs');
    const [outputsField, , outputsHelpers] = useField<RoutineVersionOutputShape[]>('routineVersion.outputs');
    const [resourceListField, , resourceListHelpers] = useField<ResourceList>('routineVersion.resourceList');
    const [tagsField] = useField<TagShape[]>('routineVersion.root.tags');
    const [versionlabelField] = useField<string>('routineVersion.versionLabel');
    const [versionsField] = useField<{ versionLabel: string }[]>('routineVersion.root.versions');

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
                <Typography variant="h6" ml={1} mr={1}>{`(${(indexField.value ?? 0) + 1} of ${(listField.value?.items?.length ?? 1)})`}</Typography>
                {/* Version */}
                <VersionDisplay
                    currentVersion={{ versionLabel: versionlabelField.value }}
                    prefix={" - "}
                    versions={versionsField.value ?? []}
                />
                {/* Button to open in full page */}
                {!isInternalField.value && (
                    <Tooltip title="Open in full page">
                        <IconButton onClick={toGraph}>
                            <OpenInNewIcon fill={palette.primary.contrastText} />
                        </IconButton>
                    </Tooltip>
                )}
                {/* Close button */}
                <IconButton onClick={onCancel} sx={{
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
                isLoading={false}
                ref={ref}
                style={{
                    display: 'block',
                    width: 'min(700px, 100vw - 16px)',
                    margin: 'auto',
                    paddingLeft: 'env(safe-area-inset-left)',
                    paddingRight: 'env(safe-area-inset-right)',
                    paddingBottom: 'calc(64px + env(safe-area-inset-bottom))',
                }}
            >
                <Grid container spacing={2}>
                    {/* owner, project, isPrivate, etc. */}
                    <Grid item xs={12}>
                        <RelationshipList
                            isEditing={isEditing}
                            isFormDirty={dirty}
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
                            languages={languages}
                            zIndex={zIndex}
                        /> : <SelectLanguageMenu
                            currentLanguage={language}
                            handleCurrent={setLanguage}
                            languages={languages}
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
                                    max={listField.value?.items?.length ?? 1}
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
                                language,
                                multiline: true,
                                maxRows: 3,
                            }}
                            title={t('Description')}
                        />
                    </Grid>
                    {/* Instructions */}
                    <Grid item xs={12} sm={6}>
                        <EditableTextCollapse
                            component='TranslatedMarkdown'
                            isEditing={isEditing}
                            name="instructions"
                            props={{
                                language,
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
                                versions={(versionsField.value ?? []).map(v => v.versionLabel)}
                            />
                        </Grid>
                    }
                    {/* Inputs */}
                    {(canUpdate || (inputsField.value?.length > 0)) && <Grid item xs={12} sm={6}>
                        <InputOutputContainer
                            isEditing={canUpdate}
                            handleUpdate={inputsHelpers.setValue as any}
                            isInput={true}
                            language={language}
                            list={inputsField.value}
                            zIndex={zIndex}
                        />
                    </Grid>}
                    {/* Outputs */}
                    {(canUpdate || (outputsField.value?.length > 0)) && <Grid item xs={12} sm={6}>
                        <InputOutputContainer
                            isEditing={canUpdate}
                            handleUpdate={outputsHelpers.setValue as any}
                            isInput={false}
                            language={language}
                            list={outputsField.value}
                            zIndex={zIndex}
                        />
                    </Grid>}
                    {
                        (canUpdate || (exists(resourceListField.value) && Array.isArray(resourceListField.value.resources) && resourceListField.value.resources.length > 0)) && <Grid item xs={12} mb={2}>
                            <ResourceListHorizontal
                                title={'Resources'}
                                list={resourceListField.value}
                                canUpdate={canUpdate}
                                handleUpdate={(newList) => { resourceListHelpers.setValue(newList) }}
                                mutate={false}
                                zIndex={zIndex}
                            />
                        </Grid>
                    }
                    <Grid item xs={12} marginBottom={4}>
                        {
                            canUpdate ? <TagSelector name='routineVersion.root.tags' /> :
                                <TagList parentId={''} tags={(tagsField.value ?? []) as any[]} />
                        }
                    </Grid>
                </Grid>
                {canUpdate && <GridSubmitButtons
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