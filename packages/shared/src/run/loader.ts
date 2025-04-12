import { DbObject, ModelType, ProjectVersion, RoutineVersion } from "../api/types.js";
import { LRUCache } from "../utils/lruCache.js";
import { PROJECT_CACHE_LIMIT, PROJECT_CACHE_MAX_SIZE_BYTES, ROUTINE_CACHE_LIMIT, ROUTINE_CACHE_MAX_SIZE_BYTES } from "./consts.js";
import { Id, Location, LocationData, LocationStack, RunConfig } from "./types.js";

/**
 * Handles loading routine and project information required for running a routine or project.
 * 
 * Initially fetches the data from the server, with subsequent calls using the cache.
 */
export abstract class RunLoader {
    /**
     * Map of loaded routine versions, keyed by routine ID.
     */
    protected routineCache: LRUCache<Id, RoutineVersion> = new LRUCache(ROUTINE_CACHE_LIMIT, ROUTINE_CACHE_MAX_SIZE_BYTES);
    /**
     * Map of loaded project versions, keyed by project ID.
     */
    protected projectCache: LRUCache<Id, ProjectVersion> = new LRUCache(PROJECT_CACHE_LIMIT, PROJECT_CACHE_MAX_SIZE_BYTES);

    /**
     * Fetch the object and subroutine (if applicable) for a given location.
     * 
     * @param location The location to fetch
     * @param config The run configuration
     * @returns The fetched data
     */
    public abstract fetchLocation(location: Location, config: RunConfig): Promise<LocationData | null>;

    /**
     * Load the object and subroutine (if applicable) for a given location, and store it in the cache.
     * 
     * @param location The location to load
     * @param config The run configuration
     * @returns The loaded data
     */
    public async loadLocation(location: Location, config: RunConfig): Promise<LocationData | null> {
        const cached = this.findInCache(location);
        if (cached) {
            return cached;
        }
        const loaded = await this.fetchLocation(location, config);
        if (!loaded) {
            return null;
        }
        if (loaded.object) {
            this.addToCache(loaded.object);
        }
        if (loaded.subroutine) {
            this.addToCache(loaded.subroutine);
        }
        return loaded;
    }

    /**
     * Load the full location stack, and store every object and subroutine in the cache.
     * 
     * @param locationStack The location stack to load
     * @param config The run configuration
     * @returns The last loaded object, or null if not found
     */
    public async loadLocationStack(locationStack: LocationStack, config: RunConfig): Promise<LocationData | null> {
        let current: LocationData | null = null;
        for (const location of locationStack) {
            current = await this.loadLocation(location, config);
        }
        return current;
    }

    /**
     * Called whenever the cache changes. Can be used to store the cache 
     * in redis or localStorage, so that it can be reloaded later without 
     * having to fetch everything again.
     */
    protected abstract onCacheChange(): void;

    /**
     * Add an object to the relevant cache.
     * 
     * @param object The object to cache
     */
    private addToCache(object: DbObject<"ProjectVersion" | "RoutineVersion">): void {
        if (object.__typename === ModelType.RoutineVersion) {
            this.routineCache.set(object.id, object as RoutineVersion);
        } else if (object.__typename === ModelType.ProjectVersion) {
            this.projectCache.set(object.id, object as ProjectVersion);
        }
        this.onCacheChange();
    }

    /**
     * Finds the object for a given location in the cache.
     * 
     * @param location The location to find
     * @returns The cached object, or null if not found
     */
    private findInCache(location: Location): LocationData | null {
        const cachedObject = location.__typename === ModelType.RoutineVersion
            ? this.routineCache.get(location.objectId)
            : this.projectCache.get(location.objectId);
        const cachedSubroutine = location.__typename === ModelType.RoutineVersion && location.subroutineId
            ? this.routineCache.get(location.subroutineId)
            : null;
        if (!cachedObject) {
            return null;
        }
        return { object: cachedObject, subroutine: cachedSubroutine || null };
    }
}
