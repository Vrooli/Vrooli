import { type PassableLogger } from "../../consts/commonTypes.js";
import { LATEST_CONFIG_VERSION, parseObject, stringifyObject, type StringifyMode } from "./utils.js";

const DEFAULT_STRINGIFY_MODE: StringifyMode = "json";

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
 * Resource definition for storing links and external data
 */
export interface ConfigResource {
    /** Unique identifier for the resource */
    id: string;
    /** Link to the resource */
    link: string;
    /** Position in the list of resources */
    index?: number;
    /** Purpose of the resource */
    usedFor: ResourceUsedFor;
    /** Resource name */
    name?: string;
    /** Resource description */
    description?: string;
}

/**
 * Base configuration object for all entity types
 */
export interface BaseConfigObject {
    /** Store the version number for future compatibility */
    __version: string;
    /** Resources attached to this entity */
    resources?: ConfigResource[];
    /** Additional metadata as key-value pairs */
    metadata?: Record<string, any>;
}

/**
 * Base configuration class that all entity config classes extend
 */
export class BaseConfig<T extends BaseConfigObject = BaseConfigObject> {
    __version: string;
    resources: ConfigResource[];
    metadata: Record<string, any>;

    constructor(data: T) {
        this.__version = data.__version ?? LATEST_CONFIG_VERSION;
        this.resources = data.resources ?? [];
        this.metadata = data.metadata ?? {};
    }

    /**
     * Creates a config instance from a stringified config
     */
    static deserialize<C extends BaseConfig, T extends BaseConfigObject>(
        data: string | null | undefined,
        ConfigClass: new (data: T) => C,
        logger: PassableLogger,
        { mode = DEFAULT_STRINGIFY_MODE }: { mode?: StringifyMode } = {},
    ): C {
        const obj = data ? parseObject<T>(data, mode, logger) : null;
        if (!obj) {
            return new ConfigClass({
                __version: LATEST_CONFIG_VERSION,
                resources: [],
                metadata: {},
            } as unknown as T);
        }
        return new ConfigClass(obj);
    }

    /**
     * Serializes the config to a string
     */
    serialize(mode: StringifyMode = DEFAULT_STRINGIFY_MODE): string {
        return stringifyObject(this.export(), mode);
    }

    /**
     * Exports the config to a plain object
     */
    export(): T {
        return {
            __version: this.__version,
            resources: this.resources,
            metadata: this.metadata,
        } as unknown as T;
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
    removeResource(resourceId: string): void {
        this.resources = this.resources.filter(resource => resource.id !== resourceId);
    }

    /**
     * Updates a resource in the config
     */
    updateResource(resourceId: string, updates: Partial<ConfigResource>): void {
        this.resources = this.resources.map(resource =>
            resource.id === resourceId
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

    /**
     * Sets metadata value
     */
    setMetadata(key: string, value: any): void {
        this.metadata[key] = value;
    }

    /**
     * Gets metadata value
     */
    getMetadata(key: string): any {
        return this.metadata[key];
    }
}
