import { Box, Checkbox, Collapse, Container, FormControlLabel, Grid, IconButton, TextField, Tooltip, Typography, useTheme } from '@mui/material';
import { InputOutputListItemProps } from '../types';
import { inputCreate, InputType, outputCreate } from '@local/shared';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { containerShadow } from 'styles';
import {
    Delete as DeleteIcon,
    ExpandLess as ExpandLessIcon,
    ExpandMore as ExpandMoreIcon,
} from '@mui/icons-material';
import { getTranslation, jsonToString, standardToFieldData, updateArray } from 'utils';
import { useFormik } from 'formik';
import { ListStandard, NewObject, RoutineInput, RoutineOutput, Standard } from 'types';
import { BaseStandardInput, PreviewSwitch, Selector, StandardSelectSwitch } from 'components';
import { FieldData } from 'forms/types';
import { generateInputComponent } from 'forms/generators';

type InputTypeOption = { label: string, value: InputType }
/**
 * Supported input types
 */
export const InputTypeOptions: InputTypeOption[] = [
    {
        label: 'Text',
        value: InputType.TextField,
    },
    {
        label: 'JSON',
        value: InputType.JSON,
    },
    {
        label: 'Integer',
        value: InputType.QuantityBox
    },
    {
        label: 'Radio (Select One)',
        value: InputType.Radio,
    },
    {
        label: 'Checkbox (Select any)',
        value: InputType.Checkbox,
    },
    {
        label: 'Switch (On/Off)',
        value: InputType.Switch,
    },
    // {
    //     label: 'File Upload',
    //     value: InputType.Dropzone,
    // },
    {
        label: 'Markdown',
        value: InputType.Markdown
    },
]

export const InputOutputListItem = ({
    isEditing,
    index,
    isInput,
    isOpen,
    item,
    handleOpen,
    handleClose,
    handleDelete,
    handleUpdate,
    language,
    session,
    zIndex,
}: InputOutputListItemProps) => {
    console.log('inputoutputlistitem', item?.standard);
    const { palette } = useTheme();

    // Handle standard select switch
    const [standard, setStandard] = useState<ListStandard | null>(item?.standard ?? null);
    // Handle input type selector
    const [inputType, setInputType] = useState<InputTypeOption>(InputTypeOptions[1]);
    const handleInputTypeSelect = useCallback((event: any) => {
        setInputType(event.target.value)
    }, []);

    // Handle standard schema
    const [schema, setSchema] = useState<FieldData | null>(null);
    const handleSchemaUpdate = useCallback((schema: FieldData) => {
        // Ignore if standard is already set
        if (standard) return;
        setSchema(schema);
    }, [standard]);
    const [schemaKey] = useState(`input-output-schema-${Math.random().toString(36).substring(2, 15)}`);

    const onSwitchChange = useCallback((s: ListStandard | null) => {
        setSchema(null);
        setStandard(s);
    }, []);

    /**
     * Update object when standard is changed
     */
    useEffect(() => {
        // Check if standard has changed
        if (item?.standard?.id === standard?.id) return;
        // TODO handle tags for all handleupdate calls. move to utility function
        /**
         * const tagsAdd = tags.length > 0 ? {
                tagsCreate: tags.filter(t => !t.id).map(t => ({ tag: t.tag })),
                tagsConnect: tags.filter(t => t.id).map(t => (t.id)),
            } : {};
         */
        handleUpdate(index, {
            ...item,
            standard: standard || null,
        })
    }, [handleUpdate, index, item, standard]);
    /**
     * Update object when schema (custom standard) is changed
     * Schemas mock a standard, so they are wrapped in a new standard object
     */
    useEffect(() => {
        if (!schema) return;
        handleUpdate(index, {
            ...item,
            standard: {
                default: schema.props?.defaultValue ?? null,
                type: schema.type,
                props: JSON.stringify(schema.props),
                yup: JSON.stringify(schema.yup),
            } as Standard
        })
    }, [handleUpdate, index, item, schema]);

    type Translation = NewObject<(RoutineInput | RoutineOutput)['translations'][0]>;
    const getTranslationsUpdate = useCallback((language: string, translation: Translation) => {
        // Find translation
        const index = item.translations.findIndex(t => language === t.language);
        // Add to array, or update if found
        return index >= 0 ? updateArray(item.translations, index, translation) : [...item.translations, translation];
    }, [item.translations]);

    /**
     * Current schema, either generated from standard or custom 
     */
    const generatedSchema = useMemo<FieldData | null>(() => {
        if (schema) return schema;
        if (!standard) return null;
        return standardToFieldData({
            fieldName: schemaKey,
            description: getTranslation(item, 'description', [language]) ?? getTranslation(standard, 'description', [language]),
            props: standard?.props ?? '',
            name: standard?.name ?? '',
            type: standard?.type ?? '',
            yup: standard?.yup ?? null,
        });
    }, [schema, standard, schemaKey, item, language]);

    const formik = useFormik({
        initialValues: {
            description: getTranslation(item, 'description', [language]) ?? '',
            isRequired: true,
            name: item.name ?? '',
            // Value of generated input component preview
            [schemaKey]: jsonToString((generatedSchema?.props as any)?.format),
        },
        enableReinitialize: true,
        validationSchema: isInput ? inputCreate : outputCreate,
        onSubmit: (values) => {
            // Update translations
            const allTranslations = getTranslationsUpdate(language, {
                language,
                description: values.description,
            })
            handleUpdate(index, {
                ...item,
                name: values.name,
                isRequired: isInput ? values.isRequired : undefined,
                translations: allTranslations,
            } as any);
        },
    });

    const toggleOpen = useCallback(() => {
        if (isOpen) {
            formik.handleSubmit();
            handleClose(index);
        }
        else handleOpen(index);
    }, [isOpen, handleOpen, index, formik, handleClose]);

    const [isPreviewOn, setIsPreviewOn] = useState<boolean>(false);
    const onPreviewChange = useCallback((isOn: boolean) => { setIsPreviewOn(isOn); }, []);

    return (
        <Box
            id={`${isInput ? 'input' : 'output'}-item-${index}`}
            sx={{
                ...containerShadow,
                zIndex: 1,
                borderRadius: '8px',
                background: 'white',
                overflow: 'overlay',
                flexGrow: 1,
                marginBottom: 2,
            }}
        >
            {/* Top bar, with expand/collapse icon */}
            <Container
                onClick={toggleOpen}
                sx={{
                    background: isInput ? (palette.mode === 'light' ? '#79addf' : '#2668a7') : (palette.mode === 'light' ? '#c15c6d' : '#9e2d40'),
                    color: 'white',
                    display: 'flex',
                    alignItems: 'center',
                    justifyContent: 'center',
                    overflow: 'hidden',
                    height: '48px', // Lighthouse SEO requirement
                    padding: '0.1em',
                    textAlign: 'center',
                    cursor: 'pointer',
                    '&:hover': {
                        filter: `brightness(120%)`,
                        transition: 'filter 0.2s',
                    },
                }}
            >
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
                {isOpen ?
                    <ExpandLessIcon sx={{ marginLeft: 'auto' }} /> :
                    <ExpandMoreIcon sx={{ marginLeft: 'auto' }} />
                }
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
                            helperText={formik.touched.name && formik.errors.name}
                        /> : <Typography variant="h6">{`Name: ${formik.values.name}`}</Typography>}
                    </Grid>
                    <Grid item xs={12}>
                        <Grid item xs={12}>
                            {isEditing ? <TextField
                                fullWidth
                                id="description"
                                name="description"
                                label="description"
                                value={formik.values.description}
                                multiline
                                maxRows={3}
                                onBlur={(e) => { formik.handleBlur(e); formik.handleSubmit() }}
                                onChange={formik.handleChange}
                                error={formik.touched.description && Boolean(formik.errors.description)}
                                helperText={formik.touched.description && formik.errors.description}
                            /> : <Typography variant="body2">{`Description: ${formik.values.description}`}</Typography>}
                        </Grid>
                    </Grid>
                    {/* Select standard */}
                    <Grid item xs={12}>
                        <StandardSelectSwitch
                            disabled={!isEditing}
                            session={session}
                            selected={standard}
                            onChange={onSwitchChange}
                            zIndex={zIndex}
                        />
                    </Grid>
                    {/* Standard build/preview */}
                    <Grid item xs={12}>
                        <PreviewSwitch
                            isPreviewOn={isPreviewOn}
                            onChange={onPreviewChange}
                            sx={{
                                marginBottom: 2
                            }}
                        />
                        {
                            isPreviewOn ?
                                (Boolean(generatedSchema) && generateInputComponent({
                                    data: generatedSchema ?? schema as FieldData,
                                    disabled: true,
                                    formik,
                                    session,
                                    onUpload: () => { },
                                    zIndex,
                                })) :
                                // Only editable if standard not selected
                                <Box>
                                    <Selector
                                        disabled={Boolean(standard)}
                                        fullWidth
                                        options={InputTypeOptions}
                                        selected={standard?.type ? (InputTypeOptions.find(option => option.value === standard.type) ?? InputTypeOptions[0]) : inputType}
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
                                        inputType={(standard?.type as InputType) ?? inputType.value}
                                        isEditing={!Boolean(standard)}
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