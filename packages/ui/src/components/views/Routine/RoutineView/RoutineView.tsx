import { Box, Button, CircularProgress, Dialog, Grid, IconButton, LinearProgress, Stack, Tooltip, Typography, useTheme } from "@mui/material"
import { useLocation } from '@shared/route';
import { APP_LINKS, VoteFor } from "@shared/consts";
import { useMutation, useLazyQuery } from "@apollo/client";
import { routine, routineVariables } from "graphql/generated/routine";
import { routineQuery } from "graphql/query";
import { MouseEvent, useCallback, useEffect, useMemo, useState } from "react";
import {
    AccountTree as GraphIcon,
    DoneAll as MarkAsCompleteIcon,
    PlayCircle as StartIcon,
} from "@mui/icons-material";
import { ObjectActionMenu, BuildView, LinkButton, ReportsLink, ResourceListHorizontal, RunPickerMenu, RunView, SelectLanguageMenu, StarButton, StatusButton, UpTransition, UpvoteDownvote } from "components";
import { RoutineViewProps } from "../types";
import { formikToRunInputs, getLanguageSubtag, getLastUrlPart, getOwnedByString, getPreferredLanguage, getRoutineStatus, getTranslation, getUserLanguages, initializeRoutine, ObjectType, parseSearchParams, PubSub, runInputsCreate, setSearchParams, standardToFieldData, Status, toOwnedBy, useReactSearch } from "utils";
import { Routine, Run } from "types";
import { runCompleteMutation } from "graphql/mutation";
import { mutationWrapper } from "graphql/utils/mutationWrapper";
import { CommentFor, NodeType, StarFor } from "graphql/generated/globalTypes";
import { ObjectAction, ObjectActionComplete } from "components/dialogs/types";
import { containerShadow } from "styles";
import { validate as uuidValidate } from 'uuid';
import { runComplete, runCompleteVariables } from "graphql/generated/runComplete";
import { useFormik } from "formik";
import { FieldData } from "forms/types";
import { generateInputWithLabel } from "forms/generators";
import { CommentContainer, ContentCollapse, TextCollapse } from "components/containers";
import { EditIcon, EllipsisIcon } from "@shared/icons";

const statsHelpText =
    `Statistics are calculated to measure various aspects of a routine. 
    
**Complexity** is a rough measure of the maximum amount of effort it takes to complete a routine. This takes into account the number of inputs, the structure of its subroutine graph, and the complexity of every subroutine.

**Simplicity** is calculated similarly to complexity, but takes the shortest path through the subroutine graph.

There will be many more statistics in the near future.`

export const RoutineView = ({
    partialData,
    session,
    zIndex,
}: RoutineViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    // Fetch data
    const id = useMemo(() => getLastUrlPart(), []);
    const [getData, { data, loading }] = useLazyQuery<routine, routineVariables>(routineQuery, { errorPolicy: 'all' });
    const [routine, setRoutine] = useState<Routine | null>(null);
    useEffect(() => {
        if (uuidValidate(id)) { getData({ variables: { input: { id } } }); }
    }, [getData, id])
    useEffect(() => {
        if (!data) return;
        setRoutine(data.routine);
    }, [data]);
    const updateRoutine = useCallback((newRoutine: Routine) => { setRoutine(newRoutine); }, [setRoutine]);

    const search = useReactSearch(null);
    const { runId } = useMemo(() => ({
        runId: typeof search.run === 'string' && uuidValidate(search.run) ? search.run : null,
    }), [search]);

    const availableLanguages = useMemo<string[]>(() => (routine?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [routine?.translations]);
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { canStar, canVote, title, description, instructions, status, statusMessages } = useMemo(() => {
        const permissions = routine?.permissionsRoutine;
        const { messages: statusMessages, status } = getRoutineStatus(routine ?? partialData);
        return {
            canStar: permissions?.canStar === true,
            canVote: permissions?.canVote === true,
            title: getTranslation(routine, 'title', [language]) ?? getTranslation(partialData, 'title', [language]),
            description: getTranslation(routine, 'description', [language]) ?? getTranslation(partialData, 'description', [language]),
            instructions: getTranslation(routine, 'instructions', [language]) ?? getTranslation(partialData, 'instructions', [language]),
            status,
            statusMessages,
        };
    }, [routine, language, partialData]);

    useEffect(() => {
        document.title = `${title} | Vrooli`;
    }, [title]);

    const ownedBy = useMemo<string | null>(() => getOwnedByString(routine, [language]), [routine, language]);
    const toOwner = useCallback(() => { toOwnedBy(routine, setLocation) }, [routine, setLocation]);

    const [isBuildOpen, setIsBuildOpen] = useState<boolean>(Boolean(parseSearchParams(window.location.search)?.build));
    /**
     * If routine ID is not valid, create default routine data
     */
    useEffect(() => {
        if (!id || !uuidValidate(id)) {
            setRoutine(initializeRoutine(language))
        }
    }, [id, language]);
    const viewGraph = useCallback(() => {
        setSearchParams(setLocation, {
            build: true,
        })
        setIsBuildOpen(true);
    }, [setLocation]);
    const stopBuild = useCallback(() => {
        setIsBuildOpen(false)
    }, []);


    const [isRunOpen, setIsRunOpen] = useState(false)
    const [selectRunAnchor, setSelectRunAnchor] = useState<any>(null);
    const handleRunSelect = useCallback((run: Run | null) => {
        // If run is null, it means the routine will be opened without a run
        if (!run) {
            setSearchParams(setLocation, {
                run: "test",
                step: [1],
            })
        }
        // Otherwise, open routine where last left off in run
        else {
            setSearchParams(setLocation, {
                run: run.id,
                step: run.steps?.length > 0 ? run.steps[run.steps.length - 1].step : undefined,
            })
        }
        setIsRunOpen(true);
    }, [setLocation]);
    const handleSelectRunClose = useCallback(() => setSelectRunAnchor(null), []);

    const handleRunDelete = useCallback((run: Run) => {
        if (!routine) return;
        setRoutine({
            ...routine,
            runs: routine.runs.filter(r => r.id !== run.id),
        });
    }, [routine]);

    const handleRunAdd = useCallback((run: Run) => {
        if (!routine) return;
        setRoutine({
            ...routine,
            runs: [run, ...routine.runs],
        });
    }, [routine]);

    const runRoutine = useCallback((e: any) => {
        // Validate routine before trying to run
        if (!routine || !uuidValidate(routine.id)) {
            PubSub.get().publishSnack({ message: 'Error loading routine.', severity: 'error' });
            return;
        }
        // Find first node
        const firstNode = routine?.nodes?.find(node => node.type === NodeType.Start);
        if (!firstNode) {
            PubSub.get().publishSnack({ message: 'Routine invalid - cannot run.', severity: 'Error' });
            return;
        }
        // If run specified use that
        if (runId) {
            handleRunSelect({ id: runId } as Run);
        }
        // Otherwise, open dialog to select runs
        else {
            setSelectRunAnchor(e.currentTarget);
        }
    }, [handleRunSelect, routine, runId]);
    const stopRoutine = () => { setIsRunOpen(false) };

    const onEdit = useCallback(() => {
        // Depends on if we're in a search popup or a normal routine page
        const isMultiStep = (Array.isArray(routine?.nodes) && (routine?.nodes as Routine['nodes']).length > 1) ||
            (Array.isArray(routine?.nodeLinks) && (routine?.nodeLinks as Routine['nodeLinks']).length > 0);
        // If multi step, navigate to build page
        if (isMultiStep) {
            setSearchParams(setLocation, {
                build: true,
                edit: true,
            });
            setIsBuildOpen(true);
        }
        // Otherwise, edit as single step
        else {
            setLocation(`${APP_LINKS.Routine}/edit/${id}`);
        }
    }, [routine?.nodes, routine?.nodeLinks, setLocation, id]);

    // More menu
    const [moreMenuAnchor, setMoreMenuAnchor] = useState<any>(null);
    const openMoreMenu = useCallback((ev: MouseEvent<any>) => {
        setMoreMenuAnchor(ev.currentTarget);
        ev.preventDefault();
    }, []);
    const closeMoreMenu = useCallback(() => setMoreMenuAnchor(null), []);

    const onMoreActionStart = useCallback((action: ObjectAction) => {
        switch (action) {
            case ObjectAction.Edit:
                onEdit();
                break;
            case ObjectAction.Stats:
                //TODO
                break;
        }
    }, [onEdit]);

    const onMoreActionComplete = useCallback((action: ObjectActionComplete, data: any) => {
        switch (action) {
            case ObjectActionComplete.VoteDown:
            case ObjectActionComplete.VoteUp:
                if (data.vote.success) {
                    setRoutine({
                        ...routine,
                        isUpvoted: action === ObjectActionComplete.VoteUp,
                    } as any)
                }
                break;
            case ObjectActionComplete.Star:
            case ObjectActionComplete.StarUndo:
                if (data.star.success) {
                    setRoutine({
                        ...routine,
                        isStarred: action === ObjectActionComplete.Star,
                    } as any)
                }
                break;
            case ObjectActionComplete.Fork:
                setLocation(`${APP_LINKS.Routine}/${data.fork.routine.id}`);
                break;
            case ObjectActionComplete.Copy:
                setLocation(`${APP_LINKS.Routine}/${data.copy.routine.id}`);
                break;
        }
    }, [routine, setLocation]);

    // The schema and formik keys for the form
    const formValueMap = useMemo<{ [fieldName: string]: FieldData } | null>(() => {
        if (!routine) return null;
        const schemas: { [fieldName: string]: FieldData } = {};
        for (let i = 0; i < routine.inputs?.length; i++) {
            const currInput = routine.inputs[i];
            if (!currInput.standard) continue;
            const currSchema = standardToFieldData({
                description: getTranslation(currInput, 'description', getUserLanguages(session), false) ?? getTranslation(currInput.standard, 'description', getUserLanguages(session), false),
                fieldName: `inputs-${currInput.id}`,
                helpText: getTranslation(currInput, 'helpText', getUserLanguages(session), false),
                props: currInput.standard.props,
                name: currInput.name ?? currInput.standard?.name,
                type: currInput.standard.type,
                yup: currInput.standard.yup,
            });
            if (currSchema) {
                schemas[currSchema.fieldName] = currSchema;
            }
        }
        return schemas;
    }, [routine, session]);
    const formik = useFormik({
        initialValues: Object.entries(formValueMap ?? {}).reduce((acc, [key, value]) => {
            acc[key] = value.props.defaultValue ?? '';
            return acc;
        }, {}),
        enableReinitialize: true,
        onSubmit: () => { },
    });

    const [runComplete] = useMutation<runComplete, runCompleteVariables>(runCompleteMutation);
    const markAsComplete = useCallback(() => {
        if (!routine) return;
        mutationWrapper({
            mutation: runComplete,
            input: {
                id: routine.id,
                exists: false,
                title: title ?? 'Unnamed Routine',
                version: routine?.version ?? '',
                ...runInputsCreate(formikToRunInputs(formik.values)),
            },
            successMessage: () => 'Routine completed!🎉',
            onSuccess: () => {
                PubSub.get().publishCelebration();
                setLocation(APP_LINKS.Home)
            },
        })
    }, [formik.values, routine, runComplete, setLocation, title]);

    /**
     * If routine has nodes (i.e. is not just this page), display "View Graph" and "Start" (or "Continue") buttons. 
     * Otherwise, display "Mark as Complete" button.
     */
    const actions = useMemo(() => {
        // If routine has no nodes
        if (!routine?.nodes?.length) {
            // Only show if logged in
            if (!session?.id) return null;
            return (
                <Grid container spacing={1}>
                    <Grid item xs={12}>
                        <Button startIcon={<MarkAsCompleteIcon />} fullWidth onClick={markAsComplete} color="secondary">Mark as Complete</Button>
                    </Grid>
                </Grid>
            )
        }
        // If routine has nodes
        return (
            <Grid container spacing={1}>
                <Grid item xs={12} sm={6}>
                    <Button startIcon={<GraphIcon />} fullWidth onClick={viewGraph} color="secondary">View Graph</Button>
                </Grid>
                {/* Show continue if routine already has progress TODO */}
                {status === Status.Valid && <Grid item xs={12} sm={6}>
                    {routine && routine.runs?.length > 0 ?
                        <Button startIcon={<StartIcon />} fullWidth onClick={runRoutine} color="secondary">Continue</Button> :
                        <Button startIcon={<StartIcon />} fullWidth onClick={runRoutine} color="secondary">Start Now</Button>
                    }
                </Grid>}
            </Grid>
        )
    }, [routine, viewGraph, status, runRoutine, session?.id, markAsComplete]);

    /**
     * Copy current value of input to clipboard
     * @param fieldName Name of input
     */
    const copyInput = useCallback((fieldName: string) => {
        const input = formik.values[fieldName];
        if (input) {
            navigator.clipboard.writeText(input);
            PubSub.get().publishSnack({ message: 'Copied to clipboard.', severity: 'success' });
        } else {
            PubSub.get().publishSnack({ message: 'Input is empty.', severity: 'error' });
        }
    }, [formik]);

    const resourceList = useMemo(() => {
        if (!routine ||
            !Array.isArray(routine.resourceLists) ||
            routine.resourceLists.length < 1 ||
            routine.resourceLists[0].resources.length < 1) return null;
        return <ResourceListHorizontal
            title={'Resources'}
            list={routine.resourceLists[0]}
            canEdit={false}
            handleUpdate={() => { }} // Intentionally blank
            loading={loading}
            session={session}
            zIndex={zIndex}
        />
    }, [loading, routine, session, zIndex]);

    /**
     * Display body or loading indicator
     */
    const body = useMemo(() => {
        if (loading) return (
            <Box sx={{
                minHeight: 'min(300px, 25vh)',
                display: 'flex',
                justifyContent: 'center',
                alignItems: 'center',
            }}>
                <CircularProgress color="secondary" />
            </Box>
        )
        return (
            <>
                {/* Stack that shows routine info, such as resources, description, inputs/outputs */}
                <Stack direction="column" spacing={1} padding={1}>
                    {/* Resources */}
                    {resourceList}
                    {/* Description */}
                    <TextCollapse title="Description" text={description} />
                    {/* Instructions */}
                    <TextCollapse title="Instructions" text={instructions} />
                    {/* Auto-generated inputs */}
                    {
                        Object.keys(formik.values).length > 0 && <Box>
                            {
                                Object.values(formValueMap ?? {}).map((fieldData: FieldData, index: number) => (
                                    generateInputWithLabel({
                                        copyInput,
                                        disabled: false,
                                        fieldData,
                                        formik: formik,
                                        index,
                                        session,
                                        textPrimary: palette.background.textPrimary,
                                        onUpload: () => { },
                                        zIndex,
                                    })
                                ))
                            }
                        </Box>
                    }
                    {/* Stats */}
                    {
                        Array.isArray(routine?.nodes) && (routine as any).nodes.length > 0 &&
                        <ContentCollapse title="Stats" helpText={statsHelpText}>
                            <Typography variant="body1">Complexity: {routine?.complexity}</Typography>
                            <Typography variant="body1">Simplicity: {routine?.simplicity}</Typography>
                        </ContentCollapse>
                    }
                </Stack>
                {/* Actions */}
                {actions}
            </>
        )
    }, [loading, resourceList, description, palette.background.textPrimary, instructions, formik, formValueMap, routine, actions, session, zIndex, copyInput]);

    return (
        <>
            {/* Chooses which run to use */}
            <RunPickerMenu
                anchorEl={selectRunAnchor}
                handleClose={handleSelectRunClose}
                onAdd={handleRunAdd}
                onDelete={handleRunDelete}
                onSelect={handleRunSelect}
                routine={routine}
                session={session}
            />
            {/* Dialog for running routine */}
            <Dialog
                id="run-routine-view-dialog"
                fullScreen
                open={isRunOpen}
                onClose={stopRoutine}
                TransitionComponent={UpTransition}
                sx={{
                    zIndex: zIndex + 1,
                }}
            >
                {routine && <RunView
                    handleClose={stopRoutine}
                    routine={routine}
                    session={session}
                    zIndex={zIndex + 1}
                />}
            </Dialog>
            {/* Dialog for building routine */}
            <Dialog
                id="run-routine-view-dialog"
                fullScreen
                open={isBuildOpen}
                onClose={stopBuild}
                TransitionComponent={UpTransition}
                sx={{
                    zIndex: zIndex + 1,
                }}
            >
                <BuildView
                    handleClose={stopBuild}
                    loading={loading}
                    onChange={updateRoutine}
                    routine={routine}
                    session={session}
                    zIndex={zIndex + 1}
                />
            </Dialog>
            {/* Popup menu displayed when "More" ellipsis pressed */}
            <ObjectActionMenu
                isUpvoted={routine?.isUpvoted}
                isStarred={routine?.isStarred}
                objectId={id ?? ''}
                objectName={title ?? ''}
                objectType={ObjectType.Routine}
                anchorEl={moreMenuAnchor}
                title='Routine Options'
                onActionStart={onMoreActionStart}
                onActionComplete={onMoreActionComplete}
                onClose={closeMoreMenu}
                permissions={routine?.permissionsRoutine}
                session={session}
                zIndex={zIndex + 1}
            />
            <Stack direction="column" sx={{
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                margin: 'auto',
                // xs: 100vh - navbar (64px) - bottom nav (56px) - iOS nav bar
                // md: 100vh - navbar (80px)
                minHeight: { xs: 'calc(100vh - 64px - 56px - env(safe-area-inset-bottom))', md: 'calc(100vh - 80px)' },
                marginBottom: 8,
            }}>
                <Box sx={{
                    ...containerShadow,
                    background: palette.background.paper,
                    width: 'min(100%, 700px)',
                    borderRadius: 1,
                    overflow: 'overlay',
                }}>
                    {/* Main container */}
                    <Box sx={{
                        overflowY: 'auto',
                        overflow: 'overlay',
                    }}>
                        {/* Heading container */}
                        <Stack direction="column" spacing={1} sx={{
                            display: 'flex',
                            alignItems: 'center',
                            justifyContent: 'center',
                            padding: 2,
                            marginBottom: 1,
                            background: palette.primary.main,
                            color: palette.primary.contrastText,
                        }}>
                            {/* Title */}
                            <Stack direction="row" textAlign="center">
                                {loading ?
                                    <LinearProgress color="inherit" sx={{
                                        borderRadius: 1,
                                        width: '50vw',
                                        height: 8,
                                        marginTop: '12px !important',
                                        marginBottom: '12px !important',
                                        maxWidth: '300px',
                                    }} /> :
                                    <>
                                        <Typography variant="h5">{title}</Typography>
                                        {
                                            Array.isArray(routine?.nodes) && routine!.nodes.length > 0 && <StatusButton
                                                status={status}
                                                messages={statusMessages}
                                                sx={{
                                                    marginTop: 'auto',
                                                    marginBottom: 'auto',
                                                    marginLeft: 2,
                                                }}
                                            />
                                        }
                                    </>
                                }
                            </Stack>
                            <Stack direction="row" spacing={1} sx={{
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                            }}>
                                {ownedBy && (
                                    <LinkButton
                                        onClick={toOwner}
                                        text={ownedBy}
                                    />
                                )}
                                <Typography variant="body1"> - {routine?.version}</Typography>
                                <SelectLanguageMenu
                                    availableLanguages={availableLanguages}
                                    canDropdownOpen={availableLanguages.length > 1}
                                    currentLanguage={language}
                                    handleCurrent={setLanguage}
                                    session={session}
                                    zIndex={zIndex}
                                />
                                {routine?.permissionsRoutine?.canEdit && <Tooltip title="Edit routine">
                                    <IconButton
                                        aria-label="Edit routine"
                                        size="small"
                                        onClick={onEdit}
                                    >
                                        <EditIcon fill={palette.secondary.light} />
                                    </IconButton>
                                </Tooltip>}
                            </Stack>
                        </Stack>
                        {/* Body container */}
                        <Box sx={{
                            padding: 2,
                        }}>
                            {body}
                        </Box>
                    </Box>
                    {/* Action buttons */}
                    <Stack direction="row" spacing={1} sx={{
                        display: 'flex',
                        marginRight: 'auto',
                        alignItems: 'center',
                        padding: 2,
                    }}>
                        {canVote && <UpvoteDownvote
                            direction="row"
                            session={session}
                            objectId={routine?.id ?? ''}
                            voteFor={VoteFor.Routine}
                            isUpvoted={routine?.isUpvoted}
                            score={routine?.score}
                            onChange={(isUpvote) => { routine && setRoutine({ ...routine, isUpvoted: isUpvote }); }}
                        />}
                        {canStar && <StarButton
                            session={session}
                            objectId={routine?.id ?? ''}
                            showStars={false}
                            starFor={StarFor.Routine}
                            isStar={routine?.isStarred ?? false}
                            stars={routine?.stars ?? 0}
                            onChange={(isStar: boolean) => { routine && setRoutine({ ...routine, isStarred: isStar }) }}
                            tooltipPlacement="bottom"
                        />}
                        <ReportsLink 
                            href={`${APP_LINKS.Routine}/reports/${routine?.id}`}
                            reports={routine?.reportsCount}
                        />
                        <Tooltip title="More options">
                            <IconButton
                                aria-label="More"
                                size="small"
                                onClick={openMoreMenu}
                                sx={{
                                    display: 'block',
                                    marginLeft: 'auto',
                                    marginRight: 1,
                                    padding: 0,
                                }}
                            >
                                <EllipsisIcon fill={palette.primary.contrastText} />
                            </IconButton>
                        </Tooltip>
                    </Stack>
                    {/* Comments Container */}
                    <CommentContainer
                        language={language}
                        objectId={id ?? ''}
                        objectType={CommentFor.Routine}
                        session={session}
                        zIndex={zIndex}
                    />
                </Box>
            </Stack>
        </>
    )
}