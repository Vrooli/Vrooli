import { type RoutineVersion } from "../api/types.js";
import { type PassableLogger } from "../consts/commonTypes.js";
import { type GraphConfig, RoutineVersionConfig } from "../shape/configs/routine.js";
import { getTranslation } from "../translations/translationTools.js";
import { type RunSubroutineResult } from "./executor.js";
import { type IOKey, type IOMap, type IOValue, type RunStateMachineServices, type RunTriggeredBy, type SubroutineContext } from "./types.js";

/**
 * Handles context management for running subroutines.
 * This is important for keeping track of inputs and outputs, as well as 
 * generating context for AI responses.
 * 
 * NOTE: This handles the context for a single multi-step subroutine. 
 * What this means is that every nested multi-step subroutine gets its own 
 * isolated context that accumulates inputs and outputs at each step. 
 * When the multi-step subroutine is finished, context is merged back into the 
 * parent context.
 */
export class SubroutineContextManager {
    /**
     * Initializes a new subroutine context.
     * 
     * @param initialValues Optional initial values to seed the context's inputs.
     *                      Keys and values will be used to pre-populate the inputs.
     * @returns The new context
     */
    static initializeContext(initialValues: IOMap = {}): SubroutineContext {
        const allInputsMap = { ...initialValues };
        const allInputsList = Object.entries(initialValues).map(([key, value]) => ({ key, value }));

        return {
            currentTask: {
                description: "",
                instructions: "",
                name: "",
            },
            allInputsList,
            allInputsMap,
            allOutputsList: [],
            allOutputsMap: {},
        };
    }


    /**
     * Updates the subroutine context with the given inputs and outputs.
     * 
     * @param context The current subroutine context
     * @param inputs The new inputs to add to the context
     * @param outputs The new outputs to add to the context
     * @returns The updated context
     */
    static updateContext(
        context: SubroutineContext | null | undefined,
        inputs: IOMap,
        outputs: IOMap,
    ): SubroutineContext {
        if (!context) {
            context = this.initializeContext();
        }

        // Update inputs
        for (const [key, value] of Object.entries(inputs)) {
            context.allInputsMap[key] = value;
            context.allInputsList.push({ key, value });
        }

        // Update outputs
        for (const [key, value] of Object.entries(outputs)) {
            context.allOutputsMap[key] = value;
            context.allOutputsList.push({ key, value });
        }

        return context;
    }

    /**
     * Helper method to prepare a subroutine context.
     * 
     * This method takes the parent routine configuration and subroutine (child)
     * configuration, and uses the parent's mapping configuration to determine which
     * IO values (with composite keys) should be passed into the subroutine.
     * 
     * NOTE: This method produces an object keyed by the subroutine’s input names for easier processing during a single-step run. 
     * It must be further processed by `prepareMultiStepSubroutineContext` or `mapSubroutineResultToParentKeys` to change 
     * the keys to the proper composite keys before merging into the parent context.
     * 
     * @param parentRoutine The parent routine (from the current location’s object)
     * @param nodeId The ID of the current node in the parent graph (i.e. where the subroutine is located in the parent graph)
     * @param parentSubcontext The existing subcontext (I/O values) from the parent branch
     * @param services The state machine's services
     * @returns An object with keys that are the subroutine's input names, and values that are the parent's IO values.
     */
    static async mapParentContextToSubroutineInputs(
        parentRoutine: RoutineVersion,
        nodeId: string,
        parentSubcontext: SubroutineContext,
        services: RunStateMachineServices,
    ): Promise<IOMap | null> {
        // Deserialize the parent's graph configuration.
        const { graph: parentGraph } = RoutineVersionConfig.parse(parentRoutine, services.logger, { useFallbacks: true });
        if (!parentGraph) {
            services.logger.error("mapParentContextToSubroutineInputs: Invalid parent graph configuration.");
            return null;
        }

        // Retrieve the parent's navigator to obtain input mapping information.
        const parentNavigator = services.navigatorFactory.getNavigator(parentGraph.__type);
        if (!parentNavigator) {
            services.logger.error(`mapParentContextToSubroutineInputs: No navigator found for parent graph type ${parentGraph.__type}`);
            return null;
        }

        // 1. Find which inputs for the child node are linked to io values in the parent graph. 
        //    The result should be an object that maps the child node's input names (as they appear in the parent graph) 
        //    to the parent graph's io name composite keys (i.e. `root.${ioName}` or `${nodeId}.${ioName}`).
        //
        //    In BPMN graphs, the node would list inputs like this:
        //    <bpmn:callActivity id="callActivityA" name="Call A" calledElement="A">
        //    //...
        //    <vrooli:input name="inputA" fromContext="callActivityB.inputC" />
        //    //...
        //    
        //    Which should give us an object like this:
        //    {
        //        inputA: "callActivityB.inputC",
        //        //...
        //    }
        const nodeInputNameToParentContextKeyMap = await parentNavigator.getIONamesPassedIntoNode({
            config: parentGraph,
            nodeId,
            services,
        });

        // 2. Create an object that maps the keys from step 1 to the subroutine's input names as they appear in the subroutine's formInput.
        //    
        //    For BPMN, we can get this easily by using the `activityMap` that accompanies the parent graph's config.
        //    {
        //        callActivityA: {
        //            //...
        //            inputMap: {
        //                inputA: "subroutineInputA",
        //            },
        //        },
        //    }
        //
        //    Which would give us an object like this:
        //    {
        //        inputA: "subroutineInputA",
        //    }
        const nodeInputNameToSubroutineInputNameMap = parentGraph.getIONamesToSubroutineInputNames(nodeId);

        // 3. Using these two objects, we can go from nodeInputName -> parentGraphInputName, and nodeInputName -> subroutineInputName.
        //    Now we'll create a new object that goes from subroutineInputName -> parentGraphInputVALUE.
        // 
        //    It should look like this:
        //    {
        //        subroutineInputA: parentSubcontext["callActivityB.inputC"],
        //    }
        const subroutineInputNameToParentContextValueMap = Object.fromEntries(
            Object.entries(nodeInputNameToSubroutineInputNameMap)
                .map(([nodeInputName, subroutineInputName]) => {
                    const parentContextKey = nodeInputNameToParentContextKeyMap[nodeInputName];
                    if (!parentContextKey) {
                        return null;
                    }
                    const parentContextValue = parentSubcontext.allInputsMap[parentContextKey] ?? parentSubcontext.allOutputsMap[parentContextKey];
                    if (!parentContextValue) {
                        return null;
                    }
                    return [subroutineInputName, parentContextValue];
                })
                .filter(Boolean) as [IOKey, IOValue][],
        );

        // Return this object for further processing by routine type-specific methods.
        return subroutineInputNameToParentContextValueMap;
    }

    /**
     * Helper method to prepare the subroutine context for a multi-step subroutine.
     * 
     * This method takes the result of `mapParentContextToSubroutineInputs` and uses it to
     * create the initial subcontext values for a multi-step subroutine. This entails 
     * converting the keys from subroutine input names to child context keys, using the 
     * child's graph config.
     * 
     * @param subroutineInputNameToParentContextValueMap The result of `mapParentContextToSubroutineInputs`
     * @param childGraph The graph config for the child subroutine
     * @returns An object that can be used to initialize the subroutine context for a multi-step subroutine
     */
    static async prepareMultiStepSubroutineContext(
        subroutineInputNameToParentContextValueMap: IOMap,
        childGraph: GraphConfig,
    ): Promise<IOMap | null> {
        // `prepareSubroutineContext` completed the first 3 steps...
        // 4. Now using the CHILD's graph config, create a map of the subroutine's starting input names to its routine's input names.
        //
        //    For BPMN, we can get this easily by using the `rootContext` that accompanies the child's graph config.
        //    {
        //        inputMap: {
        //            inputA: "subroutineInputA",
        //        },
        //    }
        //
        //    Which would give us an object like this:
        //    {
        //        root.inputA: "subroutineInputA",
        //    }
        //
        const childContextKeyToSubroutineInputNameMap = childGraph.getRootIONamesToRoutineInputNames();

        // 5. Now we'll create a new object with the same keys as step 4, but with the values from the parent context. We'll use the object from 
        //    step 3 to find the correct parent context key. It should look like this:
        //
        //    {
        //        root.inputA: parentContext["callActivityB.inputC"],
        //    }
        const initialSubcontextValues: IOMap = {};
        for (const [childContextKey, subroutineInputName] of Object.entries(childContextKeyToSubroutineInputNameMap)) {
            const parentContextValue = subroutineInputNameToParentContextValueMap[subroutineInputName];
            if (!parentContextValue) {
                continue;
            }
            initialSubcontextValues[childContextKey] = parentContextValue;
        }

        // Return the initial subcontext values
        return initialSubcontextValues;
    }

    /**
     * Builds a subroutine context from a set of initial values.
     * 
     * @param initialValues The initial values for the subroutine context
     * @param subroutine The subroutine to build the context for
     * @param parentContext The parent context for the subroutine
     * @param userData The user data for the run
     * @returns A new subroutine context
     */
    static buildSubroutineContext(
        initialValues: IOMap,
        subroutine: RoutineVersion,
        parentContext: SubroutineContext,
        userData: RunTriggeredBy,
    ): SubroutineContext {
        // Get information about the subroutine and use it to build the currentTask
        const { description, instructions, name } = getTranslation(subroutine, userData.languages, true);
        const currentTask: SubroutineContext["currentTask"] = {
            description: description ?? "",
            instructions: instructions ?? "",
            name: name ?? "",
        };

        // Get the overall task for the subroutine
        const overallTask: SubroutineContext["overallTask"] = parentContext.overallTask ?? parentContext.currentTask;

        // Build the subroutine context
        const subroutineContext: SubroutineContext = {
            ...SubroutineContextManager.initializeContext(initialValues),
            currentTask,
            overallTask,
        };

        // Return the subroutine context
        return subroutineContext;
    }

    /**
     * Helper method to convert inputs and outputs returned by `SubroutineExecutor.run` to the correct format - 
     * a composite key of the form `${nodeId}.${ioName}`.
     * 
     * Functionally, this is similar to `prepareMultiStepSubroutineContext`. That method goes from an object keyed by 
     * subroutine input names to and object keyed by child context keys. Here, we go from an object keyed by subroutine input/output 
     * names to an object keyed by PARENT context keys.
     * 
     * @param subroutineResult The result of `SubroutineExecutor.run`, containing inputs and outputs
     * @param nodeId The ID of the node that the inputs belong to
     * @param parentRoutine The parent routine (from the current location’s object)
     * @param logger The logger to use for logging
     * @returns The converted inputs or outputs
     */
    static mapSubroutineResultToParentKeys(
        subroutineResult: Pick<RunSubroutineResult, "inputs" | "outputs">,
        nodeId: string,
        parentRoutine: RoutineVersion,
        logger: PassableLogger,
    ): Pick<RunSubroutineResult, "inputs" | "outputs"> | null {
        // Deserialize the parent's graph configuration.
        const { graph: parentGraph } = RoutineVersionConfig.parse(parentRoutine, logger, { useFallbacks: true });
        if (!parentGraph) {
            logger.error("mapSubroutineResultToParentKeys: Invalid parent graph configuration.");
            return null;
        }

        // Get the node's maps for node input -> subroutine input and node output -> subroutine output
        const nodeInputNameToSubroutineInputNameMap = parentGraph.getIONamesToSubroutineInputNames(nodeId);
        const nodeOutputNameToSubroutineInputNameMap = parentGraph.getIONamesToSubroutineOutputNames(nodeId);

        // Reverse the maps so we can go from subroutine input/output names to node input/output names
        const subroutineInputNameToNodeInputNameMap = Object.fromEntries(
            Object.entries(nodeInputNameToSubroutineInputNameMap).map(([nodeInputName, subroutineInputName]) => [subroutineInputName, nodeInputName]),
        );
        const subroutineOutputNameToNodeOutputNameMap = Object.fromEntries(
            Object.entries(nodeOutputNameToSubroutineInputNameMap).map(([nodeOutputName, subroutineOutputName]) => [subroutineOutputName, nodeOutputName]),
        );

        // Initialize the result
        const result: Pick<RunSubroutineResult, "inputs" | "outputs"> = {
            inputs: {},
            outputs: {},
        };

        // Convert the inputs
        for (const [subroutineInputName, value] of Object.entries(subroutineResult.inputs)) {
            const nodeInputName = subroutineInputNameToNodeInputNameMap[subroutineInputName];
            if (!nodeInputName) {
                continue;
            }
            // Add the value to the result as a composite key
            result.inputs[`${nodeId}.${nodeInputName}`] = value;
        }

        // Convert the outputs
        for (const [subroutineOutputName, value] of Object.entries(subroutineResult.outputs)) {
            const nodeOutputName = subroutineOutputNameToNodeOutputNameMap[subroutineOutputName];
            if (!nodeOutputName) {
                continue;
            }
            // Add the value to the result as a composite key
            result.outputs[`${nodeId}.${nodeOutputName}`] = value;
        }

        // Return the converted inputs and outputs
        return result;
    }

    //TODO this might need to be done in the server. The server should load the context for the branch, trim it down to fit in the context window, and then send it to the AI.
    // /**
    //  * Exports the subroutine context to a JSON object, to be used in AI responses.
    //  * 
    //  * Adds the most important information first, then adds additional context until 
    //  * we reach the context size limit.
    //  * 
    //  * @param context The current subroutine context
    //  * @param runConfig The run configuration to use for exporting the context, including user-defined context size limits
    //  * @param aiConfig The AI configuration to use for exporting the context, including model-defined context size limits
    //  * @returns The exported context
    //  */
    // public exportContext(
    //     context: SubroutineContext,
    //     runConfig: RunConfig,
    //     aiConfig: ModelInfo
    // ): Record<string, unknown> {
    //     // Determine AI context size limit
    //     const aiContextSizeLimit = aiConfig.contextWindow
    // }
}
