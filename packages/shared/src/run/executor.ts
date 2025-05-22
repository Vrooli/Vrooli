import { ModelType, ResourceSubType, type ResourceVersion, RunStepStatus } from "../api/types.js";
import { type PassableLogger } from "../consts/commonTypes.js";
import { InputType } from "../consts/model.js";
import { CodeLanguage } from "../consts/ui.js";
import { type FormInputType, type FormSchema } from "../forms/types.js";
import { DUMMY_ID } from "../id/snowflake.js";
import { RoutineVersionConfig } from "../shape/configs/routine.js";
import { getTranslation } from "../translations/translationTools.js";
import { BranchManager } from "./branch.js";
import { SubroutineContextManager } from "./context.js";
import { type BranchLocationDataMap, type BranchProgress, BranchStatus, type ExecuteStepResult, type IOMap, type InitializedRunState, InputGenerationStrategy, type Location, type RunConfig, type RunProgress, type RunProgressStep, type RunStateMachineServices, SubroutineExecutionStrategy, type SubroutineIOMapping, type SubroutineInputDisplayInfo, type SubroutineOutputDisplayInfo, type SubroutineOutputDisplayInfoProps } from "./types.js";

/**
 * The result of running a subroutine.
 */
export type RunSubroutineResult = {
    /**
     * The cost (amount of credits spent as a stringified bigint) of running the routine as a stringified bigint.
     */
    cost: string;
    /** 
     * The inputs of the routine.
     * Used to update the context.
     */
    inputs: IOMap;
    /** 
     * The outputs of the routine.
     * Used to update the context.
     */
    outputs: IOMap;
    /**
     * The updated branch status, if we're now paused or failed.
     */
    updatedBranchStatus?: BranchStatus;
}

/**
 * This handles the execution of a subroutine, including:
 * - Input/output formatting
 * - Generating missing inputs
 * - Cost estimation
 * - Executing the subroutine
 * 
 * Without this class, runs would step through the graph but never actually do anything.
 */
export abstract class SubroutineExecutor {
    /**
     * Determines if the given routine is a single-step routine.
     * 
     * @param routine The routine to check
     * @returns True if the routine is a single-step routine, false otherwise
     */
    public isSingleStepRoutine(resource: ResourceVersion): boolean {
        return resource.resourceSubType !== ResourceSubType.RoutineMultiStep
            && resource.resourceSubType.startsWith("Routine");
    }

    /**
     * Determines if the given routine is a multi-step routine.
     * 
     * @param routine The routine to check
     * @returns True if the routine is a multi-step routine, false otherwise
     */
    public isMultiStepRoutine(resource: ResourceVersion): boolean {
        return resource.resourceSubType === ResourceSubType.RoutineMultiStep;
    }

    /**
     * Finds the FormElement that corresponds to the given input or output
     * 
     * @param schema The schema to search
     * @param io The input or output to find
     * @returns The FormElement with the same name as the input or output, or undefined if it is not found
     */
    private findFormElement(schema: FormSchema | undefined, io: RoutineVersionInput | RoutineVersionOutput): FormInputType | undefined {
        if (!schema) {
            return undefined;
        }
        const key = io.name;
        if (!key) {
            return undefined;
        }
        const element = schema.elements.find((element) => Object.prototype.hasOwnProperty.call(element, "fieldName") && (element as FormInputType).fieldName === key);
        if (!element) {
            return undefined;
        }
        return element as FormInputType;
    }

    /**
     * Converts a FormElement and input/output into props to tell the LLM exactly what we expect it to generate.
     * 
     * @param formElement The FormElement to convert
     * @param io The input or output element to convert
     * @returns The converted props
     */
    private findElementStructure(formElement: FormInputType | undefined, io: RoutineVersionInput | RoutineVersionOutput): { defaultValue: unknown | undefined, props: SubroutineOutputDisplayInfoProps } {
        const result: { defaultValue: unknown | undefined, props: SubroutineOutputDisplayInfoProps } = { defaultValue: undefined, props: { type: "Text", schema: undefined } };
        // There are three cases:
        // 1. The io has a standard version attached, which defines a specific schema to adhere to
        // 2. The formElement exists, which can be used to infer the type and schema based on the element's type and properties
        // 3. Neither are present, so we fallback to plain text
        //
        // Case 1
        if (io.standardVersion) {
            const { codeLanguage, props } = io.standardVersion;
            result.props.type = codeLanguage === CodeLanguage.Json ? "JSON schema"
                : codeLanguage === CodeLanguage.Yaml ? "YAML schema"
                    : "Unknown";
            result.props.schema = props;
        }
        // Case 2
        else if (formElement) {
            switch (formElement.type) {
                case InputType.Text: {
                    result.props.type = "Text";
                    const { defaultValue, maxChars } = formElement.props;
                    if (defaultValue !== undefined) {
                        result.defaultValue = defaultValue;
                    }
                    if (maxChars !== undefined) {
                        result.props.maxLength = maxChars;
                    }
                    break;
                }
                case InputType.Switch: {
                    result.props.type = "Boolean";
                    const { defaultValue } = formElement.props;
                    if (defaultValue !== undefined) {
                        result.defaultValue = defaultValue;
                    }
                    break;
                }
                case InputType.IntegerInput: {
                    result.props.type = "Number";
                    const { allowDecimal, defaultValue, max, min } = formElement.props;
                    if (allowDecimal === false) {
                        result.props.type = "Integer";
                    }
                    if (defaultValue !== undefined) {
                        result.defaultValue = defaultValue;
                    }
                    if (max !== undefined) {
                        result.props.max = max;
                    }
                    if (min !== undefined) {
                        result.props.min = min;
                    }
                    break;
                }
                case InputType.Slider: {
                    result.props.type = "Number";
                    const { defaultValue, max, min } = formElement.props;
                    if (defaultValue !== undefined) {
                        result.defaultValue = defaultValue;
                    }
                    if (max !== undefined) {
                        result.props.max = max;
                    }
                    if (min !== undefined) {
                        result.props.min = min;
                    }
                    break;
                }
                case InputType.Checkbox: {
                    result.props.type = "List of values from the provided options";
                    const { allowCustomValues, defaultValue, options, maxCustomValues, maxSelection, minSelection } = formElement.props;
                    if (allowCustomValues === true) {
                        const supportsMultipleCustomValues = maxCustomValues !== undefined && maxCustomValues > 1;
                        result.props.type = supportsMultipleCustomValues
                            ? "List of values, using the provided options and custom values"
                            : "List of values from the provided options, with up to one custom value";
                    }
                    if (defaultValue !== undefined) {
                        result.defaultValue = defaultValue;
                    }
                    if (maxCustomValues !== undefined) {
                        result.props.maxCustomValues = maxCustomValues;
                    }
                    if (maxSelection !== undefined) {
                        result.props.maxSelection = maxSelection;
                    }
                    if (minSelection !== undefined) {
                        result.props.minSelection = minSelection;
                    }
                    result.props.options = options;
                    break;
                }
                case InputType.Radio: {
                    result.props.type = "One of the values from the provided options";
                    const { defaultValue, options } = formElement.props;
                    if (defaultValue !== undefined) {
                        result.defaultValue = defaultValue;
                    }
                    result.props.options = options;
                    break;
                }
                case InputType.Selector: {
                    // TODO: For now, assume that options are objects with a "value" property
                    result.props.type = "One of the values from the provided options";
                    const { defaultValue, multiple, options, noneOption } = formElement.props;
                    if (defaultValue !== undefined) {
                        result.defaultValue = defaultValue;
                    }
                    if (multiple === true) {
                        result.props.type = "List of values from the provided options";
                    }
                    if (noneOption === true) {
                        result.props.type = "One of the values from the provided options, or null";
                    }
                    result.props.options = options;
                    break;
                }
                case InputType.LanguageInput: {
                    result.props.type = "List of language codes";
                    const { defaultValue } = formElement.props;
                    if (defaultValue !== undefined) {
                        result.defaultValue = defaultValue;
                    }
                    break;
                }
                // This is a special case, as we'll be finding the link later using search 
                // instead of generating it with an LLM. But we still want to grab the data 
                // required to perform the search.
                case InputType.LinkItem: {
                    result.props.type = "Item ID (e.g. RoutineVersion:123-456-789)";
                    const { defaultValue, limitTo } = formElement.props;
                    if (defaultValue !== undefined) {
                        result.defaultValue = defaultValue;
                    }
                    if (limitTo) {
                        result.props.limitTo = limitTo;
                    }
                    break;
                }
                case InputType.LinkUrl: {
                    result.props.type = "URL";
                    const { acceptedHosts, defaultValue } = formElement.props;
                    if (defaultValue !== undefined) {
                        result.defaultValue = defaultValue;
                    }
                    if (acceptedHosts) {
                        result.props.acceptedHosts = acceptedHosts;
                    }
                    break;
                }
                case InputType.TagSelector: {
                    result.props.type = "List of alphanumeric tags (a.k.a. hashtags, keywords)";
                    const { defaultValue } = formElement.props;
                    if (defaultValue !== undefined) {
                        result.defaultValue = defaultValue;
                    }
                    break;
                }
                case InputType.JSON: {
                    result.props.type = "JSON object"; // Defaults to JSON unless otherwise specified
                    const { defaultValue, limitTo } = formElement.props;
                    if (defaultValue !== undefined) {
                        result.defaultValue = defaultValue;
                    }
                    if (limitTo && Array.isArray(limitTo) && limitTo.length > 0) {
                        result.props.type = "Code in one of these languages: " + limitTo.join(", ");
                    }
                    break;
                }
                // Like LinkItem, we can't use generation for Dropzone. We'll have to support this with search instead.
                case InputType.Dropzone: {
                    result.props.type = "File";
                    const { acceptedFileTypes, defaultValue, maxFiles } = formElement.props;
                    if (acceptedFileTypes) {
                        result.props.acceptedFileTypes = acceptedFileTypes;
                    }
                    if (defaultValue !== undefined) {
                        result.defaultValue = defaultValue;
                    }
                    if (maxFiles !== undefined && maxFiles > 1) {
                        result.props.type = "Files";
                        result.props.maxFiles = maxFiles;
                    }
                    break;
                }
                default: {
                    // Add error logging here if needed
                    break;
                }
            }
        }
        return result;
    }

    /**
     * Builds a mapping of routine input and output names to an object containing the current value
     * (if any) and whether that input is required.
     *
     * @param routine The routine whose inputs we need to check.
     * @param providedInputs The IOMap containing any already provided values.
     * @param logger The logger to use for logging.
     * @returns A record mapping each input and output name to its current value and required flag.
     */
    private buildIOMapping(
        routine: RoutineVersion,
        providedInputs: IOMap,
        logger: PassableLogger,
    ): SubroutineIOMapping {
        // Initialize the mapping
        const mapping: SubroutineIOMapping = { inputs: {}, outputs: {} };

        // Some input information can be grabbed from the routine's inputs and outputs directly.
        // Other information can be found in the routine's form schemas
        const { formInput, formOutput } = RoutineVersionConfig.parse(routine, logger, { useFallbacks: true });
        const formInputSchema = formInput?.schema;
        const formOutputSchema = formOutput?.schema;

        // Process inputs
        routine.inputs.forEach((input) => {
            if (!input.name) {
                return;
            }

            const formElement = this.findFormElement(formInputSchema, input);
            const { defaultValue, props } = this.findElementStructure(formElement, input);

            const description = formElement?.description ?? undefined;
            const isRequired = input.isRequired ?? false;
            const name = input.name;
            const value = providedInputs[name];

            mapping.inputs[name] = {
                defaultValue,
                description,
                isRequired,
                name,
                props,
                value,
            };
        });

        // Process outputs
        routine.outputs.forEach((output) => {
            if (!output.name) {
                return;
            }

            const formElement = this.findFormElement(formOutputSchema, output);
            const { defaultValue, props } = this.findElementStructure(formElement, output);

            const description = formElement?.description ?? undefined;
            const name = output.name;
            const value = providedInputs[name];

            mapping.outputs[name] = {
                defaultValue,
                description,
                name,
                props,
                value,
            };
        });

        return mapping;
    }

    /**
     * Generates a dummy value based on the type of the input/output.
     * 
     * NOTE: We can't guarantee that the dummy value will be valid, especially if the expected type is a specific 
     * schema or has props like "min" or "max". But we can at least provide the correct primitive type 
     * (e.g. string, number, boolean, etc.) easily enough.
     * 
     * @param ioInfo The input or output info to generate a dummy value for
     * @returns A dummy value
     */
    private generateDummyValue(ioInfo: SubroutineOutputDisplayInfo | SubroutineInputDisplayInfo): unknown {
        if (ioInfo.defaultValue !== undefined) {
            return ioInfo.defaultValue;
        }
        if (!ioInfo.props) {
            return undefined;
        }
        const { type } = ioInfo.props;
        if (type === "Boolean") {
            return true;
        }
        if (type === "Integer") {
            return 0;
        }
        if (type === "Number") {
            return 0;
        }
        if (type.startsWith("List of")) {
            return [];
        }
        if (type.startsWith("One of")) {
            return "dummy_value";
        }
        if (type === "URL") {
            return "https://example.com";
        }
        if (type.startsWith("JSON")) {
            return {};
        }
        if (type.startsWith("YAML")) {
            return {};
        }
        if (type.startsWith("Code in")) {
            return "dummy_value";
        }
        return "dummy_value";
    }

    /**
     * Generates dummy inputs and outputs for a subroutine.
     * Useful during test mode, when we want to check if a subroutine can run without 
     * calling anything or using any credits.
     * 
     * @param ioMapping A mapping of input and output names to their current value and required flag.
     * @returns ioMapping with required inputs and outputs filled in with dummy values
     */
    private async dummyRun(ioMapping: SubroutineIOMapping): Promise<Omit<RunSubroutineResult, "updatedBranchStatus">> {
        const result: RunSubroutineResult = {
            inputs: {},
            outputs: {},
            cost: BigInt(0).toString(),
        };

        for (const inputKey in ioMapping.inputs) {
            const input = ioMapping.inputs[inputKey];
            // Skip inputs that aren't required or already have a value
            if (!input || !input.isRequired || input.value !== undefined) {
                continue;
            }
            // Generate a dummy value for the input
            result.inputs[inputKey] = this.generateDummyValue(input);
        }

        for (const outputKey in ioMapping.outputs) {
            const output = ioMapping.outputs[outputKey];
            // Skip outputs that already have a value
            if (!output || output.value !== undefined) {
                continue;
            }
            // Generate a dummy value for the output
            result.outputs[outputKey] = this.generateDummyValue(output);
        }

        return result;
    }

    /**
     * Runs a subroutine.
     * 
     * NOTE: This should only be called for single-step routines. 
     * Multi-step routines are instead used for navigation.
     * 
     * @param subroutineInstanceId The ID of the subroutine instance
     * @param routine The routine to run
     * @param ioMapping A mapping of input and output names to their current value and required flag.
     * @param runConfig The overall run configuration
     * @returns The inputs and outputs of the subroutine (for updating the subcontext), as well as the cost of running the routine
     */
    public abstract runSubroutine(subroutineInstanceId: string, routine: ResourceVersion, ioMapping: SubroutineIOMapping, runConfig: RunConfig): Promise<Omit<RunSubroutineResult, "updatedBranchStatus">>;

    /**
     * Generates missing inputs for a subroutine.
     * 
     * NOTE: This should only be called for single-step routines.
     * 
     * @param subroutineInstanceId The ID of the subroutine instance
     * @param routine The routine to generate missing inputs for
     * @param ioMapping A mapping of input and output names to their current value and required flag.
     * @param runConfig The overall run configuration
     * @returns The missing inputs
     */
    public abstract generateMissingInputs(subroutineInstanceId: string, routine: ResourceVersion, ioMapping: SubroutineIOMapping, runConfig: RunConfig): Promise<Omit<RunSubroutineResult, "updatedBranchStatus">>;

    /**
     * Runs a subroutine.
     * 
     * NOTE 1: This should only be called for single-step routines. 
     * Multi-step routines are instead used for navigation.
     * 
     * @param routine The routine to run
     * @param providedInputs A map of subroutine input names to their values, for every input value we already have
     * @param runConfig The overall run configuration
     * @param branch The branch that is running the subroutine
     * @param services The state machine's services
     * @param state The state of the state machine
     * @returns The inputs and outputs of the subroutine (for updating the subcontext), as well as the cost of running the routine
     */
    public async run(
        routine: ResourceVersion,
        providedInputs: IOMap,
        runConfig: RunConfig,
        branch: BranchProgress,
        services: RunStateMachineServices,
        state: InitializedRunState,
    ): Promise<RunSubroutineResult> {
        const result: RunSubroutineResult = {
            cost: BigInt(0).toString(),
            inputs: {},
            outputs: {},
            updatedBranchStatus: undefined,
        };

        // Don't run for multi-step routines
        if (this.isMultiStepRoutine(routine)) {
            throw new Error("Multi-step routines should not be run directly");
        }

        // Build a mapping of inputs with their provided value (or undefined) and required flag.
        const ioMapping = this.buildIOMapping(routine, providedInputs, services.logger);

        // Check if there are missing required inputs
        const missingRequiredInputNames = Object.entries(ioMapping.inputs)
            .filter(([, input]) => input.isRequired && input.value === undefined)
            .map(([name]) => name);

        // Don't continue if we need to manually fill in missing inputs
        if (missingRequiredInputNames.length > 0 && runConfig.decisionConfig.inputGeneration === InputGenerationStrategy.Manual) {
            services.notifier?.sendMissingInputsRequest(state.runIdentifier.runId, state.runIdentifier.type, branch);
            result.updatedBranchStatus = BranchStatus.Waiting;
            return result;
        }

        // Don't continue if we need to manually confirm execution
        if (runConfig.decisionConfig.subroutineExecution === SubroutineExecutionStrategy.Manual && !branch.manualExecutionConfirmed) {
            // But if we can automatically generate inputs, do that first
            if (missingRequiredInputNames.length > 0 && runConfig.decisionConfig.inputGeneration !== InputGenerationStrategy.Manual) {
                const missingInputs = await this.generateMissingInputs(branch.subroutineInstanceId, routine, ioMapping, runConfig);
                result.inputs = missingInputs;
                result.cost = missingInputs.cost;
            }
            // Send a request to the client to confirm manual execution
            services.notifier?.sendManualExecutionConfirmationRequest(state.runIdentifier.runId, state.runIdentifier.type, branch);
            result.updatedBranchStatus = BranchStatus.Waiting;
            return result;
        }

        // Perform the real run or a dummy run
        const inTestMode = runConfig.testMode === true;
        const { inputs: updatedInputs, outputs: updatedOutputs, cost: updatedCost } = inTestMode
            ? await this.dummyRun(ioMapping)
            : await this.runSubroutine(branch.subroutineInstanceId, routine, ioMapping, runConfig);
        result.inputs = updatedInputs;
        result.outputs = updatedOutputs;
        result.cost = updatedCost;

        return result;
    }

    /**
    * Estimates the max cost of running a subroutine.
    * 
    * NOTE: This should only be called for single-step routines.
    * 
    * @param subroutineInstanceId The ID of the subroutine instance
    * @param routine The routine to generate missing inputs for
    * @param runConfig The overall run configuration
    * @returns The maximum cost in credits of running the subroutine, as a stringified bigint
    */
    public abstract estimateCost(subroutineInstanceId: string, routine: ResourceVersion, runConfig: RunConfig): Promise<string>;

    /**
     * Initializes a new run step progress
     * 
     * @param location The location of the step
     * @param subroutine The subroutine that the step is for
     * @param state The state of the state machine
     * @returns A new run step progress
     */
    private createStepProgress(
        location: Location,
        subroutine: ResourceVersion | null,
        state: InitializedRunState,
    ): RunProgressStep {
        return {
            ...location,
            complexity: 0,
            contextSwitches: 0,
            id: DUMMY_ID,
            startedAt: new Date(Date.now()),
            completedAt: undefined,
            name: getTranslation(subroutine, state.userData.languages, true).name ?? "",
            order: 0, // This will be updated later in `updateRunAfterStep`
            status: RunStepStatus.InProgress,
        };
    }

    // TODO need to create/update run_routine_io records for each input and output
    /**
     * Execute the next step in a branch
     * 
     * NOTE 1: We can't update the run progress in here, since multiple 
     * branches might be executing in parallel (leading to race conditions).
     * Instead, we have to return the updated branch and run progress,
     * and let the caller (runUntilDone) handle data updates.
     * 
     * NOTE 2: This should not be used to step through the graph. The only 
     * new branches that should be created are when a multi-step routine 
     * starts.
     * 
     * @param run The run the branch belongs to
     * @param branch The branch to execute
     * @param branchLocationDataMap The location data for each branch
     * @param services The state machine's services
     * @param state The state of the state machine
     * @returns The updated branch and run data
     */
    public async executeStep(
        run: RunProgress,
        branch: BranchProgress,
        branchLocationDataMap: BranchLocationDataMap,
        services: RunStateMachineServices,
        state: InitializedRunState,
    ): Promise<ExecuteStepResult> {
        // Initialize the result
        const result: ExecuteStepResult = {
            branchId: branch.branchId,
            branchStatus: branch.status,
            deferredDecisions: [],
            creditsSpent: BigInt(0).toString(),
            newLocations: null,
            step: null,
            subroutineRun: null,
        };

        // Find the location data for the branch
        const locationData = branchLocationDataMap[branch.branchId];
        if (!locationData) {
            services.logger.error(`executeStep: Branch ${branch.branchId} not found in branchLocationDataMap for run ${run.runId}`);
            branch.status = BranchStatus.Failed;
            return result;
        }

        // Find or create a step to track the step progress and store metrics
        let step: RunProgressStep | undefined = undefined;
        if (branch.stepId) {
            step = run.steps.find(s => s.id === branch.stepId);
        }
        if (!step) {
            step = this.createStepProgress(locationData.location, locationData.subroutine, state);
        }
        result.step = step;

        // Skip if branch is not active
        if (branch.status !== BranchStatus.Active) {
            return result;
        }

        const { location, object, subroutine, subcontext: parentContext } = locationData;
        const nodeId = location.locationId;

        // Run the appropriate action based on the object type
        if (object.__typename === ModelType.RoutineVersion) {
            // Action only needed if there is a subroutine
            if (!subroutine) {
                services.logger.info(`executeStep: No subroutine found at ${branch.locationStack[branch.locationStack.length - 1]?.locationId}`);
                return result;
            }
            // If it's a single-step routine, run it
            const isSingleStep = this.isSingleStepRoutine(subroutine);
            if (isSingleStep) {
                // Build context to pass into subroutine
                const knownSubroutineInputs = await SubroutineContextManager.mapParentContextToSubroutineInputs(object, nodeId, parentContext, services);
                if (!knownSubroutineInputs) {
                    services.logger.error(`executeStep: Child context could not be prepared for single-step subroutine ${subroutine.id}`);
                    result.branchStatus = BranchStatus.Failed;
                    return result;
                }

                // Execute the subroutine
                const subroutineResult = await this.run(
                    subroutine,
                    knownSubroutineInputs,
                    run.config,
                    branch,
                    services,
                    state,
                );

                // Parse the result
                const properlyKeyedResult = SubroutineContextManager.mapSubroutineResultToParentKeys(subroutineResult, nodeId, object, services.logger);
                if (!properlyKeyedResult) {
                    services.logger.error(`executeStep: Inputs or outputs could not be converted to composite keys for single-step subroutine ${subroutine.id}`);
                    result.branchStatus = BranchStatus.Failed;
                    return result;
                }
                result.subroutineRun = {
                    inputs: properlyKeyedResult.inputs,
                    outputs: properlyKeyedResult.outputs,
                };
                result.creditsSpent = (BigInt(result.creditsSpent) + BigInt(subroutineResult.cost)).toString();
                // If the branch status is being updated, this indicates that the subroutine has NOT completed
                const isSubroutineCompleted = subroutineResult.updatedBranchStatus === undefined;
                if (isSubroutineCompleted) {
                    result.branchStatus = subroutineResult.updatedBranchStatus as BranchStatus;
                }
                // Update step data
                if (result.step) {
                    result.step.complexity = subroutine.complexity;
                    // If completed, set the completion data
                    if (isSubroutineCompleted) {
                        result.step.completedAt = new Date(Date.now());
                        result.step.status = RunStepStatus.Completed;
                    }
                }
            }
            // If it's a multi-step routine, we need to call getAvailableStartLocations with the subroutine config and create new branches
            const isMultiStep = this.isMultiStepRoutine(subroutine);
            if (isMultiStep) {
                // Deserialize the child's graph configuration.
                const { graph: childGraph } = RoutineVersionConfig.parse(subroutine, services.logger, { useFallbacks: true });
                if (!childGraph) {
                    services.logger.error("executeStep: Invalid child graph configuration.");
                    result.branchStatus = BranchStatus.Failed;
                    return result;
                }
                // Get the child's navigator
                const childNavigator = services.navigatorFactory.getNavigator(childGraph.__type);
                if (!childNavigator) {
                    services.logger.error(`executeStep: No navigator found for child graph type ${childGraph.__type}`);
                    result.branchStatus = BranchStatus.Failed;
                    return result;
                }

                // Build context to pass into subroutine
                const subroutineInputNameToParentContextValueMap = await SubroutineContextManager.mapParentContextToSubroutineInputs(object, nodeId, parentContext, services);
                if (!subroutineInputNameToParentContextValueMap) {
                    services.logger.error(`executeStep: subroutineInputNameToParentContextValueMap could not be prepared for multi-step subroutine ${subroutine.id}`);
                    result.branchStatus = BranchStatus.Failed;
                    return result;
                }
                const initialChildContextValues = await SubroutineContextManager.prepareMultiStepSubroutineContext(subroutineInputNameToParentContextValueMap, childGraph);
                if (!initialChildContextValues) {
                    services.logger.error(`executeStep: initialChildContextValues could not be prepared for multi-step subroutine ${subroutine.id}`);
                    result.branchStatus = BranchStatus.Failed;
                    return result;
                }
                const childContext = SubroutineContextManager.buildSubroutineContext(initialChildContextValues, subroutine, parentContext, state.userData);

                // Ask the subroutine's navigator for the available start locations.
                const decisionKey = services.pathSelectionHandler.generateDecisionKey(branch, "executeStep-startLocations");
                const decision = await childNavigator.getAvailableStartLocations({
                    config: childGraph,
                    decisionKey,
                    services,
                    subroutine,
                    subcontext: childContext,
                });
                if (decision.deferredDecisions) {
                    result.deferredDecisions = decision.deferredDecisions;
                    result.branchStatus = BranchStatus.Waiting;
                    return result;
                }
                if (decision.triggerBranchFailure) {
                    result.branchStatus = BranchStatus.Failed;
                    return result;
                }
                // Assign the new start locations to the result.
                result.newLocations = {
                    initialContext: childContext,
                    locations: decision.nextLocations,
                    supportsParallelExecution: childNavigator.supportsParallelExecution,
                };
                // Update step data
                if (result.step) {
                    // The complexity of triggering a multi-step subroutine is 1. Each subroutine inside it will push its own step to the run, so 
                    // we don't need to worry about the total complexity.
                    // 
                    // This setup is useful because sometimes we skip subroutines in the multi-step routine, and we don't want to 
                    // count the complexity of the subroutines that weren't run.
                    result.step.complexity = 1;
                }
                // When the result is processed (see updateRunAfterStep), new branches will be created
                // for each location in decision.nextLocations, and the current branch will be marked as waiting.
                return result;
            }
            return result;
        } else if (object.__typename === ModelType.ProjectVersion) {
            // Projects are for navigation only, so no action needed
            return result;
        } else {
            services.logger.error(`executeStep: Unsupported object type: ${(object as { __typename: unknown }).__typename}`);
            result.branchStatus = BranchStatus.Failed;
            return result;
        }
    }

    /**
     * Helper function to update the branch and run progress after a branch is stepped.
     * 
     * @param run The run progress
     * @param branch The branch that was stepped
     * @param stepResult The result of the step
     * @param services The state machine's services
     * @param state The state of the state machine
     */
    public updateRunAfterStep(
        run: RunProgress,
        branch: BranchProgress,
        stepResult: ExecuteStepResult,
        services: RunStateMachineServices,
        state: InitializedRunState,
    ): void {
        // Update status
        branch.status = stepResult.branchStatus;
        // Update credits spent
        run.metrics.creditsSpent = (BigInt(run.metrics.creditsSpent) + BigInt(stepResult.creditsSpent)).toString();
        // Add or update the step progress if it exists
        if (stepResult.step) {
            const stepId = stepResult.step.id;
            const existingStepIndex = run.steps.findIndex(step => step.id === stepId);
            if (existingStepIndex !== -1) {
                // Update the step
                run.steps[existingStepIndex] = stepResult.step;
            } else {
                // Add the step
                run.steps.push({
                    ...stepResult.step,
                    order: run.steps.reduce((max, step) => Math.max(max, step.order), 0) + 1,
                });
                // // Assign the step to the branch
                // branch.stepId = stepResult.step.id;
                // Update run metrics
                run.metrics.complexityCompleted += stepResult.step.complexity;
                run.metrics.stepsRun++;
            }
            // If the step in progress, make sure the branch is assigned to it
            if (stepResult.step.status === RunStepStatus.InProgress) {
                branch.stepId = stepId;
            }
            // Otherwise, remove the branch's assignment to it
            else {
                branch.stepId = null;
            }
        }
        // Add deferred decisions
        if (stepResult.deferredDecisions && stepResult.deferredDecisions.length > 0) {
            // Update the decision strategy
            const updatedDecisions = services.pathSelectionHandler.updateDecisionOptions(run, stepResult.deferredDecisions);
            // Update the run progress
            run.decisions = updatedDecisions;
            // Send a decision request to the client
            for (const decision of stepResult.deferredDecisions) {
                services.notifier?.sendDecisionRequest(state.runIdentifier.runId, decision);
            }
            // If the branch is still active, put it in a waiting state
            if (branch.status === BranchStatus.Active) {
                branch.status = BranchStatus.Waiting;
            }
        }
        // Add new branches (should only happen when a multi-step subroutine is encountered)
        if (stepResult.newLocations && stepResult.newLocations.locations.length > 0) {
            const { initialContext, locations, supportsParallelExecution } = stepResult.newLocations;
            // Find the ID of the multi-step subroutine all of the new branches are part of
            const firstLocation = locations[0];
            if (!firstLocation) {
                throw new Error("No location found in step result");
            }
            const subroutineId = firstLocation.objectId;
            // Use a new subroutine instance ID for the new branches
            const subroutineInstanceId = BranchManager.generateSubroutineInstanceId(subroutineId);
            // Add it to the current branch, so we can link the new branches to it
            branch.childSubroutineInstanceId = subroutineInstanceId;
            // Initialize the subcontext for the new branches
            run.subcontexts[subroutineInstanceId] = initialContext;
            // Create the new branches
            const newBranches = BranchManager.forkBranches(branch, locations, subroutineInstanceId, supportsParallelExecution, branch.locationStack);
            // Add them to the run branches
            run.branches.push(...newBranches);
            // Put the current branch in a waiting state until the new branches are completed
            branch.status = BranchStatus.Waiting;
        }
        // Update subcontext (should only happen when a single-step subroutine is encountered)
        if (stepResult.subroutineRun) {
            const { inputs, outputs } = stepResult.subroutineRun;
            const existingSubcontext = run.subcontexts[branch.subroutineInstanceId];
            run.subcontexts[branch.subroutineInstanceId] = SubroutineContextManager.updateContext(existingSubcontext, inputs, outputs);
        }
    }
}
