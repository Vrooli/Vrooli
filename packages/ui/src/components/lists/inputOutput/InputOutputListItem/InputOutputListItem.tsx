import { Box, Checkbox, Collapse, Container, FormControlLabel, Grid, IconButton, TextField, Tooltip, Typography, useTheme } from '@mui/material';
import { InputOutputListItemProps } from '../types';
import { inputCreate, InputType, outputCreate } from '@local/shared';
import { useCallback, useEffect, useState } from 'react';
import { containerShadow } from 'styles';
import {
    Delete as DeleteIcon,
    ExpandLess as ExpandLessIcon,
    ExpandMore as ExpandMoreIcon,
} from '@mui/icons-material';
import { getTranslation, updateArray } from 'utils';
import { useFormik } from 'formik';
import { ListStandard, NewObject, RoutineInput, RoutineOutput, Standard } from 'types';
import { BaseStandardInput, Selector, StandardSelectSwitch } from 'components';
import { FieldData } from 'forms/types';

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
}: InputOutputListItemProps) => {
    const { palette } = useTheme();
    console.log('iolistitem')

    // Handle standard select switch
    const [standard, setStandard] = useState<ListStandard | null>(null);
    const onSwitchChange = useCallback((s: ListStandard | null) => { console.log('on switch change'); setStandard(s) }, []);

    // Handle input type selector
    const [inputType, setInputType] = useState<InputTypeOption>(InputTypeOptions[1]);
    const handleInputTypeSelect = useCallback((event: any) => {
        setInputType(event.target.value)
    }, []);

    // Handle standard schema
    const [schema, setSchema] = useState<FieldData | null>(null);
    const handleSchemaUpdate = useCallback((schema: FieldData) => { setSchema(schema); }, []);
    const [schemaKey] = useState(`input-output-schema-${Math.random().toString(36).substring(2, 15)}`);

    useEffect(() => {
        // Check if standard has changed
        if (item?.standard?.id === standard?.id) return;
        console.log('updating standardddd....', item?.standard, standard)
        handleUpdate(index, {
            ...item,
            standard: standard || null,
        })
    }, [handleUpdate, index, item, standard]);
    // Custom schemas mean a new standard will be created. 
    // So we must wrap the schema in a new standard object
    useEffect(() => {
        if (!schema) return;
        console.log('updating schema....', item?.standard, schema)
        handleUpdate(index, {
            ...item,
            standard: {
                default: schema.props?.defaultValue ?? null,
                type: schema.type,
                props: JSON.stringify(schema.props),
                yup: JSON.stringify(schema.yup),
            } as Standard
        })
    })

    type Translation = NewObject<(RoutineInput | RoutineOutput)['translations'][0]>;
    const getTranslationsUpdate = useCallback((language: string, translation: Translation) => {
        // Find translation
        const index = item.translations.findIndex(t => language === t.language);
        // Add to array, or update if found
        return index >= 0 ? updateArray(item.translations, index, translation) : [...item.translations, translation];
    }, [item.translations]);

    const formik = useFormik({
        initialValues: {
            description: getTranslation(item, 'description', [language]) ?? '',
            isRequired: true,
            name: item.name ?? '',
        },
        enableReinitialize: true,
        validationSchema: isInput ? inputCreate : outputCreate,
        onSubmit: (values) => {
            console.log('formik handlesubmit😭')
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
        console.log('toggle open')
        if (isOpen) {
            formik.handleSubmit();
            handleClose(index);
        }
        else handleOpen(index);
    }, [isOpen, handleOpen, index, formik, handleClose]);

    return (
        <Box
            id={`${isInput ? 'input' : 'output'}-item-${index}`}
            onBlur={() => { formik.handleSubmit() }}
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
                        <TextField
                            fullWidth
                            id="name"
                            name="name"
                            label="identifier"
                            value={formik.values.name}
                            onBlur={formik.handleBlur}
                            onChange={formik.handleChange}
                            error={formik.touched.name && Boolean(formik.errors.name)}
                            helperText={formik.touched.name && formik.errors.name}
                        />
                    </Grid>
                    <Grid item xs={12}>
                        <Grid item xs={12}>
                            <TextField
                                fullWidth
                                id="description"
                                name="description"
                                label="description"
                                value={formik.values.description}
                                rows={3}
                                onBlur={formik.handleBlur}
                                onChange={formik.handleChange}
                                error={formik.touched.description && Boolean(formik.errors.description)}
                                helperText={formik.touched.description && formik.errors.description}
                            />
                        </Grid>
                    </Grid>
                    {/* Select standard */}
                    <Grid item xs={12}>
                        <StandardSelectSwitch session={session} selected={standard} onChange={onSwitchChange} />
                    </Grid>
                    {
                        !standard && (
                            <Grid item xs={12}>
                                <Selector
                                    fullWidth
                                    options={InputTypeOptions}
                                    selected={inputType}
                                    handleChange={handleInputTypeSelect}
                                    getOptionLabel={(option: InputTypeOption) => option.label}
                                    inputAriaLabel='input-type-selector'
                                    label="Type"
                                />
                            </Grid>
                        )
                    }
                    {
                        !standard && (
                            <Grid item xs={12}>
                                <BaseStandardInput
                                    key={schemaKey}
                                    inputType={inputType.value}
                                    isEditing={isEditing}
                                    schema={schema}
                                    onChange={handleSchemaUpdate}
                                />
                            </Grid>
                        )
                    }
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