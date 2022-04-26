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
import { FormSchema, InputType } from 'forms/types';
import { BaseForm } from 'forms';
import { organizationSearchSchema, projectSearchSchema, routineSearchSchema, standardSearchSchema, userSearchSchema } from './schemas';
import { useReactPath } from 'utils';
import { APP_LINKS } from '@local/shared';

// const organizationFormSchema: FormSchema = {
//     formLayout: {
//         title: "Search Organizations",
//         direction: "column",
//         rowSpacing: 5,
//     },
//     fields: [
//         {
//             fieldName: "isOpenToNewMembers",
//             label: "Accepting new members?",
//             type: InputType.Radio,
//             props: {
//                 defaultValue: null,
//                 row: true,
//                 options: [
//                     { label: "Yes", value: true },
//                     { label: "No", value: false },
//                     { label: "Don't Care", value: null },
//                 ]
//             }
//         },
//         {
//             fieldName: "minimumStars",
//             label: "Minimum Stars",
//             type: InputType.QuantityBox,
//             props: {
//                 min: 0,
//                 defaultValue: 5,
//             }
//         },
//         {
//             fieldName: "testCheckboxMulti",
//             label: "Test Checkbox Multi",
//             type: InputType.Checkbox,
//             props: {
//                 color: "secondary",
//                 options: [{ label: "Boop" }, { label: "Beep" }, { label: "Bop" }],
//             }
//         },
//         {
//             fieldName: "testDropzone",
//             label: "Test Dropzone",
//             type: InputType.Dropzone,
//             props: {
//                 maxFiles: 1,
//             }
//         },
//         // {
//         //     fieldName: "testJSONInput",
//         //     label: "Test JSON Input",
//         //     type: InputType.JSON,
//         //     props: {
//         //         description: "This is a JSON input",
//         //     }
//         // }
//     ]
// }

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
        switch (path) {
            case APP_LINKS.SearchOrganizations:
                setSchema(organizationSearchSchema);
                break;
            case APP_LINKS.SearchProjects:
                setSchema(projectSearchSchema);
                break;
            case APP_LINKS.SearchRoutines:
                setSchema(routineSearchSchema);
                break;
            case APP_LINKS.SearchStandards:
                setSchema(standardSearchSchema);
                break;
            case APP_LINKS.SearchUsers:
                setSchema(userSearchSchema);
                break;
            default:
                setSchema(null);
                break;
        }
    }, [path]);

    const handleSubmit = useCallback((values: any) => {
        console.log('AdvancedSearchDialog handleSubmit', values);
        //TODO
    }, []);

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
            <DialogContent>
                {schema && <BaseForm
                    onSubmit={handleSubmit}
                    schema={schema}
                />}
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
                        onClick={handleSearch}
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
        </Dialog>
    )
}