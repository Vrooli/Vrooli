import { Box, Button, Checkbox, Collapse, Container, FormControlLabel, Grid, IconButton, TextField, Tooltip, Typography } from '@mui/material';
import { InputOutputListItemProps } from '../types';
import { inputCreate, outputCreate } from '@local/shared';
import { useCallback, useState } from 'react';
import { Standard } from 'types';
import { containerShadow } from 'styles';
import {
    Delete as DeleteIcon,
    ExpandLess as ExpandLessIcon,
    ExpandMore as ExpandMoreIcon,
} from '@mui/icons-material';
import { getTranslation } from 'utils';
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
}: InputOutputListItemProps) => {
    const [standard, setStandard] = useState<Standard | null>(null);

    const formik = useFormik({
        initialValues: {
            description: getTranslation(item, 'description', ['en'], true) ?? '',
            isRequired: true,
            name: item.name ?? '',
        },
        validationSchema: isInput ? inputCreate : outputCreate,
        onSubmit: (values) => {
            handleUpdate(index, {
                ...values,
                standard,
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
                    <>
                        <Typography variant="h6" sx={{
                            fontWeight: 'bold',
                            margin: '0',
                            padding: '0',
                            paddingRight: '0.5em',
                        }}>
                            {formik.values.name}
                        </Typography>
                        <Typography variant="body2" sx={{
                            margin: '0',
                            padding: '0',
                        }}>
                            {formik.values.description}
                        </Typography>
                    </>
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
            <Collapse
                in={isOpen}
                sx={{ margin: isOpen ? 1 : 0 }}
            >
                <Grid container spacing={2}>
                    <Grid item xs={12}>
                        <TextField
                            fullWidth
                            id="name"
                            name="name"
                            label="name"
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
                    <Grid item xs={12}>
                        <Button
                            color="secondary"
                            sx={{

                            }}>
                            {standard ? standard.name : 'Select standard'}
                        </Button>
                    </Grid>
                    <Grid item xs={12}>
                        <Tooltip placement={'right'} title='Is this input mandatory?'>
                            <FormControlLabel
                                disabled={!isEditing}
                                label='Required'
                                control={
                                    <Checkbox
                                        id='routine-info-dialog-is-internal'
                                        size="small"
                                        name='isRequired'
                                        value='isRequired'
                                        color='secondary'
                                        checked={formik.values.isRequired}
                                        onChange={formik.handleChange}
                                    />
                                }
                            />
                        </Tooltip>
                    </Grid>
                </Grid>
            </Collapse>
        </Box>
    )
}