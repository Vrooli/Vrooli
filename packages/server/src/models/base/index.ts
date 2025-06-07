import { ModelType, lowercaseFirstLetter } from "@vrooli/shared";
import fs from "fs";
import path from "path";
import { fileURLToPath } from "url";
import { CustomError } from "../../events/error.js";
import { logger } from "../../events/logger.js";
import type { Danger, Displayer, Duplicator, Formatter, ModelLogic, Mutater, Searcher, Validator } from "../types.js";

export type GenericModelLogic = ModelLogic<any, any, any>;
type ObjectMap = { [key in ModelType]: GenericModelLogic | Record<string, never> };
type LogicProps = "danger" | "dbTable" | "dbTranslationTable" | "display" | "duplicate" | "format" | "idField" | "mutate" | "search" | "validate";
type GetLogicReturn<
    Logic extends LogicProps,
> = {
    danger: "danger" extends Logic ? Danger : never,
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
        const modelNames = Object.keys(ModelType) as (keyof typeof ModelMap.prototype.map)[];
        const dirname = path.dirname(fileURLToPath(import.meta.url));

        // Only include models with existing logic files
        const validModelNames = modelNames.filter(modelName => {
            const fileName = lowercaseFirstLetter(modelName);
            const jsExists = fs.existsSync(`${dirname}/${fileName}.js`);
            const tsExists = fs.existsSync(`${dirname}/${fileName}.ts`);
            return jsExists || tsExists;
        });

        // Dynamically import each model module in parallel
        const importPromises = validModelNames.map(async (modelName) => {
            const fileName = lowercaseFirstLetter(modelName);
            const filePathWithoutExtension = `${dirname}/${fileName}`;
            const importPath = fs.existsSync(`${filePathWithoutExtension}.js`)
                ? `./${fileName}.js`
                : `./${fileName}.ts`;
            try {
                const module = await import(importPath);
                this.map[modelName] = module[`${modelName}Model`] || {};

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
                logger.warning(`Failed to load model ${modelName}Model at ${importPath}. There is likely a circular dependency. Try changing all imports to be relative`, { trace: "0202", error });
            }
        });

        // Wait for all promises to settle
        await Promise.all(importPromises);
    }

    public static isModel(objectType: ModelType | `${ModelType}`): boolean {
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
        objectType: ModelType | `${ModelType}` | undefined,
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
            console.log("0024 error", { caller, objectType });
            throw new CustomError("0024", "InternalError", { caller, objectType });
        }
        return (isModelObject ? ModelMap.instance.map[objectType] : undefined) as ThrowError extends true ? T : (T | undefined);
    }

    public static getLogic<
        Logic extends readonly LogicProps[],
        ThrowError extends boolean = true,
    >(
        props: Logic,
        objectType: `${ModelType}`,
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
