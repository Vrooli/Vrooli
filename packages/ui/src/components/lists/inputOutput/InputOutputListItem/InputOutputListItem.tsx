import { Box, Button, Checkbox, Collapse, Container, FormControlLabel, Grid, IconButton, Stack, TextField, Tooltip, Typography } from '@mui/material';
import { InputOutputListItemProps } from '../types';
import { inputCreate, outputCreate } from '@local/shared';
import { useCallback } from 'react';
import { containerShadow } from 'styles';
import {
    Close as CloseIcon,
    Delete as DeleteIcon,
    ExpandLess as ExpandLessIcon,
    ExpandMore as ExpandMoreIcon,
} from '@mui/icons-material';
import { getTranslation, getUserLanguages } from 'utils';
import { useFormik } from 'formik';

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
    handleOpenStandardSelect,
    handleRemoveStandard,
    language,
    session,
}: InputOutputListItemProps) => {

    type Translation = {
        language: string;
        description: string;
    };
    const getTranslationsUpdate = useCallback((language: string, translation: Translation) => {
        // Find translation
        const index = item.translations.findIndex(t => language === t.language);
        // If language exists, update
        if (index >= 0) {
            const newTranslations = [...item.translations];
            newTranslations[index] = { ...translation } as any;
            return newTranslations;
        }
        // Otherwise, add new
        else {
            return [...item.translations, translation];
        }
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
            // Update translations
            const allTranslations = getTranslationsUpdate(language, {
                language,
                description: values.description,
            })
            console.log('all translations', allTranslations);
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
    }, [isOpen, handleClose, handleOpen, index]);

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
                    background: isInput ? '#79addf' : '#c15c6d',
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
                {isOpen ?
                    <ExpandLessIcon sx={{ marginLeft: 'auto' }} /> :
                    <ExpandMoreIcon sx={{ marginLeft: 'auto' }} />
                }
            </Container>
            <Collapse in={isOpen}>
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
                        {/* If no standard selected, show "Select standard" button. Otherwise, show label and remove icon */}
                        {item.standard ? (
                            <Stack direction="row" spacing={1} alignItems="center">
                                <Typography variant="body2">Standard: </Typography>
                                <Typography variant="body2">{item.standard.name}</Typography>
                                <Tooltip placement="top" title={`Remove ${item.standard.name}`}>
                                    <IconButton color="inherit" onClick={() => handleRemoveStandard(index)} aria-label="delete" sx={{
                                        height: 'fit-content',
                                        marginTop: 'auto',
                                        marginBottom: 'auto',
                                    }}>
                                        <CloseIcon sx={{
                                            fill: '#ff6a6a',
                                        }} />
                                    </IconButton>
                                </Tooltip>
                            </Stack>
                        ) : (
                            <Button
                                color="secondary"
                                onClick={() => handleOpenStandardSelect(index)}
                            >
                                Select standard (optional)
                            </Button>
                        )
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
                                    />
                                }
                            />
                        </Tooltip>
                    </Grid>}
                </Grid>
            </Collapse>
        </Box>
    )
}