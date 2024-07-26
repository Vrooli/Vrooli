import { InputType, PassableLogger, RoutineType } from "@local/shared";
import { FormSchema } from "forms/types";
import i18next from "i18next";
import { BotStyle, ConfigCallData, ConfigCallDataGenerate } from "views/objects/routine/RoutineTypeForms/RoutineTypeForms";

/**
 * @returns The default input form object for a routine. This is a function 
 * to avoid using thes same object in multiple places, which can lead to
 * unexpected behavior if the object is modified.
 */
export function defaultSchemaInput(): FormSchema {
    return { containers: [], elements: [] };
}

/**
 * @returns The default output form object for a routine. This is a function
 * to avoid using the same object in multiple places, which can lead to
 * unexpected behavior if the object is modified.
 */
export function defaultSchemaOutput(): FormSchema {
    return { containers: [], elements: [] };
}

/**
 * @return The default output form object for a Generate routine, 
 * which always returns text (for now, since we only call LLMs)
 */
export function defaultSchemaOutputGenerate(): FormSchema {
    return {
        containers: [],
        elements: [
            {
                fieldName: "response",
                id: "response",
                label: i18next.t("Response"),
                props: {
                    placeholder: "Model response will be displayed here",
                },
                type: InputType.Text,
            },
        ],
    };
}

export const defaultConfigFormInputMap: { [key in RoutineType]: (() => FormSchema) } = {
    [RoutineType.Action]: () => defaultSchemaInput(),
    [RoutineType.Api]: () => defaultSchemaInput(),
    [RoutineType.Code]: () => defaultSchemaInput(),
    [RoutineType.Data]: () => defaultSchemaInput(),
    [RoutineType.Generate]: () => defaultSchemaInput(),
    [RoutineType.Informational]: () => defaultSchemaInput(),
    [RoutineType.MultiStep]: () => defaultSchemaInput(),
    [RoutineType.SmartContract]: () => defaultSchemaInput(),
};

export const defaultConfigFormOutputMap: { [key in RoutineType]: (() => FormSchema) } = {
    [RoutineType.Action]: () => defaultSchemaOutput(),
    [RoutineType.Api]: () => defaultSchemaOutput(),
    [RoutineType.Code]: () => defaultSchemaOutput(),
    [RoutineType.Data]: () => defaultSchemaOutput(),
    [RoutineType.Generate]: () => defaultSchemaOutputGenerate(),
    [RoutineType.Informational]: () => defaultSchemaOutput(),
    [RoutineType.MultiStep]: () => defaultSchemaOutput(),
    [RoutineType.SmartContract]: () => defaultSchemaOutput(),
};

export function parseSchema(
    value: unknown,
    defaultSchemaFn: () => FormSchema,
    logger: PassableLogger,
    schemaType: string,
): FormSchema {
    let parsedValue: FormSchema;

    try {
        parsedValue = JSON.parse(typeof value === "string" ? value : JSON.stringify(value));
    } catch (error) {
        logger.error(`Error parsing schema ${schemaType}: ${JSON.stringify(error)}`);
        parsedValue = defaultSchemaFn();
    }

    // Ensure the parsed value contains `containers` and `elements` arrays
    if (typeof parsedValue !== "object" || parsedValue === null) {
        parsedValue = defaultSchemaFn();
    }
    if (!Array.isArray(parsedValue.containers)) {
        parsedValue.containers = [];
    }
    if (!Array.isArray(parsedValue.elements)) {
        parsedValue.elements = [];
    }

    return parsedValue;
}

export function parseSchemaInput(
    value: unknown,
    routineType: RoutineType | null | undefined,
    logger: PassableLogger,
): FormSchema {
    return parseSchema(value, () => {
        if (!routineType) return defaultSchemaInput();
        return defaultConfigFormInputMap[routineType]();
    }, logger, "input");
}

export function parseSchemaOutput(
    value: unknown,
    routineType: RoutineType | null | undefined,
    logger: PassableLogger,
): FormSchema {
    return parseSchema(value, () => {
        if (!routineType) return defaultSchemaOutput();
        return defaultConfigFormOutputMap[routineType]();
    }, logger, "output");
}

export function defaultConfigCallData(): ConfigCallData {
    return {};
}

export function defaultConfigCallDataGenerate(): ConfigCallDataGenerate {
    return {
        botStyle: BotStyle.Default,
        maxTokens: null,
        model: null,
        respondingBot: null,
    };
}

export const defaultConfigCallDataMap: { [key in RoutineType]: (() => ConfigCallData) } = {
    [RoutineType.Action]: () => defaultConfigCallData(),
    [RoutineType.Api]: () => defaultConfigCallData(),
    [RoutineType.Code]: () => defaultConfigCallData(),
    [RoutineType.Data]: () => defaultConfigCallData(),
    [RoutineType.Generate]: () => defaultConfigCallDataGenerate(),
    [RoutineType.Informational]: () => defaultConfigCallData(),
    [RoutineType.MultiStep]: () => defaultConfigCallData(),
    [RoutineType.SmartContract]: () => defaultConfigCallData(),
};

export function parseConfigCallData(
    value: unknown,
    routineType: RoutineType | null | undefined,
    logger: PassableLogger,
): ConfigCallData {
    let parsedValue: ConfigCallData;
    const defaultConfigCallData = routineType ? (defaultConfigCallDataMap[routineType] || {}) : {};

    try {
        parsedValue = JSON.parse(typeof value === "string" ? value : JSON.stringify(value));
    } catch (error) {
        logger.error(`Error parsing configCallData: ${JSON.stringify(error)}`);
        parsedValue = defaultConfigCallData;
    }

    if (typeof parsedValue !== "object" || parsedValue === null) {
        parsedValue = defaultConfigCallData;
    }

    return parsedValue;
}