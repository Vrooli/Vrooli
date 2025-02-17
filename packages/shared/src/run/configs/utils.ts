import { PassableLogger } from "../../consts/commonTypes.js";

export type StringifyMode = "json";// Add more as needed. E.g. "yaml", "xml"

export function stringifyObject(data: object, mode: StringifyMode): string {
    switch (mode) {
        case "json":
            return JSON.stringify(data);
        default:
            throw new Error(`Unsupported stringify mode: ${mode}`);
    }
}

export function parseObject<T extends object>(data: string, mode: StringifyMode, logger: PassableLogger): T | null {
    try {
        switch (mode) {
            case "json":
                return JSON.parse(data);
            default:
                throw new Error(`Unsupported parse mode: ${mode}`);
        }
    } catch (error) {
        logger.error(`Error parsing data: ${JSON.stringify(error)}`);
        return null;
    }
}
