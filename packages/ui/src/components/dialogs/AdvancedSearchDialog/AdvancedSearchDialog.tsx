/**
 * Displays all search options for an organization
 */
import {
    Box,
    Button,
    Dialog,
    DialogContent,
    FormControl,
    FormControlLabel,
    FormLabel,
    Grid,
    IconButton,
    Radio,
    RadioGroup,
    Stack,
    Tooltip,
    Typography
} from '@mui/material';
import { useMemo } from 'react';
import { AdvancedSearchDialogProps } from '../types';
import {
    Cancel as CancelIcon,
    Close as CloseIcon,
    Search as SearchIcon,
} from '@mui/icons-material';
import { useFormik } from 'formik';
import { QuantityBox } from 'components/inputs';
import { FormSchema, InputType } from 'forms/types';
import { BaseForm } from 'forms';

// const isOpenToNewMembersQuery = input.isOpenToNewMembers ? { isOpenToNewMembers: true } : {};
// const languagesQuery = input.languages ? { translations: { some: { language: { in: input.languages } } } } : {};
// const minStarsQuery = input.minStars ? { stars: { gte: input.minStars } } : {};
// const projectIdQuery = input.projectId ? { projects: { some: { projectId: input.projectId } } } : {};
// const resourceListsQuery = input.resourceLists ? { resourceLists: { some: { translations: { some: { title: { in: input.resourceLists } } } } } } : {};
// const resourceTypesQuery = input.resourceTypes ? { resourceLists: { some: { usedFor: ResourceListUsedFor.Display as any, resources: { some: { usedFor: { in: input.resourceTypes } } } } } } : {};
// const routineIdQuery = input.routineId ? { routines: { some: { id: input.routineId } } } : {};
// const userIdQuery = input.userId ? { members: { some: { userId: input.userId, role: { in: [MemberRole.Admin, MemberRole.Owner] } } } } : {};
// const reportIdQuery = input.reportId ? { reports: { some: { id: input.reportId } } } : {};
// const standardIdQuery = input.standardId ? { standards: { some: { id: input.standardId } } } : {};
// const tagsQuery = input.tags ? { tags: { some: { tag: { tag: { in: input.tags } } } } } : {};

const organizationFormSchema: FormSchema = {
    formLayout: {
        title: "Search Organizations",
        direction: "column",
        rowSpacing: 5,
    },
    fields: [
        {
            fieldName: "isOpenToNewMembers",
            label: "Accepting new members?",
            type: InputType.Radio,
            props: {
                defaultValue: null,
                row: true,
                options: [
                    { label: "Yes", value: true },
                    { label: "No", value: false },
                    { label: "Don't Care", value: null },
                ]
            }
        },
        {
            fieldName: "minimumStars",
            label: "Minimum Stars",
            type: InputType.QuantityBox,
            props: {
                min: 10,
            }
        }
    ]
}

export const AdvancedSearchDialog = ({
    handleClose,
    handleSearch,
    isOpen,
    session,
}: AdvancedSearchDialogProps) => {

    const formik = useFormik({
        initialValues: {
            isOpenToNewMembers: null,
            minStars: 0,
        },
        onSubmit: (values) => {
            //TODO
        },
    });

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
                <BaseForm
                    onSubmit={formik.handleSubmit}
                    schema={organizationFormSchema}
                />
                {/* <Stack direction="column" spacing={4}>
                    // Accepting new members
                    <FormControl component="fieldset">
                        <FormLabel component="legend">Accepting New Members?</FormLabel>
                        <RadioGroup
                            aria-label="isOpenToNewMembers"
                            name="isOpenToNewMembers"
                            row={true}
                            value={formik.values.isOpenToNewMembers}
                            onBlur={formik.handleBlur}
                            onChange={formik.handleChange}
                        >
                            <FormControlLabel
                                value={true}
                                control={<Radio />}
                                label={"Yes"}
                            />
                            <FormControlLabel
                                value={false}
                                control={<Radio />}
                                label={"No"}
                            />
                            <FormControlLabel
                                value={null}
                                control={<Radio />}
                                label={"Don't Care"}
                            />
                        </RadioGroup>
                    </FormControl>
                    // Min Stars
                    <QuantityBox
                        id="minStars"
                        label="Minimum Stars"
                        min={0}
                        handleChange={(newValue: number) => { formik.setFieldValue('minStars', newValue) }}
                        value={formik.values.minStars}
                        tooltip="Minimum number of stars"
                    />
                </Stack> */}
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