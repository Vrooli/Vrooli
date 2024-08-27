/**
 * This handles handles storing and retrieving "cookies", which may or
 * may not be permitted by the user's cookie preferences. I say "cookies" in 
 * quotes because they are not actually cookies, but rather localStorage. It is 
 * unclear whether EU's Cookie Law applies to localStorage, but it is better to
 * be safe than sorry.
 */
import { ActiveFocusMode, FocusMode, GqlModelType, LlmTaskInfo, NavigableObject } from "@local/shared";
import { chatMatchHash } from "./codes";
import { getDeviceInfo } from "./display/device";
import { LocalStorageLruCache } from "./localStorageLruCache";

const MAX_CACHE_SIZE = 300;
const KB_1 = 1024;
// eslint-disable-next-line no-magic-numbers
const CACHE_LIMIT_2KB = KB_1 * 2;
// eslint-disable-next-line no-magic-numbers
const CACHE_LIMIT_128KB = KB_1 * 128;
// eslint-disable-next-line no-magic-numbers
const CACHE_LIMIT_256KB = KB_1 * 256;
// eslint-disable-next-line no-magic-numbers
const CACHE_LIMIT_512KB = KB_1 * 512;
// eslint-disable-next-line no-magic-numbers
const CACHE_LIMT_1MB = KB_1 * 1024;

const FORM_CACHE_LIMIT = 100;
const ALLOW_FORM_CACHING_CACHE_LIMIT = 50;
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

export type CreateType = "Api" | "Bot" | "Chat" | "DataConverter" | "DataStructure" | "Note" | "Project" | "Prompt" | "Question" | "Reminder" | "Routine" | "SmartContract" | "Team";
export type ThemeType = "light" | "dark";

type SimpleStoragePayloads = {
    CreateOrder: string[],
    Preferences: CookiePreferences,
    Theme: ThemeType,
    FontSize: number,
    Language: string,
    LastTab: string | null,
    IsLeftHanded: boolean,
    FocusModeActive: ActiveFocusMode | null,
    FocusModeAll: FocusMode[],
    ShowMarkdown: boolean,
    SideMenuState: boolean,
}
type SimpleStorageType = keyof SimpleStoragePayloads;

type SimpleStorageInfo<T extends keyof SimpleStoragePayloads> = {
    __type: keyof CookiePreferences;
    check: (value: unknown) => boolean
    fallback: SimpleStoragePayloads[T];
    shape?: (value: SimpleStoragePayloads[T]) => SimpleStoragePayloads[T];
};

type CacheStoragePayloads = {
    AllowFormCaching: AllowFormCaching,
    ChatMessageTree: ChatMessageTreeCookie,
    ChatParticipants: ChatGroupCookie,
    FormData: FormCacheEntry,
    PartialData: CookiePartialData, // Used to store partial data for the page before the full data is loaded
}
type CacheStorageType = keyof CacheStoragePayloads;

export function getStorageItem<T extends SimpleStorageType | string>(
    name: T,
    typeCheck: (value: unknown) => boolean,
): (T extends SimpleStorageType ? SimpleStoragePayloads[T] : unknown) | undefined {
    const cookie = localStorage.getItem(name);
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
        shape: (value) => Math.max(8, Math.min(24, value)),
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
    Preferences: {
        __type: "strictlyNecessary",
        check: (value) => typeof value === "object",
        fallback: {
            strictlyNecessary: false,
            performance: false,
            functional: false,
            targeting: false,
        },
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
    ShowMarkdown: {
        __type: "functional",
        check: (value) => typeof value === "boolean",
        fallback: false,
    },
    SideMenuState: {
        __type: "strictlyNecessary",
        check: (value) => typeof value === "boolean",
        fallback: false,
    },
    Theme: {
        __type: "functional",
        check: (value) => value === "light" || value === "dark",
        fallback: window.matchMedia && window.matchMedia("(prefers-color-scheme: light)").matches ? "light" : "dark",
    },
};

/**
 * Sets a cookie only if the user has permitted the cookie's type. 
 * For strictly necessary cookies it will be set regardless of user 
 * preferences.
 * @param cookieType Cookie type to check
 * @param callback Callback function to call if cookie is allowed
 * @param fallback Optional fallback value to use if cookie is not allowed
 */
export function ifAllowed(cookieType: keyof CookiePreferences, callback: () => unknown, fallback?: any) {
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
export function getCookieFormData(formId: string): FormCacheEntry | undefined {
    return ifAllowed("functional", () => {
        return formDataCache.get(formId);
    });
}
export function setCookieFormData(formId: string, data: FormCacheEntry) {
    return ifAllowed("functional", () => {
        formDataCache.set(formId, data);
    });
}
export function removeCookieFormData(formId: string) {
    return ifAllowed("functional", () => {
        formDataCache.remove(formId);
    });
}


/** Indicates if the form for this objectType/ID pair has disabled auto-save */
type AllowFormCaching = boolean;
const formCachingCache = new LocalStorageLruCache<AllowFormCaching>("allowFormCaching", ALLOW_FORM_CACHING_CACHE_LIMIT, CACHE_LIMIT_2KB);
export function getCookieAllowFormCache(objectType: GqlModelType | `${GqlModelType}`, objectId: string): AllowFormCaching {
    return ifAllowed("functional", () => {
        const result = formCachingCache.get(`${objectType}:${objectId}`);
        if (typeof result === "boolean") return result;
        return true; // Default to true
    });
}
export function setCookieAllowFormCache(objectType: GqlModelType | `${GqlModelType}`, objectId: string, data: AllowFormCaching) {
    return ifAllowed("functional", () => {
        formCachingCache.set(`${objectType}:${objectId}`, data);
    });
}
export function removeCookieAllowFormCache(objectType: GqlModelType | `${GqlModelType}`, objectId: string) {
    return ifAllowed("functional", () => {
        formCachingCache.remove(`${objectType}:${objectId}`);
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
export function getCookieMatchingChat(userIds: string[], task?: string): string | undefined {
    return ifAllowed("functional", function getCookieMatchingChatCallback() {
        return chatGroupCache.get(chatMatchHash(userIds, task))?.chatId;
    });
}
export function setCookieMatchingChat(chatId: string, userIds: string[], task?: string) {
    return ifAllowed("functional", function setCookieMatchingChatCallback() {
        chatGroupCache.set(chatMatchHash(userIds, task), { chatId });
    });
}
export function removeCookieMatchingChat(userIds: string[], task?: string) {
    return ifAllowed("functional", function removeCookieMatchingChatCallback() {
        chatGroupCache.remove(chatMatchHash(userIds, task));
    });
}
export function removeCookiesWithChatId(chatId: string) {
    return ifAllowed("functional", function removeCookiesWithChatIdCallback() {
        chatGroupCache.removeKeysWithValue((_, value) => value.chatId === chatId);
    });
}

type MessageTasks = {
    tasks: LlmTaskInfo[];
}
const llmTasksCache = new LocalStorageLruCache<MessageTasks>("llmTasks", LLM_TASKS_CACHE_LIMIT, CACHE_LIMT_1MB);
export function getCookieTasksForMessage(messageId: string): MessageTasks | undefined {
    return ifAllowed("functional", () => {
        const existing = llmTasksCache.get(messageId);
        if (typeof existing === "object" && Array.isArray(existing.tasks)) {
            return existing;
        }
    });
}
export const setCookieTaskForMessage = (messageId: string, task: LlmTaskInfo) => ifAllowed("functional", () => {
    const existing = getCookieTasksForMessage(messageId) || { tasks: [] };
    const taskIndex = existing.tasks.findIndex(t => t.taskId === task.taskId);
    if (taskIndex > -1) {
        existing.tasks[taskIndex] = task; // Replace existing task with the same ID
    } else {
        existing.tasks.push(task); // Add new task
    }
    llmTasksCache.set(messageId, existing);
});
export const removeTaskForMessage = (messageId: string, taskId: string) => ifAllowed("functional", () => {
    const existing = getCookieTasksForMessage(messageId);
    if (!existing || !existing.tasks.length) return;
    const updated = { tasks: existing.tasks.filter(t => t.taskId !== taskId) };
    if (updated.tasks.length) {
        llmTasksCache.set(messageId, updated);
    } else {
        llmTasksCache.remove(messageId); // Remove the entry if no tasks left
    }
});
export const removeTasksForMessage = (messageId: string) => ifAllowed("functional", () => {
    llmTasksCache.remove(messageId);
});

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
const getCache = (): Cache => getOrSetCookie(
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

/** Shape knownData to replace idRoot and handleRoot with proper root object */
const shapeKnownData = <T extends CookiePartialData>(knownData: T): T => ({
    ...knownData,
    idRoot: undefined,
    handleRoot: undefined,
    ...(knownData.idRoot || knownData.handleRoot ? {
        root: {
            ...knownData.root,
            id: knownData.idRoot,
            handle: knownData.handleRoot,
        },
    } : {}),
});
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
export const setCookiePartialData = (partialData: CookiePartialData, dataType: DataType) => ifAllowed("functional", () => {
    ifAllowed("functional", () => {
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
        if (cache.order.length >= MAX_CACHE_SIZE) {
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
});
export const removeCookiePartialData = (partialData: CookiePartialData) => ifAllowed("functional", () => {
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
