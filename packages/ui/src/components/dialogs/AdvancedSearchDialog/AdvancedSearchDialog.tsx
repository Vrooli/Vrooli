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
    Typography,
    useTheme
} from '@mui/material';
import { useEffect, useMemo, useState } from 'react';
import { AdvancedSearchDialogProps } from '../types';
import {
    Cancel as CancelIcon,
    Close as CloseIcon,
    Search as SearchIcon,
} from '@mui/icons-material';
import { useFormik } from 'formik';
import { FieldData, FormSchema } from 'forms/types';
import { organizationSearchSchema, projectSearchSchema, routineSearchSchema, runSearchSchema, standardSearchSchema, userSearchSchema } from './schemas';
import { ObjectType } from 'utils';
import { generateDefaultProps, generateGrid, generateYupSchema } from 'forms/generators';
import { Tag } from 'types';

/**
 * Maps object types to their corresponding search schemas
 */
const objectToSchema: { [key in ObjectType]?: FormSchema } = {
    [ObjectType.Organization]: organizationSearchSchema,
    [ObjectType.Project]: projectSearchSchema,
    [ObjectType.Routine]: routineSearchSchema,
    [ObjectType.Run]: runSearchSchema,
    [ObjectType.Standard]: standardSearchSchema,
    [ObjectType.User]: userSearchSchema,
};

const getSchema = (objectType: ObjectType): FormSchema | undefined => {
    if (objectType in objectToSchema) return objectToSchema[objectType];
    return undefined;
}

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

const shapeFormikRun = (values: { [x: string]: any }) => ({
    status: values.status,
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
 * Maps object types to their corresponding search queries
 */
const shapeFormik = {
    [ObjectType.Organization]: shapeFormikOrganization,
    [ObjectType.Project]: shapeFormikProject,
    [ObjectType.Routine]: shapeFormikRoutine,
    [ObjectType.Run]: shapeFormikRun,
    [ObjectType.Standard]: shapeFormikStandard,
    [ObjectType.User]: shapeFormikUser,
}

const getFormik = (objectType: ObjectType): any => {
    if (objectType in shapeFormik) return shapeFormik[objectType];
    return undefined;
}

export const AdvancedSearchDialog = ({
    handleClose,
    handleSearch,
    isOpen,
    objectType,
    session,
    zIndex,
}: AdvancedSearchDialogProps) => {
    const theme = useTheme();
    // Search schema to use
    const [schema, setSchema] = useState<FormSchema | undefined>(getSchema(objectType));
    useEffect(() => { setSchema(getSchema(objectType)) }, [objectType]);

    // Get field inputs from schema, and add default values
    const fieldInputs = useMemo<FieldData[]>(() => schema?.fields ? generateDefaultProps(schema.fields) : [], [schema?.fields]);

    // Parse default values from fieldInputs, to use in formik
    const initialValues = useMemo(() => {
        let values: { [x: string]: any } = {};
        fieldInputs.forEach((field) => {
            values[field.fieldName] = field.props.defaultValue;
        });
        return values;
    }, [fieldInputs])

    // Generate yup validation schema
    const validationSchema = useMemo(() => schema ? generateYupSchema(schema) : undefined, [schema]);

    /**
     * Controls updates and validation of form
     */
    const formik = useFormik({
        initialValues,
        enableReinitialize: true,
        validationSchema,
        onSubmit: (values) => {
            // Shape values to match search query
            const formikShape = getFormik(objectType);
            if (formikShape) {
                const searchValue = formikShape(values);
                handleSearch(searchValue);
            }
            handleClose();
        },
    });
    const grid = useMemo(() => {
        if (!schema) return null;
        return generateGrid({
            childContainers: schema.containers,
            fields: schema.fields,
            formik,
            layout: schema.formLayout,
            onUpload: () => { },
            session,
            theme,
            zIndex,
        })
    }, [schema, formik, session, theme, zIndex])

    /**
     * Title bar with help button and close icon
     */
    const titleBar = useMemo(() => (
        <Box
            sx={{
                background: theme.palette.primary.dark,
                color: theme.palette.primary.contrastText,
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'space-between',
                padding: 2,
            }}
        >
            <Typography component="h2" variant="h5" textAlign="center" sx={{ marginLeft: 'auto', paddingLeft: 2, paddingRight: 2 }}>
                {'Advanced Search'}
            </Typography>
            <Box sx={{ marginLeft: 'auto' }}>
                <IconButton
                    edge="start"
                    onClick={(e) => { handleClose() }}
                >
                    <CloseIcon sx={{ fill: theme.palette.primary.contrastText }} />
                </IconButton>
            </Box>
        </Box>
    ), [handleClose, theme])

    return (
        <Dialog
            id="advanced-search-dialog"
            open={isOpen}
            onClose={handleClose}
            scroll="body"
            sx={{
                zIndex,
                '& .MuiDialogContent-root': {
                    background: theme.palette.background.default,
                    color: theme.palette.background.textPrimary,
                    minWidth: 'min(400px, 100%)',
                },
                '& .MuiPaper-root': {
                    margin: { xs: 0, sm: 2, md: 4 },
                    maxWidth: { xs: '100%!important', md: 'calc(100% - 64px)' },
                },
            }}
        >
            {titleBar}
            <form onSubmit={formik.handleSubmit}>
                <DialogContent sx={{
                    padding: { xs: 1, sm: 2 },
                }}>
                    {grid}
                </DialogContent>
                {/* Search/Cancel buttons */}
                <Grid container spacing={1} sx={{
                    background: theme.palette.primary.dark,
                    maxWidth: 'min(700px, 100%)',
                    margin: 0,
                }}>
                    <Grid item xs={6} p={1} sx={{ paddingTop: 0 }}>
                        <Button
                            fullWidth
                            startIcon={<SearchIcon />}
                            type="submit"
                        >Search</Button>
                    </Grid>
                    <Grid item xs={6} p={1} sx={{ paddingTop: 0 }}>
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