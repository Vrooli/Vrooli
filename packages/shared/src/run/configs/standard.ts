import { StandardType } from "../../api/types.js";
import { type PassableLogger } from "../../consts/commonTypes.js";
import { LATEST_RUN_CONFIG_VERSION } from "../consts.js";
import { BaseConfig, BaseConfigObject } from "./baseConfig.js";
import { parseObject, type StringifyMode } from "./utils.js";

const DEFAULT_STRINGIFY_MODE: StringifyMode = "json";

/**
 * Represents all data that can be stored in a standard's stringified config.
 * 
 * This includes configuration that doesn't need to be queried while searching
 * (e.g., validation options, format settings). Properties like standardType
 * are stored directly in the StandardVersion model.
 */
export interface StandardVersionConfigObject extends BaseConfigObject {
    /** Validation settings */
    validation?: {
        /** Whether to enable strict validation */
        strictMode?: boolean;
        /** Custom validation rules */
        rules?: Record<string, unknown>;
        /** Error messages for validation failures */
        errorMessages?: Record<string, string>;
    };
    /** Format settings for display */
    format?: {
        /** Default presentation format */
        defaultFormat?: string;
        /** Custom formatting options */
        options?: Record<string, unknown>;
    };
    /** Compatibility information */
    compatibility?: {
        /** Minimum system requirements */
        minimumRequirements?: Record<string, string>;
        /** Known compatibility issues */
        knownIssues?: string[];
        /** Compatible with standards */
        compatibleWith?: string[];
    };
    /** Compliance information */
    compliance?: {
        /** Standards this complies with */
        compliesWith?: string[];
        /** Certification information */
        certifications?: Array<{
            /** Name of certification */
            name: string;
            /** Issuing organization */
            issuer: string;
            /** Date of certification */
            date?: string;
            /** Expiration date */
            expiration?: string;
        }>;
    };
    /** JSON Schema for the standard */
    jsonSchema?: string;
}

/**
 * Top-level Standard config that encapsulates all standard-related configuration data.
 */
export class StandardVersionConfig extends BaseConfig<StandardVersionConfigObject> {
    validation?: StandardVersionConfigObject["validation"];
    format?: StandardVersionConfigObject["format"];
    compatibility?: StandardVersionConfigObject["compatibility"];
    compliance?: StandardVersionConfigObject["compliance"];
    jsonSchema?: StandardVersionConfigObject["jsonSchema"];

    standardType: StandardType;

    constructor({
        data,
        standardType,
    }: {
        data: StandardVersionConfigObject,
        standardType: StandardType,
    }) {
        super(data);
        this.validation = data.validation;
        this.format = data.format;
        this.compatibility = data.compatibility;
        this.compliance = data.compliance;
        this.jsonSchema = data.jsonSchema;

        this.standardType = standardType;
    }

    /**
     * Creates a StandardVersionConfig from a standard version object
     */
    static createFromStandardVersion(
        standardVersion: {
            data?: string | null;
            standardType: StandardType;
        },
        logger: PassableLogger,
        options: { mode?: StringifyMode } = {},
    ): StandardVersionConfig {
        const mode = options.mode || DEFAULT_STRINGIFY_MODE;
        const obj = standardVersion.data ? parseObject<StandardVersionConfigObject>(standardVersion.data, mode, logger) : null;
        if (!obj) {
            return StandardVersionConfig.default({
                standardType: standardVersion.standardType,
            });
        }
        return new StandardVersionConfig({
            data: obj,
            standardType: standardVersion.standardType,
        });
    }

    /**
     * Creates a default StandardVersionConfig
     */
    static default({
        standardType,
    }: {
        standardType: StandardType,
    }): StandardVersionConfig {
        const data: StandardVersionConfigObject = {
            __version: LATEST_RUN_CONFIG_VERSION,
            resources: [],
            metadata: {},
        };
        return new StandardVersionConfig({
            data,
            standardType,
        });
    }

    /**
     * Exports the config to a plain object
     */
    override export(): StandardVersionConfigObject {
        return {
            ...super.export(),
            validation: this.validation,
            format: this.format,
            compatibility: this.compatibility,
            compliance: this.compliance,
            jsonSchema: this.jsonSchema,
        };
    }

    /**
     * Sets validation settings
     */
    setValidation(validation: StandardVersionConfigObject["validation"]): void {
        this.validation = validation;
    }

    /**
     * Sets format settings
     */
    setFormat(format: StandardVersionConfigObject["format"]): void {
        this.format = format;
    }

    /**
     * Sets compatibility information
     */
    setCompatibility(compatibility: StandardVersionConfigObject["compatibility"]): void {
        this.compatibility = compatibility;
    }

    /**
     * Sets compliance information
     */
    setCompliance(compliance: StandardVersionConfigObject["compliance"]): void {
        this.compliance = compliance;
    }

    /**
     * Sets JSON schema
     */
    setJsonSchema(jsonSchema: StandardVersionConfigObject["jsonSchema"]): void {
        this.jsonSchema = jsonSchema;
    }
} 
