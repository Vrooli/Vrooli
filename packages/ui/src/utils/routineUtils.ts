import { RoutineType } from "@local/shared";
import { BotStyle, ConfigCallData, ConfigCallDataGenerate } from "views/objects/routine/RoutineTypeForms/RoutineTypeForms";
import { FormSchema } from "../forms/types";

export const defaultSchemaInput = { containers: [], elements: [] };
export const defaultSchemaOutput = { containers: [], elements: [] };

export function parseSchemaInputOutput(value: unknown, defaultValue: FormSchema): FormSchema {
    let parsedValue: FormSchema;

    try {
        parsedValue = JSON.parse(typeof value === "string" ? value : JSON.stringify(value));
    } catch (error) {
        console.error("Error parsing schema", error);
        parsedValue = defaultValue;
    }

    // Ensure the parsed value contains `containers` and `elements` arrays
    if (typeof parsedValue !== "object" || parsedValue === null) {
        parsedValue = defaultValue;
    }
    if (!Array.isArray(parsedValue.containers)) {
        parsedValue.containers = [];
    }
    if (!Array.isArray(parsedValue.elements)) {
        parsedValue.elements = [];
    }

    return parsedValue;
}

export const defaultConfigCallDataGenerate: ConfigCallDataGenerate = {
    botStyle: BotStyle.Default,
    model: null,
    respondingBot: null,
};
const defaultConfigDataMap: { [key in RoutineType]?: ConfigCallData } = {
    [RoutineType.Generate]: defaultConfigCallDataGenerate,
};


export function parseConfigCallData(value: unknown, routineType: RoutineType | null | undefined): ConfigCallData {
    let parsedValue: ConfigCallData;
    const defaultConfigCallData = routineType ? (defaultConfigDataMap[routineType] || {}) : {};

    try {
        parsedValue = JSON.parse(typeof value === "string" ? value : JSON.stringify(value));
    } catch (error) {
        console.error("Error parsing configCallData", error);
        parsedValue = defaultConfigCallData;
    }

    if (typeof parsedValue !== "object" || parsedValue === null) {
        parsedValue = defaultConfigCallData;
    }

    return parsedValue;
}
