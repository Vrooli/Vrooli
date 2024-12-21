import { CommentFor, FindByIdInput, FormSchema, LINKS, ResourceListShape, ResourceList as ResourceListType, RoutineShape, RoutineSingleStepViewSearchParams, RoutineType, RoutineVersion, RunIOManager, RunProject, RunRoutine, RunStatus, RunnableRoutineVersion, Tag, TagShape, UrlTools, base36ToUuid, endpointGetRoutineVersion, endpointGetRunRoutine, exists, generateInitialValues, generateRoutineInitialValues, generateYupSchema, getTranslation, noop, noopSubmit, parseConfigCallData, parseSchemaInput, parseSchemaOutput, uuid, uuidToBase36, uuidValidate } from "@local/shared";
import { Box, Divider, Stack, Typography, styled, useTheme } from "@mui/material";
import { RunButton } from "components/buttons/RunButton/RunButton";
import { SideActionsButtons } from "components/buttons/SideActionsButtons/SideActionsButtons";
import { CommentContainer } from "components/containers/CommentContainer/CommentContainer";
import { ContentCollapse } from "components/containers/ContentCollapse/ContentCollapse";
import { TextCollapse } from "components/containers/TextCollapse/TextCollapse";
import { SelectLanguageMenu } from "components/dialogs/SelectLanguageMenu/SelectLanguageMenu";
import { ObjectActionsRow } from "components/lists/ObjectActionsRow/ObjectActionsRow";
import { RelationshipList } from "components/lists/RelationshipList/RelationshipList";
import { ResourceList } from "components/lists/ResourceList/ResourceList";
import { TagList } from "components/lists/TagList/TagList";
import { TopBar } from "components/navigation/TopBar/TopBar";
import { DateDisplay } from "components/text/DateDisplay/DateDisplay";
import { StatsCompact } from "components/text/StatsCompact/StatsCompact";
import { VersionDisplay } from "components/text/VersionDisplay/VersionDisplay";
import { SessionContext } from "contexts";
import { Formik, useFormikContext } from "formik";
import { useObjectActions } from "hooks/objectActions";
import { useSocketRun, useUpsertRunRoutine } from "hooks/runs";
import { useErrorPopover } from "hooks/useErrorPopover";
import { useLazyFetch } from "hooks/useLazyFetch";
import { useManagedObject } from "hooks/useManagedObject";
import { EditIcon } from "icons";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { addSearchParams, useLocation } from "route";
import { FormSection, ScrollBox, SideActionsButton } from "styles";
import { FormErrors, PartialWithType } from "types";
import { ObjectAction } from "utils/actions/objectActions";
import { getCurrentUser } from "utils/authentication/session";
import { ELEMENT_IDS } from "utils/consts";
import { getDisplay } from "utils/display/listTools";
import { firstString } from "utils/display/stringTools";
import { getLanguageSubtag, getPreferredLanguage, getUserLanguages } from "utils/display/translationTools";
import { openObject } from "utils/navigation/openObject";
import { PubSub } from "utils/pubsub";
import { routineTypes } from "utils/search/schemas/routine";
import { routineSingleStepInitialValues } from "../RoutineSingleStepUpsert/RoutineSingleStepUpsert";
import { RoutineApiForm, RoutineDataConverterForm, RoutineDataForm, RoutineFormPropsBase, RoutineGenerateForm, RoutineInformationalForm, RoutineSmartContractForm } from "../RoutineTypeForms/RoutineTypeForms";
import { RoutineSingleStepViewProps } from "../types";

/**
 * If no existing routine found, consider it an error if we are not creating a new routine
 */
function handleInvalidUrlParams() {
    PubSub.get().publish("snack", { messageKey: "InvalidUrlId", severity: "Error" });
}

type RoutineSingleStepTypeViewProps = {
    existing: PartialWithType<RoutineVersion>;
    isGetRoutineLoading: boolean;
    schemaInput: FormSchema;
    schemaOutput: FormSchema;
}

const IncompleteWarningLabel = styled(Typography)(({ theme }) => ({
    color: theme.palette.background.textSecondary,
    fontStyle: "italic",
}));

function RoutineSingleStepTypeView({
    existing,
    isGetRoutineLoading,
    schemaInput,
    schemaOutput,
}: RoutineSingleStepTypeViewProps) {
    const [, setLocation] = useLocation();
    const session = useContext(SessionContext);
    const { errors, isSubmitting, isValidating, resetForm, setSubmitting, setValues, values } = useFormikContext<FormErrors>();

    const routineTypeOption = useMemo(function routineTypeMemo() {
        return routineTypes.find((option) => option.type === existing.routineType);
    }, [existing.routineType]);

    const configCallData = useMemo(function configCallDataMemo() {
        return parseConfigCallData(existing.configCallData, existing.routineType, console);
    }, [existing.configCallData, existing.routineType]);

    const { createRun, updateRun, isCreatingRunRoutine, isUpdatingRunRoutine } = useUpsertRunRoutine();
    const [run, setRun] = useState<RunRoutine | null>(null);
    useEffect(function updateUrlOnRunChange() {
        if (run) {
            addSearchParams(setLocation, { runId: uuidToBase36(run.id) } as RoutineSingleStepViewSearchParams);
        }
    }, [run, setLocation]);

    const onRunChange = useCallback(function onRunChangeCallback(newRun: RunRoutine | null) {
        setRun(newRun);
        // Update formik values with run inputs
        if (!newRun) {
            resetForm();
            return;
        }
        const values = generateRoutineInitialValues({
            configFormInput: parseSchemaInput(existing.configFormInput, existing.routineType, console),
            configFormOutput: parseSchemaOutput(existing.configFormOutput, existing.routineType, console),
            logger: console,
            runInputs: newRun.inputs,
            runOutputs: newRun.outputs,
        });
        setValues(values as FormErrors);
    }, [existing.configFormInput, existing.configFormOutput, existing.routineType, resetForm, setValues]);

    const [getRun, { data: runData, loading: isLoadingGetRun }] = useLazyFetch<FindByIdInput, RunRoutine>(endpointGetRunRoutine);
    useEffect(function fetchRunFromUrl() {
        const searchParams = UrlTools.parseSearchParams(LINKS.RoutineSingleStep);
        if (searchParams.runId) {
            const runId = base36ToUuid(searchParams.runId);
            if (uuidValidate(runId)) {
                getRun({ id: runId });
            }
        }
    }, [getRun]);
    useEffect(function updateRunFromUrl() {
        if (runData) {
            onRunChange(runData);
        }
    }, [onRunChange, runData]);

    const getFormValues = useCallback(function getFormValuesCallback() {
        return values;
    }, [values]);
    const {
        handleRunSubroutine,
        isGeneratingOutputs,
    } = useSocketRun({
        getFormValues,
        handleRunUpdate: onRunChange as (run: RunRoutine | RunProject) => unknown,
        run,
        runnableObject: existing as RunnableRoutineVersion,
    });

    const { openPopover, Popover } = useErrorPopover({ errors, onSetSubmitting: setSubmitting });
    const hasFormErrors = useMemo(() => Object.values(errors ?? {}).some((value) => exists(value)), [errors]);
    const handleCompleteStep = useCallback(function handleCompleteStepCallback(_event: React.MouseEvent | React.TouchEvent) {
        const isLoggedIn = uuidValidate(getCurrentUser(session).id);
        if (!isLoggedIn) {
            PubSub.get().publish("proDialog");
            return;
        }
        if (!uuidValidate(existing.id)) {
            console.error("Routine ID is not a valid UUID");
            return;
        }
        // Create run if it doesn't exist
        if (!run) {
            const runRoutineId = uuid();
            const { inputsCreate } = new RunIOManager(console).runInputsUpdate({
                existingIO: [],
                formData: values as object,
                routineIO: existing.inputs ?? [],
                runRoutineId,
            });
            createRun({
                completedComplexity: existing.complexity ?? 1,
                id: runRoutineId,
                inputsCreate,
                objectId: existing.id as string,
                objectName: getDisplay(existing).title,
                status: hasFormErrors ? RunStatus.InProgress : RunStatus.Completed,
                onSuccess: function handleSuccess(data) {
                    setRun(data);
                    PubSub.get().publish("snack", {
                        messageKey: "RunCompleted",
                        buttonKey: "View",
                        buttonClicked: () => { openObject(data, setLocation); },
                        severity: "Success",
                    });
                    PubSub.get().publish("celebration");
                },
            });
        }
        // Update run if it does exist
        else {
            const runRoutineId = run.id;
            const { inputsCreate, inputsUpdate, inputsDelete } = new RunIOManager(console).runInputsUpdate({
                existingIO: run.inputs,
                formData: values as object,
                routineIO: existing.inputs ?? [],
                runRoutineId,
            });
            console.log("completing step with inputs", inputsCreate, inputsUpdate, inputsDelete);
            updateRun({
                id: runRoutineId,
                inputsCreate,
                inputsUpdate,
                inputsDelete,
                status: RunStatus.Completed,
                onSuccess: function handleSuccess(data) {
                    setRun(data);
                    PubSub.get().publish("snack", {
                        messageKey: "RunUpdated",
                        buttonKey: "View",
                        buttonClicked: () => { openObject(data, setLocation); },
                        severity: "Success",
                    });
                    PubSub.get().publish("celebration");
                },
            });
        }
    }, [createRun, existing, hasFormErrors, run, session, setLocation, updateRun, values]);
    const handleRunStep = useCallback(function handleRunStepCallback() {
        if (!run) {
            // Create the run
            createRun({
                completedComplexity: existing.complexity ?? 1,
                objectId: existing.id as string,
                objectName: getDisplay(existing).title,
                onSuccess: function handleSuccess(data) {
                    setRun(data);
                    handleRunSubroutine();
                },
            });
        } else {
            // We have a run, proceed to run the subroutine
            handleRunSubroutine();
        }
    }, [createRun, existing, handleRunSubroutine, run]);
    const handleClearRun = useCallback(function handleClearRunCallback() {
        setRun(null);
        resetForm();
    }, [resetForm]);

    // Type-specific components
    const routineTypeComponents = useMemo(function routineTypeComponentsMemo() {
        if (!existing) return null;

        const isLoading = isCreatingRunRoutine || isUpdatingRunRoutine || isGetRoutineLoading || isLoadingGetRun || isGeneratingOutputs || isSubmitting || isValidating;
        const isLoggedIn = uuidValidate(getCurrentUser(session).id);
        const isDeleted = existing.isDeleted ?? existing.root?.isDeleted ?? false;
        const canRun = existing.you?.canRun ?? false;
        const hasErrors = hasFormErrors;
        const routineTypeBaseProps: RoutineFormPropsBase = {
            configCallData,
            disabled: false,
            display: "view",
            handleCompleteStep,
            handleClearRun,
            handleRunStep,
            hasErrors,
            isCompleteStepDisabled: isLoading || !isLoggedIn || isDeleted || !canRun || hasFormErrors,
            isPartOfMultiStepRoutine: false,
            isRunStepDisabled: isLoading || !isLoggedIn || isDeleted || !canRun || hasFormErrors,
            isRunningStep: isGeneratingOutputs,
            onConfigCallDataChange: noop, // Only used in edit mode
            onRunChange,
            onSchemaInputChange: noop, // Only used in edit mode
            onSchemaOutputChange: noop, // Only used in edit mode
            schemaInput,
            schemaOutput,
            routineId: existing.id as string,
            routineName: getDisplay(existing).title,
            run,
        };

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
            case RoutineType.SmartContract:
                return <RoutineSmartContractForm {...routineTypeBaseProps} />;
            default:
                return null;
        }
    }, [configCallData, existing, handleClearRun, handleCompleteStep, handleRunStep, hasFormErrors, isCreatingRunRoutine, isGeneratingOutputs, isGetRoutineLoading, isLoadingGetRun, isSubmitting, isUpdatingRunRoutine, isValidating, onRunChange, run, schemaInput, schemaOutput, session]);

    return (
        <>
            <Popover />
            <FormSection id={ELEMENT_IDS.RoutineTypeForm}>
                <ContentCollapse
                    title={`Type: ${routineTypeOption?.label ?? "Unknown"}`}
                    helpText={routineTypeOption?.description ?? ""}
                >
                    {/* warning label when routine is marked as incomplete */}
                    {existing.isComplete !== true && <IncompleteWarningLabel variant="caption">
                        This routine is marked as incomplete. It may change in the future.
                    </IncompleteWarningLabel>}
                    <br />
                    <br />
                    {routineTypeComponents}
                </ContentCollapse>
            </FormSection>
        </>
    );
}

const statsHelpText =
    "Statistics are calculated to measure various aspects of a routine. \n\n**Complexity** is a rough measure of the maximum amount of effort it takes to complete a routine. This takes into account the number of inputs, the structure of its subroutine graph, and the complexity of every subroutine.\n\n**Simplicity** is calculated similarly to complexity, but takes the shortest path through the subroutine graph.\n\nThere will be many more statistics in the near future.";

const excludedActionRowActions = [ObjectAction.Comment, ObjectAction.Edit, ObjectAction.VoteDown, ObjectAction.VoteUp] as const;
const basicInfoStackStyle = {
    marginLeft: "auto",
    marginRight: "auto",
    width: "min(100%, 700px)",
    padding: 2,
} as const;
const tagListStyle = { marginTop: 4 } as const;

export function RoutineSingleStepView({
    display,
    onClose,
}: RoutineSingleStepViewProps) {
    const session = useContext(SessionContext);
    const { palette } = useTheme();
    const [, setLocation] = useLocation();
    const { t } = useTranslation();
    const [language, setLanguage] = useState<string>(getUserLanguages(session)[0]);

    const { isLoading: isGetRoutineLoading, object: existing, permissions, setObject: setRoutineVersion } = useManagedObject<RoutineVersion>({
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

    const [isGraphOpen, setIsGraphOpen] = useState(false);
    const openGraph = useCallback(() => {
        setIsGraphOpen(true);
    }, []);
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

    const initialValues = useMemo(() => routineSingleStepInitialValues(session, existing), [existing, session]);
    const resourceList = useMemo<ResourceListShape | null | undefined>(() => initialValues.resourceList as ResourceListShape | null | undefined, [initialValues]);
    const tags = useMemo<TagShape[] | null | undefined>(() => (initialValues.root as RoutineShape)?.tags as TagShape[] | null | undefined, [initialValues]);

    const schemaInput = useMemo(function schemaInputMemo() {
        return parseSchemaInput(existing.configFormInput, existing.routineType, console);
    }, [existing.configFormInput, existing.routineType]);
    const schemaOutput = useMemo(function schemaOutputMemo() {
        return parseSchemaOutput(existing.configFormOutput, existing.routineType, console);
    }, [existing.configFormOutput, existing.routineType]);

    const inputInitialValues = useMemo(function inputInitialValuesMemo() {
        return generateInitialValues(schemaInput.elements, "input");
    }, [schemaInput]);
    const inputValidationSchema = useMemo(function inputValidationSchemaMemo() {
        return schemaInput ? generateYupSchema(schemaInput, "input") : undefined;
    }, [schemaInput]);

    const resourceListParent = useMemo(function resourceListParent() {
        return { __typename: "RoutineVersion", id: existing?.id ?? "" } as const;
    }, [existing?.id]);

    return (
        <ScrollBox>
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
                        id={ELEMENT_IDS.ResourceCards}
                        list={resourceList as unknown as ResourceListType}
                        canUpdate={false}
                        handleUpdate={noop}
                        loading={isGetRoutineLoading}
                        parent={resourceListParent}
                    />}
                    {(!!description || !!instructions) && <FormSection>
                        <TextCollapse
                            title="Description"
                            text={description}
                            loading={isGetRoutineLoading}
                            loadingLines={2}
                        />
                        <TextCollapse
                            title="Instructions"
                            text={instructions}
                            loading={isGetRoutineLoading}
                            loadingLines={4}
                        />
                    </FormSection>}
                    {existing?.routineType && Object.values(RoutineType).includes(existing.routineType) && (
                        <Formik
                            enableReinitialize={true}
                            initialValues={inputInitialValues}
                            onSubmit={noop}
                            validationSchema={inputValidationSchema}
                        >
                            <RoutineSingleStepTypeView
                                existing={existing}
                                isGetRoutineLoading={isGetRoutineLoading}
                                schemaInput={schemaInput}
                                schemaOutput={schemaOutput}
                            />
                        </Formik>
                    )}
                    {/* Tags */}
                    {exists(tags) && tags.length > 0 && <TagList
                        maxCharacters={30}
                        parentId={existing?.id ?? ""}
                        tags={tags as Tag[]}
                        sx={tagListStyle}
                    />}
                    <Box display="flex" flexDirection="column" gap={1}>
                        <Divider />
                        <Box display="flex" flexDirection="row" justifyContent="space-between" alignItems="center">
                            <Box display="flex" flexDirection="row" alignItems="center" gap={1}>
                                <DateDisplay
                                    loading={isGetRoutineLoading}
                                    showIcon={true}
                                    timestamp={existing?.created_at}
                                />
                                <VersionDisplay
                                    currentVersion={existing}
                                    prefix={" - v"}
                                    versions={existing?.root?.versions}
                                />
                            </Box>
                            <StatsCompact
                                handleObjectUpdate={() => { }}
                                object={existing}
                            />
                        </Box>
                        <Divider />
                        <ObjectActionsRow
                            actionData={actionData}
                            exclude={excludedActionRowActions} // Handled elsewhere
                            object={existing}
                        />
                    </Box>
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
        </ScrollBox>
    );
}
