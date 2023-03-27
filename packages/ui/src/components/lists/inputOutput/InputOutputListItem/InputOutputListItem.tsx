import { Box, Checkbox, Collapse, Container, FormControlLabel, Grid, IconButton, TextField, Tooltip, Typography, useTheme } from '@mui/material';
import { InputType, Session, StandardVersion } from '@shared/consts';
import { DeleteIcon, ExpandLessIcon, ExpandMoreIcon, ReorderIcon } from '@shared/icons';
import { uuid } from '@shared/uuid';
import { routineVersionInputValidation, routineVersionOutputValidation } from '@shared/validation';
import { GeneratedInputComponent } from 'components/inputs/generated';
import { MarkdownInput } from 'components/inputs/MarkdownInput/MarkdownInput';
import { PreviewSwitch } from 'components/inputs/PreviewSwitch/PreviewSwitch';
import { Selector } from 'components/inputs/Selector/Selector';
import { BaseStandardInput } from 'components/inputs/standards';
import { StandardVersionSelectSwitch } from 'components/inputs/StandardVersionSelectSwitch/StandardVersionSelectSwitch';
import { useFormik } from 'formik';
import { FieldData } from 'forms/types';
import Markdown from 'markdown-to-jsx';
import { useCallback, useContext, useEffect, useMemo, useState } from 'react';
import { linkColors } from 'styles';
import { getCurrentUser } from 'utils/authentication/session';
import { InputTypeOption, InputTypeOptions } from 'utils/consts';
import { getTranslation, getUserLanguages } from 'utils/display/translationTools';
import { SessionContext } from 'utils/SessionContext';
import { jsonToString, standardVersionToFieldData, updateArray } from 'utils/shape/general';
import { RoutineVersionInputShape, RoutineVersionInputTranslationShape } from 'utils/shape/models/routineVersionInput';
import { RoutineVersionOutputTranslationShape } from 'utils/shape/models/routineVersionOutput';
import { StandardVersionShape } from 'utils/shape/models/standardVersion';
import { InputOutputListItemProps } from '../types';

const defaultStandardVersion = (
    item: InputOutputListItemProps['item'],
    session: Session | undefined,
    generatedSchema?: FieldData | null,
): StandardVersionShape => ({
    id: uuid(),
    default: JSON.stringify(generatedSchema?.props?.defaultValue ?? null),
    isComplete: true,
    isPrivate: false, // TODO not sure if this should be true or false
    standardType: generatedSchema?.type ?? InputTypeOptions[0].value,
    props: JSON.stringify(generatedSchema?.props ?? '{}'),
    yup: JSON.stringify(generatedSchema?.yup ?? '{}'),
    root: {
        id: uuid(),
        owner: { __typename: 'User', id: getCurrentUser(session)!.id! },
        permissions: JSON.stringify({}),
        isInternal: true,
        isPrivate: false, // TODO not sure if this should be true or false
        tags: [],
    },
    translations: [{
        id: uuid(),
        language: getUserLanguages(session)[0],
        name: `${item.name}-schema`,
    }],
    versionLabel: '1.0.0',
})

const toFieldData = (schemaKey: string, item: InputOutputListItemProps['item'], language: string): FieldData | null => {
    if (!item.standardVersion || item.standardVersion.root.isInternal === false) return null;
    return standardVersionToFieldData({
        fieldName: schemaKey,
        description: getTranslation(item as RoutineVersionInputShape, [language]).description ?? getTranslation(item.standardVersion, [language]).description,
        helpText: getTranslation(item as RoutineVersionInputShape, [language], false).helpText ?? getTranslation(item.standardVersion, [language], false).helpText,
        props: item.standardVersion.props ?? '',
        name: item.name ?? getTranslation(item.standardVersion, [language]).name ?? '',
        standardType: item.standardVersion.standardType ?? InputTypeOptions[0].value,
        yup: item.standardVersion.yup ?? null,
    })
}

//TODO handle language change somehow
export const InputOutputListItem = ({
    isEditing,
    index,
    isInput,
    isOpen,
    item,
    handleOpen,
    handleClose,
    handleDelete,
    handleReorder,
    handleUpdate,
    language,
    zIndex,
}: InputOutputListItemProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [schemaKey] = useState(`input-output-schema-${Math.random().toString(36).substring(2, 15)}`);

    const [standardVersion, setStandardVersion] = useState<StandardVersionShape>(item.standardVersion ?? defaultStandardVersion(item, session));
    useEffect(() => {
        setStandardVersion(item.standardVersion ?? defaultStandardVersion(item, session));
    }, [item, session])

    const canUpdateStandardVersion = useMemo(() => isEditing && standardVersion.root.isInternal === true, [isEditing, standardVersion.root.isInternal]);

    /**
     * Schema only available when defining custom (internal) standard
     */
    const [generatedSchema, setGeneratedSchema] = useState<FieldData | null>(null);

    // Handle standard schema
    const handleSchemaUpdate = useCallback((schema: FieldData) => {
        if (!canUpdateStandardVersion) return;
        setGeneratedSchema(schema);
    }, [canUpdateStandardVersion]);

    const handleInputTypeSelect = useCallback((selected: InputTypeOption) => {
        if (selected.value === item.standardVersion?.standardType) return;
        const existingStandardVersion = item.standardVersion ?? defaultStandardVersion(item, session);
        setGeneratedSchema(toFieldData(schemaKey, {
            ...item,
            standardVersion: {
                ...existingStandardVersion,
                standardType: (selected ?? InputTypeOptions[0]).value,
            }
        }, language));
    }, [item, language, schemaKey, session]);

    type Translation = RoutineVersionInputTranslationShape | RoutineVersionOutputTranslationShape;
    const getTranslationsUpdate = useCallback((language: string, translation: Translation) => {
        // Find translation
        const index = item.translations?.findIndex(t => language === t.language) ?? -1;
        // Add to array, or update if found
        return index >= 0 ? updateArray(item.translations!, index, translation) : [...item.translations!, translation];
    }, [item.translations]);

    const formik = useFormik({
        initialValues: {
            id: item.id,
            description: getTranslation(item as RoutineVersionInputShape, [language]).description ?? '',
            helpText: getTranslation(item as RoutineVersionInputShape, [language]).helpText ?? '',
            isRequired: true,
            name: item.name ?? '' as string,
            // Value of generated input component preview
            [schemaKey]: jsonToString((generatedSchema?.props as any)?.format),
        },
        enableReinitialize: true,
        validationSchema: isInput ? routineVersionInputValidation.create({}) : routineVersionOutputValidation.create({}),
        onSubmit: (values) => {
            // Update translations
            const allTranslations = getTranslationsUpdate(language, {
                id: uuid(),
                language,
                description: values.description,
                helpText: values.helpText,
            })
            handleUpdate(index, {
                ...item,
                name: values.name,
                isRequired: isInput ? values.isRequired : undefined,
                translations: allTranslations as any,
                standardVersion: !canUpdateStandardVersion ? standardVersion : defaultStandardVersion(item, session, generatedSchema),
            } as RoutineVersionInputShape);
        },
    });

    const toggleOpen = useCallback(() => {
        if (isOpen) {
            formik.handleSubmit();
            handleClose(index);
        }
        else handleOpen(index);
    }, [isOpen, handleOpen, index, formik, handleClose]);

    const [isPreviewOn, setIsPreviewOn] = useState<boolean>(!canUpdateStandardVersion);
    const onPreviewChange = useCallback((isOn: boolean) => { setIsPreviewOn(isOn); }, []);
    const onSwitchChange = useCallback((s: StandardVersion | null) => {
        if (s && s.root.isInternal === false) {
            setStandardVersion(s as any)
            setIsPreviewOn(true);
        } else {
            setStandardVersion(defaultStandardVersion(item, session))
            setIsPreviewOn(false);
        }
    }, [item, session]);

    const openReorderDialog = useCallback((e: any) => {
        e.stopPropagation();
        handleReorder(index);
    }, [index, handleReorder]);

    return (
        <Box
            id={`${isInput ? 'input' : 'output'}-item-${index}`}
            sx={{
                zIndex: 1,
                background: 'white',
                overflow: 'hidden',
                flexGrow: 1,
            }}
        >
            {/* Top bar, with expand/collapse icon */}
            <Container
                onClick={toggleOpen}
                sx={{
                    background: palette.primary.main,
                    color: 'white',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'left',
                    overflow: 'hidden',
                    height: '48px', // Lighthouse SEO requirement
                    textAlign: 'center',
                    cursor: 'pointer',
                    paddingLeft: '8px !important',
                    paddingRight: '8px !important',
                    '&:hover': {
                        filter: `brightness(120%)`,
                        transition: 'filter 0.2s',
                    },
                }}
            >
                {/* Show order in list */}
                <Tooltip placement="top" title="Order">
                    <Typography variant="h6" sx={{
                        margin: '0',
                        marginRight: 1,
                        padding: '0',
                    }}>
                        {index + 1}.
                    </Typography>
                </Tooltip>
                {/* Show delete icon if editing */}
                {isEditing && (
                    <Tooltip placement="top" title={`Delete ${isInput ? 'input' : 'output'}. This will not delete the standard`}>
                        <IconButton color="inherit" onClick={() => handleDelete(index)} aria-label="delete" sx={{
                            height: 'fit-content',
                            marginTop: 'auto',
                            marginBottom: 'auto',
                        }}>
                            <DeleteIcon fill={'white'} />
                        </IconButton>
                    </Tooltip>
                )}
                {/* Show name and description if closed */}
                {!isOpen && (
                    <Box sx={{ display: 'flex', alignItems: 'center', overflow: 'hidden' }}>
                        <Typography variant="h6" sx={{
                            fontWeight: 'bold',
                            margin: '0',
                            padding: '0',
                            paddingRight: '0.5em',
                            overflow: 'hidden',
                            textOverflow: 'ellipsis',
                            whiteSpace: 'nowrap',
                        }}>
                            {formik.values.name}
                        </Typography>
                        <Typography variant="body2" sx={{
                            margin: '0',
                            padding: '0',
                            overflow: 'hidden',
                            textOverflow: 'ellipsis',
                            whiteSpace: 'nowrap',
                        }}>
                            {formik.values.description}
                        </Typography>
                    </Box>
                )}
                {/* Show reorder icon if editing */}
                {isEditing && <IconButton onClick={openReorderDialog} sx={{ marginLeft: 'auto' }}>
                    <ReorderIcon />
                </IconButton>}
                <IconButton sx={{ marginLeft: isEditing ? 'unset' : 'auto' }}>
                    {isOpen ?
                        <ExpandMoreIcon fill={palette.secondary.contrastText} /> :
                        <ExpandLessIcon fill={palette.secondary.contrastText} />
                    }
                </IconButton>
            </Container>
            <Collapse in={isOpen} sx={{
                background: palette.background.paper,
                color: palette.background.textPrimary,
            }}>
                <Grid container spacing={2} sx={{ padding: 1, ...linkColors(palette) }}>
                    <Grid item xs={12}>
                        {isEditing ? <TextField
                            fullWidth
                            id="name"
                            name="name"
                            label="identifier"
                            value={formik.values.name}
                            onBlur={(e) => { formik.handleBlur(e); formik.handleSubmit() }}
                            onChange={formik.handleChange}
                            error={formik.touched.name && Boolean(formik.errors.name)}
                            helperText={formik.touched.name ? (formik.errors.name as any) : null}
                        /> : <Typography variant="h6">{`Name: ${formik.values.name}`}</Typography>}
                    </Grid>
                    <Grid item xs={12}>
                        {isEditing ? <TextField
                            fullWidth
                            id="description"
                            name="description"
                            placeholder="Short description (optional)"
                            value={formik.values.description}
                            multiline
                            maxRows={3}
                            onBlur={(e) => { formik.handleBlur(e); formik.handleSubmit() }}
                            onChange={formik.handleChange}
                            error={formik.touched.description && Boolean(formik.errors.description)}
                            helperText={formik.touched.description && formik.errors.description}
                        /> : <Typography variant="body2">{`Description: ${formik.values.description}`}</Typography>}
                    </Grid>
                    <Grid item xs={12}>
                        {isEditing ? <MarkdownInput
                            id="helpText"
                            placeholder="Detailed information (optional)"
                            value={formik.values.helpText}
                            minRows={3}
                            onChange={(newText: string) => formik.setFieldValue('helpText', newText)}
                            error={formik.touched.helpText && Boolean(formik.errors.helpText)}
                            helperText={formik.touched.helpText ? formik.errors.helpText as string : null}
                        /> : formik.values.helpText.length > 0 ? <Markdown>{`Defailed information: ${formik.values.helpText}`}</Markdown> :
                            null}
                    </Grid>
                    {/* Select standard */}
                    <Grid item xs={12}>
                        <StandardVersionSelectSwitch
                            disabled={!isEditing}
                            selected={!canUpdateStandardVersion ? { root: { name: standardVersion.root.name ?? '' } } : null}
                            onChange={onSwitchChange}
                            zIndex={zIndex}
                        />
                    </Grid>
                    {/* Standard build/preview */}
                    <Grid item xs={12}>
                        {canUpdateStandardVersion && <PreviewSwitch
                            isPreviewOn={isPreviewOn}
                            onChange={onPreviewChange}
                            sx={{
                                marginBottom: 2
                            }}
                        />}
                        {
                            (isPreviewOn || !canUpdateStandardVersion) ?
                                ((standardVersion || generatedSchema) && <GeneratedInputComponent
                                    disabled={true}
                                    fieldData={standardVersion ?
                                        standardVersionToFieldData({
                                            fieldName: schemaKey,
                                            description: getTranslation(item as RoutineVersionInputShape, [language]).description ?? getTranslation(standardVersion, [language]).description,
                                            helpText: getTranslation(item as RoutineVersionInputShape, [language], false).helpText ?? getTranslation(standardVersion, [language], false).helpText,
                                            props: standardVersion?.props ?? '',
                                            name: item.name ?? getTranslation(standardVersion, [language]).name ?? '',
                                            standardType: standardVersion?.standardType ?? InputTypeOptions[0].value,
                                            yup: standardVersion.yup ?? null,
                                        }) as FieldData :
                                        generatedSchema as FieldData}
                                    onUpload={() => { }}
                                    zIndex={zIndex}
                                />) :
                                // Only editable if standard not selected and is editing
                                <Box>
                                    <Selector
                                        fullWidth
                                        options={InputTypeOptions}
                                        selected={InputTypeOptions.find(option => option.value === generatedSchema?.type) ?? InputTypeOptions[0]}
                                        handleChange={handleInputTypeSelect}
                                        getOptionLabel={(option: InputTypeOption) => option.label}
                                        inputAriaLabel='input-type-selector'
                                        label="Type"
                                        sx={{ marginBottom: 2 }}
                                    />
                                    <BaseStandardInput
                                        fieldName={schemaKey}
                                        inputType={(generatedSchema?.type as InputType) ?? InputTypeOptions[0].value}
                                        isEditing={isEditing}
                                        schema={generatedSchema}
                                        onChange={handleSchemaUpdate}
                                        storageKey={schemaKey}
                                    />
                                </Box>
                        }
                    </Grid>
                    {isInput && <Grid item xs={12}>
                        <Tooltip placement={'right'} title='Is this input mandatory?'>
                            <FormControlLabel
                                disabled={!isEditing}
                                label='Required'
                                control={
                                    <Checkbox
                                        id='routine-info-dialog-is-internal'
                                        size="small"
                                        name='isRequired'
                                        color='secondary'
                                        checked={formik.values.isRequired}
                                        onChange={formik.handleChange}
                                        onBlur={(e) => { formik.handleBlur(e); formik.handleSubmit() }}
                                    />
                                }
                            />
                        </Tooltip>
                    </Grid>}
                </Grid>
            </Collapse>
        </Box >
    )
}