import { CommentFor, ResourceList as ResourceListType, RoutineVersion, RunRoutine, RunRoutineCompleteInput, Tag, endpointGetRoutineVersion, endpointPutRunRoutineComplete, exists, noop, noopSubmit, parseSearchParams, setDotNotationValue } from "@local/shared";
import { Box, Button, Divider, IconButton, Stack, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api";
import { RunButton } from "components/buttons/RunButton/RunButton";
import { SideActionsButtons } from "components/buttons/SideActionsButtons/SideActionsButtons";
import { CommentContainer, containerProps } from "components/containers/CommentContainer/CommentContainer";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { TextCollapse } from "components/containers/TextCollapse/TextCollapse";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { GeneratedInputComponentWithLabel } from "components/inputs/generated";
import { ObjectActionsRow } from "components/lists/ObjectActionsRow/ObjectActionsRow";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TagList } from "components/lists/TagList/TagList";
import { ResourceList } from "components/lists/resource";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { StatsCompact } from "components/text/StatsCompact/StatsCompact";
import { Title } from "components/text/Title/Title";
import { VersionDisplay } from "components/text/VersionDisplay/VersionDisplay";
import { SessionContext } from "contexts/SessionContext";
import { Formik, useFormik } from "formik";
import { FieldData } from "forms/types";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useObjectActions } from "hooks/useObjectActions";
import { useObjectFromUrl } from "hooks/useObjectFromUrl";
import { EditIcon, RoutineIcon, SuccessIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { setSearchParams, useLocation } from "route";
import { ObjectAction } from "utils/actions/objectActions";
import { getCurrentUser } from "utils/authentication/session";
import { firstString } from "utils/display/stringTools";
import { getLanguageSubtag, getPreferredLanguage, getTranslation, getUserLanguages } from "utils/display/translationTools";
import { openObject } from "utils/navigation/openObject";
import { PubSub } from "utils/pubsub";
import { formikToRunInputs, runInputsCreate } from "utils/runUtils";
import { standardVersionToFieldData } from "utils/shape/general";
import { ResourceListShape } from "utils/shape/models/resourceList";
import { RoutineShape } from "utils/shape/models/routine";
import { TagShape } from "utils/shape/models/tag";
import { BuildView } from "views/objects/routine/BuildView/BuildView";
import { routineInitialValues } from "../RoutineUpsert/RoutineUpsert";
import { RoutineViewProps } from "../types";

const statsHelpText =
    "Statistics are calculated to measure various aspects of a routine. \n\n**Complexity** is a rough measure of the maximum amount of effort it takes to complete a routine. This takes into account the number of inputs, the structure of its subroutine graph, and the complexity of every subroutine.\n\n**Simplicity** is calculated similarly to complexity, but takes the shortest path through the subroutine graph.\n\nThere will be many more statistics in the near future.";

export const RoutineView = ({
    display,
    isOpen,
    onClose,
}: RoutineViewProps) => {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);

    const { isLoading, object: existing, permissions, setObject: setRoutineVersion } = useObjectFromUrl<RoutineVersion>({
        ...endpointGetRoutineVersion,
        onInvalidUrlParams: ({ build }) => {
            // Throw error if we are not creating a new routine
            if (!build || build !== true) PubSub.get().publish("snack", { messageKey: "InvalidUrlId", severity: "Error" });
        },
        objectType: "RoutineVersion",
    });
    console.log("this is the existing", existing);

    const availableLanguages = useMemo<string[]>(() => (existing?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [existing?.translations]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const { name, description, instructions } = useMemo(() => {
        const { description, instructions, name } = getTranslation(existing, [language]);
        return { name, description, instructions };
    }, [existing, language]);

    const [isBuildOpen, setIsBuildOpen] = useState<boolean>(Boolean(parseSearchParams()?.build));
    const viewGraph = useCallback(() => {
        setSearchParams(setLocation, { build: true });
        setIsBuildOpen(true);
    }, [setLocation]);
    const stopBuild = useCallback(() => {
        setIsBuildOpen(false);
    }, []);

    const handleRunDelete = useCallback((run: RunRoutine) => {
        if (!existing) return;
        setRoutineVersion(setDotNotationValue(existing, "you.runs", existing.you?.runs?.filter(r => r.id !== run.id)) ?? existing);
    }, [existing, setRoutineVersion]);

    const handleRunAdd = useCallback((run: RunRoutine) => {
        if (!existing) return;
        setRoutineVersion(setDotNotationValue(existing, "you.runs", [run, ...(existing.you?.runs ?? [])]) ?? existing);
    }, [existing, setRoutineVersion]);

    const [isAddCommentOpen, setIsAddCommentOpen] = useState(false);
    const openAddCommentDialog = useCallback(() => { setIsAddCommentOpen(true); }, []);
    const closeAddCommentDialog = useCallback(() => { setIsAddCommentOpen(false); }, []);

    const actionData = useObjectActions({
        object: existing,
        objectType: "Routine",
        openAddCommentDialog,
        setLocation,
        setObject: setRoutineVersion,
    });

    // The schema and formik keys for the form
    const formValueMap = useMemo<{ [fieldName: string]: FieldData } | null>(() => {
        if (!existing.inputs || !Array.isArray(existing.inputs)) return null;
        const schemas: { [fieldName: string]: FieldData } = {};
        for (let i = 0; i < existing.inputs?.length; i++) {
            const currInput = existing.inputs[i];
            if (!currInput.standardVersion) continue;
            const currSchema = standardVersionToFieldData({
                description: getTranslation(currInput, getUserLanguages(session), false).description ?? getTranslation(currInput.standardVersion, getUserLanguages(session), false).description,
                fieldName: `inputs-${currInput.id}`,
                helpText: getTranslation(currInput, getUserLanguages(session), false).helpText,
                props: currInput.standardVersion.props,
                name: currInput.name ?? getTranslation(currInput.standardVersion, getUserLanguages(session), false).name ?? "",
                standardType: currInput.standardVersion.standardType,
                yup: currInput.standardVersion.yup,
            });
            if (currSchema) {
                schemas[currSchema.fieldName] = currSchema;
            }
        }
        return schemas;
    }, [existing, session]);
    const formik = useFormik({
        initialValues: Object.entries(formValueMap ?? {}).reduce((acc, [key, value]) => {
            acc[key] = value.props.defaultValue ?? "";
            return acc;
        }, {}),
        enableReinitialize: true,
        onSubmit: noopSubmit,
    });

    const [runComplete] = useLazyFetch<RunRoutineCompleteInput, RunRoutine>(endpointPutRunRoutineComplete);
    const markAsComplete = useCallback(() => {
        if (!existing.id) return;
        fetchLazyWrapper<RunRoutineCompleteInput, RunRoutine>({
            fetch: runComplete,
            inputs: {
                id: existing.id,
                exists: false,
                name: name ?? "Unnamed Routine",
                ...runInputsCreate(formikToRunInputs(formik.values), existing.id),
            },
            successMessage: (data) => ({
                messageKey: "RoutineCompleted",
                buttonKey: "View",
                buttonClicked: () => { openObject(data, setLocation); },
            }),
            onSuccess: (data) => {
                PubSub.get().publish("celebration");
            },
        });
    }, [formik.values, existing, runComplete, setLocation, name]);

    /**
     * Copy current value of input to clipboard
     * @param fieldName Name of input
     */
    const copyInput = useCallback((fieldName: string) => {
        const input = formik.values[fieldName];
        if (input) {
            navigator.clipboard.writeText(input);
            PubSub.get().publish("snack", { messageKey: "CopiedToClipboard", severity: "Success" });
        } else {
            PubSub.get().publish("snack", { messageKey: "InputEmpty", severity: "Error" });
        }
    }, [formik]);

    const initialValues = useMemo(() => routineInitialValues(session, existing), [existing, session]);
    const resourceList = useMemo<ResourceListShape | null | undefined>(() => initialValues.resourceList as ResourceListShape | null | undefined, [initialValues]);
    const tags = useMemo<TagShape[] | null | undefined>(() => (initialValues.root as RoutineShape)?.tags as TagShape[] | null | undefined, [initialValues]);

    return (
        <>
            <TopBar
                display={display}
                onClose={onClose}
                title={firstString(name, t("Routine", { count: 1 }))}
                below={availableLanguages.length > 1 && <SelectLanguageMenu
                    currentLanguage={language}
                    handleCurrent={setLanguage}
                    languages={availableLanguages}
                />}
            />
            <Formik
                enableReinitialize={true}
                initialValues={initialValues}
                onSubmit={noopSubmit}
            >
                {(formik) => <Stack direction="column" spacing={4} sx={{
                    marginLeft: "auto",
                    marginRight: "auto",
                    width: "min(100%, 700px)",
                    padding: 2,
                }}>
                    <BuildView
                        display="dialog"
                        handleCancel={stopBuild}
                        onClose={stopBuild}
                        handleSubmit={noop}
                        isEditing={false}
                        isOpen={isBuildOpen}
                        loading={isLoading}
                        routineVersion={existing as RoutineVersion}
                        translationData={{
                            language,
                            languages: availableLanguages,
                            setLanguage,
                            handleAddLanguage: noop,
                            handleDeleteLanguage: noop,
                        }}
                    />
                    {/* Relationships */}
                    <RelationshipList
                        isEditing={false}
                        objectType={"Routine"}
                    />
                    {/* Resources */}
                    {exists(resourceList) && Array.isArray(resourceList.resources) && resourceList.resources.length > 0 && <ResourceList
                        horizontal
                        list={resourceList as unknown as ResourceListType}
                        canUpdate={false}
                        // eslint-disable-next-line @typescript-eslint/no-empty-function
                        handleUpdate={() => { }}
                        loading={isLoading}
                        parent={{ __typename: "RoutineVersion", id: existing?.id ?? "" }}
                    />}
                    {/* Box with description and instructions */}
                    {(!!description || !!instructions) && <Stack direction="column" spacing={4} sx={containerProps(palette)}>
                        {/* Description */}
                        <TextCollapse
                            title="Description"
                            text={description}
                            loading={isLoading}
                            loadingLines={2}
                        />
                        {/* Instructions */}
                        <TextCollapse
                            title="Instructions"
                            text={instructions}
                            loading={isLoading}
                            loadingLines={4}
                        />
                    </Stack>}
                    {/* Box with inputs, if this is a single-step routine */}
                    {existing.nodes?.length === 0 && existing.nodeLinks?.length === 0 && <Box sx={containerProps(palette)}>
                        <ContentCollapse
                            isOpen={Object.keys(formValueMap ?? {}).length <= 1} // Default to open if there is one or less inputs
                            title="Inputs"
                        >
                            {Object.values(formValueMap ?? {}).map((fieldData: FieldData, index: number) => (
                                <GeneratedInputComponentWithLabel
                                    key={fieldData.fieldName}
                                    copyInput={copyInput}
                                    disabled={false}
                                    fieldData={fieldData}
                                    index={index}
                                    textPrimary={palette.background.textPrimary}
                                    onUpload={() => { }}
                                />
                            ))}
                            {getCurrentUser(session).id && <Button
                                startIcon={<SuccessIcon />}
                                fullWidth
                                onClick={markAsComplete}
                                color="secondary"
                                variant="outlined"
                                sx={{ marginTop: 2 }}
                            >{t("MarkAsComplete")}</Button>}
                        </ContentCollapse>
                    </Box>}
                    {/* "View Graph" button if this is a multi-step routine */}
                    {
                        existing?.nodes?.length ? <Box>
                            <Title
                                title={t("ThisIsMultiStep")}
                                help={"ThisIsMultiStepHelp"}
                                variant="subheader"
                            />
                            <Button
                                startIcon={<RoutineIcon />}
                                fullWidth
                                onClick={viewGraph}
                                color="secondary"
                                variant="outlined"
                            >{t("ViewGraph")}</Button>
                        </Box> : null
                    }
                    {/* Tags */}
                    {exists(tags) && tags.length > 0 && <TagList
                        maxCharacters={30}
                        parentId={existing?.id ?? ""}
                        tags={tags as Tag[]}
                        sx={{ marginTop: 4 }}
                    />}
                    <Box>
                        {/* Date and version labels */}
                        <Stack direction="row" spacing={1} mt={2} mb={1}>
                            {/* Date created */}
                            <DateDisplay
                                loading={isLoading}
                                showIcon={true}
                                timestamp={existing?.created_at}
                            />
                            <VersionDisplay
                                currentVersion={existing}
                                prefix={" - v"}
                                versions={existing?.root?.versions}
                            />
                        </Stack>
                        {/* Votes, reports, and other basic stats */}
                        <StatsCompact
                            handleObjectUpdate={() => { }}
                            object={existing}
                        />
                        {/* Action buttons */}
                        <ObjectActionsRow
                            actionData={actionData}
                            exclude={[ObjectAction.Edit, ObjectAction.VoteDown, ObjectAction.VoteUp]} // Handled elsewhere
                            object={existing}
                        />
                    </Box>
                    <Divider />
                    <CommentContainer
                        forceAddCommentOpen={isAddCommentOpen}
                        language={language}
                        objectId={existing?.id ?? ""}
                        objectType={CommentFor.RoutineVersion}
                        onAddCommentClose={closeAddCommentDialog}
                    />
                </Stack>}
            </Formik>
            {/* Edit button (if canUpdate) and run button, positioned at bottom corner of screen */}
            <SideActionsButtons
                // Treat as a dialog when build view is open
                display={isBuildOpen ? "dialog" : display}
            >
                {/* Edit button */}
                {permissions.canUpdate ? (
                    <IconButton aria-label={t("UpdateRoutine")} onClick={() => { actionData.onActionStart(ObjectAction.Edit); }} sx={{ background: palette.secondary.main }}>
                        <EditIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </IconButton>
                ) : null}
                {/* Play button fixed to bottom of screen, to start routine (if multi-step) */}
                {existing?.nodes?.length ? <RunButton
                    canUpdate={permissions.canUpdate}
                    handleRunAdd={handleRunAdd as any}
                    handleRunDelete={handleRunDelete as any}
                    isBuildGraphOpen={isBuildOpen}
                    isEditing={false}
                    runnableObject={existing}
                /> : null}
            </SideActionsButtons>
        </>
    );
};
