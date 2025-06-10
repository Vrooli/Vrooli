/**
 * Enums extracted from run/types.ts to break circular dependencies
 */

/** 
 * How subroutine inputs should be generated.
 */
export enum InputGenerationStrategy {
    /** Let the AI generate the inputs */
    Auto = "Auto",
    /** Let the user provide the inputs */
    Manual = "Manual"
}

/**
 * The type of path selection strategy to use for the run, 
 * when we're in a situation where there are multiple outgoing branches 
 * and we can only pick one or a subset of them.
 */
export enum PathSelectionStrategy {
    /** Pick the first available branch */
    AutoPickFirst = "AutoPickFirst",
    /** Use an AI model to decide which branch to pick */
    AutoPickLLM = "AutoPickLLM",
    /** Pick a random branch */
    AutoPickRandom = "AutoPickRandom",
    /** Let the user decide which branch to pick */
    ManualPick = "ManualPick"
}

/**
 * How subroutines should be executed.
 */
export enum SubroutineExecutionStrategy {
    /** Run the subroutine as soon as the inputs are generated */
    Auto = "Auto",
    /** Make the user press a button to run the subroutine */
    Manual = "Manual"
}

/**
 * Determines which bot personas to use for a routine.
 */
export enum BotStyle {
    // The default bot
    Default = "Default",
    // Will use ConfigCallDataGenerate.respondingBot
    Specific = "Specific",
    // Don't use a bot
    None = "None",
}

/** The status of a run branch. */
export enum BranchStatus {
    /** The branch is currently running with no blockers. */
    Active = "Active",
    /** The branch has completed successfully. */
    Completed = "Completed",
    /** The branch has failed. */
    Failed = "Failed",
    /** The branch is waiting for some condition to be met before continuing. */
    Waiting = "Waiting",
}