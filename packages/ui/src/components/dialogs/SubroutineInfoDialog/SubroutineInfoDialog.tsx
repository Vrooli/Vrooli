/**
 * Drawer to display a routine list item's info on the build page. 
 * Swipes up from bottom of screen
 */
import { useCallback, useEffect, useMemo, useState } from 'react';
import {
    Box,
    Grid,
    IconButton,
    SwipeableDrawer,
    TextField,
    Tooltip,
    Typography,
    useTheme,
} from '@mui/material';
import { SubroutineInfoDialogProps } from '../types';
import { defaultRelationships, defaultResourceList, getMinimumVersion, getUserLanguages, RoutineVersionInputShape, RoutineVersionOutputShape, TagShape, usePromptBeforeUnload, useTranslatedFields } from 'utils';
import { routineVersionTranslationValidation, routineVersionValidation } from '@shared/validation';
import { EditableTextCollapse, GridSubmitButtons, InputOutputContainer, LanguageInput, QuantityBox, RelationshipButtons, ResourceListHorizontal, SelectLanguageMenu, TagList, TagSelector, VersionDisplay, VersionInput } from 'components';
import { useFormik } from 'formik';
import { DUMMY_ID, uuid } from '@shared/uuid';
import { CloseIcon, OpenInNewIcon } from '@shared/icons';
import { RelationshipItemRoutineVersion, RelationshipsObject } from 'components/inputs/types';
import { getCurrentUser } from 'utils/authentication';
import { NodeRoutineListItem, ResourceList } from '@shared/consts';

export const SubroutineInfoDialog = ({
    data,
    defaultLanguage,
    handleUpdate,
    handleReorder,
    handleViewFull,
    isEditing,
    open,
    session,
    onClose,
    zIndex,
}: SubroutineInfoDialogProps) => {
    const { palette } = useTheme();
    const { id: userId } = useMemo(() => getCurrentUser(session), [session]);

    const subroutine = useMemo<NodeRoutineListItem | undefined>(() => {
        if (!data?.node || !data?.routineItemId) return undefined;
        return data.node.routineList.items.find(r => r.id === data.routineItemId);
    }, [data]);

    // Handle relationships
    const [relationships, setRelationships] = useState<RelationshipsObject>(defaultRelationships(isEditing, session));
    const onRelationshipsChange = useCallback((change: Partial<RelationshipsObject>) => setRelationships({ ...relationships, ...change }), [relationships]);

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
        setInputsList(subroutine?.routineVersion?.inputs ?? []);
        setOutputsList(subroutine?.routineVersion?.outputs ?? []);
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
                            <RelationshipButtons
                                isEditing={isEditing}
                                isFormDirty={formik.dirty}
                                objectType={'Routine'}
                                onRelationshipsChange={onRelationshipsChange}
                                relationships={relationships}
                                session={session}
                                zIndex={zIndex}
                            />
                        </Grid>
                        <Grid item xs={12}>
                            {canUpdate ? <LanguageInput
                                currentLanguage={language}
                                handleAdd={handleAddLanguage}
                                handleDelete={handleDeleteLanguage}
                                handleCurrent={setLanguage}
                                session={session}
                                translations={formik.values.translationsUpdate}
                                zIndex={zIndex}
                            /> : <SelectLanguageMenu
                                currentLanguage={language}
                                handleCurrent={setLanguage}
                                session={session}
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
                                    <Typography variant="h6">Order</Typography>
                                    <QuantityBox
                                        id="subroutine-position"
                                        disabled={!canUpdate}
                                        label="Order"
                                        min={1}
                                        max={data?.node?.routineList?.items?.length ?? 1}
                                        tooltip="The order of this subroutine in its parent routine"
                                        value={formik.values.index}
                                        handleChange={(value: number) => {
                                            formik.setFieldValue('index', value);
                                            handleReorder(data?.node?.id ?? '', subroutine?.index ?? 0, value - 1);
                                        }}
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
                                    versionInfo={formik.values.versionInfo}
                                    versions={subroutine?.routineVersion?.root?.versions ?? []}
                                    onBlur={formik.handleBlur}
                                    onChange={(newVersionInfo) => {
                                        formik.setFieldValue('versionInfo', newVersionInfo);
                                        setRelationships({
                                            ...relationships,
                                            isComplete: false,
                                        })
                                    }}
                                    error={formik.touched.versionInfo?.versionLabel && Boolean(formik.errors.versionInfo?.versionLabel)}
                                    helperText={formik.touched.versionInfo?.versionLabel ? formik.errors.versionInfo?.versionLabel : null}
                                />
                            </Grid>
                        }
                        {/* Inputs */}
                        {(canUpdate || (inputsList?.length > 0)) && <Grid item xs={12} sm={6}>
                            <InputOutputContainer
                                isEditing={canUpdate}
                                handleUpdate={handleInputsUpdate}
                                isInput={true}
                                language={language}
                                list={inputsList}
                                session={session}
                                zIndex={zIndex}
                            />
                        </Grid>}
                        {/* Outputs */}
                        {(canUpdate || (outputsList?.length > 0)) && <Grid item xs={12} sm={6}>
                            <InputOutputContainer
                                isEditing={canUpdate}
                                handleUpdate={handleOutputsUpdate}
                                isInput={false}
                                language={language}
                                list={outputsList}
                                session={session}
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
                                    session={session}
                                    mutate={false}
                                    zIndex={zIndex}
                                />
                            </Grid>
                        }
                        <Grid item xs={12} marginBottom={4}>
                            {
                                canUpdate ? <TagSelector
                                    handleTagsUpdate={handleTagsUpdate}
                                    session={session}
                                    tags={tags}
                                /> :
                                    <TagList session={session} parentId={''} tags={subroutine?.routineVersion?.root?.tags ?? []} />
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