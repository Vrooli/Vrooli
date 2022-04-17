/**
 * Drawer to display overall routine info on the build page. 
 * Swipes left from right of screen
 */
import { useCallback, useMemo, useState } from 'react';
import {
    Cancel as CancelIcon,
    Close as CloseIcon,
    Delete as DeleteIcon,
    Info as InfoIcon,
    Update as UpdateIcon,
    QueryStats as StatsIcon,
    ForkRight as ForkIcon,
} from '@mui/icons-material';
import {
    Box,
    Checkbox,
    Drawer,
    FormControlLabel,
    IconButton,
    Link,
    List,
    ListItem,
    ListItemIcon,
    ListItemText,
    Stack,
    SwipeableDrawer,
    TextField,
    Tooltip,
    Typography,
} from '@mui/material';
import { BaseObjectAction, BuildInfoDialogProps } from '../types';
import Markdown from 'markdown-to-jsx';
import { MarkdownInput, ResourceListHorizontal } from 'components';
import { getOwnedByString, getTranslation, Pubs, toOwnedBy } from 'utils';
import { APP_LINKS } from '@local/shared';
import { ResourceList, User } from 'types';
import { useLocation } from 'wouter';
import { useFormik } from 'formik';
import { routineUpdateForm as validationSchema } from '@local/shared';

export const BuildInfoDialog = ({
    handleAction,
    handleUpdate,
    isEditing,
    language,
    routine,
    session,
    sxs,
}: BuildInfoDialogProps) => {
    const [, setLocation] = useLocation();

    const ownedBy = useMemo<string | null>(() => getOwnedByString(routine, [language]), [routine, language]);
    const toOwner = useCallback(() => { toOwnedBy(routine, setLocation) }, [routine, setLocation]);

    /**
     * Determines which action buttons to display
     */
    const actions = useMemo(() => {
        // [value, label, icon]
        const results: [BaseObjectAction, string, any][] = [];
        // If editing, show "Update", "Cancel", and "Delete" buttons
        if (isEditing) {
            results.push(
                [BaseObjectAction.Update, 'Update', UpdateIcon],
                [BaseObjectAction.UpdateCancel, 'Cancel', CancelIcon],
                [BaseObjectAction.Delete, 'Delete', DeleteIcon],
            )
        }
        // If not editing, show "Stats" and "Fork" buttons
        else {
            results.push(
                [BaseObjectAction.Stats, 'Stats', StatsIcon],
                [BaseObjectAction.Fork, 'Fork', ForkIcon],
            )
        }
        return results;
    }, [isEditing]);

    // Handle update
    const formik = useFormik({
        initialValues: {
            description: getTranslation(routine, 'description', [language]) ?? '',
            instructions: getTranslation(routine, 'instructions', [language]) ?? '',
            isInternal: routine?.isInternal ?? false,
            title: getTranslation(routine, 'title', [language]) ?? '',
            version: routine?.version ?? '',
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            handleUpdate({
                ...routine,
                isInternal: values.isInternal,
                version: values.version,
                translations: [{
                    language,
                    title: values.title,
                    description: values.description,
                    instructions: values.instructions,
                }],
            } as any);
        },
    });

    // Open boolean for drawer
    const [open, setOpen] = useState(false);
    const toggleOpen = () => setOpen(o => !o);
    const closeMenu = () => {
        // If not editing, just close 
        if (!isEditing) {
            setOpen(false);
            return;
        }
        // If editing, try to save changes
        if (formik.isValid) {
            formik.handleSubmit();
            setOpen(false);
        } else {
            PubSub.publish(Pubs.Snack, { message: 'Please fix errors before closing.', severity: 'Error' });
        }
    };

    const resourceList = useMemo(() => {
        if (!routine || 
            !Array.isArray(routine.resourceLists) ||
            routine.resourceLists.length < 1 ||
            routine.resourceLists[0].resources.length < 1) return null;
        return <ResourceListHorizontal
            title={'Resources'}
            list={(routine as any).resourceLists[0]}
            canEdit={false}
            handleUpdate={() => { }} // Intentionally blank
            session={session}
        />
    }, [routine, session]);

    return (
        <>
            <IconButton edge="start" color="inherit" aria-label="menu" onClick={toggleOpen}>
                <InfoIcon sx={sxs?.icon} />
            </IconButton>
            {/* TODO SwipeableDrawer breaking */}
            <Drawer
                anchor="right"
                open={open}
                // onOpen={() => { }} // Intentionally empty
                onClose={closeMenu}
                sx={{
                    '& .MuiDrawer-paper': {
                        background: (t) => t.palette.background.default,
                        borderLeft: `1px solid ${(t) => t.palette.text.primary}`,
                        maxWidth: { xs: '100%', sm: '75%', md: '50%', lg: '40%', xl: '30%' },
                    }
                }}
            >
                {/* Title bar */}
                <Box sx={{
                    display: 'flex',
                    flexDirection: 'row',
                    alignItems: 'center',
                    background: (t) => t.palette.primary.dark,
                    color: (t) => t.palette.primary.contrastText,
                    padding: 1,
                }}>
                    {/* Title, created by, and version  */}
                    <Stack direction="column" spacing={1} alignItems="center" sx={{ marginLeft: 'auto' }}>
                        <Typography variant="h5">{getTranslation(routine, 'title', [language])}</Typography>
                        <Stack direction="row" spacing={1}>
                            {ownedBy ? (
                                <Link onClick={toOwner}>
                                    <Typography variant="body1" sx={{ color: (t) => t.palette.primary.contrastText, cursor: 'pointer' }}>{ownedBy} - </Typography>
                                </Link>
                            ) : null}
                            <Typography variant="body1">{routine?.version}</Typography>
                        </Stack>
                    </Stack>
                    <IconButton onClick={closeMenu} sx={{
                        color: (t) => t.palette.primary.contrastText,
                        borderRadius: 0,
                        borderBottom: `1px solid ${(t) => t.palette.primary.dark}`,
                        justifyContent: 'end',
                        flexDirection: 'row-reverse',
                        marginLeft: 'auto',
                    }}>
                        <CloseIcon fontSize="large" />
                    </IconButton>
                </Box>
                {/* Main content */}
                {/* Stack that shows routine info, such as resources, description, inputs/outputs */}
                <Stack direction="column" spacing={2} padding={1}>
                    {/* Resources */}
                    {resourceList}
                    {/* Description */}
                    <Box sx={{
                        padding: 1,
                        border: `1px solid ${(t) => t.palette.primary.dark}`,
                        borderRadius: 1,
                    }}>
                        <Typography variant="h6">Description</Typography>
                        {
                            isEditing ? (
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
                            ) : (
                                <Markdown>{getTranslation(routine, 'description', [language]) ?? ''}</Markdown>
                            )
                        }
                    </Box>
                    {/* Instructions */}
                    <Box sx={{
                        padding: 1,
                        border: `1px solid ${(t) => t.palette.background.paper}`,
                        borderRadius: 1,
                    }}>
                        <Typography variant="h6">Instructions</Typography>
                        {
                            isEditing ? (
                                <MarkdownInput
                                    id="instructions"
                                    placeholder="Instructions"
                                    value={formik.values.instructions}
                                    minRows={3}
                                    onChange={(newText: string) => formik.setFieldValue('instructions', newText)}
                                    error={formik.touched.instructions && Boolean(formik.errors.instructions)}
                                    helperText={formik.touched.instructions ? formik.errors.instructions as string : null}
                                />
                            ) : (
                                <Markdown>{getTranslation(routine, 'instructions', [language]) ?? ''}</Markdown>
                            )
                        }
                    </Box>
                    {/* Inputs/Outputs TODO*/}
                    {/* Is internal checkbox */}
                    <Tooltip placement={'top'} title='Indicates if this routine is shown in search results'>
                        <FormControlLabel
                            disabled={!isEditing}
                            label='Internal'
                            control={
                                <Checkbox
                                    id='routine-info-dialog-is-internal'
                                    size="small"
                                    name='isInternal'
                                    color='secondary'
                                    checked={formik.values.isInternal}
                                    onChange={formik.handleChange}
                                />
                            }
                        />
                    </Tooltip>
                </Stack>
                {/* List of actions that can be taken, such as viewing stats, forking, and deleting */}
                <List sx={{ marginTop: 'auto' }}>
                    {actions.map(([value, label, Icon]) => (
                        <ListItem
                            key={value}
                            button
                            onClick={() => handleAction(value)}
                        >
                            <ListItemIcon>
                                <Icon />
                            </ListItemIcon>
                            <ListItemText primary={label} />
                        </ListItem>
                    ))}
                </List>
            </Drawer>
        </>
    );
}