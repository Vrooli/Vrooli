/**
 * Drawer to display a routine list item's info on the build page. 
 * Swipes up from bottom of screen
 */
import {
    Box,
    Grid,
    IconButton,
    SwipeableDrawer,
    TextField,
    Tooltip,
    Typography,
    useTheme
} from '@mui/material';
import { NodeRoutineListItem, ResourceList } from '@shared/consts';
import { CloseIcon, OpenInNewIcon } from '@shared/icons';
import { DUMMY_ID, uuid } from '@shared/uuid';
import { routineVersionTranslationValidation, routineVersionValidation } from '@shared/validation';
import { GridSubmitButtons } from 'components/buttons/GridSubmitButtons/GridSubmitButtons';
import { EditableTextCollapse } from 'components/containers/EditableTextCollapse/EditableTextCollapse';
import { IntegerInput } from 'components/inputs/IntegerInput/IntegerInput';
import { LanguageInput } from 'components/inputs/LanguageInput/LanguageInput';
import { TagSelector } from 'components/inputs/TagSelector/TagSelector';
import { VersionInput } from 'components/inputs/VersionInput/VersionInput';
import { InputOutputContainer } from 'components/lists/inputOutput';
import { RelationshipList } from 'components/lists/RelationshipList/RelationshipList';
import { ResourceListHorizontal } from 'components/lists/resource';
import { TagList } from 'components/lists/TagList/TagList';
import { RelationshipItemRoutineVersion } from 'components/lists/types';
import { VersionDisplay } from 'components/text/VersionDisplay/VersionDisplay';
import { useFormik } from 'formik';
import { useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { useTranslation } from 'react-i18next';
import { getCurrentUser } from 'utils/authentication/session';
import { defaultResourceList } from 'utils/defaults/resourceList';
import { getUserLanguages } from 'utils/display/translationTools';
import { usePromptBeforeUnload } from 'utils/hooks/usePromptBeforeUnload';
import { useTranslatedFields } from 'utils/hooks/useTranslatedFields';
import { SessionContext } from 'utils/SessionContext';
import { getMinimumVersion } from 'utils/shape/general';
import { RoutineVersionInputShape } from 'utils/shape/models/routineVersionInput';
import { RoutineVersionOutputShape } from 'utils/shape/models/routineVersionOutput';
import { TagShape } from 'utils/shape/models/tag';
import { SelectLanguageMenu } from '../SelectLanguageMenu/SelectLanguageMenu';
import { SubroutineInfoDialogProps } from '../types';

export const SubroutineInfoDialog = ({
    data,
    defaultLanguage,
    handleUpdate,
    handleReorder,
    handleViewFull,
    isEditing,
    open,
    onClose,
    zIndex,
}: SubroutineInfoDialogProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const { t } = useTranslation();

    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    const subroutine = useMemo<NodeRoutineListItem | undefined>(() => {
        if (!data?.node || !data?.routineItemId) return undefined;
        return data.node.routineList.items.find(r => r.id === data.routineItemId);
    }, [data]);

    // Handle inputs
    const [inputsList, setInputsList] = useState<RoutineVersionInputShape[]>([]);
    const handleInputsUpdate = useCallback((updatedList: RoutineVersionInputShape[]) => {
        setInputsList(updatedList);
    }, [setInputsList]);

    // Handle outputs
    const [outputsList, setOutputsList] = useState<RoutineVersionOutputShape[]>([]);
    const handleOutputsUpdate = useCallback((updatedList: RoutineVersionOutputShape[]) => {
        setOutputsList(updatedList);
    }, [setOutputsList]);

    // Handle resources
    const [resourceList, setResourceList] = useState<ResourceList>(defaultResourceList);
    const handleResourcesUpdate = useCallback((updatedList: ResourceList) => setResourceList(updatedList), [setResourceList]);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>([]);
    const handleTagsUpdate = useCallback((updatedList: TagShape[]) => { setTags(updatedList); }, [setTags]);

    useEffect(() => {
        setRelationships({
            isComplete: subroutine?.routineVersion?.isComplete ?? false,
            isPrivate: subroutine?.routineVersion?.isPrivate ?? false,
            owner: subroutine?.routineVersion?.root?.owner ?? null,
            parent: null,
            // parent: subroutine?.routineVersion?.parent ?? null, TODO
            project: null //TODO
        });
        setInputsList(subroutine?.routineVersion?.inputs ?? [] as RoutineVersionInputShape[]);
        setOutputsList(subroutine?.routineVersion?.outputs ?? [] as RoutineVersionOutputShape[]);
        setResourceList(subroutine?.routineVersion?.resourceList ?? { id: uuid() } as any);
        setTags(subroutine?.routineVersion?.root?.tags ?? []);
    }, [subroutine?.routineVersion]);

    // Handle update
    const formik = useFormik({
        initialValues: {
            index: (subroutine?.index ?? 0) + 1,
            isComplete: subroutine?.routineVersion?.isComplete ?? true,
            isInternal: subroutine?.routineVersion?.root?.isInternal ?? false,
            translationsUpdate: subroutine?.routineVersion?.translations ?? [{
                id: DUMMY_ID,
                language: getUserLanguages(session)[0],
                description: '',
                instructions: '',
                name: '',
            }],
            versionInfo: {
                versionIndex: subroutine?.routineVersion?.root?.versions?.length ?? 0,
                versionLabel: subroutine?.routineVersion?.versionLabel ?? '1.0.0',
                versionNotes: '',
            }
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema: routineVersionValidation.update({ minVersion: getMinimumVersion(subroutine?.routineVersion?.root?.versions ?? []) }),
        onSubmit: (values) => {
            if (!subroutine) return;
            handleUpdate({
                ...subroutine,
                index: Math.max(values.index - 1, 0), // Formik index starts at 1, for user convenience
                routine: {
                    ...subroutine.routineVersion,
                    isInternal: values.isInternal,
                    isComplete: relationships.isComplete,
                    isPrivate: relationships.isPrivate,
                    owner: relationships.owner,
                    parent: relationships.parent as RelationshipItemRoutineVersion | null,
                    project: relationships.project,
                    inputs: inputsList,
                    outputs: outputsList,
                    resourceList: resourceList,
                    tags,
                    translations: values.translationsUpdate,
                    ...values.versionInfo
                }
            } as any);
        },
    });
    usePromptBeforeUnload({ shouldPrompt: formik.dirty });

    // On index change, update the index of the subroutine
    useEffect(() => {
        if (formik.values.index !== subroutine?.index) {
            handleReorder(data?.node?.id ?? '', subroutine?.index ?? 0, formik.values.index - 1);
        }
    }, [data?.node?.id, formik.values.index, handleReorder, subroutine?.id, subroutine?.index]);

    const {
        handleAddLanguage,
        handleDeleteLanguage,
        language,
        languages,
        onTranslationBlur,
        onTranslationChange,
        setLanguage,
        translations,
    } = useTranslatedFields({
        defaultLanguage,
        fields: ['description', 'instructions', 'name'],
        formik,
        formikField: 'translationsUpdate',
        validationSchema: routineVersionTranslationValidation.update({}),
    });
    useEffect(() => {
        if (languages.length === 0 && formik.values.translationsUpdate.length > 0) {
            setLanguage(formik.values.translationsUpdate[0].language);
        }
    }, [formik, languages, setLanguage])

    const canUpdate = useMemo<boolean>(() => isEditing && (subroutine?.routineVersion?.root?.isInternal || subroutine?.routineVersion?.root?.owner?.id === userId || subroutine?.routineVersion?.you?.canUpdate === true), [isEditing, subroutine?.routineVersion?.root?.isInternal, subroutine?.routineVersion?.root?.owner?.id, subroutine?.routineVersion?.you?.canUpdate, userId]);

    /**
     * Navigate to the subroutine's build page
     */
    const toGraph = useCallback(() => {
        handleViewFull();
    }, [handleViewFull]);

    /**
     * Before closing, update subroutine if changes were made
     */
    const handleClose = useCallback(() => {
        if (canUpdate && formik.dirty) {
            formik.submitForm();
        } else {
            onClose();
        }
    }, [formik, canUpdate, onClose]);

    /**
     * Close without saving
     */
    const handleCancel = useCallback(() => {
        formik.resetForm();
        onClose();
    }, [formik, onClose]);

    return (
        <SwipeableDrawer
            anchor="bottom"
            variant='persistent'
            open={open}
            onOpen={() => { }}
            onClose={handleClose}
            sx={{
                zIndex,
                '& .MuiDrawer-paper': {
                    background: palette.background.default,
                    borderTop: 'none',
                }
            }}
        >
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
                <Typography variant="h5">{translations.name}</Typography>
                <Typography variant="h6" ml={1} mr={1}>{`(${(subroutine?.index ?? 0) + 1} of ${(data?.node?.routineList?.items?.length ?? 1)})`}</Typography>
                {/* Version */}
                <VersionDisplay
                    currentVersion={subroutine?.routineVersion}
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
            {/* Main content */}
            <Box sx={{
                padding: 2,
                overflowY: 'auto',
            }}>
                <form onSubmit={formik.handleSubmit}>
                    {/* Position, description and instructions */}
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
                                <TextField
                                    fullWidth
                                    id="name"
                                    name="name"
                                    label="Name"
                                    value={translations.name}
                                    onBlur={onTranslationBlur}
                                    onChange={onTranslationChange}
                                    error={translations.touchedName && Boolean(translations.errorName)}
                                    helperText={translations.touchedName && translations.errorName}
                                />
                            </Grid>
                        }
                        {/* Description */}
                        <Grid item xs={12} sm={6}>
                            <EditableTextCollapse
                                isEditing={canUpdate}
                                propsTextField={{
                                    fullWidth: true,
                                    id: "description",
                                    name: "description",
                                    InputLabelProps: { shrink: true },
                                    value: translations.description,
                                    multiline: true,
                                    maxRows: 3,
                                    onBlur: onTranslationBlur,
                                    onChange: onTranslationChange,
                                    error: translations.touchedDescription && Boolean(translations.errorDescription),
                                    helperText: translations.touchedDescription && translations.errorDescription,
                                }}
                                text={translations.description}
                                title="Description"
                            />
                        </Grid>
                        {/* Instructions */}
                        <Grid item xs={12} sm={6}>
                            <EditableTextCollapse
                                isEditing={isEditing}
                                propsMarkdownInput={{
                                    id: "instructions",
                                    placeholder: "Instructions",
                                    value: translations.instructions,
                                    minRows: 3,
                                    onChange: (newText: string) => onTranslationChange({ target: { name: 'instructions', value: newText } }),
                                    error: translations.touchedInstructions && Boolean(translations.errorInstructions),
                                    helperText: translations.touchedInstructions ? translations.errorInstructions as string : null,
                                }}
                                text={translations.instructions}
                                title="Instructions"
                            />
                        </Grid>
                        {
                            canUpdate && <Grid item xs={12}>
                                <VersionInput
                                    fullWidth
                                    id="version"
                                    name="version"
                                    version={formik.values.versionLabel}
                                    versions={subroutine?.routineVersion?.root?.versions ?? []}
                                    onBlur={formik.handleBlur}
                                    onChange={(newVersion) => {
                                        formik.setFieldValue('versionLabel', newVersion);
                                        setRelationships({
                                            ...relationships,
                                            isComplete: false,
                                        })
                                    }}
                                    error={formik.touched.versionLabel?.versionLabel && Boolean(formik.errors.versionLabel?.versionLabel)}
                                    helperText={formik.touched.versionLabel?.versionLabel ? formik.errors.versionLabel?.versionLabel : null}
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
                            (canUpdate || (resourceList?.resources?.length > 0)) && <Grid item xs={12} mb={2}>
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
                    {/* Save/Cancel buttons */}
                    {
                        canUpdate && <Grid container spacing={1}>
                            <GridSubmitButtons
                                display="dialog"
                                errors={translations.errorsWithTranslations}
                                isCreate={false}
                                loading={formik.isSubmitting}
                                onCancel={handleCancel}
                                onSetSubmitting={formik.setSubmitting}
                                onSubmit={formik.handleSubmit}
                            />
                        </Grid>
                    }
                </form>
            </Box>
        </SwipeableDrawer>
    );
}