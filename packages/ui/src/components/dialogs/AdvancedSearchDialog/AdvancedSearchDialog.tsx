/**
 * Displays all search options for an organization
 */
import {
    Box,
    Button,
    Dialog,
    DialogContent,
    Grid,
    IconButton,
    Typography
} from '@mui/material';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { AdvancedSearchDialogProps } from '../types';
import {
    Cancel as CancelIcon,
    Close as CloseIcon,
    Search as SearchIcon,
} from '@mui/icons-material';
import { useFormik } from 'formik';
import { FieldData, FormSchema, InputType } from 'forms/types';
import { BaseForm } from 'forms';
import { organizationSearchSchema, projectSearchSchema, routineSearchSchema, standardSearchSchema, userSearchSchema } from './schemas';
import { useReactPath } from 'utils';
import { APP_LINKS } from '@local/shared';
import { generateDefaultProps, generateGrid, generateYupSchema } from 'forms/generators';
import { Tag } from 'types';

/**
 * Maps routes to their corresponding search schemas
 */
const routeToSchema = {
    [APP_LINKS.SearchOrganizations]: organizationSearchSchema,
    [APP_LINKS.SearchProjects]: projectSearchSchema,
    [APP_LINKS.SearchRoutines]: routineSearchSchema,
    [APP_LINKS.SearchStandards]: standardSearchSchema,
    [APP_LINKS.SearchUsers]: userSearchSchema,
};

const yesNoDontCareToSearch = (value: 'yes' | 'no' | 'dontCare'): boolean | undefined => {
    switch (value) {
        case 'yes':
            return true;
        case 'no':
            return false;
        case 'dontCare':
            return undefined;
    }
};

const languagesToSearch = (languages: string[] | undefined): string[] | undefined => {
    if (Array.isArray(languages)) {
        if (languages.length === 0) return undefined;
        return languages;
    }
    return undefined;
};

const tagsToSearch = (tags: Tag[] | undefined): string[] | undefined => {
    if (Array.isArray(tags)) {
        if (tags.length === 0) return undefined;
        return tags.map(({ tag }) => tag);
    }
    return undefined;
};

const shapeFormikOrganization = (values: { [x: string]: any }) => ({
    isOpenToNewMembers: yesNoDontCareToSearch(values.isOpenToNewMembers),
    minStars: values.minStars,
    languages: languagesToSearch(values.languages),
    tags: tagsToSearch(values.tags),
})

const shapeFormikProject = (values: { [x: string]: any }) => ({
    isComplete: yesNoDontCareToSearch(values.isComplete),
    minScore: values.minScore,
    minStars: values.minStars,
    languages: languagesToSearch(values.languages),
    tags: tagsToSearch(values.tags),
})

const shapeFormikRoutine = (values: { [x: string]: any }) => ({
    isComplete: yesNoDontCareToSearch(values.isComplete),
    minScore: values.minScore,
    minStars: values.minStars,
    minComplexity: values.minComplexity,
    maxComplexity: values.maxComplexity === 0 ? undefined : values.maxComplexity,
    minSimplicity: values.minSimplicity,
    maxSimplicity: values.maxSimplicity === 0 ? undefined : values.maxSimplicity,
    languages: languagesToSearch(values.languages),
    tags: tagsToSearch(values.tags),
})

const shapeFormikStandard = (values: { [x: string]: any }) => ({
    minScore: values.minScore,
    minStars: values.minStars,
    languages: languagesToSearch(values.languages),
    tags: tagsToSearch(values.tags),
})

const shapeFormikUser = (values: { [x: string]: any }) => ({
    minStars: values.minStars,
    languages: languagesToSearch(values.languages),
})

/**
 * Shapes formik values to match the search query
 */
const shapeFormik = {
    [APP_LINKS.SearchOrganizations]: shapeFormikOrganization,
    [APP_LINKS.SearchProjects]: shapeFormikProject,
    [APP_LINKS.SearchRoutines]: shapeFormikRoutine,
    [APP_LINKS.SearchStandards]: shapeFormikStandard,
    [APP_LINKS.SearchUsers]: shapeFormikUser,
}

export const AdvancedSearchDialog = ({
    handleClose,
    handleSearch,
    isOpen,
    session,
}: AdvancedSearchDialogProps) => {
    // Search schema to use
    const [schema, setSchema] = useState<FormSchema | null>(null);

    // Use path to determine which form schema to use
    const path = useReactPath();
    useEffect(() => {
        if (path in routeToSchema) {
            setSchema(routeToSchema[path]);
        }
    }, [path]);

    // Get field inputs from schema, and add default values
    const fieldInputs = useMemo<FieldData[]>(() => schema?.fields ? generateDefaultProps(schema.fields) : [], [schema?.fields]);
    console.log('fieldInputs', fieldInputs);

    // Parse default values from fieldInputs, to use in formik
    const initialValues = useMemo(() => {
        let values: { [x: string]: any } = {};
        fieldInputs.forEach((field) => {
            values[field.fieldName] = field.props.defaultValue;
        });
        return values;
    }, [fieldInputs])
    console.log('initialValues', initialValues);

    // Generate yup validation schema
    const validationSchema = useMemo(() => schema ? generateYupSchema(schema) : undefined, [schema]);
    console.log('validationSchema', validationSchema);

    /**
     * Controls updates and validation of form
     */
    const formik = useFormik({
        initialValues,
        enableReinitialize: true,
        validationSchema,
        onSubmit: (values) => {
            console.log('in advanced search submit!!!!!!!!!!!😎', values);
            // Shape values to match search query
            const searchValues = shapeFormik[path](values);
            handleSearch(searchValues);
        },
    });
    console.log('formik values', formik.values)
    console.log('formik error', formik.errors);
    const grid = useMemo(() => {
        if (!schema) return null;
        return generateGrid(schema.formLayout, schema.containers, schema.fields, formik, () => { })
    }, [schema, formik])

    /**
     * Title bar with help button and close icon
     */
    const titleBar = useMemo(() => (
        <Box sx={{
            background: (t) => t.palette.primary.dark,
            color: (t) => t.palette.primary.contrastText,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'space-between',
            padding: 2,
        }}>
            <Typography component="h2" variant="h4" textAlign="center" sx={{ marginLeft: 'auto', paddingLeft: 2, paddingRight: 2 }}>
                {'Advanced Search'}
            </Typography>
            <Box sx={{ marginLeft: 'auto' }}>
                <IconButton
                    edge="start"
                    onClick={(e) => { handleClose() }}
                >
                    <CloseIcon sx={{ fill: (t) => t.palette.primary.contrastText }} />
                </IconButton>
            </Box>
        </Box>
    ), [])

    return (
        <Dialog
            open={isOpen}
            onClose={handleClose}
            sx={{
                '& .MuiDialogContent-root': { overflow: 'visible', background: '#cdd6df' },
                '& .MuiDialog-paper': { overflow: 'visible' }
            }}
        >
            {titleBar}
            <form onSubmit={formik.handleSubmit}>
                <DialogContent>
                    {grid}
                </DialogContent>
                {/* Search/Cancel buttons */}
                <Grid container spacing={1} sx={{
                    background: (t) => t.palette.primary.dark,
                    maxWidth: 'min(700px, 100%)',
                    margin: 0,
                }}>
                    <Grid item xs={12} sm={6} p={1} sx={{ paddingTop: 0 }}>
                        <Button
                            fullWidth
                            startIcon={<SearchIcon />}
                            type="submit"
                        >Search</Button>
                    </Grid>
                    <Grid item xs={12} sm={6} p={1} sx={{ paddingTop: 0 }}>
                        <Button
                            fullWidth
                            startIcon={<CancelIcon />}
                            onClick={handleClose}
                        >Cancel</Button>
                    </Grid>
                </Grid>
            </form>
        </Dialog>
    )
}