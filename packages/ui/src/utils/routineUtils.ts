import { Routine } from "types";

/**
 * Calculates the percentage of the run that has been completed.
 * @param completedComplexity The number of completed steps. Ideally shouldn't exceed the 
 * totalComplexity, since steps shouldn't be counted multiple times.
 * @param totalComplexity The total number of steps.
 * @returns The percentage of the run that has been completed, 0-100.
 */
export const getRunPercentComplete = (
    completedComplexity: number | null | undefined,
    totalComplexity: number | null | undefined,
) => {
    if (!completedComplexity || !totalComplexity || totalComplexity === 0) return 0;
    const percentage = Math.round(completedComplexity as number / totalComplexity * 100);
    return Math.min(percentage, 100);
}

/**
 * Determines if two location arrays match, where a location array is an array of step indices
 * @param locationA The first location array 
 * @param locationB The second location array
 * @return True if the location arrays match, false otherwise
 */
export const locationArraysMatch = (locationA: number[], locationB: number[]): boolean => {
    if (locationA.length !== locationB.length) return false;
    for (let i = 0; i < locationA.length; i++) {
        if (locationA[i] !== locationB[i]) return false;
    }
    return true;
}

/**
 * Determines if a routine has subroutines. This may be simple, depending on 
 * how much information is provided
 * @param routine The routine to check
 * @return True if the routine has subroutines, false otherwise
 */
export const routineHasSubroutines = (routine: Partial<Routine>): boolean => {
    // If routine has nodes or links, we know it has subroutines
    if (routine.nodes && routine.nodes.length > 0) return true;
    if (routine.nodeLinks && routine.nodeLinks.length > 0) return true;
    // Complexity is calculated from nodes and inputs, so a complexity > the number of inputs 
    // indicates that the routine has multiple steps
    if (routine.complexity && routine.inputs) return routine.complexity > routine.inputs.length;
    // Can do the same with simplicity, if complexity not provided
    if (routine.simplicity && routine.inputs) return routine.simplicity > routine.inputs.length;
    // If these cases fail, there is no other information we can use to determine if the routine has subroutines
    return false;
}