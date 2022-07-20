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
import { getTranslation, InputTranslationShape, jsonToString, OutputTranslationShape, StandardShape, standardToFieldData, updateArray } from 'utils';
import { useFormik } from 'formik';
import { Standard } from 'types';
import { BaseStandardInput, PreviewSwitch, Selector, StandardSelectSwitch } from 'components';
import { FieldData } from 'forms/types';
import { generateInputComponent } from 'forms/generators';
import { v4 as uuid } from 'uuid';

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

const defaultStandard = (item: InputOutputListItemProps['item']): StandardShape => ({
    __typename: 'Standard',
    id: uuid(),
    default: null,
    isInternal: true,
    type: InputTypeOptions[0].value,
    props: '{}',
    yup: '{}',
    name: `${item.name}-schema`,
    tags: [],
    translations: [],
})

const toFieldData = (schemaKey: string, item: InputOutputListItemProps['item'], language: string): FieldData | null => {
    if (!item.standard || item.standard.isInternal === false) return null;
    return standardToFieldData({
        fieldName: schemaKey,
        description: getTranslation(item, 'description', [language]) ?? getTranslation(item.standard, 'description', [language]),
        props: item.standard?.props ?? '',
        name: item.standard?.name ?? '',
        type: item.standard?.type ?? InputTypeOptions[0].value,
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
    handleUpdate,
    language,
    session,
    zIndex,
}: InputOutputListItemProps) => {
    console.log('rendering item')
    const { palette } = useTheme();
    const [schemaKey] = useState(`input-output-schema-${Math.random().toString(36).substring(2, 15)}`);

    const [externalStandard, setExternalStandard] = useState<StandardShape | null>(item.standard);
    // True if using existing standard
    const isExternal = useMemo<boolean>(() => externalStandard ? externalStandard.isInternal === false : false, [externalStandard]);

    /**
     * Schema only available when defining custom (internal) standard
     */
    const [generatedSchema, setGeneratedSchema] = useState<FieldData | null>(toFieldData(schemaKey, {
        ...item,
        standard: {
            ...(item.standard || defaultStandard(item)),
        }
    }, language));

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

    useEffect(() => {
        if (item.standard && item.standard.isInternal === false) {
            setExternalStandard(item.standard)
        } else {
            setGeneratedSchema(toFieldData(schemaKey, item, language))
        }
    }, [item, language, schemaKey, setExternalStandard]);

    // Handle standard schema
    const handleSchemaUpdate = useCallback((schema: FieldData) => {
        console.log('handleschemaupdate', schema)
        setGeneratedSchema(schema);
    }, []);

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
            isRequired: true,
            name: item.name ?? '' as string,
            // Value of generated input component preview
            [schemaKey]: jsonToString((generatedSchema?.props as any)?.format),
        },
        enableReinitialize: true,
        validationSchema: isInput ? inputCreate : outputCreate,
        onSubmit: (values) => {
            console.log('on formik submit', values)
            // Update translations
            const allTranslations = getTranslationsUpdate(language, {
                id: uuid(),
                language,
                description: values.description,
            })
            const standard: StandardShape = (isExternal && externalStandard) ? externalStandard : {
                __typename: 'Standard',
                id: uuid(),
                default: generatedSchema?.props?.defaultValue ?? null,
                isInternal: true,
                type: generatedSchema?.type ?? InputTypeOptions[0].value,
                props: JSON.stringify(generatedSchema?.props ?? '{}'),
                yup: JSON.stringify(generatedSchema?.yup ?? '{}'),
                name: `${item.name}-standard`,
                tags: [],
                translations: [],
            }
            handleUpdate(index, {
                ...item,
                name: values.name,
                isRequired: isInput ? values.isRequired : undefined,
                translations: allTranslations,
                standard,
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

    const [isPreviewOn, setIsPreviewOn] = useState<boolean>(isExternal);
    const onPreviewChange = useCallback((isOn: boolean) => { setIsPreviewOn(isOn); }, []);
    const onSwitchChange = useCallback((s: Standard | null) => {
        setIsPreviewOn(Boolean(s));
        if (s && s.isInternal === false) {
            setExternalStandard(s)
        } else {
            setGeneratedSchema(toFieldData(schemaKey, {
                ...item,
                standard: s,
            }, language))
        }
    }, [item, language, schemaKey]);

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
                            helperText={formik.touched.name ? (formik.errors.name as any) : null}
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
                            selected={isExternal ? { name: externalStandard?.name ?? '' } : null}
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
                                ((externalStandard || generatedSchema) && generateInputComponent({
                                    data: externalStandard ?
                                        standardToFieldData({
                                            fieldName: schemaKey,
                                            description: getTranslation(item, 'description', [language]) ?? getTranslation(externalStandard, 'description', [language]),
                                            props: externalStandard?.props ?? '',
                                            name: externalStandard?.name ?? '',
                                            type: externalStandard?.type ?? InputTypeOptions[0].value,
                                            yup: externalStandard.yup ?? null,
                                        }) as FieldData :
                                        generatedSchema as FieldData,
                                    disabled: true,
                                    formik,
                                    session,
                                    onUpload: () => { },
                                    zIndex,
                                })) :
                                // Only editable if standard not selected and is editing
                                <Box>
                                    <Selector
                                        disabled={!isEditing || isExternal}
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
                                        isEditing={isEditing && !isExternal}
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