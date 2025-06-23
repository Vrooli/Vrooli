import { ResourceSubType, type ResourceVersion } from "../../api/types.js";
import { type PassableLogger } from "../../consts/commonTypes.js";
import { BaseConfig, type BaseConfigObject } from "./base.js";

const LATEST_CONFIG_VERSION = "1.0";

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
    /** Schema for the standard (typically JSON Schema) */
    schema?: string;
    /** What the schema is written in */
    schemaLanguage?: string;
    /** Props that are used to generate the standard */
    props?: Record<string, unknown>;
}

/**
 * Top-level Standard config that encapsulates all standard-related configuration data.
 */
export class StandardVersionConfig extends BaseConfig<StandardVersionConfigObject> {
    validation?: StandardVersionConfigObject["validation"];
    format?: StandardVersionConfigObject["format"];
    compatibility?: StandardVersionConfigObject["compatibility"];
    compliance?: StandardVersionConfigObject["compliance"];
    schema?: StandardVersionConfigObject["schema"];
    schemaLanguage?: StandardVersionConfigObject["schemaLanguage"];
    props?: StandardVersionConfigObject["props"];

    resourceSubType: ResourceSubType;

    constructor({ config, resourceSubType }: { config: StandardVersionConfigObject, resourceSubType: ResourceSubType }) {
        super({ config });
        this.__version = config.__version ?? LATEST_CONFIG_VERSION;
        this.validation = config.validation;
        this.format = config.format;
        this.compatibility = config.compatibility;
        this.compliance = config.compliance;
        this.schema = config.schema;
        this.schemaLanguage = config.schemaLanguage;
        this.props = config.props;

        this.resourceSubType = resourceSubType;
    }


    static parse(
        version: Pick<ResourceVersion, "config" | "resourceSubType">,
        logger: PassableLogger,
        _opts?: { useFallbacks?: boolean },
    ): StandardVersionConfig {
        return super.parseBase<StandardVersionConfigObject, StandardVersionConfig>(
            version.config,
            logger,
            ({ config }) => {
                // Add fallback properties as needed
                return new StandardVersionConfig({ config, resourceSubType: version.resourceSubType ?? ResourceSubType.StandardDataStructure });
            },
        );
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
            schema: this.schema,
            schemaLanguage: this.schemaLanguage,
            props: this.props,
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
    setSchema(schema: StandardVersionConfigObject["schema"]): void {
        this.schema = schema;
    }

    /**
     * Sets the language of the schema
     */
    setSchemaLanguage(schemaLanguage: StandardVersionConfigObject["schemaLanguage"]): void {
        this.schemaLanguage = schemaLanguage;
    }
} 
