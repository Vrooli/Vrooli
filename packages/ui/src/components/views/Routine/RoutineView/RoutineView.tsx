import { Box, Button, Dialog, Palette, Stack, useTheme } from "@mui/material"
import { useLocation } from '@shared/route';
import { APP_LINKS, CommentFor, FindByIdInput, ResourceList, Routine, RunRoutine, RunRoutineCompleteInput } from "@shared/consts";
import { useMutation, useLazyQuery } from "graphql/hooks";
import { useCallback, useEffect, useMemo, useState } from "react";
import { BuildView, ResourceListHorizontal, UpTransition, VersionDisplay, SnackSeverity, ObjectTitle, StatsCompact, ObjectActionsRow, RunButton, TagList, RelationshipButtons, ColorIconButton, DateDisplay } from "components";
import { RoutineViewProps } from "../types";
import { base36ToUuid, formikToRunInputs, getLanguageSubtag, getLastUrlPart, getListItemPermissions, getPreferredLanguage, getTranslation, getUserLanguages, ObjectAction, ObjectActionComplete, openObject, parseSearchParams, PubSub, runInputsCreate, setSearchParams, standardToFieldData, TagShape, uuidToBase36 } from "utils";
import { mutationWrapper } from "graphql/utils";
import { uuid, uuidValidate } from '@shared/uuid';
import { useFormik } from "formik";
import { FieldData } from "forms/types";
import { generateInputWithLabel } from "forms/generators";
import { CommentContainer, ContentCollapse, TextCollapse } from "components/containers";
import { EditIcon, RoutineIcon, SuccessIcon } from "@shared/icons";
import { getCurrentUser } from "utils/authentication";
import { smallHorizontalScrollbar } from "components/lists/styles";
import { RelationshipsObject } from "components/inputs/types";
import { routineEndpoint, runRoutineEndpoint } from "graphql/endpoints";

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
    const [getData, { data, loading }] = useLazyQuery<Routine, FindByIdInput, 'routine'>(...routineEndpoint.findOne, { errorPolicy: 'all' });
    useEffect(() => {
        if (uuidValidate(id) || uuidValidate(versionGroupId)) getData({ variables: { id, versionGroupId } });
        // If IDs are not invalid, throw error if we are not creating a new routine
        else {
            const { build } = parseSearchParams();
            if (!build || build !== true) PubSub.get().publishSnack({ messageKey: 'InvalidUrlId', severity: SnackSeverity.Error });
        }
    }, [getData, id, versionGroupId])

    const [routine, setRoutine] = useState<Routine | null>(null);
    useEffect(() => {
        if (!data?.routine) return;
        setRoutine(data.routine);
    }, [data]);
    const updateRoutine = useCallback((newRoutine: Routine) => { setRoutine(newRoutine); }, [setRoutine]);
    const canEdit = useMemo(() => getListItemPermissions(routine as any, session).canEdit, [routine, session]);

    const availableLanguages = useMemo<string[]>(() => (routine?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [routine?.translations]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { name, description, instructions } = useMemo(() => {
        const { description, instructions, name } = getTranslation(routine ?? partialData, [language]);
        return { name, description, instructions };
    }, [routine, language, partialData]);

    useEffect(() => {
        document.title = `${name} | Vrooli`;
    }, [name]);

    const [isBuildOpen, setIsBuildOpen] = useState<boolean>(Boolean(parseSearchParams()?.build));
    const viewGraph = useCallback(() => {
        setSearchParams(setLocation, {
            build: true,
        })
        setIsBuildOpen(true);
    }, [setLocation]);
    const stopBuild = useCallback(() => {
        setIsBuildOpen(false)
    }, []);

    const handleRunDelete = useCallback((run: RunRoutine) => {
        if (!routine) return;
        setRoutine({
            ...routine,
            runs: routine.runs.filter(r => r.id !== run.id),
        });
    }, [routine]);

    const handleRunAdd = useCallback((run: RunRoutine) => {
        if (!routine) return;
        setRoutine({
            ...routine,
            runs: [run, ...routine.runs],
        });
    }, [routine]);

    const onEdit = useCallback(() => {
        id && setLocation(`${APP_LINKS.Routine}/edit/${uuidToBase36(id)}`);
    }, [setLocation, id]);

    const [isAddCommentOpen, setIsAddCommentOpen] = useState(false);
    const openAddCommentDialog = useCallback(() => { setIsAddCommentOpen(true); }, []);
    const closeAddCommentDialog = useCallback(() => { setIsAddCommentOpen(false); }, []);

    const onActionStart = useCallback((action: ObjectAction) => {
        switch (action) {
            case ObjectAction.Comment:
                openAddCommentDialog();
                break;
            case ObjectAction.Edit:
                onEdit();
                break;
            case ObjectAction.Stats:
                //TODO
                break;
        }
    }, [onEdit, openAddCommentDialog]);

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

    const [runComplete] = useMutation<RunRoutine, RunRoutineCompleteInput, 'runRoutineComplete'>(...runRoutineEndpoint.complete);
    const markAsComplete = useCallback(() => {
        if (!routine) return;
        mutationWrapper<RunRoutine, RunRoutineCompleteInput>({
            mutation: runComplete,
            input: {
                id: routineVersion.id,
                exists: false,
                name: name ?? 'Unnamed Routine',
                ...runInputsCreate(formikToRunInputs(formik.values)),
            },
            successMessage: () => ({ key: 'RoutineCompleted' }),
            onSuccess: () => {
                PubSub.get().publishCelebration();
                setLocation(APP_LINKS.Home)
            },
        })
    }, [formik.values, routine, runComplete, setLocation, name]);

    /**
     * Copy current value of input to clipboard
     * @param fieldName Name of input
     */
    const copyInput = useCallback((fieldName: string) => {
        const input = formik.values[fieldName];
        if (input) {
            navigator.clipboard.writeText(input);
            PubSub.get().publishSnack({ messageKey: 'CopiedToClipboard', severity: SnackSeverity.Success });
        } else {
            PubSub.get().publishSnack({ messageKey: 'InputEmpty', severity: SnackSeverity.Error });
        }
    }, [formik]);

    // Handle relationships
    const [relationships, setRelationships] = useState<RelationshipsObject>({
        isComplete: true,
        isPrivate: false,
        owner: null,
        parent: null,
        project: null,
    });
    const onRelationshipsChange = useCallback((newRelationshipsObject: Partial<RelationshipsObject>) => 
        setRelationships({ ...relationships, ...newRelationshipsObject }), [relationships]);

    // Handle resources
    const [resourceList, setResourceList] = useState<ResourceList>({ id: uuid() } as any);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>((partialData?.tags as TagShape[] | undefined) ?? []);

    useEffect(() => {
        setRelationships({
            isComplete: routine?.isComplete ?? false,
            isPrivate: routine?.isPrivate ?? false,
            owner: routine?.owner ?? null,
            parent: null,
            // parent: routine?.parent ?? null, TODO
            project: null, //TODO
        });
        setResourceList(routine?.resourceList ?? { id: uuid() } as any);
        setTags(routine?.tags ?? []);
    }, [routine]);

    return (
        <Box sx={{
            marginLeft: 'auto',
            marginRight: 'auto',
            width: 'min(100%, 700px)',
            padding: 2,
        }}>
            {/* Edit button (if canEdit) and run button, positioned at bottom corner of screen */}
            <Stack direction="row" spacing={2} sx={{
                position: 'fixed',
                display: 'flex',
                alignItems: 'center',
                justifyContent: 'center',
                zIndex: zIndex + 2,
                bottom: 0,
                right: 0,
                // Accounts for BottomNav, BuildView, and edit/cancel buttons in BuildView
                marginBottom: {
                    xs: !isBuildOpen ? 'calc(56px + 16px + env(safe-area-inset-bottom))' : 'calc(16px + env(safe-area-inset-bottom))',
                    md: 'calc(16px + env(safe-area-inset-bottom))'
                },
                marginLeft: 'calc(16px + env(safe-area-inset-left))',
                marginRight: 'calc(16px + env(safe-area-inset-right))',
                height: 'calc(64px + env(safe-area-inset-bottom))',
            }}>
                {/* Edit button */}
                {canEdit ? (
                    <ColorIconButton aria-label="confirm-name-change" background={palette.secondary.main} onClick={() => { onActionStart(ObjectAction.Edit) }} >
                        <EditIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </ColorIconButton>
                ) : null}
                {/* Play button fixed to bottom of screen, to start routine (if multi-step) */}
                {routine?.nodes?.length ? <RunButton
                    canEdit={canEdit}
                    handleRunAdd={handleRunAdd}
                    handleRunDelete={handleRunDelete}
                    isBuildGraphOpen={isBuildOpen}
                    isEditing={false}
                    routine={routine}
                    session={session}
                    zIndex={zIndex}
                /> : null}
            </Stack>
            {/* Dialog for building routine */}
            {routine && <Dialog
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
                    handleCancel={stopBuild}
                    handleClose={stopBuild}
                    handleSubmit={() => { }} //Intentionally blank, since this is a read-only view
                    isEditing={false}
                    loading={loading}
                    owner={relationships.owner}
                    routine={routine}
                    session={session}
                    translationData={{
                        language,
                        setLanguage,
                        handleAddLanguage: () => { },
                        handleDeleteLanguage: () => { },
                        translations: routine.translations,
                    }}
                    zIndex={zIndex + 1}
                />
            </Dialog>}
            <ObjectTitle
                language={language}
                loading={loading}
                title={name}
                session={session}
                setLanguage={setLanguage}
                translations={routine?.translations ?? partialData?.translations ?? []}
                zIndex={zIndex}
            />
            {/* Relationships */}
            <RelationshipButtons
                isEditing={false}
                objectType={'Routine'}
                onRelationshipsChange={onRelationshipsChange}
                relationships={relationships}
                session={session}
                zIndex={zIndex}
            />
            {/* Resources */}
            {Array.isArray(resourceList.resources) && resourceList.resources.length > 0 && <ResourceListHorizontal
                title={'Resources'}
                list={resourceList}
                canEdit={false}
                handleUpdate={() => { }} // Intentionally blank
                loading={loading}
                session={session}
                zIndex={zIndex}
            />}
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
                    {getCurrentUser(session).id && <Button
                        startIcon={<SuccessIcon />}
                        fullWidth
                        onClick={markAsComplete}
                        color="secondary"
                        sx={{ marginTop: 2 }}
                    >Mark as Complete</Button>}
                </ContentCollapse>
            </Box>}
            {/* "View Graph" button if this is a multi-step routine */}
            {routine?.nodes?.length ? <Button startIcon={<RoutineIcon />} fullWidth onClick={viewGraph} color="secondary">View Graph</Button> : null}
            {/* Tags */}
            {tags.length > 0 && <TagList
                maxCharacters={30}
                parentId={routine?.id ?? ''}
                session={session}
                tags={tags as any[]}
                sx={{ ...smallHorizontalScrollbar(palette), marginTop: 4 }}
            />}
            {/* Date and version labels */}
            <Stack direction="row" spacing={1} mt={2} mb={1}>
                {/* Date created */}
                <DateDisplay
                    loading={loading}
                    showIcon={true}
                    timestamp={routine?.created_at}
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
                exclude={[ObjectAction.Edit, ObjectAction.VoteDown, ObjectAction.VoteUp]} // Handled elsewhere
                onActionStart={onActionStart}
                onActionComplete={onActionComplete}
                object={routine}
                session={session}
                zIndex={zIndex}
            />
            {/* Comments */}
            <Box sx={containerProps(palette)}>
                <CommentContainer
                    forceAddCommentOpen={isAddCommentOpen}
                    language={language}
                    objectId={id ?? ''}
                    objectType={CommentFor.RoutineVersion}
                    onAddCommentClose={closeAddCommentDialog}
                    session={session}
                    zIndex={zIndex}
                />
            </Box>
        </Box>
    )
}