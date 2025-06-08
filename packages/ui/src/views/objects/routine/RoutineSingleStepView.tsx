import Box from "@mui/material/Box";
import Divider from "@mui/material/Divider";
import IconButton from "@mui/material/IconButton";
import Stack from "@mui/material/Stack";
import Typography from "@mui/material/Typography";
import { styled, useTheme } from "@mui/material/styles";
import { CommentFor, FormBuilder, LINKS, ResourceSubType, RunStatus, UrlTools, endpointsResource, endpointsRun, exists, generatePK, getTranslation, noop, noopSubmit, validatePK, type FindByIdInput, type ResourceListShape, type ResourceList as ResourceListType, type Resource, type RoutineSingleStepViewSearchParams, type ResourceVersion, type RunRoutine, type Tag, type TagShape } from "@vrooli/shared";
import { RoutineVersionConfig } from "@vrooli/shared/shape/configs/routine.js";
import { Formik, useFormikContext } from "formik";

// Mapping from old RoutineType enum to new ResourceSubType values
const RoutineType = {
    Action: ResourceSubType.RoutineInternalAction,
    Api: ResourceSubType.RoutineApi,
    Code: ResourceSubType.RoutineCode,
    Data: ResourceSubType.RoutineData,
    Generate: ResourceSubType.RoutineGenerate,
    Informational: ResourceSubType.RoutineInformational,
    SmartContract: ResourceSubType.RoutineSmartContract,
} as const;
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { RunButton } from "../../../components/buttons/RunButton.js";
import { SideActionsButtons } from "../../../components/buttons/SideActionsButtons.js";
import { CommentContainer } from "../../../components/containers/CommentContainer.js";
import { ContentCollapse } from "../../../components/containers/ContentCollapse.js";
import { TextCollapse } from "../../../components/containers/TextCollapse.js";
import { SelectLanguageMenu } from "../../../components/dialogs/SelectLanguageMenu/SelectLanguageMenu.js";
import { ObjectActionsRow } from "../../../components/lists/ObjectActionsRow/ObjectActionsRow.js";
import { RelationshipList } from "../../../components/lists/RelationshipList/RelationshipList.js";
import { ResourceList } from "../../../components/lists/ResourceList/ResourceList.js";
import { TagList } from "../../../components/lists/TagList/TagList.js";
import { TopBar } from "../../../components/navigation/TopBar.js";
import { DateDisplay } from "../../../components/text/DateDisplay.js";
import { StatsCompact } from "../../../components/text/StatsCompact.js";
import { VersionDisplay } from "../../../components/text/VersionDisplay.js";
import { SessionContext } from "../../../contexts/session.js";
import { useObjectActions } from "../../../hooks/objectActions.js";
import { useUpsertRunRoutine } from "../../../hooks/runs.js";
import { useErrorPopover } from "../../../hooks/useErrorPopover.js";
import { useLazyFetch } from "../../../hooks/useFetch.js";
import { useManagedObject } from "../../../hooks/useManagedObject.js";
import { IconCommon } from "../../../icons/Icons.js";
import { useLocation } from "../../../route/router.js";
import { addSearchParams } from "../../../route/searchParams.js";
import { FormSection, ScrollBox } from "../../../styles.js";
import { type FormErrors, type PartialWithType } from "../../../types.js";
import { ObjectAction } from "../../../utils/actions/objectActions.js";
import { getCurrentUser } from "../../../utils/authentication/session.js";
import { ELEMENT_IDS } from "../../../utils/consts.js";
import { getDisplay } from "../../../utils/display/listTools.js";
import { firstString } from "../../../utils/display/stringTools.js";
import { getLanguageSubtag, getPreferredLanguage, getUserLanguages } from "../../../utils/display/translationTools.js";
import { openObject } from "../../../utils/navigation/openObject.js";
import { PubSub } from "../../../utils/pubsub.js";
import { routineTypes } from "../../../utils/search/schemas/resource.js";
import { routineSingleStepInitialValues } from "./RoutineSingleStepUpsert.js";
import { RoutineApiForm, RoutineDataConverterForm, RoutineDataForm, RoutineGenerateForm, RoutineInformationalForm, RoutineSmartContractForm, type RoutineFormPropsBase } from "./RoutineTypeForms.js";
import { type RoutineSingleStepViewProps } from "./types.js";

/**
 * If no existing routine found, consider it an error if we are not creating a new routine
 */
function handleInvalidUrlParams() {
    PubSub.get().publish("snack", { messageKey: "InvalidUrlId", severity: "Error" });
}

type RoutineSingleStepTypeViewProps = {
    config: ResourceVersionConfig;
    existing: PartialWithType<ResourceVersion>;
    isGetRoutineLoading: boolean;
}

const IncompleteWarningLabel = styled(Typography)(({ theme }) => ({
    color: theme.palette.background.textSecondary,
    fontStyle: "italic",
}));

function RoutineSingleStepTypeView({
    config,
    existing,
    isGetRoutineLoading,
}: RoutineSingleStepTypeViewProps) {
    const [, setLocation] = useLocation();
    const session = useContext(SessionContext);
    const { errors, isSubmitting, isValidating, resetForm, setSubmitting, setValues, values } = useFormikContext<FormErrors>();

    const routineTypeOption = useMemo(function routineTypeMemo() {
        return routineTypes.find((option) => option.type === existing.resourceSubType);
    }, [existing.routineType]);

    const onRunChange = useCallback(function onRunChangeCallback(newRun: RunRoutine | null) {
        setRun(newRun);
        // Update formik values with run inputs
        if (!newRun) {
            resetForm();
            return;
        }
        const values = FormBuilder.generateInitialValuesFromRoutineConfig(config, newRun);
        setValues(values as FormErrors);
    }, [config, resetForm, setValues]);

    const { createRun, updateRun, isCreatingRunRoutine, isUpdatingRunRoutine } = useUpsertRunRoutine();
    const [run, setRun] = useState<RunRoutine | null>(null);
    useEffect(function updateUrlOnRunChange() {
        if (run) {
            addSearchParams(setLocation, { runId: run.id } as RoutineSingleStepViewSearchParams);
        }
    }, [run, setLocation]);

    const [getRun, { data: runData, loading: isLoadingGetRun }] = useLazyFetch<FindByIdInput, RunRoutine>(endpointsRun.findOne);
    useEffect(function fetchRunFromUrl() {
        const searchParams = UrlTools.parseSearchParams(LINKS.RoutineSingleStep);
        if (searchParams.runId) {
            if (validatePK(searchParams.runId)) {
                getRun({ id: searchParams.runId });
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
    // const {
    //     handleRunSubroutine,
    //     isGeneratingOutputs,
    // } = useSocketRun({
    //     getFormValues,
    //     handleRunUpdate: onRunChange as (run: RunRoutine | RunProject) => unknown,
    //     run,
    //     runnableObject: existing as ResourceVersion,
    // });

    const { openPopover, Popover } = useErrorPopover({ errors, onSetSubmitting: setSubmitting });
    const hasFormErrors = useMemo(() => Object.values(errors ?? {}).some((value) => exists(value)), [errors]);
    const handleCompleteStep = useCallback(function handleCompleteStepCallback(_event: React.MouseEvent | React.TouchEvent) {
        const isLoggedIn = validatePK(getCurrentUser(session).id);
        if (!isLoggedIn) {
            PubSub.get().publish("proDialog");
            return;
        }
        if (!validatePK(existing.id)) {
            console.error("Routine ID is not a valid UUID");
            return;
        }
        // Create run if it doesn't exist
        if (!run) {
            const runRoutineId = generatePK();
            // TODO: Need to implement RunIOManager replacement
            const inputsCreate: any[] = [];
            /* const { inputsCreate } = new RunIOManager(console).runInputsUpdate({
                existingIO: [],
                formData: values as object,
                routineIO: existing.inputs ?? [],
                runRoutineId,
            }); */
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
            // TODO: Need to implement RunIOManager replacement
            const inputsCreate: any[] = [];
            const inputsUpdate: any[] = [];
            const inputsDelete: any[] = [];
            /* const { inputsCreate, inputsUpdate, inputsDelete } = new RunIOManager(console).runInputsUpdate({
                existingIO: run.inputs,
                formData: values as object,
                routineIO: existing.inputs ?? [],
                runRoutineId,
            }); */
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
                    // handleRunSubroutine();
                },
            });
        } else {
            // We have a run, proceed to run the subroutine
            // handleRunSubroutine();
        }
    }, [createRun, existing, run]);
    const handleClearRun = useCallback(function handleClearRunCallback() {
        setRun(null);
        resetForm();
    }, [resetForm]);

    // Type-specific components
    const routineTypeComponents = useMemo(function routineTypeComponentsMemo() {
        if (!existing) return null;

        const isLoading = isCreatingRunRoutine || isUpdatingRunRoutine || isGetRoutineLoading || isLoadingGetRun || isSubmitting || isValidating; // || isGeneratingOutputs
        const isLoggedIn = validatePK(getCurrentUser(session).id);
        const isDeleted = existing.isDeleted ?? existing.root?.isDeleted ?? false;
        const canRun = existing.you?.canRun ?? false;
        const hasErrors = hasFormErrors;
        const routineTypeBaseProps: RoutineFormPropsBase = {
            config,
            disabled: false,
            display: "view",
            handleCompleteStep,
            handleClearRun,
            handleRunStep,
            hasErrors,
            isCompleteStepDisabled: isLoading || !isLoggedIn || isDeleted || !canRun || hasFormErrors,
            isPartOfMultiStepRoutine: false,
            isRunStepDisabled: isLoading || !isLoggedIn || isDeleted || !canRun || hasFormErrors,
            isRunningStep: false, // isGeneratingOutputs,
            onCallDataActionChange: noop, // Only used in edit mode
            onCallDataApiChange: noop, // Only used in edit mode
            onCallDataCodeChange: noop, // Only used in edit mode
            onCallDataGenerateChange: noop, // Only used in edit mode
            onCallDataSmartContractChange: noop, // Only used in edit mode
            onFormInputChange: noop, // Only used in edit mode
            onFormOutputChange: noop, // Only used in edit mode
            onGraphChange: noop, // Only used in edit mode
            onRunChange,
            routineId: existing.id as string,
            routineName: getDisplay(existing).title,
            run,
        };

        switch (existing.resourceSubType) {
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
    }, [config, existing, handleClearRun, handleCompleteStep, handleRunStep, hasFormErrors, isCreatingRunRoutine, isGetRoutineLoading, isLoadingGetRun, isSubmitting, isUpdatingRunRoutine, isValidating, onRunChange, run, session]);

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
    "Statistics are calculated to measure various aspects of a routine. \n\n**Complexity** is a rough measure of the maximum amount of effort it takes to complete a routine. This takes into account the number of inputs, the structure of its subroutine graph, and the complexity of every subroutine.\n\nThere will be many more statistics in the near future.";

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

    const { isLoading: isGetRoutineLoading, object: existing, permissions, setObject: setResourceVersion } = useManagedObject<ResourceVersion>({
        ...endpointsResource.findOne,
        onInvalidUrlParams: handleInvalidUrlParams,
        objectType: "ResourceVersion",
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
        objectType: "ResourceVersion",
        openAddCommentDialog,
        setLocation,
        setObject: setResourceVersion,
    });
    const handleEditStart = useCallback(function handleEditStartCallback() {
        actionData.onActionStart(ObjectAction.Edit);
    }, [actionData]);

    const initialValues = useMemo(() => routineSingleStepInitialValues(session, existing), [existing, session]);
    const resourceList = useMemo<ResourceListShape | null | undefined>(() => initialValues.resourceList as ResourceListShape | null | undefined, [initialValues]);
    const tags = useMemo<TagShape[] | null | undefined>(() => (initialValues.root as Resource)?.tags as TagShape[] | null | undefined, [initialValues]);

    const config = useMemo(function configMemo() {
        return ResourceVersionConfig.parse({ config: existing.config, routineType: existing.resourceSubType ?? RoutineType.Informational }, console);
    }, [existing.config, existing.resourceSubType]);

    const { formInitialValues, formValidationSchema } = useMemo(function initialValuesMemo() {
        const initialValues = FormBuilder.generateInitialValuesFromRoutineConfig(config, existing.resourceSubType ?? RoutineType.Informational);
        const validationSchema = FormBuilder.generateYupSchemaFromRoutineConfig(config, existing.resourceSubType ?? RoutineType.Informational);
        return { formInitialValues: initialValues, formValidationSchema: validationSchema };
    }, [config, existing.resourceSubType]);

    const resourceListParent = useMemo(function resourceListParent() {
        return { __typename: "ResourceVersion", id: existing?.id ?? "" } as const;
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
                    {existing?.resourceSubType && Object.values(RoutineType).includes(existing.resourceSubType) && (
                        <Formik
                            enableReinitialize={true}
                            initialValues={formInitialValues}
                            onSubmit={noop}
                            validationSchema={formValidationSchema}
                        >
                            <RoutineSingleStepTypeView
                                config={config}
                                existing={existing}
                                isGetRoutineLoading={isGetRoutineLoading}
                            />
                        </Formik>
                    )}
                    {/* Tags */}
                    {exists(tags) && tags.length > 0 && <TagList
                        maxCharacters={30}
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
                                    timestamp={existing?.createdAt}
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
                        objectType={CommentFor.ResourceVersion}
                        onAddCommentClose={closeAddCommentDialog}
                    />
                </Stack>}
            </Formik>
            {/* Edit button (if canUpdate) and run button, positioned at bottom corner of screen */}
            <SideActionsButtons
                // Treat as a dialog when build view is open
                display={isGraphOpen ? "Dialog" : display}
            >
                {/* Edit button */}
                {permissions.canUpdate ? (
                    <IconButton
                        aria-label={t("UpdateRoutine")}
                        onClick={handleEditStart}
                    >
                        <IconCommon name="Edit" />
                    </IconButton>
                ) : null}
                {/* Play button fixed to bottom of screen, to start routine (if multi-step) */}
                {Boolean(existing) && existing?.resourceSubType !== RoutineType.Informational ? <RunButton
                    isEditing={false}
                    objectType="ResourceVersion"
                    runnableObject={existing as ResourceVersion}
                /> : null}
            </SideActionsButtons>
        </ScrollBox>
    );
}
