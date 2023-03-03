/**
 * Displays all search options for an organization
 */
import {
    Box,
    Button,
    Dialog,
    Grid,
    useTheme
} from '@mui/material';
import { useEffect, useMemo, useState } from 'react';
import { AdvancedSearchDialogProps } from '../types';
import { useFormik } from 'formik';
import { FieldData, FormSchema } from 'forms/types';
import { generateDefaultProps, generateYupSchema } from 'forms/generators';
import { convertFormikForSearch, convertSearchForFormik, searchTypeToParams } from 'utils';
import { DialogTitle, GeneratedGrid, GridActionButtons } from 'components';
import { CancelIcon, SearchIcon } from '@shared/icons';
import { useTranslation } from 'react-i18next';
import { parseSearchParams } from '@shared/route';

const titleAria = 'advanced-search-dialog-title';

export const AdvancedSearchDialog = ({
    handleClose,
    handleSearch,
    isOpen,
    searchType,
    session,
    zIndex,
}: AdvancedSearchDialogProps) => {
    const theme = useTheme();
    const { t } = useTranslation();

    // Search schema to use
    const [schema, setSchema] = useState<FormSchema | null>(null);
    useEffect(() => {
        async function getSchema() {
            setSchema(searchType in searchTypeToParams ? (await searchTypeToParams[searchType]()).advancedSearchSchema : null)
        }
        getSchema();
    }, [searchType]);

    // Parse default values to use in formik
    const initialValues = useMemo(() => {
        // Calculate initial values from schema, to use if values not already in URL
        const fieldInputs: FieldData[] = generateDefaultProps(schema?.fields ?? []);
        // Parse search params from URL, and filter out search fields that are not in schema
        const urlValues = schema ? convertSearchForFormik(parseSearchParams(), schema) : {} as { [key: string]: any };
        // Filter out search params that are not in schema
        let values: { [x: string]: any } = {};
        // Add fieldInputs to values
        fieldInputs.forEach((field) => {
            values[field.fieldName] = field.props.defaultValue;
        });
        // Add or replace urlValues to values
        Object.keys(urlValues).forEach((key) => {
            const currValue = urlValues[key];
            if (currValue !== undefined) values[key] = currValue;
        });
        return values;
    }, [schema])

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
            if (schema) {
                const searchValue = convertFormikForSearch(values, schema);
                handleSearch(searchValue);
            }
            handleClose();
        },
    });
    useEffect(() => {
        console.log('schema changed', schema);
    }, [schema])
    useEffect(() => {
        console.log('formik changed', formik);
    }, [formik])
    useEffect(() => {
        console.log('session changed', session);
    }, [session])

    return (
        <Dialog
            id="advanced-search-dialog"
            open={isOpen}
            onClose={handleClose}
            scroll="body"
            aria-labelledby={titleAria}
            sx={{
                zIndex,
                '& .MuiDialogContent-root': {
                    minWidth: `min(${theme.breakpoints.values.sm}px, 100%)`,
                },
                '& .MuiPaper-root': {
                    margin: { xs: 0, sm: 2, md: 4 },
                    maxWidth: { xs: '100%!important', sm: 'calc(100% - 64px)' },
                    minHeight: { xs: '100vh', sm: 'auto' },
                    display: { xs: 'block', sm: 'inline-block' },
                    background: theme.palette.background.default,
                    color: theme.palette.background.textPrimary,
                },
                // Remove ::after element that is added to the dialog
                '& .MuiDialog-container::after': {
                    content: 'none',
                },
            }}
        >
            <DialogTitle
                ariaLabel={titleAria}
                title={t(`AdvancedSearch`)}
                onClose={handleClose}
            />
            <form onSubmit={formik.handleSubmit}>
                <Box sx={{
                    padding: { xs: 1, sm: 2 },
                    margin: 'auto',
                    display: 'flex',
                    justifyContent: 'center',
                    alignItems: 'center',
                }}>
                    {schema && <GeneratedGrid
                        childContainers={schema.containers}
                        fields={schema.fields}
                        formik={formik}
                        layout={schema.formLayout}
                        onUpload={() => { }}
                        session={session}
                        theme={theme}
                        zIndex={zIndex}
                    />}
                </Box>
                <Box sx={{
                    padding: { xs: 1, sm: 2 },
                    margin: 'auto',
                    display: 'flex',
                    justifyContent: 'center',
                    alignItems: 'center',
                }}>
                    {schema && <GeneratedGrid
                        childContainers={schema.containers}
                        fields={schema.fields}
                        formik={formik}
                        layout={schema.formLayout}
                        onUpload={() => { }}
                        session={session}
                        theme={theme}
                        zIndex={zIndex}
                    />}
                </Box>
                <Box sx={{
                    padding: { xs: 1, sm: 2 },
                    margin: 'auto',
                    display: 'flex',
                    justifyContent: 'center',
                    alignItems: 'center',
                }}>
                    {schema && <GeneratedGrid
                        childContainers={schema.containers}
                        fields={schema.fields}
                        formik={formik}
                        layout={schema.formLayout}
                        onUpload={() => { }}
                        session={session}
                        theme={theme}
                        zIndex={zIndex}
                    />}
                </Box>
                {/* Search/Cancel buttons */}
                <GridActionButtons display="dialog">
                    <Grid item xs={6} p={1} sx={{ paddingTop: 0 }}>
                        <Button
                            fullWidth
                            startIcon={<SearchIcon />}
                            type="submit"
                        >{t(`Search`)}</Button>
                    </Grid>
                    <Grid item xs={6} p={1} sx={{ paddingTop: 0 }}>
                        <Button
                            fullWidth
                            startIcon={<CancelIcon />}
                            onClick={handleClose}
                        >{t(`Cancel`)}</Button>
                    </Grid>
                </GridActionButtons>
            </form>
        </Dialog>
    )
}