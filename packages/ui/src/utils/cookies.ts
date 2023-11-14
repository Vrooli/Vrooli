/**
 * This handles handles storing and retrieving "cookies", which may or
 * may not be permitted by the user's cookie preferences. I say "cookies" in 
 * quotes because they are not actually cookies, but rather localStorage. It is 
 * unclear whether EU's Cookie Law applies to localStorage, but it is better to
 * be safe than sorry.
 */
import { ActiveFocusMode, COOKIE, exists, FocusMode, ValueOf } from "@local/shared";
import { AssistantTask, NavigableObject } from "types";
import { hashChatMatchData } from "./hash";

/**
 * Handles storing and retrieving cookies, which may or 
 * may not be permitted by the user's cookie preferences
 */
export const Cookies = {
    ...COOKIE,
    PartialData: "partialData", // Used to store partial data for the page before the full data is loaded
    Preferences: "cookiePreferences",
    Theme: "theme",
    FontSize: "fontSize",
    Language: "language",
    LastTab: "lastTab",
    IsLeftHanded: "isLeftHanded",
    FocusModeActive: "focusModeActive",
    FocusModeAll: "focusModeAll",
    ShowMarkdown: "showMarkdown",
    SideMenuState: "sideMenuState",
    ChatMessageTree: "chatMessageTree",
    ChatParticipants: "chatParticipants",
    FormData: "formData",
};
export type Cookies = ValueOf<typeof Cookies>;

/**
 * Preferences for the user's cookie settings
 */
export type CookiePreferences = {
    strictlyNecessary: boolean;
    performance: boolean;
    functional: boolean;
    targeting: boolean;
}

const getCookie = <T>(name: Cookies | string, typeCheck: (value: unknown) => value is T): T | undefined => {
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
};

const setCookie = (name: Cookies | string, value: unknown) => { localStorage.setItem(name, JSON.stringify(value)); };

/**
 * Gets a cookie if it exists, otherwise sets it to the default value. 
 * Assumes that you have already checked that the cookie is allowed.
 */
export const getOrSetCookie = <T>(name: Cookies | string, typeCheck: (value: unknown) => value is T, defaultValue?: T): T | undefined => {
    const cookie = getCookie(name, typeCheck);
    if (exists(cookie)) return cookie;
    if (exists(defaultValue)) setCookie(name, defaultValue);
    return defaultValue;
};

/**
 * Finds the user's cookie preferences.
 * @returns CookiePreferences object, or null if not set
 */
export const getCookiePreferences = (): CookiePreferences | null => {
    const cookie = getCookie(Cookies.Preferences, (value: unknown): value is CookiePreferences => typeof value === "object");
    if (!cookie) return null;
    return {
        strictlyNecessary: cookie.strictlyNecessary || true,
        performance: cookie.performance || false,
        functional: cookie.functional || false,
        targeting: cookie.targeting || false,
    };
};

/**
 * Sets the user's cookie preferences.
 * @param preferences CookiePreferences object
 */
export const setCookiePreferences = (preferences: CookiePreferences) => {
    setCookie(Cookies.Preferences, preferences);
};

/**
 * Sets a cookie only if the user has permitted the cookie's type. 
 * For strictly necessary cookies it will be set regardless of user 
 * preferences.
 * @param cookieType Cookie type to check
 * @param callback Callback function to call if cookie is allowed
 * @param fallback Optional fallback value to use if cookie is not allowed
 */
const ifAllowed = (cookieType: keyof CookiePreferences, callback: () => unknown, fallback?: any) => {
    const preferences = getCookiePreferences();
    if (cookieType === "strictlyNecessary" || preferences?.[cookieType]) {
        return callback();
    }
    else {
        console.warn(`Not allowed to get/set cookie ${cookieType}`);
        return fallback;
    }
};

type ThemeType = "light" | "dark";
export const getCookieTheme = <T extends ThemeType | undefined>(fallback?: T): T =>
    ifAllowed("functional",
        () => getOrSetCookie(Cookies.Theme, (value: unknown): value is ThemeType => value === "light" || value === "dark", fallback),
        fallback,
    );
export const setCookieTheme = (theme: ThemeType) => ifAllowed("functional", () => setCookie(Cookies.Theme, theme));

export const getCookieFontSize = <T extends number | undefined>(fallback?: T): T =>
    ifAllowed("functional",
        () => {
            const size = getOrSetCookie(Cookies.FontSize, (value: unknown): value is number => typeof value === "number", fallback);
            // Ensure font size is not too small or too large. This would make the UI unusable.
            return size ? Math.max(8, Math.min(24, size)) : undefined;
        },
        fallback,
    );
export const setCookieFontSize = (fontSize: number) => ifAllowed("functional", () => setCookie(Cookies.FontSize, fontSize));

export const getCookieLanguage = <T extends string | undefined>(fallback?: T): T =>
    ifAllowed("functional",
        () => getOrSetCookie(Cookies.Language, (value: unknown): value is string => typeof value === "string", fallback),
        fallback,
    );
export const setCookieLanguage = (language: string) => ifAllowed("functional", () => setCookie(Cookies.Language, language));

export const getCookieIsLeftHanded = <T extends boolean | undefined>(fallback?: T): T =>
    ifAllowed("functional",
        () => getOrSetCookie(Cookies.IsLeftHanded, (value: unknown): value is boolean => typeof value === "boolean", fallback),
        fallback,
    );
export const setCookieIsLeftHanded = (isLeftHanded: boolean) => ifAllowed("functional", () => setCookie(Cookies.IsLeftHanded, isLeftHanded));

export const getCookieActiveFocusMode = <T extends ActiveFocusMode | undefined>(fallback?: T): T =>
    ifAllowed("functional",
        () => getOrSetCookie(Cookies.FocusModeActive, (value: unknown): value is ActiveFocusMode => typeof value === "object", fallback),
        fallback,
    );
export const setCookieActiveFocusMode = (focusMode: ActiveFocusMode | null) => ifAllowed("functional", () => setCookie(Cookies.FocusModeActive, focusMode));

export const getCookieAllFocusModes = <T extends FocusMode[]>(fallback?: T): T =>
    ifAllowed("functional",
        () => getOrSetCookie(Cookies.FocusModeAll, (value: unknown): value is FocusMode[] => Array.isArray(value), fallback),
        fallback,
    );
export const setCookieAllFocusModes = (modes: FocusMode[]) => ifAllowed("functional", () => setCookie(Cookies.FocusModeAll, modes));

export const getCookieSideMenuState = <T extends boolean | undefined>(id: string, fallback?: T): T =>
    ifAllowed("functional",
        () => {
            const allMenus = getOrSetCookie(Cookies.SideMenuState, (value: unknown): value is Record<string, boolean> => typeof value === "object" && value !== null, {});
            return allMenus?.[id] ?? fallback;
        },
        fallback,
    );

export const setCookieSideMenuState = (id: string, state: boolean) => ifAllowed("functional", () => {
    const allMenus = getOrSetCookie(Cookies.SideMenuState, (value: unknown): value is Record<string, boolean> => typeof value === "object" && value !== null, {});
    setCookie(Cookies.SideMenuState, allMenus ? { ...allMenus, [id]: state } : { [id]: state });
});

export const getCookieShowMarkdown = <T extends boolean | undefined>(fallback?: T): T =>
    ifAllowed("functional",
        () => getOrSetCookie(Cookies.ShowMarkdown, (value: unknown): value is boolean => typeof value === "boolean", fallback),
        fallback,
    );
export const setCookieShowMarkdown = (showMarkdown: boolean) => ifAllowed("functional", () => setCookie(Cookies.ShowMarkdown, showMarkdown));

export const getCookieLastTab = <T>(id: string, fallback?: T): T | undefined => ifAllowed("functional", () => {
    const lastTab = getOrSetCookie(`${Cookies.LastTab}-${id}`, (value: unknown): value is string => typeof value === "string", undefined);
    return lastTab as unknown as T;
}, fallback);
export const setCookieLastTab = <T>(id: string, tabType: T) => ifAllowed("functional", () => setCookie(`${Cookies.LastTab}-${id}`, tabType));





/** Represents the stored values for a particular form identified by objectType and objectId */
type FormCacheEntry = {
    [key: string]: any; // This would be the form data
}

type FormCache = {
    formData: { [formId: string]: FormCacheEntry };
    order: string[]; // Order of formIds in cache, for FIFO
};

const MAX_FORM_CACHE_SIZE = 100; // You can set this to an appropriate limit

/** Get the form data cache */
const getFormDataCache = (): FormCache => getOrSetCookie(
    Cookies.FormData,
    (value: unknown): value is FormCache =>
        typeof value === "object" &&
        typeof (value as Partial<FormCache>)?.formData === "object" &&
        Array.isArray((value as Partial<FormCache>)?.order),
    {
        formData: {},
        order: [],
    }, // Default value
) as FormCache;

export const getCookieFormData = (formId: string): FormCacheEntry | undefined =>
    ifAllowed("functional", () => {
        const cache = getFormDataCache();
        return cache.formData[formId];
    });

export const setCookieFormData = (formId: string, data: FormCacheEntry) => ifAllowed("functional", () => {
    const cache = getFormDataCache();

    // If formId already exists, remove from order for reinsertion at end
    const existingIndex = cache.order.indexOf(formId);
    if (existingIndex !== -1) {
        cache.order.splice(existingIndex, 1);
    }

    // If cache is full, remove the oldest form data
    if (cache.order.length >= MAX_FORM_CACHE_SIZE) {
        const oldestFormId = cache.order.shift();
        if (oldestFormId) {
            delete cache.formData[oldestFormId];
        }
    }

    // Store the new/updated form data and update order
    cache.formData[formId] = data;
    cache.order.push(formId);

    setCookie(Cookies.FormData, cache);
});

export const removeCookieFormData = (formId: string) => ifAllowed("functional", () => {
    const cache = getFormDataCache();

    // Remove form data from cache
    const existingIndex = cache.order.indexOf(formId);
    if (existingIndex !== -1) {
        cache.order.splice(existingIndex, 1);
        delete cache.formData[formId];

        setCookie(Cookies.FormData, cache);
    }
});






/** A chat message ID and which way it branches */
export type ChatMessageBranch = {
    messageId: string;
    selectedChildId: string;
}
type ChatMessageTreeCookie = {
    /** Which branches you're viewing */
    branches: ChatMessageBranch[];
    /** The message you viewed last */
    locationId: string;
}
type ChatMessageCache = {
    chats: { [chatId: string]: ChatMessageTreeCookie };
    order: string[]; // Order of chatIds in cache, for FIFO
};

const MAX_CHAT_CACHE_SIZE = 100;

/** Get the chat message cache */
const getChatMessageCache = (): ChatMessageCache => getOrSetCookie(
    Cookies.ChatMessageTree,
    (value: unknown): value is ChatMessageCache =>
        typeof value === "object" &&
        typeof (value as Partial<ChatMessageCache>)?.chats === "object" &&
        Array.isArray((value as Partial<ChatMessageCache>)?.order),
    {
        chats: {},
        order: [],
    }, // Default value
) as ChatMessageCache;

export const getCookieMessageTree = (chatId: string): ChatMessageTreeCookie | undefined =>
    ifAllowed("functional", () => {
        const cache = getChatMessageCache();
        return cache.chats[chatId];
    });

export const setCookieMessageTree = (chatId: string, data: ChatMessageTreeCookie) => ifAllowed("functional", () => {
    const cache = getChatMessageCache();

    // If chatId already exists, remove from order for reinsertion at end
    const existingIndex = cache.order.indexOf(chatId);
    if (existingIndex !== -1) {
        cache.order.splice(existingIndex, 1);
    }

    // If cache is full, remove the oldest chat
    if (cache.order.length >= MAX_CHAT_CACHE_SIZE) {
        const oldestChatId = cache.order.shift();
        if (oldestChatId) {
            delete cache.chats[oldestChatId];
        }
    }

    // Store the new/updated chat data and update order
    cache.chats[chatId] = data;
    cache.order.push(chatId);

    setCookie(Cookies.ChatMessageTree, cache);
});





type ChatGroupCookie = {
    /** The last chatId for the group of userIds */
    chatId: string;
};

type ChatMatchCache = {
    chats: { [participantsHash: string]: ChatGroupCookie };
    order: string[]; // Array of participantsHash in cache, for FIFO
};

const MAX_CHAT_MATCH_CACHE_SIZE = 100;

const getChatMatchCache = (): ChatMatchCache => getOrSetCookie(
    Cookies.ChatParticipants,
    (value: unknown): value is ChatMatchCache =>
        typeof value === "object" &&
        typeof (value as Partial<ChatMatchCache>)?.chats === "object" &&
        Array.isArray((value as Partial<ChatMatchCache>)?.order),
    {
        chats: {},
        order: [], // Default value for order
    }, // Default value
) as ChatMatchCache;

export const getCookieMatchingChat = (userIds: string[], task?: AssistantTask): string | undefined => ifAllowed("functional", () => {
    const cache = getChatMatchCache();
    const participantsHash = hashChatMatchData(userIds, task);
    return cache.chats[participantsHash]?.chatId;
});

export const setCookieMatchingChat = (chatId: string, userIds: string[], task?: AssistantTask) => ifAllowed("functional", () => {
    const cache = getChatMatchCache();
    const participantsHash = hashChatMatchData(userIds, task);

    // If participantsHash already exists, remove from order for reinsertion at end
    const existingIndex = cache.order.indexOf(participantsHash);
    if (existingIndex !== -1) {
        cache.order.splice(existingIndex, 1);
    } else if (cache.order.length >= MAX_CHAT_MATCH_CACHE_SIZE) {
        // If cache is full, remove the oldest participants group
        const oldestParticipantsHash = cache.order.shift();
        if (oldestParticipantsHash) {
            delete cache.chats[oldestParticipantsHash];
        }
    }
    // Store the new/updated participants group chatId and update order
    cache.chats[participantsHash] = { chatId };
    cache.order.push(participantsHash);

    setCookie(Cookies.ChatParticipants, cache);
});
export const removeCookieMatchingChat = (userIds: string[], task?: AssistantTask) => ifAllowed("functional", () => {
    const cache = getChatMatchCache();
    const participantsHash = hashChatMatchData(userIds, task);

    // Remove participantsHash from cache
    const existingIndex = cache.order.indexOf(participantsHash);
    if (existingIndex !== -1) {
        cache.order.splice(existingIndex, 1);
        delete cache.chats[participantsHash];

        setCookie(Cookies.ChatParticipants, cache);
    }
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

const MAX_CACHE_SIZE = 300;

/** Get object containing all cached objects */
const getCache = (): Cache => getOrSetCookie(
    Cookies.PartialData, // Cookie name
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
export const getCookiePartialData = <T extends CookiePartialData>(knownData: CookiePartialData): T =>
    ifAllowed("functional",
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
            setCookie(Cookies.PartialData, cache);
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

        setCookie(Cookies.PartialData, cache);
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
    setCookie(Cookies.PartialData, cache);
});
