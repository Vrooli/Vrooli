import { GqlModelType, lowercaseFirstLetter } from "@local/shared";
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";
import { CustomError } from "../../events/error";
import { logger } from "../../events/logger";
import { Displayer, Duplicator, Formatter, ModelLogic, Mutater, Searcher, Validator } from "../types";

export type GenericModelLogic = ModelLogic<any, any, any>;
type ObjectMap = { [key in GqlModelType]: GenericModelLogic | Record<string, never> };
type LogicProps = "dbTable" | "dbTranslationTable" | "display" | "duplicate" | "format" | "idField" | "mutate" | "search" | "validate";
type GetLogicReturn<
    Logic extends LogicProps,
> = {
    dbTable: "dbTable" extends Logic ? string : never,
    dbTranslationTable: "dbTranslationTable" extends Logic ? string : never,
    display: "display" extends Logic ? Displayer<any> : never,
    duplicate: "duplicate" extends Logic ? Duplicator<any, any> : never,
    format: "format" extends Logic ? Formatter<any> : never,
    idField: string,
    mutate: "mutate" extends Logic ? Mutater<any> : never,
    search: "search" extends Logic ? Searcher<any, any> : never,
    validate: "validate" extends Logic ? Validator<any> : never,
}
type Undefinable<T> = {
    [K in keyof T]: T[K] | undefined;
}

function getCallerFunctionName(): string {
    try {
        throw new Error();
    } catch (e) {
        if (e instanceof Error) {
            const stack = e.stack?.split("\n");
            if (!stack || stack.length < 4) return "unknown";
            // The current function is at stack[1], so the caller is at stack[2]
            const match = stack[3].match(/at (\S+)/);
            return match && match[1] || "unknown";
        }
        return "unknown";
    }
}

export class ModelMap {
    private static instance: ModelMap;

    public map: ObjectMap = {} as ObjectMap;

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }

    private async initializeMap() {
        const modelNames = Object.keys(GqlModelType) as (keyof typeof ModelMap.prototype.map)[];
        const dirname = path.dirname(fileURLToPath(import.meta.url));

        // Create a promise for each model import and process them in parallel
        const importPromises = modelNames.map(async (modelName) => {
            const fileName = lowercaseFirstLetter(modelName);
            const modelPath = `./${fileName}`;
            const filePathWithoutExtension = `${dirname}/${fileName}`;
            try {
                if (fs.existsSync(`${filePathWithoutExtension}.js`) || fs.existsSync(`${filePathWithoutExtension}.ts`)) {
                    this.map[modelName] = (await import(modelPath))[`${modelName}Model`];
                } else {
                    this.map[modelName] = {};
                }
                // If the model has a translation table, add that to the map as well
                const translationTable = this.map[modelName].dbTranslationTable;
                if (translationTable) {
                    const __typename = this.map[modelName].__typename + "Translation";
                    this.map[__typename] = {
                        __typename,
                        dbTable: translationTable,
                        idField: "id",
                    };
                }
            } catch (error) {
                this.map[modelName] = {};
                logger.warning(`Failed to load model ${modelName}Model at ${modelPath}. There is likely a circular dependency. Try changing all imports to be relative`, { trace: "0202", error });
            }
        });

        // Wait for all promises to settle
        await Promise.all(importPromises);
    }

    public static isModel(objectType: GqlModelType | `${GqlModelType}`): boolean {
        if (!ModelMap.instance) {
            const caller = getCallerFunctionName();
            throw new Error(`ModelMap was never initialized by caller ${caller}`);
        }
        const model = ModelMap.instance.map[objectType];
        return model && typeof model === "object" && Object.prototype.hasOwnProperty.call(model, "dbTable");
    }

    public static get<
        T extends GenericModelLogic,
        ThrowError extends boolean = true,
    >(
        objectType: GqlModelType | `${GqlModelType}` | undefined,
        throwErrorIfNotFound: ThrowError = true as ThrowError,
        errorTrace?: string,
    ): ThrowError extends true ? T : (T | undefined) {
        if (!objectType) {
            if (throwErrorIfNotFound) {
                const caller = errorTrace ?? getCallerFunctionName();
                throw new CustomError("0023", "InternalError", { caller });
            } else {
                return {} as ThrowError extends true ? T : undefined;
            }
        }
        const isModelObject = this.isModel(objectType);
        if (!isModelObject && throwErrorIfNotFound) {
            const caller = errorTrace ?? getCallerFunctionName();
            throw new CustomError("0024", "InternalError", { caller, objectType });
        }
        return (isModelObject ? ModelMap.instance.map[objectType] : undefined) as ThrowError extends true ? T : (T | undefined);
    }

    public static getLogic<
        Logic extends readonly LogicProps[],
        ThrowError extends boolean = true,
    >(
        props: Logic,
        objectType: `${GqlModelType}`,
        throwErrorIfNotFound: ThrowError = true as ThrowError,
        errorTrace?: string,
    ): ThrowError extends true ? GetLogicReturn<Logic[number]> : Undefinable<GetLogicReturn<Logic[number]>> {
        const object = ModelMap.get(objectType, throwErrorIfNotFound);
        if (!object) return {} as ThrowError extends true ? GetLogicReturn<Logic[number]> : Undefinable<GetLogicReturn<Logic[number]>>;
        // Loop through requested types to validate that all requested types exist
        for (const field of props) {
            // Get logic function
            let logic = object[field];
            // If this is for "idField" and it doesn't exist, default to "id"
            if (field === "idField" && !logic) {
                logic = "id";
                object[field] = logic;
            }
            // Make sure logic function exists
            if (!logic && throwErrorIfNotFound) {
                const caller = errorTrace ?? getCallerFunctionName();
                throw new CustomError("0367", "InternalError", { caller, objectType, field });
            }
        }
        return object as unknown as GetLogicReturn<Logic[number]>;
    }

    public static async init() {
        if (!ModelMap.instance) {
            ModelMap.instance = new ModelMap();
        }
        await ModelMap.instance.initializeMap();
    }

    public static isInitialized() {
        return !!ModelMap.instance;
    }
}
