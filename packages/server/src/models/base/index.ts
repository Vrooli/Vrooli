import { GqlModelType } from "@local/shared";
import { logger } from "../../events/logger";
import { ModelLogic } from "../types";

type ObjectMap = { [key in GqlModelType]: ModelLogic<any, any, any> | Record<string, never> };

export class ObjectMapSingleton {
    private static instance: ObjectMapSingleton;

    public map: ObjectMap = {} as ObjectMap;

    // eslint-disable-next-line @typescript-eslint/no-empty-function
    private constructor() { }

    private async initializeMap() {
        const modelNames = Object.keys(GqlModelType) as (keyof typeof ObjectMapSingleton.prototype.map)[];
        for (const modelName of modelNames) {
            try {
                this.map[modelName] = (await import(`./${modelName.toLowerCase()}`))[`${modelName}Model`];
            } catch (error) {
                this.map[modelName] = {};
            }
        }
    }

    public static getInstance(): ObjectMapSingleton {
        if (!ObjectMapSingleton.instance) {
            ObjectMapSingleton.instance = new ObjectMapSingleton();
            logger.info("ObjectMapSingleton was never initialized. Returning an empty instance.", { trace: "0003" });
        }
        return ObjectMapSingleton.instance;
    }

    public static async init() {
        if (!ObjectMapSingleton.instance) {
            ObjectMapSingleton.instance = new ObjectMapSingleton();
        }
        await ObjectMapSingleton.instance.initializeMap();
    }
}
