import { Box, Button, Grid, IconButton, Slider, Stack, TextField, Tooltip, Typography } from '@mui/material';
import { HelpButton, NodeGraphContainer } from 'components';
import { useCallback, useEffect, useMemo, useState } from 'react';
import { routineQuery } from 'graphql/query';
import { useMutation, useQuery } from '@apollo/client';
import { routineUpdateMutation } from 'graphql/mutation';
import { mutationWrapper } from 'graphql/utils/wrappers';
import { routine } from 'graphql/generated/routine';
import { Pubs } from 'utils';
import {
    Close as CloseIcon,
    Done as DoneIcon,
    Edit as EditIcon,
    Info as InfoIcon,
    Restore as RestoreIcon,
    Update as UpdateIcon
} from '@mui/icons-material';
import { Routine } from 'types';
import isEqual from 'lodash/isEqual';
import { withStyles } from '@mui/styles';
import helpMarkdown from './OrchestratorHelp.md';
import { useRoute } from 'wouter';
import { APP_LINKS } from '@local/shared';

const TERTIARY_COLOR = '#95f3cd';
const STATUS_COLOR = {
    Valid: '#00d51e',
    Invalid: '#c72121',
}

const CustomSlider = withStyles({
    root: {
        color: '#dc697d',
        width: 'calc(100% - 32px)',
        left: '11px', // 16px - 1/4 of thumb diameter
    },
})(Slider);

export const RoutineOrchestratorPage = () => {
    // Get routine ID from URL
    const [, params] = useRoute(`${APP_LINKS.Orchestrate}/:id`);
    const id: string = useMemo(() => params?.id ?? '', [params]);
    // Queries routine data
    const { data: routineData } = useQuery<routine>(routineQuery, { variables: { input: { id } } });
    const [routine, setRoutine] = useState<Routine | null>(null);
    const [changedRoutine, setChangedRoutine] = useState<Routine | null>(null);
    useEffect(() => { setRoutine(routineData?.routine ?? null) }, [routineData]);
    // Routine mutator
    const [routineUpdate, { loading }] = useMutation<any>(routineUpdateMutation);
    // The routine's status (valid/invalid)
    const [isValid, setIsValid] = useState<boolean>(false);
    // Determines the size of the nodes and edges
    const [scale, setScale] = useState<number>(1);
    const [isEditing, setIsEditing] = useState<boolean>(false);
    // Used for editing the title of the routine
    const [titleActive, setTitleActive] = useState<boolean>(false);
    const toggleTitle = useCallback(() => setTitleActive(a => !a), []);
    const saveTitle = useCallback(() => {
        //TODO
        setTitleActive(false);
    }, []);
    const cancelTitle = useCallback(() => {
        //TODO
        setTitleActive(false);
    }, []);

    useEffect(() => {
        setChangedRoutine(routine);
    }, [routine]);

    const handleScaleChange = (_event: any, newScale: number | number[]) => {
        console.log('HANDLE SCALE CHANGE', newScale);
        setScale(newScale as number);
    };

    const startEditing = useCallback(() => setIsEditing(true), []);

    /**
     * Mutates routine data
     */
    const updateRoutine = useCallback(() => {
        if (!changedRoutine || isEqual(routine, changedRoutine)) {
            PubSub.publish(Pubs.Snack, { message: 'No changes detected', severity: 'error' });
            return;
        }
        if (!changedRoutine.id) {
            PubSub.publish(Pubs.Snack, { message: 'Cannot update: Invalid routine data', severity: 'error' });
            return;
        }
        mutationWrapper({
            mutation: routineUpdate,
            input: changedRoutine,
            successMessage: () => 'Routine updated.',
        })
    }, [changedRoutine, routine, routineUpdate])

    const revertChanges = useCallback(() => {
        setChangedRoutine(routine);
    }, [routine])

    // Parse markdown from .md file
    const [helpText, setHelpText] = useState<string>('');
    useEffect(() => {
        fetch(helpMarkdown).then((r) => r.text()).then((text) => { setHelpText(text) });
    }, []);

    /**
     * Displays metadata about the routine orchestration.
     * On the left is a status indicator, which lets you know if the routine is valid or not.
     * In the middle is the title of the routine. Once clicked, the information bar converts to 
     * a text input field, which allows you to edit the title of the routine.
     * To the right is a button to switch to the metadata view/edit component. You can view/edit the 
     * title, descriptions, instructions, inputs, outputs, tags, etc.
     */
    const informationBar = useMemo(() => {
        const title = titleActive ? (
            <Stack direction="row" spacing={1} alignItems="center">
                {/* Component for editing title */}
                <TextField
                    autoFocus
                    id="title"
                    name="title"
                    autoComplete="routine-title"
                    label="Title"
                    value={changedRoutine?.title ?? ''}
                    onChange={() => { }}
                    sx={{ cursor: 'pointer' }}
                />
                {/* Buttons for saving/cancelling edit */}
                <IconButton aria-label="confirm-title-change" onClick={saveTitle}>
                    <DoneIcon sx={{ fill: '#40dd43' }} />
                </IconButton>
                <IconButton aria-label="cancel-title-change" onClick={cancelTitle} color="secondary">
                    <CloseIcon sx={{ fill: '#ff2a2a' }} />
                </IconButton>
            </Stack>
        ) : (
            <Typography
                onClick={toggleTitle}
                component="h2"
                variant="h5"
                textAlign="center"
                sx={{ cursor: 'pointer' }}
            >{changedRoutine?.title ?? ''}</Typography>
        );
        return (
            <Stack
                direction="row"
                spacing={2}
                width="100%"
                justifyContent="space-between"
                sx={{
                    background: (t) => t.palette.primary.light,
                    color: (t) => t.palette.primary.contrastText,
                }}
            >
                {/* Status indicator */}
                <Tooltip title={isValid ? "Routine is valid" : "Routine is invalid"}>
                    <Box sx={{
                        borderRadius: 1,
                        border: `2px solid ${isValid ? STATUS_COLOR.Valid : STATUS_COLOR.Invalid}`,
                        color: isValid ? STATUS_COLOR.Valid : STATUS_COLOR.Invalid,
                        height: 'fit-content',
                        margin: 'auto 8px',
                        fontWeight: 'bold',
                    }}>{isValid ? 'Valid' : 'Invalid'}</Box>
                </Tooltip>
                {/* Routine title label/TextField and action buttons */}
                {title}
                <Stack direction="row" spacing={1}>
                    {/* Help button */}
                    <HelpButton markdown={helpText} sxRoot={{ margin: "auto" }} sx={{ color: TERTIARY_COLOR }} />
                    {/* Switch to routine metadata page */}
                    <IconButton aria-label="show-routine-metadata" onClick={() => { }} color="secondary">
                        <EditIcon sx={{ fill: TERTIARY_COLOR }} />
                    </IconButton>
                </Stack>
            </Stack>
        )
    }, [isValid, titleActive, toggleTitle]);

    let optionsBar = useMemo(() => (
        <Grid
            container
            p={2}
            sx={{
                background: (t) => t.palette.primary.light,
                display: 'flex',
                justifyContent: 'center',
                alignItems: 'center',
            }}
        >
            {
                isEditing ?
                    (
                        <>
                            <Grid item xs={12} sm={6} pt={0}>
                                <CustomSlider
                                    aria-label="graph-scale"
                                    min={0.25}
                                    max={1}
                                    defaultValue={1}
                                    step={0.01}
                                    value={scale}
                                    valueLabelDisplay="auto"
                                    onChange={handleScaleChange}
                                />
                            </Grid>
                            <Grid item xs={12} sm={3} sx={{ padding: 1 }}>
                                <Button
                                    fullWidth
                                    startIcon={<UpdateIcon />}
                                    onClick={updateRoutine}
                                    disabled={loading || !isEqual(routine, changedRoutine)}
                                >Update</Button>
                            </Grid>
                            <Grid item xs={12} sm={3}>
                                <Button
                                    fullWidth
                                    startIcon={<RestoreIcon />}
                                    onClick={revertChanges}
                                    disabled={loading || isEqual(routine, changedRoutine)}
                                >Revert</Button>
                            </Grid>
                        </>
                    ) :
                    (
                        <>
                            <Grid item xs={12} sm={9}>
                                <CustomSlider
                                    aria-label="graph-scale"
                                    min={0.01}
                                    max={1}
                                    defaultValue={1}
                                    step={0.01}
                                    value={scale}
                                    valueLabelDisplay="auto"
                                    onChange={handleScaleChange} />
                            </Grid>
                            <Grid item xs={12} sm={3}>
                                <Button
                                    fullWidth
                                    startIcon={<EditIcon />}
                                    onClick={startEditing}
                                    disabled={loading}
                                >Update</Button>
                            </Grid>
                        </>
                    )
            }
        </Grid >
    ), [changedRoutine, isEditing, loading, revertChanges, routine, scale, startEditing, updateRoutine])

    return (
        <Stack direction="column" spacing={0} sx={{
            minWidth: '100%',
            paddingTop: '10vh',
            height: { xs: 'calc(90vh - 56px)', md: '90vh' },
        }}>
            {/* Information bar */}
            {informationBar}
            <NodeGraphContainer
                scale={scale}
                isEditable={true}
                labelVisible={true}
                nodes={changedRoutine?.nodes ?? []}
                links={changedRoutine?.nodeLinks ?? []}
            />
            {optionsBar}
        </Stack>
    )
};