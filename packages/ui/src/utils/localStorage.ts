import { AITaskInfo, ActiveFocusMode, FocusMode, ModelType, NavigableObject, ProjectVersion, RoutineVersion, TaskContextInfo, isObject } from "@local/shared";
import { chatMatchHash } from "./codes.js";
import { FONT_SIZE_MAX, FONT_SIZE_MIN } from "./consts.js";
import { getDeviceInfo } from "./display/device.js";

const KB_1 = 1024;
// eslint-disable-next-line no-magic-numbers
const CACHE_LIMIT_128KB = KB_1 * 128;
// eslint-disable-next-line no-magic-numbers
const CACHE_LIMIT_256KB = KB_1 * 256;
// eslint-disable-next-line no-magic-numbers
const CACHE_LIMIT_512KB = KB_1 * 512;
// eslint-disable-next-line no-magic-numbers
const CACHE_LIMT_1MB = KB_1 * 1024;

const FORM_CACHE_LIMIT = 100;
const PARTIAL_DATA_CACHE_LIMIT = 300;
const CHAT_MESSAGE_TREE_CACHE_LIMIT = 100;
const CHAT_GROUP_CACHE_LIMIT = 100;
const LLM_TASKS_CACHE_LIMIT = 100;

/**
 * Preferences for the user's cookie settings
 */
export type CookiePreferences = {
    strictlyNecessary: boolean;
    performance: boolean;
    functional: boolean;
    targeting: boolean;
}

type RunLoaderCache = {
    projects: { [projectId: string]: ProjectVersion };
    routines: { [routineId: string]: RoutineVersion };
};

export type CreateType = "Api" | "Bot" | "Chat" | "DataConverter" | "DataStructure" | "Note" | "Project" | "Prompt" | "Question" | "Reminder" | "Routine" | "SmartContract" | "Team";
export type ThemeType = "light" | "dark";

type SimpleStoragePayloads = {
    CreateOrder: string[],
    FocusModeActive: ActiveFocusMode | null,
    FocusModeAll: FocusMode[],
    FontSize: number,
    IsLeftHanded: boolean,
    Language: string,
    LastTab: string | null,
    MenuState: boolean,
    Preferences: CookiePreferences,
    RunLoaderCache: RunLoaderCache,
    ShowBotWarning: boolean,
    ShowMarkdown: boolean,
    SingleStepRoutineOrder: string[],
    Theme: ThemeType,
    AdvancedInputSettings: {
        enterWillSubmit: boolean;
        showToolbar: boolean;
        isWysiwyg: boolean;
    },
}
type SimpleStorageType = keyof SimpleStoragePayloads;

type SimpleStorageInfo<T extends keyof SimpleStoragePayloads> = {
    __type: keyof CookiePreferences;
    check: (value: unknown) => boolean
    fallback: SimpleStoragePayloads[T];
    shape?: (value: SimpleStoragePayloads[T]) => SimpleStoragePayloads[T];
};

type CacheStoragePayloads = {
    ChatMessageTree: ChatMessageTreeCookie,
    ChatParticipants: ChatGroupCookie,
    FormData: FormCacheEntry,
    PartialData: CookiePartialData, // Used to store partial data for the page before the full data is loaded
}
type CacheStorageType = keyof CacheStoragePayloads;

interface GetLocalStorageKeysProps {
    prefix?: string;
    suffix?: string;
}

export const noPreferences = {
    strictlyNecessary: false,
    performance: false,
    functional: false,
    targeting: false,
};
export const fullPreferences = {
    strictlyNecessary: true,
    performance: true,
    functional: true,
    targeting: true,
};

export const cookies: { [T in SimpleStorageType]: SimpleStorageInfo<T> } = {
    CreateOrder: {
        __type: "functional",
        check: (value) => Array.isArray(value) && value.every(v => typeof v === "string"),
        fallback: [],
    },
    FocusModeActive: {
        __type: "functional",
        check: (value) => typeof value === "object" || value === null,
        fallback: null,
    },
    FocusModeAll: {
        __type: "functional",
        check: (value) => Array.isArray(value),
        fallback: [],
    },
    FontSize: {
        __type: "functional",
        check: (value) => typeof value === "number",
        fallback: 14,
        // Ensure font size is not too small or too large. This would make the UI unusable.
        shape: (value) => Math.max(FONT_SIZE_MIN, Math.min(FONT_SIZE_MAX, value)),
    },
    IsLeftHanded: {
        __type: "functional",
        check: (value) => typeof value === "boolean",
        fallback: false,
    },
    Language: {
        __type: "functional",
        check: (value) => typeof value === "string",
        fallback: "en",
    },
    LastTab: {
        __type: "functional",
        check: (value) => typeof value === "string" && value !== "",
        fallback: null,
    },
    MenuState: {
        __type: "strictlyNecessary",
        check: (value) => typeof value === "boolean",
        fallback: false,
    },
    Preferences: {
        __type: "strictlyNecessary",
        check: (value) => typeof value === "object",
        fallback: noPreferences,
        // Allow all for PWAs
        shape: (value) => {
            if (getDeviceInfo().isStandalone) {
                // Use fallback with all true
                return Object.fromEntries(Object.keys(cookies.Preferences.fallback).map(k => [k, true])) as CookiePreferences;
            }
            return {
                strictlyNecessary: value.strictlyNecessary || true,
                performance: value.performance || false,
                functional: value.functional || false,
                targeting: value.targeting || false,
            };
        },
    },
    RunLoaderCache: {
        __type: "functional",
        check: (value) => typeof value === "object" && value !== null && isObject((value as { projects?: unknown }).projects) && isObject((value as { routines?: unknown }).routines),
        fallback: { projects: {}, routines: {} },
    },
    ShowBotWarning: {
        __type: "functional",
        check: (value) => typeof value === "boolean",
        fallback: true,
    },
    ShowMarkdown: {
        __type: "functional",
        check: (value) => typeof value === "boolean",
        fallback: false,
    },
    SingleStepRoutineOrder: {
        __type: "functional",
        check: (value) => Array.isArray(value) && value.every(v => typeof v === "string"),
        fallback: [],
    },
    Theme: {
        __type: "functional",
        check: (value) => value === "light" || value === "dark",
        fallback: window.matchMedia && window.matchMedia("(prefers-color-scheme: light)").matches ? "light" : "dark",
    },
    AdvancedInputSettings: {
        __type: "functional",
        check: (value) => (
            typeof value === "object" &&
            value !== null &&
            typeof (value as { enterWillSubmit?: boolean }).enterWillSubmit === "boolean" &&
            typeof (value as { showToolbar?: boolean }).showToolbar === "boolean" &&
            typeof (value as { isWysiwyg?: boolean }).isWysiwyg === "boolean"
        ),
        fallback: {
            enterWillSubmit: true,
            showToolbar: false,
            isWysiwyg: false,
        },
    },
};

/**
 * Find all keys in localStorage matching the specified options
 * @returns Array of keys
 */
export function getLocalStorageKeys({
    prefix = "",
    suffix = "",
}: GetLocalStorageKeysProps): string[] {
    const keys: string[] = [];
    for (let i = 0; i < localStorage.length; i++) {
        const key = localStorage.key(i);
        if (key && key.startsWith(prefix) && key.endsWith(suffix)) {
            keys.push(key);
        }
    }
    return keys;
}

/**
 * A simple LRU (Least Recently Used) cache that stores values in localStorage. 
 * This is useful for limiting the amount of data stored in localStorage.
 */
export class LocalStorageLruCache<ValueType> {
    private limit: number;
    private maxSize: number | null; // maxSize in bytes, null means no limit
    private namespace: string; // Unique identifier for this cache
    private cacheKeys: Array<string>; // Store keys to manage the order for LRU

    constructor(namespace: string, limit: number, maxSize: number | null = null) {
        this.namespace = namespace;
        this.limit = limit;
        this.maxSize = maxSize;
        this.cacheKeys = this.loadKeys();
    }

    private loadKeys(): Array<KeyType> {
        try {
            const keys = localStorage.getItem(this.getNamespacedKey("cacheKeys"));
            if (keys) {
                const parsedKeys = JSON.parse(keys);
                if (Array.isArray(parsedKeys)) {
                    return parsedKeys;
                }
            }
        } catch (error) {
            console.error("Error loading keys from localStorage:", error);
        }
        return [];
    }

    private saveKeys(): void {
        localStorage.setItem(this.getNamespacedKey("cacheKeys"), JSON.stringify(this.cacheKeys));
    }

    private getNamespacedKey(key: string): string {
        return `${this.namespace}:${key}`;
    }

    get(key: string): ValueType | undefined {
        try {
            const serializedValue = localStorage.getItem(this.getNamespacedKey(key));
            if (serializedValue) {
                this.touchKey(key);
                return JSON.parse(serializedValue);
            }
        } catch (error) {
            console.error("Error parsing value from localStorage", key, error);
        }
        return undefined;
    }

    set(key: string, value: ValueType): void {
        const serializedValue = JSON.stringify(value);
        if (this.maxSize != null && serializedValue.length > this.maxSize) {
            console.warn(`Skipping cache set for key ${key}: value size ${serializedValue.length} exceeds maxSize ${this.maxSize}`);
            return;
        }

        this.touchKey(key);
        localStorage.setItem(this.getNamespacedKey(key), serializedValue);
    }

    remove(key: string): void {
        // Remove the key from the cacheKeys array
        this.removeKey(key);
        // Remove the item from localStorage
        localStorage.removeItem(this.getNamespacedKey(key));
        // Save the updated keys to localStorage
        this.saveKeys();
    }

    removeKeysWithValue(predicate: (key: string, value: ValueType) => boolean): void {
        console.log("in removeKeysWithValue", this.cacheKeys);
        this.cacheKeys.forEach((key) => {
            const value = this.get(key);
            console.log("key and value", key, value);
            if (value !== undefined && predicate(key, value)) {
                console.log("removing key:", key);
                this.remove(key);
            }
        });
    }

    private touchKey(key: string): void {
        this.removeKey(key);
        this.cacheKeys.push(key);
        while (this.cacheKeys.length > this.limit) {
            const oldKey = this.cacheKeys.shift();
            if (oldKey !== undefined) {
                localStorage.removeItem(this.getNamespacedKey(oldKey));
            }
        }
        this.saveKeys();
    }

    private removeKey(key: string): void {
        const index = this.cacheKeys.indexOf(key);
        if (index > -1) {
            this.cacheKeys.splice(index, 1);
        }
    }

    size(): number {
        return this.cacheKeys.length;
    }
}

export function getStorageItem<T extends SimpleStorageType | string>(
    name: T,
    typeCheck: (value: unknown) => boolean,
): (T extends SimpleStorageType ? SimpleStoragePayloads[T] : unknown) | undefined {
    const cookie = localStorage.getItem(name);
    if (cookie === null) return undefined;
    // Try to parse
    try {
        const parsed = JSON.parse(cookie ?? "");
        if (typeCheck(parsed)) {
            return parsed;
        }
    } catch (e) {
        console.warn(`Failed to parse cookie ${name}`, cookie);
    }
    return undefined;
}

export function setStorageItem<T extends SimpleStorageType | string>(
    name: T,
    value: T extends SimpleStorageType ? SimpleStoragePayloads[T] : unknown,
) {
    localStorage.setItem(name, JSON.stringify(value));
}

/**
 * Gets a cookie if it exists, otherwise sets it to the default value. 
 * Assumes that you have already checked that the cookie is allowed.
 */
export function getOrSetCookie<T extends SimpleStorageType | string>(
    name: T,
    check: (value: unknown) => boolean,
    fallback: T extends SimpleStorageType ? SimpleStoragePayloads[T] : unknown,
): T extends SimpleStorageType ? SimpleStoragePayloads[T] : unknown {
    const cookie = getStorageItem(name, check);
    if (cookie !== undefined) return cookie;
    // NOTE: The only cookie we'll refuse to set is "Preferences", since 
    // the user needs to set these using the cookie dialog
    if (name !== "Preferences") {
        setStorageItem(name, fallback);
    }
    return fallback;
}

/**
 * Sets a cookie only if the user has permitted the cookie's type. 
 * For strictly necessary cookies it will be set regardless of user 
 * preferences.
 * @param cookieType Cookie type to check
 * @param callback Callback function to call if cookie is allowed
 * @param fallback Optional fallback value to use if cookie is not allowed
 */
export function ifAllowed(cookieType: keyof CookiePreferences, callback: () => unknown, fallback?: any) {
    // Allow all in dev mode   
    if (process.env.DEV) {
        return callback();
    }
    const preferences = getStorageItem("Preferences", cookies.Preferences.check) ?? cookies.Preferences.fallback;
    if (cookieType === "strictlyNecessary" || preferences[cookieType]) {
        return callback();
    }
    else {
        console.warn(`Not allowed to get/set cookie ${cookieType}`, preferences, localStorage.getItem("Preferences"));
        return fallback;
    }
}

/**
 * Retrieves data from localStorage, if allowed by the user's cookie preferences.
 * @param name The name of the cookie to retrieve
 * @param id Provides unique identifier for the cookie, if needed (e.g. last tab is stored for many different tabs)
 * @returns The value of the cookie, or the fallback value if the cookie is not allowed
 */
export function getCookie<T extends SimpleStorageType>(
    name: T,
    id?: string,
): SimpleStoragePayloads[T] {
    const { __type, check, fallback, shape } = cookies[name] || {};
    return ifAllowed(
        __type,
        () => {
            const key = id ? `${name}-${id}` : name;
            const storedValue = getOrSetCookie(key, check, fallback) as SimpleStoragePayloads[T];
            return storedValue !== undefined && typeof shape === "function"
                ? shape(storedValue)
                : storedValue;
        },
        fallback,
    );
}

/**
 * Stroes data in localStorage only if the user has permitted the cookie's type.
 * @param name The name of the cookie to set
 * @param value The value to store in the cookie
 * @param id Provides unique identifier for the cookie, if needed (e.g. last tab is stored for many different tabs)
 */
export function setCookie<T extends SimpleStorageType>(
    name: T,
    value: SimpleStoragePayloads[T],
    id?: string,
) {
    const { __type } = cookies[name] || {};
    const key = id ? `${name}-${id}` : name;
    ifAllowed(__type, () => { setStorageItem(key, value); });
}

export function removeCookie<T extends SimpleStorageType | CacheStorageType>(
    name: T,
    id?: string,
) {
    const key = id ? `${name}-${id}` : name;
    localStorage.removeItem(key);
}

/** Represents the stored values for a particular form identified by objectType and objectId */
type FormCacheEntry = {
    [key: string]: any; // This would be the form data
}
const formDataCache = new LocalStorageLruCache<FormCacheEntry>("formData", FORM_CACHE_LIMIT, CACHE_LIMIT_512KB);
function getFormCacheKey(
    objectType: ModelType | `${ModelType}`,
    objectId: string,
) {
    return `${objectType}:${objectId}`;
}
export function getCookieFormData(
    objectType: ModelType | `${ModelType}`,
    objectId: string,
): FormCacheEntry | undefined {
    return ifAllowed("functional", () => {
        const key = getFormCacheKey(objectType, objectId);
        return formDataCache.get(key);
    });
}
export function setCookieFormData(
    objectType: ModelType | `${ModelType}`,
    objectId: string,
    data: FormCacheEntry,
) {
    return ifAllowed("functional", () => {
        const key = getFormCacheKey(objectType, objectId);
        formDataCache.set(key, data);
    });
}
export function removeCookieFormData(
    objectType: ModelType | `${ModelType}`,
    objectId: string,
) {
    return ifAllowed("functional", () => {
        const key = getFormCacheKey(objectType, objectId);
        formDataCache.remove(key);
    });
}

/** Maps a chat message ID to the ID of the selected child branch */
export type BranchMap = Record<string, string>;
type ChatMessageTreeCookie = {
    branches: BranchMap
    /** The message you viewed last */
    locationId: string;
}
const chatMessageTreeCache = new LocalStorageLruCache<ChatMessageTreeCookie>("chatMessageTree", CHAT_MESSAGE_TREE_CACHE_LIMIT, CACHE_LIMIT_256KB);
export function getCookieMessageTree(chatId: string): ChatMessageTreeCookie | undefined {
    return ifAllowed("functional", () => {
        return chatMessageTreeCache.get(chatId);
    });
}
export function setCookieMessageTree(chatId: string, data: ChatMessageTreeCookie) {
    return ifAllowed("functional", () => {
        chatMessageTreeCache.set(chatId, data);
    });
}

type ChatGroupCookie = {
    /** The last chatId for the group of userIds */
    chatId: string;
};
const chatGroupCache = new LocalStorageLruCache<ChatGroupCookie>("chatGroup", CHAT_GROUP_CACHE_LIMIT, CACHE_LIMIT_128KB);
export function getCookieMatchingChat(userIds: string[]): string | undefined {
    return ifAllowed("functional", function getCookieMatchingChatCallback() {
        return chatGroupCache.get(chatMatchHash(userIds))?.chatId;
    });
}
export function setCookieMatchingChat(chatId: string, userIds: string[]) {
    return ifAllowed("functional", function setCookieMatchingChatCallback() {
        chatGroupCache.set(chatMatchHash(userIds), { chatId });
    });
}
export function removeCookieMatchingChat(userIds: string[]) {
    return ifAllowed("functional", function removeCookieMatchingChatCallback() {
        chatGroupCache.remove(chatMatchHash(userIds));
    });
}

type ChatTasks = {
    activeTask: AITaskInfo | null;
    contexts: { [taskId: string]: TaskContextInfo[] };
    inactiveTasks: AITaskInfo[];
}
const llmTasksCache = new LocalStorageLruCache<ChatTasks>("llmTasks", LLM_TASKS_CACHE_LIMIT, CACHE_LIMT_1MB);
export function getCookieTasksForChat(chatId: string): ChatTasks | undefined {
    return ifAllowed("functional", () => {
        const existing = llmTasksCache.get(chatId);
        const isValid =
            typeof existing === "object" &&
            typeof existing.activeTask === "object" &&
            typeof existing.contexts === "object" &&
            Array.isArray(existing.inactiveTasks);
        if (isValid) {
            return existing;
        }
    });
}
export function setCookieTasksForChat(
    chatId: string,
    { activeTask, contexts, inactiveTasks }: Partial<ChatTasks>,
) {
    return ifAllowed("functional", () => {
        const existing = getCookieTasksForChat(chatId) || { activeTask: null, contexts: {}, inactiveTasks: [] };
        if (activeTask) existing.activeTask = activeTask;
        if (contexts) existing.contexts = contexts;
        if (inactiveTasks) existing.inactiveTasks = inactiveTasks;
        llmTasksCache.set(chatId, existing);
    });
}
export function updateCookiePartialTaskForChat(chatId: string, task: Partial<AITaskInfo>) {
    return ifAllowed("functional", () => {
        const existing = getCookieTasksForChat(chatId) || { activeTask: null, contexts: {}, inactiveTasks: [] };
        // Task being updated can either be active or inactive
        if (existing.activeTask && existing.activeTask.taskId === task.taskId) {
            existing.activeTask = { ...existing.activeTask, ...task };
        } else {
            const taskIndex = existing.inactiveTasks.findIndex(t => t.taskId === task.taskId);
            if (taskIndex >= 0) {
                existing.inactiveTasks[taskIndex] = { ...existing.inactiveTasks[taskIndex], ...task };
            } else {
                console.error(`Task ${task.taskId} not found in active or inactive tasks`);
            }
        }
        llmTasksCache.set(chatId, existing);
    });
}
export function upsertCookieTaskForChat(chatId: string, task: AITaskInfo) {
    return ifAllowed("functional", () => {
        const existing = getCookieTasksForChat(chatId) || { activeTask: null, contexts: {}, inactiveTasks: [] };
        // Task being updated can either be active or inactive
        if (existing.activeTask && existing.activeTask.taskId === task.taskId) {
            existing.activeTask = task;
        } else {
            const taskIndex = existing.inactiveTasks.findIndex(t => t.taskId === task.taskId);
            if (taskIndex >= 0) {
                existing.inactiveTasks[taskIndex] = task;
            } else {
                existing.inactiveTasks = [task, ...existing.inactiveTasks];
            }
        }
        llmTasksCache.set(chatId, existing);
    });
}
export function removeTasksForChat(chatId: string) {
    return ifAllowed("functional", () => {
        llmTasksCache.remove(chatId);
    });
}

/**
 * Removes all cookies related to a chatId from the cache
 */
export function removeCookiesWithChatId(chatId: string) {
    return ifAllowed("functional", function removeCookiesWithChatIdCallback() {
        chatMessageTreeCache.remove(chatId);
        chatGroupCache.removeKeysWithValue((_, value) => value.chatId === chatId);
        llmTasksCache.remove(chatId);
    });
}

/** Supports ID data from URL params, as well as partial object */
export type CookiePartialData = {
    __typename: NavigableObject["__typename"],
    id?: string | null,
    idRoot?: string | null,
    handle?: string | null
    handleRoot?: string | null,
    root?: {
        id?: string | null,
        handle?: string | null,
    } | null,
};
/** Type of data stored */
type DataType = "list" | "full";
/** 
 * Stores multiple cached objects, in a way that allows 
 * for removing the oldest object when the cache is full.
 * 
 * Objects can be cached using their ID or handle, or root's ID or handle
 */
type Cache = {
    idMap: Record<string, CookiePartialData>,
    handleMap: Record<string, CookiePartialData>,
    idRootMap: Record<string, CookiePartialData>,
    handleRootMap: Record<string, CookiePartialData>,
    order: string[], // Order of objects in cache, for FIFO
};


/** Get object containing all cached objects */
function getCache(): Cache {
    return getOrSetCookie(
        "PartialData", // Cookie name
        (value: unknown): value is Cache =>
            typeof value === "object" &&
            typeof (value as Partial<Cache>)?.idMap === "object" &&
            typeof (value as Partial<Cache>)?.handleMap === "object" &&
            typeof (value as Partial<Cache>)?.idRootMap === "object" &&
            typeof (value as Partial<Cache>)?.handleRootMap === "object" &&
            Array.isArray((value as Partial<Cache>)?.order),
        {
            idMap: {},
            handleMap: {},
            idRootMap: {},
            handleRootMap: {},
            order: [],
        }, // Default value
    ) as Cache;
}

/** Shape knownData to replace idRoot and handleRoot with proper root object */
function shapeKnownData<T extends CookiePartialData>(knownData: T): T {
    let result = { ...knownData };
    if (knownData.idRoot || knownData.handleRoot) {
        result = {
            ...result,
            root: {
                id: knownData.idRoot,
                handle: knownData.handleRoot,
            },
        };
    }
    delete result.idRoot;
    delete result.handleRoot;
    return result;
}
export function getCookiePartialData<T extends CookiePartialData>(knownData: CookiePartialData): T {
    return ifAllowed("functional",
        () => {
            const shapedKnownData = shapeKnownData(knownData);
            // Get the cache, which hopefully includes more info on the requested object
            const cache = getCache();
            // Try to find stored data in cache
            const storedData = cache.idMap[shapedKnownData.id || ""] ||
                cache.handleMap[shapedKnownData.handle || ""] ||
                cache.idRootMap[shapedKnownData.root?.id || ""] ||
                cache.handleRootMap[shapedKnownData.root?.handle || ""];
            // Return match if found
            if (storedData && storedData.__typename === shapedKnownData.__typename) {
                return storedData;
            }
            // Otherwise return known data
            return shapedKnownData;
        },
        shapeKnownData(knownData),
    );
}
export function setCookiePartialData(partialData: CookiePartialData, dataType: DataType) {
    return ifAllowed("functional", () => {
        // Get the cache
        const cache = getCache();
        // Create a key that includes every identifier for the object
        const key = `${partialData.id}|${partialData.handle}|${partialData.root?.id}|${partialData.root?.handle}`;
        // Check if the object is already in the cache
        const isAlreadyInCache = Boolean((partialData.id && cache.idMap[partialData.id]) ||
            (partialData.handle && cache.handleMap[partialData.handle]) ||
            (partialData.root?.id && cache.idRootMap[partialData.root.id]) ||
            (partialData.root?.handle && cache.handleRootMap[partialData.root.handle]));
        // If data type is "list", don't overwrite the existing cached data
        if (isAlreadyInCache && dataType === "list") {
            return;
        }
        // Store the object in the cache, even if it already existed. Store in every map that applies, since we don't know which one will be needed
        if (partialData.id) cache.idMap[partialData.id] = partialData;
        if (partialData.handle) cache.handleMap[partialData.handle] = partialData;
        if (partialData.root?.id) cache.idRootMap[partialData.root.id] = partialData;
        if (partialData.root?.handle) cache.handleRootMap[partialData.root.handle] = partialData;
        // If the object was already in the cache, update the order so it's at the end
        if (isAlreadyInCache) {
            // Find the existing key in the order array
            const existingIndex = cache.order.indexOf(key);
            if (existingIndex !== -1) {
                // Remove the existing key
                cache.order.splice(existingIndex, 1);
                // Add it back to the end
                cache.order.push(key);
            }
            setStorageItem("PartialData", cache);
            return;
        }
        // If the cache is full, remove the oldest object
        if (cache.order.length >= PARTIAL_DATA_CACHE_LIMIT) {
            const oldestKey = cache.order.shift();
            if (oldestKey && typeof oldestKey === "string") {
                const [id, handle, idRoot, handleRoot] = oldestKey.split("|");
                delete cache.idMap[id];
                delete cache.handleMap[handle];
                delete cache.idRootMap[idRoot];
                delete cache.handleRootMap[handleRoot];
            }
        }
        // Push the new object to the end of the FIFO order
        cache.order.push(key);

        setStorageItem("PartialData", cache);
    });
}
export function removeCookiePartialData(partialData: CookiePartialData) {
    return ifAllowed("functional", () => {
        // Get the cache
        const cache = getCache();
        // Create a key that includes every identifier for the object
        const key = `${partialData.id}|${partialData.handle}|${partialData.root?.id}|${partialData.root?.handle}`;
        // Remove the object from the cache
        delete cache.idMap[partialData.id || ""];
        delete cache.handleMap[partialData.handle || ""];
        delete cache.idRootMap[partialData.root?.id || ""];
        delete cache.handleRootMap[partialData.root?.handle || ""];
        // Remove the key from the order array
        const existingIndex = cache.order.indexOf(key);
        if (existingIndex !== -1) {
            cache.order.splice(existingIndex, 1);
        }
        setStorageItem("PartialData", cache);
    });
}
