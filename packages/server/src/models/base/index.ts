import { GqlModelType, lowercaseFirstLetter } from "@local/shared";
import { PrismaDelegate } from "../../builders/types";
import { CustomError } from "../../events/error";
import { PrismaType } from "../../types";
import { Displayer, Duplicator, Formatter, ModelLogic, Mutater, Searcher, Validator } from "../types";

export type GenericModelLogic = ModelLogic<any, any, any>;
type ObjectMap = { [key in GqlModelType]: GenericModelLogic | Record<string, never> };
type LogicProps = "delegate" | "display" | "duplicate" | "format" | "idField" | "mutate" | "search" | "validate";
type GetLogicReturn<
    Logic extends LogicProps,
> = {
    delegate: "delegate" extends Logic ? ((prisma: PrismaType) => PrismaDelegate) : never,
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
        for (const modelName of modelNames) {
            try {
                this.map[modelName] = (await import(`./${lowercaseFirstLetter(modelName)}`))[`${modelName}Model`];
            } catch (error) {
                this.map[modelName] = {};
            }
        }
    }

    public static isModel(objectType: GqlModelType | `${GqlModelType}`): boolean {
        if (!ModelMap.instance) {
            const caller = getCallerFunctionName();
            throw new Error(`ModelMap was never initialized by caller ${caller}`);
        }
        const model = ModelMap.instance.map[objectType];
        return model && typeof model === "object" && Object.prototype.hasOwnProperty.call(model, "delegate");
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
                throw new CustomError("0023", "InternalError", ["en"], { caller });
            } else {
                return {} as ThrowError extends true ? T : undefined;
            }
        }
        const isModelObject = this.isModel(objectType);
        if (!isModelObject && throwErrorIfNotFound) {
            const caller = errorTrace ?? getCallerFunctionName();
            throw new CustomError("0024", "InternalError", ["en"], { caller, objectType });
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
                throw new CustomError("0367", "InternalError", ["en"], { caller, objectType, field });
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
}
