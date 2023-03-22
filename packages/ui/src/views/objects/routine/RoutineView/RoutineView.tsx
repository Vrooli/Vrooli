import { Box, Button, Dialog, Palette, Stack, useTheme } from "@mui/material";
import { CommentFor, FindVersionInput, LINKS, ResourceList, RoutineVersion, RunRoutine, RunRoutineCompleteInput } from "@shared/consts";
import { EditIcon, RoutineIcon, SuccessIcon } from "@shared/icons";
import { parseSearchParams, setSearchParams, useLocation } from '@shared/route';
import { setDotNotationValue } from "@shared/utils";
import { uuid } from '@shared/uuid';
import { routineVersionFindOne } from "api/generated/endpoints/routineVersion_findOne";
import { runRoutineComplete } from "api/generated/endpoints/runRoutine_complete";
import { useCustomMutation } from "api/hooks";
import { mutationWrapper } from "api/utils";
import { ColorIconButton } from "components/buttons/ColorIconButton/ColorIconButton";
import { RunButton } from "components/buttons/RunButton/RunButton";
import { SideActionButtons } from "components/buttons/SideActionButtons/SideActionButtons";
import { CommentContainer } from "components/containers/CommentContainer/CommentContainer";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { TextCollapse } from "components/containers/TextCollapse/TextCollapse";
import { UpTransition } from "components/dialogs/transitions";
import { GeneratedInputComponentWithLabel } from "components/inputs/generated";
import { RelationshipButtons } from "components/inputs/RelationshipButtons/RelationshipButtons";
import { ObjectActionsRow } from "components/lists/ObjectActionsRow/ObjectActionsRow";
import { ResourceListHorizontal } from "components/lists/resource";
import { smallHorizontalScrollbar } from "components/lists/styles";
import { TagList } from "components/lists/TagList/TagList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { ObjectTitle } from "components/text/ObjectTitle/ObjectTitle";
import { VersionDisplay } from "components/text/VersionDisplay/VersionDisplay";
import { useFormik } from "formik";
import { FieldData } from "forms/types";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { ObjectAction } from "utils/actions/objectActions";
import { getCurrentUser } from "utils/authentication/session";
import { defaultResourceList } from "utils/defaults/resourceList";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "utils/display/translationTools";
import { useObjectActions } from "utils/hooks/useObjectActions";
import { useObjectFromUrl } from "utils/hooks/useObjectFromUrl";
import { PubSub } from "utils/pubsub";
import { formikToRunInputs, runInputsCreate } from "utils/runUtils";
import { SessionContext } from "utils/SessionContext";
import { standardVersionToFieldData } from "utils/shape/general";
import { TagShape } from "utils/shape/models/tag";
import { BuildView } from "views/BuildView/BuildView";
import { RoutineViewProps } from "../types";

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
    display = 'page',
    partialData,
    zIndex = 200,
}: RoutineViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);

    const { id, isLoading, object: routineVersion, permissions, setObject: setRoutineVersion } = useObjectFromUrl<RoutineVersion, FindVersionInput>({
        query: routineVersionFindOne,
        onInvalidUrlParams: ({ build }) => {
            // Throw error if we are not creating a new routine
            if (!build || build !== true) PubSub.get().publishSnack({ messageKey: 'InvalidUrlId', severity: 'Error' });
        },
        partialData,
    });

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
        setSearchParams(setLocation, { build: true })
        setIsBuildOpen(true);
    }, [setLocation]);
    const stopBuild = useCallback(() => {
        setIsBuildOpen(false)
    }, []);

    const handleRunDelete = useCallback((run: RunRoutine) => {
        if (!routineVersion) return;
        setRoutineVersion(setDotNotationValue(routineVersion, 'you.runs', routineVersion.you.runs.filter(r => r.id !== run.id)));
    }, [routineVersion, setRoutineVersion]);

    const handleRunAdd = useCallback((run: RunRoutine) => {
        if (!routineVersion) return;
        setRoutineVersion(setDotNotationValue(routineVersion, 'you.runs', [run, ...routineVersion.you.runs]));
    }, [routineVersion, setRoutineVersion]);

    const [isAddCommentOpen, setIsAddCommentOpen] = useState(false);
    const openAddCommentDialog = useCallback(() => { setIsAddCommentOpen(true); }, []);
    const closeAddCommentDialog = useCallback(() => { setIsAddCommentOpen(false); }, []);

    const actionData = useObjectActions({
        object: routineVersion,
        objectType: 'Routine',
        openAddCommentDialog,
        setLocation,
        setObject: setRoutineVersion,
    });

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

    const [runComplete] = useCustomMutation<RunRoutine, RunRoutineCompleteInput>(runRoutineComplete);
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
                setLocation(LINKS.Home)
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
        <>
            <TopBar
                display={display}
                onClose={() => { }}
                titleData={{
                    titleKey: 'Routine',
                }}
            />
            <Box sx={{
                marginLeft: 'auto',
                marginRight: 'auto',
                width: 'min(100%, 700px)',
                padding: 2,
            }}>
                {/* Edit button (if canUpdate) and run button, positioned at bottom corner of screen */}
                <SideActionButtons
                    // Treat as a dialog when build view is open
                    display={isBuildOpen ? 'dialog' : display}
                    zIndex={zIndex + 2}
                >
                    {/* Edit button */}
                    {permissions.canUpdate ? (
                        <ColorIconButton aria-label="confirm-name-change" background={palette.secondary.main} onClick={() => { actionData.onActionStart(ObjectAction.Edit) }} >
                            <EditIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                        </ColorIconButton>
                    ) : null}
                    {/* Play button fixed to bottom of screen, to start routine (if multi-step) */}
                    {routineVersion?.nodes?.length ? <RunButton
                        canUpdate={permissions.canUpdate}
                        handleRunAdd={handleRunAdd as any}
                        handleRunDelete={handleRunDelete as any}
                        isBuildGraphOpen={isBuildOpen}
                        isEditing={false}
                        runnableObject={routineVersion}
                        zIndex={zIndex}
                    /> : null}
                </SideActionButtons>
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
                        loading={isLoading}
                        owner={relationships.owner}
                        routineVersion={routineVersion}
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
                    loading={isLoading}
                    title={name}
                    setLanguage={setLanguage}
                    translations={routineVersion?.translations ?? partialData?.translations ?? []}
                    zIndex={zIndex}
                />
                {/* Relationships */}
                <RelationshipButtons
                    isEditing={false}
                    objectType={'Routine'}
                    zIndex={zIndex}
                />
                {/* Resources */}
                {Array.isArray(resourceList.resources) && resourceList.resources.length > 0 && <ResourceListHorizontal
                    title={'Resources'}
                    list={resourceList}
                    canUpdate={false}
                    handleUpdate={() => { }} // Intentionally blank
                    loading={isLoading}
                    zIndex={zIndex}
                />}
                {/* Box with description and instructions */}
                <Stack direction="column" spacing={4} sx={containerProps(palette)}>
                    {/* Description */}
                    <TextCollapse title="Description" text={description} loading={isLoading} loadingLines={2} />
                    {/* Instructions */}
                    <TextCollapse title="Instructions" text={instructions} loading={isLoading} loadingLines={4} />
                </Stack>
                {/* Box with inputs, if this is a single-step routine */}
                {Object.keys(formik.values).length > 0 && <Box sx={containerProps(palette)}>
                    <ContentCollapse
                        isOpen={false}
                        title="Inputs"
                    >
                        {Object.values(formValueMap ?? {}).map((fieldData: FieldData, index: number) => (
                            <GeneratedInputComponentWithLabel
                                copyInput={copyInput}
                                disabled={false}
                                fieldData={fieldData}
                                index={index}
                                textPrimary={palette.background.textPrimary}
                                onUpload={() => { }}
                                zIndex={zIndex}
                            />
                        ))}
                        {getCurrentUser(session).id && <Button
                            startIcon={<SuccessIcon />}
                            fullWidth
                            onClick={markAsComplete}
                            color="secondary"
                            sx={{ marginTop: 2 }}
                        >{t(`MarkAsComplete`)}</Button>}
                    </ContentCollapse>
                </Box>}
                {/* "View Graph" button if this is a multi-step routine */}
                {routineVersion?.nodes?.length ? <Button startIcon={<RoutineIcon />} fullWidth onClick={viewGraph} color="secondary">View Graph</Button> : null}
                {/* Tags */}
                {tags.length > 0 && <TagList
                    maxCharacters={30}
                    parentId={routineVersion?.id ?? ''}
                    tags={tags as any[]}
                    sx={{ ...smallHorizontalScrollbar(palette), marginTop: 4 }}
                />}
                {/* Date and version labels */}
                <Stack direction="row" spacing={1} mt={2} mb={1}>
                    {/* Date created */}
                    <DateDisplay
                        loading={isLoading}
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
            /> */}
                {/* Action buttons */}
                <ObjectActionsRow
                    actionData={actionData}
                    exclude={[ObjectAction.Edit, ObjectAction.VoteDown, ObjectAction.VoteUp]} // Handled elsewhere
                    object={routineVersion}
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
                        zIndex={zIndex}
                    />
                </Box>
            </Box>
        </>
    )
}