import { RunInputCreateInput, RunInputUpdateInput } from "graphql/generated/globalTypes";
import { Routine, RunInput } from "types";
import { v4 as uuid } from "uuid";
import { hasObjectChanged, shapeRunInputCreate, shapeRunInputUpdate } from "./shape";
import { shapeUpdateList } from "./shape/shapeTools";

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
    if ((routine as any).nodesCount && (routine as any).nodesCount > 0) return true;
    // Complexity is calculated from nodes and inputs, so a complexity > the number of inputs + 1
    // indicates that the routine has multiple steps
    if (routine.complexity && routine.inputs) return routine.complexity > routine.inputs.length + 1;
    // Can do the same with simplicity, if complexity not provided
    if (routine.simplicity && routine.inputs) return routine.simplicity > routine.inputs.length + 1;
    // If these cases fail, there is no other information we can use to determine if the routine has subroutines
    return false;
}

/**
 * Converts formik into object with run input data
 * @param values The formik values object
 * @returns object where keys are inputIds, and values are the run input data
 */
export const formikToRunInputs = (values: { [x: string]: string }): { [x: string]: string } => {
    const result: { [x: string]: string } = {};
    // Get user inputs, and ignore empty values and blank strings.
    const inputValues = Object.entries(values).filter(([key, value]) =>
        key.startsWith('inputs-') &&
        typeof value === 'string' && 
        value.length > 0);
    // Input keys are in the form of inputs-<inputId>. We need the inputId
    for (const [key, value] of inputValues) {
        const inputId = key.substring(key.indexOf('-')+1)
        result[inputId] = JSON.stringify(value);
    }
    return result;
}

/**
 * Updates formik values with run input data
 * @param runInputs The run input data
 * @returns Object to pass into formik setValues function
 */
export const runInputsToFormik = (runInputs: RunInput[]): { [x: string]: string } => {
    const result: { [x: string]: string } = {};
    for (const runInput of runInputs) {
        result[`inputs-${runInput.input.id}`] = JSON.parse(runInput.data);
    }
    return result;
}

/**
 * Converts a run inputs object to a run input create object
 * @param runInputs The run inputs object
 * @returns The run input create input object
 */
export const runInputsCreate = (runInputs: { [x: string]: string }): { inputsCreate: RunInputCreateInput[] } => {
    return {
        inputsCreate: Object.entries(runInputs).map(([inputId, data]) => ({
            id: uuid(),
            data,
            inputId
        }))
    }
}

/**
 * Converts a run inputs object to a run input update object
 * @param original RunInputs[] array of existing run inputs data
 * @param updated Run input object with updated data
 * @returns The run input update input object
 */
export const runInputsUpdate = (
    original: RunInput[],
    updated: { [x: string]: string },
): { 
    inputsCreate?: RunInputCreateInput[],
    inputsUpdate?: RunInputUpdateInput[],
    inputsDelete?: string[],
} => {
    // Convert user inputs object to RunInput[]
    const updatedInputs = Object.entries(updated).map(([inputId, data]) => ({
        id: uuid(),
        data,
        input: { id: inputId }
    }))
    // Loop through original run inputs. Where input id is in updated inputs, update id
    for (const currOriginal of original) {
        const currUpdated = updatedInputs.findIndex((input) => input.input.id === currOriginal.input.id);
        if (currUpdated !== -1) {
            updatedInputs[currUpdated].id = currOriginal.id;
        }
    }
    return shapeUpdateList({ inputs: original }, { inputs: updatedInputs }, 'inputs', hasObjectChanged, shapeRunInputCreate, shapeRunInputUpdate, 'id');
}