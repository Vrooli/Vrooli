import { RoutineType, RoutineVersion } from "../api/types.js";
import { IOMap, RunBotConfig, RunConfig, SubroutineContext } from "./types.js";

/**
 * The result of running a subroutine.
 */
export type RunSubroutineResult = {
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
     * The cost (amount of credits spent as a stringified bigint) of running the routine as a stringified bigint.
     */
    cost: string;
}

/**
 * This executes the actions of a subroutine. 
 * Without this, runs would step through the graph but never actually do anything.
 */
export abstract class SubroutineExecutor {
    /**
     * Determines if the given routine is a single-step routine.
     * 
     * @param routine The routine to check
     * @returns True if the routine is a single-step routine, false otherwise
     */
    public isSingleStepRoutine(routine: RoutineVersion): boolean {
        return routine.routineType !== RoutineType.MultiStep;
    }

    /**
     * Determines if the given routine is a multi-step routine.
     * 
     * @param routine The routine to check
     * @returns True if the routine is a multi-step routine, false otherwise
     */
    public isMultiStepRoutine(routine: RoutineVersion): boolean {
        return routine.routineType === RoutineType.MultiStep;
    }

    /**
     * Generates dummy inputs and outputs for a subroutine.
     * Useful during test mode, when we want to check if a subroutine can run without 
     * calling anything or using any credits.
     * 
     * @param routine The routine to generate inputs and outputs for
     * @param subcontext The current subroutine context, containing any inputs and outputs 
     * we already have data for (and thus don't need to generate)
     */
    private async dummyRun(routine: RoutineVersion, subcontext: SubroutineContext): Promise<RunSubroutineResult> {
        const result: RunSubroutineResult = {
            inputs: {},
            outputs: {},
            cost: BigInt(0).toString(),
        };

        routine.inputs.forEach((input) => {
            // Skip inputs that aren't required
            if (!input.isRequired || !input.name) {
                return;
            }
            // Check if input already exists in the subcontext
            const existingInput = subcontext.allInputsMap[input.name];
            if (existingInput !== undefined) {
                result.inputs[input.name] = existingInput;
            } else {
                // Generate a dummy value for the input
                result.inputs[input.name] = `dummy_value_for_${input.name}`;
            }
        });
        routine.outputs.forEach((output) => {
            if (!output.name) {
                return;
            }
            // Check if output already exists in the subcontext
            const existingOutput = subcontext.allOutputsMap[output.name];
            if (existingOutput !== undefined) {
                result.outputs[output.name] = existingOutput;
            } else {
                // Generate a dummy value for the output
                result.outputs[output.name] = `dummy_value_for_${output.name}`;
            }
        });

        return result;
    }

    /**
     * Runs a subroutine.
     * 
     * NOTE: This should only be called for single-step routines. 
     * Multi-step routines are instead used for navigation.
     * 
     * @param routine The routine to run
     * @param subcontext The current subroutine context (e.g. variables, state), which may be needed to evaluate conditions
     * @param runBotConfig The overall bot configuration for the run
     * @returns The inputs and outputs of the subroutine (for updating the subcontext), as well as the cost of running the routine
     */
    public abstract runSubroutine(routine: RoutineVersion, subcontext: SubroutineContext, runBotConfig: RunBotConfig): Promise<RunSubroutineResult>;

    /**
     * Runs a subroutine.
     * 
     * NOTE 1: This should only be called for single-step routines. 
     * Multi-step routines are instead used for navigation.
     * 
     * NOTE 2: Make sure the context is keyed by subroutine input/output names, as they 
     * appear in the subroutine's RoutineVersionInputs and RoutineVersionOutputs. When 
     * working with multi-step routines, we instead use composite keys of the form `${nodeId}.${ioName}`. 
     * So make sure the right keys are passed in!
     * 
     * @param routine The routine to run
     * @param subcontext The current subroutine context (e.g. variables, state), which may be needed to evaluate conditions
     * @param runConfig The overall run configuration
     * @returns The inputs and outputs of the subroutine (for updating the subcontext), as well as the cost of running the routine
     */
    public run(routine: RoutineVersion, subcontext: SubroutineContext, runConfig: RunConfig): Promise<RunSubroutineResult> {
        // Don't run for multi-step routines
        if (this.isMultiStepRoutine(routine)) {
            throw new Error("Multi-step routines should not be run directly");
        }

        // Perform the real run or a dummy run
        const inTestMode = runConfig.testMode === true;
        const result = inTestMode
            ? this.dummyRun(routine, subcontext)
            : this.runSubroutine(routine, subcontext, runConfig.botConfig);

        return result;
    }

    /**
     * Estimates the max cost of running a subroutine.
     * 
     * @param routineType The type of routine to estimate
     * @param subcontext The current subroutine context (e.g. variables, state), which can increase the cost 
     * (e.g. if passed into an LLM)
     * @returns The maximum cost in credits of running the subroutine
     */
    public abstract estimateCost(routineType: RoutineType, subcontext: SubroutineContext): Promise<bigint>;
}
