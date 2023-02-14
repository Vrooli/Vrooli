import { Box, Button, Dialog, Palette, Stack, useTheme } from "@mui/material"
import { useLocation } from '@shared/route';
import { APP_LINKS, CommentFor, FindVersionInput, ResourceList, RoutineVersion, RunRoutine, RunRoutineCompleteInput } from "@shared/consts";
import { useMutation, useLazyQuery } from "api/hooks";
import { useCallback, useEffect, useMemo, useState } from "react";
import { BuildView, ResourceListHorizontal, UpTransition, VersionDisplay, ObjectTitle, StatsCompact, ObjectActionsRow, RunButton, TagList, RelationshipButtons, ColorIconButton, DateDisplay } from "components";
import { RoutineViewProps } from "../types";
import { formikToRunInputs, getLanguageSubtag, getYou, getPreferredLanguage, getTranslation, getUserLanguages, ObjectAction, ObjectActionComplete, openObject, parseSearchParams, PubSub, runInputsCreate, setSearchParams, standardVersionToFieldData, TagShape, uuidToBase36, parseSingleItemUrl, defaultRelationships, defaultResourceList } from "utils";
import { mutationWrapper } from "api/utils";
import { uuid } from '@shared/uuid';
import { useFormik } from "formik";
import { FieldData } from "forms/types";
import { generateInputWithLabel } from "forms/generators";
import { CommentContainer, ContentCollapse, TextCollapse } from "components/containers";
import { EditIcon, RoutineIcon, SuccessIcon } from "@shared/icons";
import { getCurrentUser } from "utils/authentication";
import { smallHorizontalScrollbar } from "components/lists/styles";
import { RelationshipsObject } from "components/inputs/types";
import { setDotNotationValue } from "@shared/utils";
import { runRoutineComplete } from "api/generated/endpoints/runRoutine";
import { routineVersionFindOne } from "api/generated/endpoints/routineVersion";

const statsHelpText =
    `Statistics are calculated to measure various aspects of a routine. \n\n**Complexity** is a rough measure of the maximum amount of effort it takes to complete a routine. This takes into account the number of inputs, the structure of its subroutine graph, and the complexity of every subroutine.\n\n**Simplicity** is calculated similarly to complexity, but takes the shortest path through the subroutine graph.\n\nThere will be many more statistics in the near future.`

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
    const urlData = useMemo(() => parseSingleItemUrl(), []);
    const [getData, { data, loading }] = useLazyQuery<RoutineVersion, FindVersionInput, 'routineVersion'>(routineVersionFindOne, 'routineVersion', { errorPolicy: 'all' });
    useEffect(() => {
        if (urlData.id || urlData.idRoot) getData({ variables: urlData });
        // If IDs are not invalid, throw error if we are not creating a new routine
        else {
            const { build } = parseSearchParams();
            if (!build || build !== true) PubSub.get().publishSnack({ messageKey: 'InvalidUrlId', severity: 'Error' });
        }
    }, [getData, urlData])

    const [routineVersion, setRoutineVersion] = useState<RoutineVersion | null>(null);
    useEffect(() => {
        if (!data?.routineVersion) return;
        setRoutineVersion(data.routineVersion);
    }, [data]);
    const updateRoutineVersion = useCallback((newRoutineVersion: RoutineVersion) => { setRoutineVersion(newRoutineVersion); }, [setRoutineVersion]);
    const { canUpdate } = useMemo(() => getYou(routineVersion), [routineVersion]);

    const availableLanguages = useMemo<string[]>(() => (routineVersion?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [routineVersion?.translations]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { name, description, instructions } = useMemo(() => {
        const { description, instructions, name } = getTranslation(routineVersion ?? partialData, [language]);
        return { name, description, instructions };
    }, [routineVersion, language, partialData]);

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
        if (!routineVersion) return;
        setRoutineVersion(setDotNotationValue(routineVersion, 'you.runs', routineVersion.you.runs.filter(r => r.id !== run.id)));
    }, [routineVersion]);

    const handleRunAdd = useCallback((run: RunRoutine) => {
        if (!routineVersion) return;
        setRoutineVersion(setDotNotationValue(routineVersion, 'you.runs', [run, ...routineVersion.you.runs]));
    }, [routineVersion]);

    const onEdit = useCallback(() => {
        routineVersion?.id && setLocation(`${APP_LINKS.Routine}/edit/${uuidToBase36(routineVersion.id)}`);
    }, [setLocation, routineVersion?.id]);

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
                if (data.success && routineVersion) {
                    setRoutineVersion(setDotNotationValue(routineVersion, 'root.you.isUpvoted', action === ObjectActionComplete.VoteUp))
                }
                break;
            case ObjectActionComplete.Star:
            case ObjectActionComplete.StarUndo:
                if (data.success && routineVersion) {
                    setRoutineVersion(setDotNotationValue(routineVersion, 'root.you.isBookmarked', action === ObjectActionComplete.Star))
                }
                break;
            case ObjectActionComplete.Fork:
                openObject(data.routine, setLocation);
                window.location.reload();
                break;
        }
    }, [routineVersion, setLocation]);

    // The schema and formik keys for the form
    const formValueMap = useMemo<{ [fieldName: string]: FieldData } | null>(() => {
        if (!routineVersion) return null;
        const schemas: { [fieldName: string]: FieldData } = {};
        for (let i = 0; i < routineVersion.inputs?.length; i++) {
            const currInput = routineVersion.inputs[i];
            if (!currInput.standardVersion) continue;
            const currSchema = standardVersionToFieldData({
                description: getTranslation(currInput, getUserLanguages(session), false).description ?? getTranslation(currInput.standardVersion, getUserLanguages(session), false).description,
                fieldName: `inputs-${currInput.id}`,
                helpText: getTranslation(currInput, getUserLanguages(session), false).helpText,
                props: currInput.standardVersion.props,
                name: currInput.name ?? currInput.standardVersion?.root?.name,
                standardType: currInput.standardVersion.standardType,
                yup: currInput.standardVersion.yup,
            });
            if (currSchema) {
                schemas[currSchema.fieldName] = currSchema;
            }
        }
        return schemas;
    }, [routineVersion, session]);
    const formik = useFormik({
        initialValues: Object.entries(formValueMap ?? {}).reduce((acc, [key, value]) => {
            acc[key] = value.props.defaultValue ?? '';
            return acc;
        }, {}),
        enableReinitialize: true,
        onSubmit: () => { },
    });

    const [runComplete] = useMutation<RunRoutine, RunRoutineCompleteInput, 'runRoutineComplete'>(runRoutineComplete, 'runRoutineComplete');
    const markAsComplete = useCallback(() => {
        if (!routineVersion) return;
        mutationWrapper<RunRoutine, RunRoutineCompleteInput>({
            mutation: runComplete,
            input: {
                id: routineVersion.id,
                exists: false,
                name: name ?? 'Unnamed Routine',
                ...runInputsCreate(formikToRunInputs(formik.values), routineVersion.id),
            },
            successMessage: () => ({ key: 'RoutineCompleted' }),
            onSuccess: () => {
                PubSub.get().publishCelebration();
                setLocation(APP_LINKS.Home)
            },
        })
    }, [formik.values, routineVersion, runComplete, setLocation, name]);

    /**
     * Copy current value of input to clipboard
     * @param fieldName Name of input
     */
    const copyInput = useCallback((fieldName: string) => {
        const input = formik.values[fieldName];
        if (input) {
            navigator.clipboard.writeText(input);
            PubSub.get().publishSnack({ messageKey: 'CopiedToClipboard', severity: 'Success' });
        } else {
            PubSub.get().publishSnack({ messageKey: 'InputEmpty', severity: 'Error' });
        }
    }, [formik]);

    // Handle relationships
    const [relationships, setRelationships] = useState<RelationshipsObject>(defaultRelationships(false, null));
    const onRelationshipsChange = useCallback((change: Partial<RelationshipsObject>) => setRelationships({ ...relationships, ...change }), [relationships]);

    // Handle resources
    const [resourceList, setResourceList] = useState<ResourceList>(defaultResourceList);

    // Handle tags
    const [tags, setTags] = useState<TagShape[]>((partialData?.root?.tags as TagShape[] | undefined) ?? []);

    useEffect(() => {
        setRelationships({
            isComplete: routineVersion?.isComplete ?? false,
            isPrivate: routineVersion?.isPrivate ?? false,
            owner: routineVersion?.root?.owner ?? null,
            parent: null,
            // parent: routine?.parent ?? null, TODO
            project: null, //TODO
        });
        setResourceList(routineVersion?.resourceList ?? { id: uuid() } as any);
        setTags(routineVersion?.root?.tags ?? []);
    }, [routineVersion]);

    return (
        <Box sx={{
            marginLeft: 'auto',
            marginRight: 'auto',
            width: 'min(100%, 700px)',
            padding: 2,
        }}>
            {/* Edit button (if canUpdate) and run button, positioned at bottom corner of screen */}
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
                {canUpdate ? (
                    <ColorIconButton aria-label="confirm-name-change" background={palette.secondary.main} onClick={() => { onActionStart(ObjectAction.Edit) }} >
                        <EditIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </ColorIconButton>
                ) : null}
                {/* Play button fixed to bottom of screen, to start routine (if multi-step) */}
                {routineVersion?.nodes?.length ? <RunButton
                    canUpdate={canUpdate}
                    handleRunAdd={handleRunAdd as any}
                    handleRunDelete={handleRunDelete as any}
                    isBuildGraphOpen={isBuildOpen}
                    isEditing={false}
                    runnableObject={routineVersion}
                    session={session}
                    zIndex={zIndex}
                /> : null}
            </Stack>
            {/* Dialog for building routine */}
            {routineVersion && <Dialog
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
                    routineVersion={routineVersion}
                    session={session}
                    translationData={{
                        language,
                        setLanguage,
                        handleAddLanguage: () => { },
                        handleDeleteLanguage: () => { },
                        translations: routineVersion.translations,
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
                translations={routineVersion?.translations ?? partialData?.translations ?? []}
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
                canUpdate={false}
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
            {routineVersion?.nodes?.length ? <Button startIcon={<RoutineIcon />} fullWidth onClick={viewGraph} color="secondary">View Graph</Button> : null}
            {/* Tags */}
            {tags.length > 0 && <TagList
                maxCharacters={30}
                parentId={routineVersion?.id ?? ''}
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
                    timestamp={routineVersion?.created_at}
                />
                <VersionDisplay
                    currentVersion={routineVersion}
                    prefix={" - "}
                    versions={routineVersion?.root?.versions}
                />
            </Stack>
            {/* Votes, reports, and other basic stats */}
            {/* <StatsCompact
                handleObjectUpdate={updateRoutineVersion}
                loading={loading}
                object={routineVersion}
                session={session}
            /> */}
            {/* Action buttons */}
            <ObjectActionsRow
                exclude={[ObjectAction.Edit, ObjectAction.VoteDown, ObjectAction.VoteUp]} // Handled elsewhere
                onActionStart={onActionStart}
                onActionComplete={onActionComplete}
                object={routineVersion}
                session={session}
                zIndex={zIndex}
            />
            {/* Comments */}
            <Box sx={containerProps(palette)}>
                <CommentContainer
                    forceAddCommentOpen={isAddCommentOpen}
                    language={language}
                    objectId={routineVersion?.id ?? ''}
                    objectType={CommentFor.RoutineVersion}
                    onAddCommentClose={closeAddCommentDialog}
                    session={session}
                    zIndex={zIndex}
                />
            </Box>
        </Box>
    )
}