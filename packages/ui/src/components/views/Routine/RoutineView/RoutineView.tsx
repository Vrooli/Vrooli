import { Box, Button, Dialog, Grid, Palette, Stack, useTheme } from "@mui/material"
import { useLocation } from '@shared/route';
import { APP_LINKS } from "@shared/consts";
import { useMutation, useLazyQuery } from "@apollo/client";
import { routine, routineVariables } from "graphql/generated/routine";
import { routineQuery } from "graphql/query";
import { useCallback, useEffect, useMemo, useState } from "react";
import { BuildView, ResourceListHorizontal, RunPickerMenu, RunView, UpTransition, OwnerLabel, VersionDisplay, SnackSeverity, ObjectTitle, StatsCompact, ObjectActionsRow } from "components";
import { RoutineViewProps } from "../types";
import { base36ToUuid, formikToRunInputs, getLanguageSubtag, getLastUrlPart, getPreferredLanguage, getTranslation, getUserLanguages, initializeRoutine, ObjectAction, ObjectActionComplete, ObjectType, openObject, parseSearchParams, PubSub, runInputsCreate, setSearchParams, standardToFieldData, useReactSearch, uuidToBase36 } from "utils";
import { Routine, Run } from "types";
import { runCompleteMutation } from "graphql/mutation";
import { mutationWrapper } from "graphql/utils/graphqlWrapper";
import { CommentFor, NodeType } from "graphql/generated/globalTypes";
import { uuidValidate } from '@shared/uuid';
import { runCompleteVariables, runComplete_runComplete } from "graphql/generated/runComplete";
import { useFormik } from "formik";
import { FieldData } from "forms/types";
import { generateInputWithLabel } from "forms/generators";
import { CommentContainer, ContentCollapse, TextCollapse } from "components/containers";
import { RoutineIcon, SuccessIcon } from "@shared/icons";
import { getCurrentUser } from "utils/authentication";

const statsHelpText =
    `Statistics are calculated to measure various aspects of a routine. 
    
**Complexity** is a rough measure of the maximum amount of effort it takes to complete a routine. This takes into account the number of inputs, the structure of its subroutine graph, and the complexity of every subroutine.

**Simplicity** is calculated similarly to complexity, but takes the shortest path through the subroutine graph.

There will be many more statistics in the near future.`

const containerProps = (palette: Palette) => ({
    boxShadow: 1,
    background: palette.background.paper,
    borderRadius: 1,
    overflow: 'overlay',
    marginTop: 4,
    marginBottom: 4,
    padding: 2,
})

export const RoutineView = ({
    partialData,
    session,
    zIndex,
}: RoutineViewProps) => {
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);

    // Fetch data
    const { id, versionGroupId } = useMemo(() => {
        // URL is /object/:versionGroupId/?:id
        const last = base36ToUuid(getLastUrlPart(0), false);
        const secondLast = base36ToUuid(getLastUrlPart(1), false);
        return {
            id: uuidValidate(secondLast) ? last : undefined,
            versionGroupId: uuidValidate(secondLast) ? secondLast : last,
        }
    }, []);
    const [getData, { data, loading }] = useLazyQuery<routine, routineVariables>(routineQuery, { errorPolicy: 'all' });
    useEffect(() => {
        if (uuidValidate(id) || uuidValidate(versionGroupId)) getData({ variables: { input: { id, versionGroupId } } });
        // If IDs are not invalid, throw error if we are not creating a new routine
        else {
            const { build } = parseSearchParams();
            if (!build || build !== true) PubSub.get().publishSnack({ message: 'Could not parse ID in URL', severity: SnackSeverity.Error });
        }
    }, [getData, id, versionGroupId])

    const [routine, setRoutine] = useState<Routine>(initializeRoutine(language));
    useEffect(() => {
        if (!data?.routine) return;
        setRoutine(data.routine);
    }, [data]);
    const updateRoutine = useCallback((newRoutine: Routine) => { setRoutine(newRoutine); }, [setRoutine]);

    const search = useReactSearch(null);
    const { runId } = useMemo(() => ({
        runId: typeof search.run === 'string' && uuidValidate(search.run) ? search.run : null,
    }), [search]);

    const availableLanguages = useMemo<string[]>(() => (routine?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [routine?.translations]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { title, description, instructions } = useMemo(() => {
        const { description, instructions, title } = getTranslation(routine ?? partialData, [language]);
        return { title, description, instructions };
    }, [routine, language, partialData]);

    useEffect(() => {
        document.title = `${title} | Vrooli`;
    }, [title]);

    const [isBuildOpen, setIsBuildOpen] = useState<boolean>(Boolean(parseSearchParams()?.build));
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
                run: uuidToBase36(run.id),
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
            PubSub.get().publishSnack({ message: 'Error loading routine.', severity: SnackSeverity.Error });
            return;
        }
        // Find first node
        const firstNode = routine?.nodes?.find(node => node.type === NodeType.Start);
        if (!firstNode) {
            PubSub.get().publishSnack({ message: 'Routine invalid - cannot run.', severity: SnackSeverity.Error });
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
        id && setLocation(`${APP_LINKS.Routine}/edit/${uuidToBase36(id)}`);
    }, [setLocation, id]);

    const onActionStart = useCallback((action: ObjectAction) => {
        switch (action) {
            case ObjectAction.Edit:
                onEdit();
                break;
            case ObjectAction.Stats:
                //TODO
                break;
        }
    }, [onEdit]);

    const onActionComplete = useCallback((action: ObjectActionComplete, data: any) => {
        switch (action) {
            case ObjectActionComplete.VoteDown:
            case ObjectActionComplete.VoteUp:
                if (data.success) {
                    setRoutine({
                        ...routine,
                        isUpvoted: action === ObjectActionComplete.VoteUp,
                    } as any)
                }
                break;
            case ObjectActionComplete.Star:
            case ObjectActionComplete.StarUndo:
                if (data.success) {
                    setRoutine({
                        ...routine,
                        isStarred: action === ObjectActionComplete.Star,
                    } as any)
                }
                break;
            case ObjectActionComplete.Fork:
                openObject(data.routine, setLocation);
                window.location.reload();
                break;
            case ObjectActionComplete.Copy:
                openObject(data.routine, setLocation);
                window.location.reload();
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
                description: getTranslation(currInput, getUserLanguages(session), false).description ?? getTranslation(currInput.standard, getUserLanguages(session), false).description,
                fieldName: `inputs-${currInput.id}`,
                helpText: getTranslation(currInput, getUserLanguages(session), false).helpText,
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

    const [runComplete] = useMutation(runCompleteMutation);
    const markAsComplete = useCallback(() => {
        if (!routine) return;
        mutationWrapper<runComplete_runComplete, runCompleteVariables>({
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
     * Copy current value of input to clipboard
     * @param fieldName Name of input
     */
    const copyInput = useCallback((fieldName: string) => {
        const input = formik.values[fieldName];
        if (input) {
            navigator.clipboard.writeText(input);
            PubSub.get().publishSnack({ message: 'Copied to clipboard.', severity: SnackSeverity.Success });
        } else {
            PubSub.get().publishSnack({ message: 'Input is empty.', severity: SnackSeverity.Error });
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

    return (
        <Box sx={{
            marginLeft: 'auto',
            marginRight: 'auto',
            width: 'min(100%, 700px)',
            padding: 2,
        }}>
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
            <ObjectTitle
                language={language}
                loading={loading}
                title={title}
                session={session}
                setLanguage={setLanguage}
                translations={routine?.translations ?? partialData?.translations ?? []}
                zIndex={zIndex}
            />
            {/* Resources */}
            {resourceList}
            {/* Box with description and instructions */}
            <Stack direction="column" spacing={4} sx={containerProps(palette)}>
                {/* Description */}
                <TextCollapse title="Description" text={description} loading={loading} loadingLines={2} />
                {/* Instructions */}
                <TextCollapse title="Instructions" text={instructions} loading={loading} loadingLines={4} />
            </Stack>
            {/* Box with inputs, if this is a single-step routine */}
            {Object.keys(formik.values).length > 0 && <Box sx={containerProps(palette)}>
                <ContentCollapse
                    isOpen={false}
                    title="Inputs"
                >
                    {Object.values(formValueMap ?? {}).map((fieldData: FieldData, index: number) => (
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
                    ))}
                    {getCurrentUser(session).id && <Grid container spacing={1} mt={1}>
                        <Grid item xs={12}>
                            <Button startIcon={<SuccessIcon />} fullWidth onClick={markAsComplete} color="secondary">Mark as Complete</Button>
                        </Grid>
                    </Grid>}
                </ContentCollapse>
            </Box>}
            {/* "View Graph" button if this is a multi-step routine */}
            {routine?.nodes?.length ? <Button startIcon={<RoutineIcon />} fullWidth onClick={viewGraph} color="secondary">View Graph</Button> : null}
            {/* Owner and version labels */}
            <Stack direction="row" spacing={1} mt={4} mb={1}>
                <OwnerLabel
                    objectType={ObjectType.Routine}
                    owner={routine?.owner}
                    session={session}
                    sxs={{ label: { color: palette.background.textPrimary } }}
                />
                <VersionDisplay
                    currentVersion={routine?.version}
                    prefix={" - "}
                    versions={routine?.versions}
                />
            </Stack>
            {/* Votes, reports, and other basic stats */}
            <StatsCompact
                handleObjectUpdate={updateRoutine}
                loading={loading}
                object={routine}
                session={session}
            />
            {/* Action buttons */}
            <ObjectActionsRow
                exclude={[ObjectAction.VoteDown, ObjectAction.VoteUp]} // Handled by StatsCompact
                onActionStart={onActionStart}
                onActionComplete={onActionComplete}
                object={routine}
                session={session}
                zIndex={zIndex}
            />
            {/* Comments */}
            <Box sx={containerProps(palette)}>
                <CommentContainer
                    language={language}
                    objectId={id ?? ''}
                    objectType={CommentFor.Routine}
                    session={session}
                    zIndex={zIndex}
                />
            </Box>
            {/* Play button fixed to bottom of screen, to start routine (if multi-step).
            If the routine is invalid, this is greyed out with a tooltip on hover or press. 
            If the routine is valid but incomplete, play button is available but user must accept alert dialog warning */}
            <RunButton />
            {/* TODO */}
        </Box>
    )
}