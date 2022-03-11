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
    FormControlLabel,
    IconButton,
    Link,
    List,
    ListItem,
    ListItemIcon,
    ListItemText,
    Stack,
    SwipeableDrawer,
    Tooltip,
    Typography,
} from '@mui/material';
import { BuildInfoDialogProps } from '../types';
import Markdown from 'markdown-to-jsx';
import { MarkdownInput, ResourceListHorizontal } from 'components';
import { DeleteRoutineDialog } from '..';
import { formatForUpdate, getTranslation, Pubs } from 'utils';
import { APP_LINKS } from '@local/shared';
import { Resource, ResourceList, User } from 'types';
import { useLocation } from 'wouter';
import { routine } from "graphql/generated/routine";
import { useMutation } from '@apollo/client';
import { useFormik } from 'formik';
import { routineUpdate as validationSchema } from '@local/shared';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { routineUpdateMutation } from 'graphql/mutation';

enum ActionOption {
    Cancel = 'cancel',
    Delete = 'delete',
    Fork = 'fork',
    Stats = 'stats',
    Update = 'update',
}

export const BuildInfoDialog = ({
    handleUpdate,
    isEditing,
    language,
    routine,
    session,
    sxs,
}: BuildInfoDialogProps) => {
    const [, setLocation] = useLocation();
    // Open boolean for drawer
    const [open, setOpen] = useState(false);
    // Open boolean for delete routine confirmation
    const [deleteOpen, setDeleteOpen] = useState(false);
    const toggleOpen = () => setOpen(o => !o);
    const closeMenu = () => setOpen(false);

    /**
     * Name of user or organization that owns this routine
     */
    const ownedBy = useMemo<string | null>(() => {
        if (!routine?.owner) return null;
        return getTranslation(routine.owner, 'username', [language]) ?? getTranslation(routine.owner, 'name', [language]);
    }, [routine?.owner, language]);

    /**
     * Navigate to owner's profile
     */
    const toOwner = useCallback(() => {
        if (!routine?.owner) {
            PubSub.publish(Pubs.Snack, { message: 'Could not find owner.', severity: 'Error' });
            return;
        }
        // Check if user or organization
        if (routine.owner.hasOwnProperty('username')) {
            setLocation(`${APP_LINKS.User}/${(routine.owner as User).username}`);
        } else {
            setLocation(`${APP_LINKS.Organization}/${routine.owner.id}`);
        }
    }, [routine?.owner, setLocation]);

    /**
     * Determines which action buttons to display
     */
    const actions = useMemo(() => {
        // [value, label, icon]
        const results: [ActionOption, string, any][] = [];
        // If editing, show "Update", "Cancel", and "Delete" buttons
        if (isEditing) {
            results.push(
                [ActionOption.Update, 'Update', UpdateIcon],
                [ActionOption.Cancel, 'Cancel', CancelIcon],
                [ActionOption.Delete, 'Delete', DeleteIcon],
            )
        }
        // If not editing, show "Stats" and "Fork" buttons
        else {
            results.push(
                [ActionOption.Stats, 'Stats', StatsIcon],
                [ActionOption.Fork, 'Fork', ForkIcon],
            )
        }
        return results;
    }, [isEditing, routine]);

    // Handle update
    const [mutation] = useMutation<routine>(routineUpdateMutation);
    const formik = useFormik({
        initialValues: {
            description: getTranslation(routine, 'description', [language]) ?? '',
            instructions: getTranslation(routine, 'instructions', [language]) ?? '',
            title: getTranslation(routine, 'title', [language]) ?? '',
            version: routine?.version ?? '',
        },
        enableReinitialize: true, // Needed because existing data is obtained from async fetch
        validationSchema,
        onSubmit: (values) => {
            mutationWrapper({
                mutation,
                input: formatForUpdate(routine, {
                    translations: [{
                        language: 'en',
                        title: values.title,
                        description: values.description,
                        instructions: values.instructions,
                    }],
                    ...values,
                }),
                onSuccess: (response) => { handleUpdate(response.data.routineUpdate) },
                onError: (response) => {
                    PubSub.publish(Pubs.Snack, { message: 'Error occurred.', severity: 'error', data: { error: response } });
                }
            })
        },
    });

    const handleAction = useCallback((option: ActionOption) => {
        //TODO
    }, []);

    return (
        <>
            <IconButton edge="start" color="inherit" aria-label="menu" onClick={toggleOpen}>
                <InfoIcon sx={sxs?.icon} />
            </IconButton>
            <SwipeableDrawer
                anchor="right"
                open={open}
                onOpen={() => { }}
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
                    {Array.isArray(routine?.resourceLists) && (routine?.resourceLists as ResourceList[]).length > 0 ? <ResourceListHorizontal
                        list={routine?.resourceLists[0] as ResourceList}
                        canEdit={isEditing}
                        handleUpdate={() => { }}
                        session={session}
                    /> : null}
                    {/* Description */}
                    <Box sx={{
                        padding: 1,
                        border: `1px solid ${(t) => t.palette.primary.dark}`,
                        borderRadius: 1,
                    }}>
                        <Typography variant="h6">Description</Typography>
                        {
                            isEditing ? (
                                <MarkdownInput
                                    id="description"
                                    placeholder="Description"
                                    value={formik.values.description}
                                    minRows={2}
                                    onChange={(newText: string) => formik.setFieldValue('description', newText)}
                                    error={formik.touched.description && Boolean(formik.errors.description)}
                                    helperText={formik.touched.description ? formik.errors.description as string : null}
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
                                    minRows={2}
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
                                    name='isInternalCheckbox'
                                    value='isInternalCheckbox'
                                    color='secondary'
                                    checked={routine?.isInternal ?? false}
                                    onChange={() => { }}
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
            </SwipeableDrawer>
            {/* Delete routine confirmation dialog */}
            <DeleteRoutineDialog
                isOpen={deleteOpen}
                routineName={getTranslation(routine, 'title', [language]) ?? ''}
                handleClose={() => setDeleteOpen(false)}
                handleDelete={() => { }}
            />
        </>
    );
}