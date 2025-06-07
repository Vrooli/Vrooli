// import { RoutineVersionInput, RoutineVersionOutput, RunRoutineIO, RunRoutineUpdateInput } from "../api/types.js";
// import { PassableLogger } from "../consts/commonTypes.js";
// import { generateInitialValues } from "../forms/defaultGenerator.js";
// import { FormSchema } from "../forms/types.js";
// import { uuid } from "../id/uuid.js";

// type IOKeyObject = { id?: string | null; name?: string | null };

// const FIELD_NAME_DELIMITER = "-";

// export type ExistingInput = Pick<RunRoutineIO, "id" | "data"> & { routineVersionInput: Pick<RoutineVersionInput, "id" | "name"> };
// export type ExistingOutput = Pick<RunRoutineIO, "id" | "data"> & { routineVersionOutput: Pick<RoutineVersionOutput, "id" | "name"> };

// type RunIO = ExistingInput | ExistingOutput;

// export type RunIOUpdateParams<IOType extends "input" | "output"> = {
//     /** Existing run inputs or outputs data */
//     existingIO: IOType extends "input" ? ExistingInput[] : ExistingOutput[],
//     /** Current input data in form */
//     formData: object,
//     /** If we're updating input or output data */
//     ioType: IOType,
//     /** Current routine's inputs or outputs */
//     routineIO: IOType extends "input" ? Pick<RoutineVersionInput, "id" | "name">[] : Pick<RoutineVersionOutput, "id" | "name">[],
//     /** ID of current run */
//     runRoutineId: string;
// }

// export type RunIOUpdateResult = Pick<RunRoutineUpdateInput, "ioCreate" | "ioUpdate" | "ioDelete">

// export class RunIOManager {
//     private logger: PassableLogger;

//     /**
//      * @param logger Logger to log errors
//      */
//     constructor(logger: PassableLogger) {
//         this.logger = logger;
//     }

//     /**
//      * Converts existing run inputs or outputs into an object for populating formik
//      * @param runIOs The run input or output data
//      * @param ioType The type of IO being parsed ('input' or 'output')
//      * @returns Object to pass into formik setValues function
//      */
//     public parseRunIO(
//         runIOs: RunIO[],
//         ioType: "input" | "output",
//     ): object {
//         const result: object = {};

//         if (!Array.isArray(runIOs)) return result;

//         for (const io of runIOs) {
//             const ioData = io[ioType as keyof RunIO] as IOKeyObject | undefined;
//             if (!ioData) continue;
//             const key = this.getIOKey(io[ioType], ioType);
//             if (!key) continue;
//             try {
//                 result[key] = JSON.parse(io.data);
//             } catch (error) {
//                 this.logger.error(`Error parsing ${ioType} data for ${ioData.id}:`, typeof error === "object" && error !== null ? error : {});
//                 // In case of parsing error, use the raw string data
//                 result[key] = io.data;
//             }
//         }

//         return result;
//     }

//     public parseRunInputs(
//         runInputs: ExistingInput[],
//     ): object {
//         return this.parseRunIO(runInputs, "input");
//     }

//     public parseRunOutputs(
//         runOutputs: ExistingOutput[],
//     ): object {
//         return this.parseRunIO(runOutputs, "output");
//     }

//     /**
//      * Converts a run inputs object to a run input update object
//      * @returns The run input update input object
//      */
//     public runIOUpdate<IOType extends "input" | "output">({
//         existingIO,
//         formData,
//         ioType,
//         routineIO,
//         runRoutineId,
//     }: RunIOUpdateParams<IOType>): RunIOUpdateResult {
//         // Initialize result
//         const result: Record<string, object> = {};

//         // Create a map of existing inputs for quick lookup
//         const existingIOMap = new Map<string, RunIO>();
//         existingIO?.forEach((io: ExistingInput | ExistingOutput) => {
//             const ioObject = io[ioType as keyof RunIO] as IOKeyObject;
//             // In case some data is missing, store keys for both name and ID
//             const keyName = this.getIOKey({ name: ioObject?.name }, ioType);
//             const keyId = this.getIOKey({ id: ioObject?.id }, ioType);
//             if (keyName) existingIOMap.set(keyName, io);
//             if (keyId) existingIOMap.set(keyId, io);
//         });

//         // Process each routine input
//         routineIO?.forEach(currRoutineIO => {
//             // Find routine input in form data
//             const keyName = this.getIOKey({ name: currRoutineIO?.name }, ioType);
//             const keyId = this.getIOKey({ id: currRoutineIO?.id }, ioType);
//             const inputData = (keyName ? formData[keyName] : undefined) || (keyId ? formData[keyId] : undefined);
//             if (inputData === undefined) return;

//             // Find existing input if it exists
//             const existingInput = (keyName ? existingIOMap.get(keyName) : undefined) || (keyId ? existingIOMap.get(keyId) : undefined);
//             // Update existing input if found
//             if (existingInput) {
//                 try {
//                     // Data must be stored as a string
//                     const stringifiedData = JSON.stringify(inputData);
//                     // Update existing input if data has changed
//                     if (existingInput.data !== stringifiedData) {
//                         const updateKey = `${ioType as IOType}sUpdate` as const;
//                         if (!result[updateKey]) {
//                             result[updateKey] = [];
//                         }
//                         (result[updateKey] as RunRoutineUpdateInput[]).push({
//                             id: existingInput.id,
//                             data: stringifiedData,
//                         });
//                     }
//                 } catch (error) {
//                     this.logger.error("Error stringifying input data", typeof error === "object" && error !== null ? error : {});
//                 }
//             }
//             // Otherwise, create new input 
//             else {
//                 const createKey = `${ioType as IOType}sCreate` as const;
//                 if (!result[createKey]) {
//                     result[createKey] = [];
//                 }
//                 // Create new input
//                 (result[createKey] as RunRoutineUpdateInput[]).push({
//                     id: uuid(),
//                     // Data must be stored as a string
//                     data: JSON.stringify(inputData),
//                     routineVersionInputConnect: currRoutineIO.id,
//                 });
//             }
//         });

//         // Return result
//         return result;
//     }

//     public runInputsUpdate(params: Omit<RunIOUpdateParams<"input">, "ioType">): RunIOUpdateResult {
//         return this.runIOUpdate({ ...params, ioType: "input" });
//     }

//     public runOutputsUpdate(params: Omit<RunIOUpdateParams<"output">, "ioType">): RunIOUpdateResult {
//         return this.runIOUpdate({ ...params, ioType: "output" });
//     }

//     /**
//      * Creates the formik key for a given input or output object
//      * @param io The input or output object. Both RunRoutineInput/Output and RoutineVersionInput/Output should work
//      * @param ioType The type of IO being sanitized ('input' or 'output')
//      * @returns A sanitized key for the input or output object, or null 
//      * if we couldn't generate a key
//      */
//     public getIOKey(
//         io: IOKeyObject,
//         ioType: "input" | "output",
//     ): string | null {
//         // Prefer to use name, but fall back to ID
//         let identifier = io?.name || io?.id;
//         if (!identifier) {
//             return null;
//         }
//         // Sanitize identifier by replacing any non-alphanumeric characters (except underscores) with underscores
//         identifier = identifier.replace(/[^a-zA-Z0-9_-]/g, "_");
//         // Prepend the ioType to the identifier so we can store both inputs and outputs in one formik object
//         return `${ioType}${FIELD_NAME_DELIMITER}${identifier}`; // Make sure the delimiter is not one of the sanitized characters
//     }

//     /**
//      * Creates the formik field name for a run IO based on a fieldName and prefix
//      * @param fieldName The field name to use
//      * @param prefix The prefix to use
//      * @returns The formik field name
//      */
//     public getFormikFieldName(fieldName: string, prefix?: string): string {
//         return prefix ? `${prefix}${FIELD_NAME_DELIMITER}${fieldName}` : fieldName;
//     }
// }

// type GenerateRoutineInitialValuesProps = {
//     configFormInput: FormSchema | null | undefined,
//     configFormOutput: FormSchema | null | undefined,
//     logger: PassableLogger,
//     runInputs: ExistingInput[] | null | undefined,
//     runOutputs: ExistingOutput[] | null | undefined,
// }

// /**
//  * Combines form schema default values with existing run data
//  * to create Formik initial values for displaying a subroutine.
//  * 
//  * Any run inputs/outputs which don't appear in the form inputs/outpus 
//  * will be ignored, as they're likely for another step.
//  * 
//  * Any form inputs/outputs which don't have a corresponding run and don't have 
//  * default values will be set to the InputType's default value or "" to avoid uncontrolled input warnings.
//  * 
//  * @param configFormInput Form schema for inputs
//  * @param configFormOutput Form schema for outputs
//  * @param runInputs Optional existing run inputs
//  * @param runOutputs Optional existing run outputs
//  * @param logger Logger to log errors
//  * @returns Tnitial values for Formik to display a subroutine
//  */
// export function generateRoutineInitialValues({
//     configFormInput,
//     configFormOutput,
//     logger,
//     runInputs,
//     runOutputs,
// }: GenerateRoutineInitialValuesProps): object {
//     let initialValues: object = {};

//     // Generate initial values from form schema elements (defaults)
//     //NOTE: possibly rewrite these to be part of new routineConfig.ts
//     const inputDefaults = generateInitialValues(configFormInput?.elements, "input");
//     const outputDefaults = generateInitialValues(configFormOutput?.elements, "output");

//     // Add all form data to result
//     initialValues = {
//         ...inputDefaults,
//         ...outputDefaults,
//     };

//     // Collect run data
//     let runData: object = {};
//     if (runInputs) {
//         runData = {
//             ...runData,
//             ...new RunIOManager(logger).parseRunInputs(runInputs),
//         };
//     }
//     if (runOutputs) {
//         runData = {
//             ...runData,
//             ...new RunIOManager(logger).parseRunOutputs(runOutputs),
//         };
//     }

//     // Override initial values with run data
//     for (const key in runData) {
//         if (Object.prototype.hasOwnProperty.call(runData, key) && Object.prototype.hasOwnProperty.call(initialValues, key)) {
//             initialValues[key] = runData[key];
//         }
//     }

//     return initialValues;
// }
export { };
//TODO 1 Probably the main thing needed from here is handling default values. LLM should be able to see them, but not necessarily use them
//TODO 2 Probably need some of the formik logic from here to display the subroutine inputs/outputs in the UI. Note that now we store inputs/outputs in the subcontexts, mapped by instance ID. Each input/output name is a composite key of the form `${nodeId}.${ioName}`.
