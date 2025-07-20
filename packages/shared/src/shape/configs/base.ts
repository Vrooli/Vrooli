import { type PassableLogger } from "../../consts/commonTypes.js";
import { LATEST_CONFIG_VERSION } from "./utils.js";


export enum ResourceUsedFor {
    Community = "Community",
    Context = "Context",
    Developer = "Developer",
    Donation = "Donation",
    ExternalService = "ExternalService",
    Feed = "Feed",
    Install = "Install",
    Learning = "Learning",
    Notes = "Notes",
    OfficialWebsite = "OfficialWebsite",
    Proposal = "Proposal",
    Related = "Related",
    Researching = "Researching",
    Scheduling = "Scheduling",
    Social = "Social",
    Tutorial = "Tutorial"
}

/**
 * Model selection strategies for AI providers
 */
export enum ModelStrategy {
    /** Always use preferredModel, fail if unavailable */
    FIXED = "fixed",
    /** Use preferredModel, then fallback chain */
    FALLBACK = "fallback",
    /** Cheapest available model */
    COST_OPTIMIZED = "cost",
    /** Best available model */
    QUALITY_FIRST = "quality",
    /** Prefer local models, fallback to cloud */
    LOCAL_FIRST = "local"
}

/**
 * Configuration for AI model selection
 */
export interface ModelConfig {
    /** How to select models */
    strategy: ModelStrategy;
    /** Default model to use */
    preferredModel?: string;
    /** Force local models only */
    offlineOnly: boolean;
}

/**
 * Resource definition for storing links and external data
 */
export interface ConfigResource {
    /** Link to the resource */
    link: string;
    /** Purpose of the resource */
    usedFor?: ResourceUsedFor | `${ResourceUsedFor}`;
    /** Display info */
    translations: {
        language: string;
        name: string;
        description?: string;
    }[];
}

/**
 * Base configuration object for all entity types
 */
export interface BaseConfigObject {
    /** Store the version number for future compatibility */
    __version: string;
    /** Resources attached to this entity */
    resources?: ConfigResource[];
}

/**
 * Base configuration class that all entity config classes extend
 */
export class BaseConfig<T extends BaseConfigObject = BaseConfigObject> {
    __version: string;
    resources: ConfigResource[];

    constructor({ config }: { config: T }) {
        this.__version = config.__version ?? LATEST_CONFIG_VERSION;
        this.resources = config.resources ?? [];
    }

    /**
     * Helper for subclasses to parse stringified config, auto-fill version/resources/etc., and forward to a factory.
     */
    protected static parseBase<T extends BaseConfigObject, C>(
        data: T | null | undefined,
        _logger: PassableLogger,
        factory: (params: { config: T }) => C,
    ): C {
        const full = {
            __version: data?.__version ?? LATEST_CONFIG_VERSION,
            resources: data?.resources ?? [],
            ...data,
        } as T;
        return factory({ config: full });
    }

    /**
     * Exports the config to a plain object
     */
    export(): T {
        const baseConfig: BaseConfigObject = {
            __version: this.__version,
            resources: this.resources,
        };
        return baseConfig as T;
    }

    /**
     * Adds a resource to the config
     */
    addResource(resource: ConfigResource): void {
        this.resources.push(resource);
    }

    /**
     * Removes a resource from the config
     */
    removeResource(index: number): void {
        this.resources = this.resources.filter((_, i) => i !== index);
    }

    /**
     * Updates a resource in the config
     */
    updateResource(index: number, updates: Partial<ConfigResource>): void {
        this.resources = this.resources.map((resource, i) =>
            i === index
                ? { ...resource, ...updates }
                : resource,
        );
    }

    /**
     * Gets resources by their usage type
     */
    getResourcesByType(usedFor: ResourceUsedFor): ConfigResource[] {
        return this.resources.filter(resource => resource.usedFor === usedFor);
    }
}
