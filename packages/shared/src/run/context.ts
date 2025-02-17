import { IOMap, type SubroutineContext } from "./types.js";

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
