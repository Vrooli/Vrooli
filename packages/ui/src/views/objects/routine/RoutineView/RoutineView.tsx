import { CommentFor, ParseSearchParamsResult, ResourceListShape, ResourceList as ResourceListType, RoutineShape, RoutineType, RoutineVersion, RunRoutine, RunRoutineCreateInput, RunStatus, RunnableRoutineVersion, Tag, TagShape, endpointGetRoutineVersion, endpointPostRunRoutine, exists, generateInitialValues, generateYupSchema, getTranslation, noop, noopSubmit, parseConfigCallData, parseSchemaInput, parseSchemaOutput, parseSearchParams, runInputsUpdate, uuid } from "@local/shared";
import { Box, Button, Divider, Stack, Typography, styled, useTheme } from "@mui/material";
import { fetchLazyWrapper } from "api/fetchWrapper";
import { RunButton } from "components/buttons/RunButton/RunButton";
import { SideActionsButtons } from "components/buttons/SideActionsButtons/SideActionsButtons";
import { CommentContainer } from "components/containers/CommentContainer/CommentContainer";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { TextCollapse } from "components/containers/TextCollapse/TextCollapse";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { ObjectActionsRow } from "components/lists/ObjectActionsRow/ObjectActionsRow";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { TagList } from "components/lists/TagList/TagList";
import { ResourceList } from "components/lists/resource";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { StatsCompact } from "components/text/StatsCompact/StatsCompact";
import { VersionDisplay } from "components/text/VersionDisplay/VersionDisplay";
import { SessionContext } from "contexts";
import { Formik } from "formik";
import { useObjectActions } from "hooks/objectActions";
import { useErrorPopover } from "hooks/useErrorPopover";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useManagedObject } from "hooks/useManagedObject";
import { EditIcon, SuccessIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { setSearchParams, useLocation } from "route";
import { FormSection, SideActionsButton } from "styles";
import { FormErrors } from "types";
import { ObjectAction } from "utils/actions/objectActions";
import { getCurrentUser } from "utils/authentication/session";
import { getDisplay } from "utils/display/listTools";
import { firstString } from "utils/display/stringTools";
import { getLanguageSubtag, getPreferredLanguage, getUserLanguages } from "utils/display/translationTools";
import { openObject } from "utils/navigation/openObject";
import { PubSub } from "utils/pubsub";
import { routineTypes } from "utils/search/schemas/routine";
import { RoutineApiForm, RoutineDataConverterForm, RoutineDataForm, RoutineGenerateForm, RoutineInformationalForm, RoutineMultiStepForm, RoutineSmartContractForm } from "../RoutineTypeForms/RoutineTypeForms";
import { routineInitialValues } from "../RoutineUpsert/RoutineUpsert";
import { RoutineViewProps } from "../types";

/**
 * If no existing routine found, consider it an error if we are not creating a new routine
 */
function handleInvalidUrlParams(params: ParseSearchParamsResult) {
    if (!params.build || params.build !== true) {
        PubSub.get().publish("snack", { messageKey: "InvalidUrlId", severity: "Error" });
    }
}

type RoutineTypeViewProps = {
    errors: FormErrors;
    isComplete: boolean | undefined;
    isLoading: boolean;
    onSetSubmitting: ((isSubmitting: boolean) => unknown);
    onSubmit: (() => unknown);
    routineType: RoutineType | undefined;
    routineTypeComponents: JSX.Element | null;
}

const IncompleteWarningLabel = styled(Typography)(({ theme }) => ({
    color: theme.palette.background.textSecondary,
    fontStyle: "italic",
}));

const formSubmitButtonStyle = { marginTop: 2 } as const;

function RoutineTypeView({
    errors,
    isComplete,
    isLoading,
    onSetSubmitting,
    onSubmit,
    routineType,
    routineTypeComponents,
}: RoutineTypeViewProps) {
    const session = useContext(SessionContext);
    const { t } = useTranslation();

    const { openPopover, Popover } = useErrorPopover({ errors, onSetSubmitting });

    const hasErrors = useMemo(() => Object.values(errors ?? {}).some((value) => exists(value)), [errors]);
    const isSubmitDisabled = useMemo(() => isLoading || hasErrors, [hasErrors, isLoading]);

    const handleSubmit = useCallback(function handleSubmitCallback(event: React.MouseEvent | React.TouchEvent) {
        if (hasErrors) {
            openPopover(event);
        } else {
            onSubmit();
        }
    }, [hasErrors, onSubmit, openPopover]);

    const routineTypeOption = useMemo(function routineTypeMemo() {
        return routineTypes.find((option) => option.type === routineType);
    }, [routineType]);

    return (
        <>
            <Popover />
            <FormSection>
                <ContentCollapse
                    title={`Type: ${routineTypeOption?.label ?? "Unknown"}`}
                    helpText={routineTypeOption?.description ?? ""}
                >
                    {/* warning label when routine is marked as incomplete */}
                    {isComplete !== true && <IncompleteWarningLabel variant="caption">
                        This routine is marked as incomplete. It may change in the future.
                    </IncompleteWarningLabel>}
                    <br />
                    <br />
                    {routineTypeComponents}
                    {routineType === RoutineType.Informational && getCurrentUser(session).id && <Button
                        disabled={isSubmitDisabled}
                        startIcon={<SuccessIcon />}
                        fullWidth
                        onClick={handleSubmit}
                        color="secondary"
                        variant="outlined"
                        sx={formSubmitButtonStyle}
                    >{t("MarkAsComplete")}</Button>}
                </ContentCollapse>
            </FormSection>
        </>
    );
}

const statsHelpText =
    "Statistics are calculated to measure various aspects of a routine. \n\n**Complexity** is a rough measure of the maximum amount of effort it takes to complete a routine. This takes into account the number of inputs, the structure of its subroutine graph, and the complexity of every subroutine.\n\n**Simplicity** is calculated similarly to complexity, but takes the shortest path through the subroutine graph.\n\nThere will be many more statistics in the near future.";

const excludedActionRowActions = [ObjectAction.Edit, ObjectAction.VoteDown, ObjectAction.VoteUp] as const;
const basicInfoStackStyle = {
    marginLeft: "auto",
    marginRight: "auto",
    width: "min(100%, 700px)",
    padding: 2,
} as const;
const tagListStyle = { marginTop: 4 } as const;

export function RoutineView({
    display,
    isOpen,
    onClose,
}: RoutineViewProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);

    const { isLoading, object: existing, permissions, setObject: setRoutineVersion } = useManagedObject<RoutineVersion>({
        ...endpointGetRoutineVersion,
        onInvalidUrlParams: handleInvalidUrlParams,
        objectType: "RoutineVersion",
    });

    const availableLanguages = useMemo<string[]>(() => (existing?.translations?.map(t => getLanguageSubtag(t.language)) ?? []), [existing?.translations]);
    useEffect(() => {
        if (availableLanguages.length === 0) return;
        setLanguage(getPreferredLanguage(availableLanguages, getUserLanguages(session)));
    }, [availableLanguages, setLanguage, session]);

    const {
        description,
        instructions,
        name,
    } = useMemo(() => {
        const { description, instructions, name } = getTranslation(existing, [language]);
        return { name, description, instructions };
    }, [existing, language]);

    const [isGraphOpen, setIsGraphOpen] = useState<boolean>(Boolean(parseSearchParams()?.build));
    const openGraph = useCallback(() => {
        setSearchParams(setLocation, { build: true });
        setIsGraphOpen(true);
    }, [setLocation]);
    const closeGraph = useCallback(() => { setIsGraphOpen(false); }, []);

    const [isAddCommentOpen, setIsAddCommentOpen] = useState(false);
    const openAddCommentDialog = useCallback(() => { setIsAddCommentOpen(true); }, []);
    const closeAddCommentDialog = useCallback(() => { setIsAddCommentOpen(false); }, []);

    const actionData = useObjectActions({
        object: existing,
        objectType: "RoutineVersion",
        openAddCommentDialog,
        setLocation,
        setObject: setRoutineVersion,
    });
    const handleEditStart = useCallback(function handleEditStartCallback() {
        actionData.onActionStart(ObjectAction.Edit);
    }, [actionData]);

    const initialValues = useMemo(() => routineInitialValues(session, existing), [existing, session]);
    const resourceList = useMemo<ResourceListShape | null | undefined>(() => initialValues.resourceList as ResourceListShape | null | undefined, [initialValues]);
    const tags = useMemo<TagShape[] | null | undefined>(() => (initialValues.root as RoutineShape)?.tags as TagShape[] | null | undefined, [initialValues]);

    const translationData = useMemo(() => ({
        language,
        languages: availableLanguages,
        setLanguage,
        handleAddLanguage: noop,
        handleDeleteLanguage: noop,
    }), [availableLanguages, language, setLanguage]);

    const configCallData = useMemo(function configCallDataMemo() {
        return parseConfigCallData(existing.configCallData, existing.routineType, console);
    }, [existing.configCallData, existing.routineType]);
    const schemaInput = useMemo(function schemaInputMemo() {
        return parseSchemaInput(existing.configFormInput, existing.routineType, console);
    }, [existing.configFormInput, existing.routineType]);
    const schemaOutput = useMemo(function schemaOutputMemo() {
        return parseSchemaOutput(existing.configFormOutput, existing.routineType, console);
    }, [existing.configFormOutput, existing.routineType]);

    const routineTypeBaseProps = useMemo(function routineTypeBasePropsMemo() {
        return {
            configCallData,
            disabled: true,
            display: "view",
            handleGenerateOutputs: noop,
            isGeneratingOutputs: false,
            onConfigCallDataChange: noop,
            onSchemaInputChange: noop,
            onSchemaOutputChange: noop,
            schemaInput,
            schemaOutput,
        } as const;
    }, [configCallData, schemaInput, schemaOutput]);

    // Type-specific components
    const routineTypeComponents = useMemo(function routineTypeComponentsMemo() {
        switch (existing.routineType) {
            case RoutineType.Api:
                return <RoutineApiForm {...routineTypeBaseProps} />;
            case RoutineType.Code:
                return <RoutineDataConverterForm {...routineTypeBaseProps} />;
            case RoutineType.Data:
                return <RoutineDataForm {...routineTypeBaseProps} />;
            case RoutineType.Generate:
                return <RoutineGenerateForm {...routineTypeBaseProps} />;
            case RoutineType.Informational:
                return <RoutineInformationalForm {...routineTypeBaseProps} />;
            case RoutineType.MultiStep:
                return <RoutineMultiStepForm
                    {...routineTypeBaseProps}
                    isGraphOpen={isGraphOpen}
                    handleGraphClose={closeGraph}
                    handleGraphOpen={openGraph}
                    handleGraphSubmit={noop}
                    nodeLinks={existing.nodeLinks}
                    nodes={existing.nodes}
                    routineId={existing.id}
                    translations={existing.translations}
                    translationData={translationData}
                />;
            case RoutineType.SmartContract:
                return <RoutineSmartContractForm {...routineTypeBaseProps} />;
            default:
                return null;
        }
    }, [closeGraph, existing.id, existing.nodeLinks, existing.nodes, existing.routineType, existing.translations, isGraphOpen, openGraph, routineTypeBaseProps, translationData]);

    const inputInitialValues = useMemo(function inputInitialValuesMemo() {
        return generateInitialValues(schemaInput.elements, "input");
    }, [schemaInput]);
    const inputValidationSchema = useMemo(function inputValidationSchemaMemo() {
        return schemaInput ? generateYupSchema(schemaInput) : undefined;
    }, [schemaInput]);

    const [runCreate] = useLazyFetch<RunRoutineCreateInput, RunRoutine>(endpointPostRunRoutine);
    const markAsComplete = useCallback((values: unknown) => {
        if (!existing.id) return;
        const runRoutineId = uuid();
        const { inputsCreate } = runInputsUpdate({
            existingIO: [],
            formData: values as object,
            logger: console,
            routineIO: existing.inputs ?? [],
            runRoutineId,
        });
        fetchLazyWrapper<RunRoutineCreateInput, RunRoutine>({
            fetch: runCreate,
            inputs: {
                id: runRoutineId,
                isPrivate: true,
                completedComplexity: existing.complexity ?? 1,
                name: getDisplay(existing, [language]).title ?? "Untitled",
                status: RunStatus.Completed,
                inputsCreate,
                routineVersionConnect: existing.id,
            },
            successMessage: (data) => ({
                messageKey: "RoutineCompleted",
                buttonKey: "View",
                buttonClicked: () => { openObject(data, setLocation); },
            }),
            onSuccess: () => {
                PubSub.get().publish("celebration");
            },
        });
    }, [existing, language, runCreate, setLocation]);

    const resourceListParent = useMemo(function resourceListParent() {
        return { __typename: "RoutineVersion", id: existing?.id ?? "" } as const;
    }, [existing?.id]);

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
                {() => <Stack direction="column" spacing={4} sx={basicInfoStackStyle}>
                    <RelationshipList
                        isEditing={false}
                        objectType={"Routine"}
                    />
                    {exists(resourceList) && Array.isArray(resourceList.resources) && resourceList.resources.length > 0 && <ResourceList
                        horizontal
                        list={resourceList as unknown as ResourceListType}
                        canUpdate={false}
                        handleUpdate={noop}
                        loading={isLoading}
                        parent={resourceListParent}
                    />}
                    {(!!description || !!instructions) && <FormSection>
                        <TextCollapse
                            title="Description"
                            text={description}
                            loading={isLoading}
                            loadingLines={2}
                        />
                        <TextCollapse
                            title="Instructions"
                            text={instructions}
                            loading={isLoading}
                            loadingLines={4}
                        />
                    </FormSection>}
                    {Boolean(routineTypeComponents) && (
                        <Formik
                            enableReinitialize={true}
                            initialValues={inputInitialValues}
                            onSubmit={markAsComplete}
                            validationSchema={inputValidationSchema}
                        >
                            {(formik) => (
                                <RoutineTypeView
                                    errors={formik.errors}
                                    isComplete={existing.isComplete}
                                    isLoading={isLoading || formik.isSubmitting || formik.isValidating}
                                    onSetSubmitting={formik.setSubmitting}
                                    onSubmit={formik.submitForm}
                                    routineType={existing.routineType}
                                    routineTypeComponents={routineTypeComponents}
                                />
                            )}
                        </Formik>
                    )}
                    {/* Tags */}
                    {exists(tags) && tags.length > 0 && <TagList
                        maxCharacters={30}
                        parentId={existing?.id ?? ""}
                        tags={tags as Tag[]}
                        sx={tagListStyle}
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
                            exclude={excludedActionRowActions} // Handled elsewhere
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
                display={isGraphOpen ? "dialog" : display}
            >
                {/* Edit button */}
                {permissions.canUpdate ? (
                    <SideActionsButton aria-label={t("UpdateRoutine")} onClick={handleEditStart}>
                        <EditIcon fill={palette.secondary.contrastText} width='36px' height='36px' />
                    </SideActionsButton>
                ) : null}
                {/* Play button fixed to bottom of screen, to start routine (if multi-step) */}
                {Boolean(existing) && existing?.routineType !== RoutineType.Informational ? <RunButton
                    isEditing={false}
                    objectType="RoutineVersion"
                    runnableObject={existing as RunnableRoutineVersion}
                /> : null}
            </SideActionsButtons>
        </>
    );
}
