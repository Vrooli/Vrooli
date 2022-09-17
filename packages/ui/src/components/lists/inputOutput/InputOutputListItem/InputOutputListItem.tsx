import { Box, Checkbox, Collapse, Container, FormControlLabel, Grid, IconButton, TextField, Tooltip, Typography, useTheme } from '@mui/material';
import { InputOutputListItemProps } from '../types';
import { InputType } from '@shared/consts';
import { inputCreate, outputCreate } from '@shared/validation';
import { useCallback, useEffect, useMemo, useState } from 'react';
import {
    Delete as DeleteIcon,
} from '@mui/icons-material';
import { getTranslation, InputTranslationShape, InputTypeOption, InputTypeOptions, jsonToString, OutputTranslationShape, StandardShape, standardToFieldData, updateArray } from 'utils';
import { useFormik } from 'formik';
import { Standard } from 'types';
import { BaseStandardInput, MarkdownInput, PreviewSwitch, Selector, StandardSelectSwitch } from 'components';
import { FieldData } from 'forms/types';
import { generateInputComponent } from 'forms/generators';
import { v4 as uuid } from 'uuid';
import Markdown from 'markdown-to-jsx';
import { ExpandLessIcon, ExpandMoreIcon, ReorderIcon } from '@shared/icons';

const defaultStandard = (item: InputOutputListItemProps['item'], generatedSchema?: FieldData | null): StandardShape => ({
    __typename: 'Standard',
    id: uuid(),
    default: JSON.stringify(generatedSchema?.props?.defaultValue ?? null),
    isInternal: true,
    type: generatedSchema?.type ?? InputTypeOptions[0].value,
    props: JSON.stringify(generatedSchema?.props ?? '{}'),
    yup: JSON.stringify(generatedSchema?.yup ?? '{}'),
    name: `${item.name}-schema`,
    tags: [],
    translations: [],
})

const toFieldData = (schemaKey: string, item: InputOutputListItemProps['item'], language: string): FieldData | null => {
    if (!item.standard || item.standard.isInternal === false) return null;
    return standardToFieldData({
        fieldName: schemaKey,
        description: getTranslation(item, 'description', [language]) ?? getTranslation(item.standard, 'description', [language]),
        helpText: getTranslation(item, 'helpText', [language], false),
        props: item.standard.props ?? '',
        name: item.standard.name ?? '',
        type: item.standard.type ?? InputTypeOptions[0].value,
        yup: item.standard.yup ?? null,
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
    session,
    zIndex,
}: InputOutputListItemProps) => {
    const { palette } = useTheme();
    const [schemaKey] = useState(`input-output-schema-${Math.random().toString(36).substring(2, 15)}`);

    const [standard, setStandard] = useState<StandardShape>(item.standard ?? defaultStandard(item));
    useEffect(() => {
        setStandard(item.standard ?? defaultStandard(item));
    }, [item])

    const canEditStandard = useMemo(() => standard.isInternal === true, [standard.isInternal]);

    /**
     * Schema only available when defining custom (internal) standard
     */
    const [generatedSchema, setGeneratedSchema] = useState<FieldData | null>(null);

    // Handle standard schema
    const handleSchemaUpdate = useCallback((schema: FieldData) => {
        if (!canEditStandard) return;
        setGeneratedSchema(schema);
    }, [canEditStandard]);

    const handleInputTypeSelect = useCallback((event: any) => {
        if (event.target.value !== item.standard?.type) {
            const newType = event.target.value?.value ?? InputTypeOptions[0].value;
            const existingStandard = item.standard ?? defaultStandard(item);
            setGeneratedSchema(toFieldData(schemaKey, {
                ...item,
                standard: {
                    ...existingStandard,
                    type: newType,
                }
            }, language));
        }
    }, [item, language, schemaKey]);

    type Translation = InputTranslationShape | OutputTranslationShape;
    const getTranslationsUpdate = useCallback((language: string, translation: Translation) => {
        // Find translation
        const index = item.translations.findIndex(t => language === t.language);
        // Add to array, or update if found
        return index >= 0 ? updateArray(item.translations, index, translation) : [...item.translations, translation];
    }, [item.translations]);

    const formik = useFormik({
        initialValues: {
            id: item.id,
            description: getTranslation(item, 'description', [language]) ?? '',
            helpText: getTranslation(item, 'helpText', [language]) ?? '',
            isRequired: true,
            name: item.name ?? '' as string,
            // Value of generated input component preview
            [schemaKey]: jsonToString((generatedSchema?.props as any)?.format),
        },
        enableReinitialize: true,
        validationSchema: isInput ? inputCreate : outputCreate,
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
                translations: allTranslations,
                standard: !canEditStandard ? standard : defaultStandard(item, generatedSchema),
            });
        },
    });

    const toggleOpen = useCallback(() => {
        if (isOpen) {
            formik.handleSubmit();
            handleClose(index);
        }
        else handleOpen(index);
    }, [isOpen, handleOpen, index, formik, handleClose]);

    const [isPreviewOn, setIsPreviewOn] = useState<boolean>(!canEditStandard);
    const onPreviewChange = useCallback((isOn: boolean) => { setIsPreviewOn(isOn); }, []);
    const onSwitchChange = useCallback((s: Standard | null) => {
        if (s && s.isInternal === false) {
            setStandard(s)
            setIsPreviewOn(true);
        } else {
            setStandard(defaultStandard(item))
            setIsPreviewOn(false);
        }
    }, [item]);

    const openReorderDialog = useCallback((e: any) => {
        e.stopPropagation();
        handleReorder(index);
    } , [index, handleReorder]);

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
                            <DeleteIcon sx={{
                                fill: 'white',
                                '&:hover': {
                                    fill: '#ff6a6a'
                                },
                                transition: 'fill 0.5s ease-in-out',
                            }} />
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
                <Grid container spacing={2} sx={{ padding: 1 }}>
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
                        <StandardSelectSwitch
                            disabled={!isEditing}
                            session={session}
                            selected={!canEditStandard ? { name: standard.name ?? '' } : null}
                            onChange={onSwitchChange}
                            zIndex={zIndex}
                        />
                    </Grid>
                    {/* Standard build/preview */}
                    <Grid item xs={12}>
                        {canEditStandard && <PreviewSwitch
                            isPreviewOn={isPreviewOn}
                            onChange={onPreviewChange}
                            sx={{
                                marginBottom: 2
                            }}
                        />}
                        {
                            (isPreviewOn || !canEditStandard) ?
                                ((standard || generatedSchema) && generateInputComponent({
                                    disabled: true,
                                    fieldData: standard ?
                                        standardToFieldData({
                                            fieldName: schemaKey,
                                            description: getTranslation(item, 'description', [language]) ?? getTranslation(standard, 'description', [language]),
                                            helpText: getTranslation(item, 'helpText', [language], false),
                                            props: standard?.props ?? '',
                                            name: standard?.name ?? '',
                                            type: standard?.type ?? InputTypeOptions[0].value,
                                            yup: standard.yup ?? null,
                                        }) as FieldData :
                                        generatedSchema as FieldData,
                                    formik,
                                    session,
                                    onUpload: () => { },
                                    zIndex,
                                })) :
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
                                        style={{
                                            marginBottom: 2
                                        }}
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