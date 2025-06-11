import { type AutocompleteOption, type Session } from "@vrooli/shared";
import { getCurrentUser } from "../authentication/session.js";
import { getLocalStorageKeys } from "../localStorage.js";
import { PubSub } from "../pubsub.js";

const MAX_HISTORY_LENGTH = 500;

export type SearchOptionHistory = { timestamp: number, option: AutocompleteOption };
type SearchHistoryObject = { [label: string]: SearchOptionHistory };

export class SearchHistory {
    private static readonly SEARCH_HISTORY_PREFIX = "search-history-";
    private static readonly MAX_HISTORY_LENGTH = MAX_HISTORY_LENGTH;

    /**
     * Validates that a stored search history object is valid
     * @param history The history object to validate
     */
    private static validateSearchHistoryObject(history: unknown): boolean {
        if (typeof history !== "object" || history === null) return false;
        if (Object.keys(history).length === 0) return true;
        for (const key in history) {
            if (typeof key !== "string") return false;
            if (typeof history[key] !== "object" || history[key] === null) return false;
            const optionHistory = history[key];
            if (typeof (optionHistory as SearchOptionHistory).timestamp !== "number") return false;
            if (typeof (optionHistory as SearchOptionHistory).option !== "object" || (optionHistory as SearchOptionHistory).option === null) return false;
        }
        return true;
    }

    /**
     * Gets search history from local storage, unique by search bar and account
     * @param searchBarId The search bar ID
     * @param userId The user ID
     */
    static getSearchHistory(searchBarId: string, userId: string): SearchHistoryObject {
        const existingHistoryString: string = localStorage.getItem(`${SearchHistory.SEARCH_HISTORY_PREFIX}${searchBarId}-${userId}`) ?? "{}";
        let existingHistory: SearchHistoryObject = {};
        // Try to parse existing history
        try {
            const parsedHistory = JSON.parse(existingHistoryString);
            // If it's not an object, set it to an empty object
            if (typeof parsedHistory !== "object") existingHistory = {};
            else existingHistory = parsedHistory;
        } catch (e) {
            existingHistory = {};
        }
        // Validate the history object
        if (!SearchHistory.validateSearchHistoryObject(existingHistory)) {
            existingHistory = {};
        }
        return existingHistory;
    }

    /**
     * For a list of options, checks if they are stored in local storage and need their bookmarks/isBookmarked updated. If so, updates them.
     * @param searchBarId The search bar ID
     * @param userId The user ID
     * @param options The options to check
     */
    static updateHistoryItems(searchBarId: string, userId: string, options: AutocompleteOption[] | undefined) {
        if (!options) return;
        // Find all search history objects in localStorage
        const searchHistoryKeys = getLocalStorageKeys({
            prefix: SearchHistory.SEARCH_HISTORY_PREFIX,
            suffix: userId,
        });
        // For each search history object, perform update
        searchHistoryKeys.forEach(key => {
            // Get history data
            let existingHistory: SearchHistoryObject = {};
            // Try to parse existing history
            try {
                const parsedHistory = JSON.parse(localStorage.getItem(key) ?? "{}");
                // If it's not an object, set it to an empty object
                if (typeof parsedHistory !== "object") existingHistory = {};
                else existingHistory = parsedHistory;
            } catch (e) {
                existingHistory = {};
            }
            // Find options that are in history, and update if bookmarks or isBookmarked are different
            const updatedHistory = options.map(option => {
                // bookmarks and isBookmarked are not in shortcuts or actions
                if (option.__typename === "Shortcut" || option.__typename === "Action") return null;
                const historyItem = existingHistory[option.label];
                if (historyItem && historyItem.option.__typename !== "Shortcut" && historyItem.option.__typename !== "Action") {
                    const { bookmarks, isBookmarked } = option;
                    if (bookmarks !== historyItem.option.bookmarks || isBookmarked !== historyItem.option.isBookmarked) {
                        return {
                            timestamp: historyItem.timestamp,
                            option: {
                                ...historyItem.option,
                                bookmarks: option.bookmarks,
                                isBookmarked: option.isBookmarked,
                            },
                        };
                    }
                }
                return null;
            }).filter(Boolean) as SearchOptionHistory[];
            // Update changed options
            if (updatedHistory.length > 0) {
                for (const historyItem of updatedHistory) {
                    existingHistory[historyItem.option.label] = historyItem;
                }
                localStorage.setItem(key, JSON.stringify(existingHistory));
            }
        });
    }

    /**
     * Clears search history from all search bars
     */
    static clearSearchHistory(session: Session) {
        const { id } = getCurrentUser(session);
        if (!id) return;
        // Find all search history objects in localStorage
        const searchHistoryKeys = getLocalStorageKeys({
            prefix: SearchHistory.SEARCH_HISTORY_PREFIX,
            suffix: id ?? "",
        });
        // Clear them
        searchHistoryKeys.forEach(key => {
            localStorage.removeItem(key);
        });
        PubSub.get().publish("snack", { messageKey: "SearchHistoryCleared", severity: "Success" });
    }

    /**
     * Removes a specific search history item based on its label
     */
    static removeSearchHistoryItem(searchBarId: string, userId: string, label: string): void {
        const key = `${SearchHistory.SEARCH_HISTORY_PREFIX}${searchBarId}-${userId}`;
        const existingHistory = SearchHistory.getSearchHistory(searchBarId, userId);
        delete existingHistory[label];
        if (Object.keys(existingHistory).length === 0) {
            localStorage.removeItem(key);
        } else {
            localStorage.setItem(key, JSON.stringify(existingHistory));
        }
    }

    /**
     * Adds a new search history item, removing the oldest if exceeding max length
     */
    static addSearchHistoryItem(searchBarId: string, userId: string, option: AutocompleteOption): void {
        const existingHistory = SearchHistory.getSearchHistory(searchBarId, userId);
        const label = option.label;
        if (!(label in existingHistory) && Object.keys(existingHistory).length >= SearchHistory.MAX_HISTORY_LENGTH) {
            // Find oldest item more efficiently
            let oldestKey = "";
            let oldestTimestamp = Infinity;
            
            // Iterate once through all keys to find the oldest
            for (const key in existingHistory) {
                const timestamp = existingHistory[key].timestamp;
                if (timestamp < oldestTimestamp) {
                    oldestTimestamp = timestamp;
                    oldestKey = key;
                }
            }
            
            if (oldestKey) {
                delete existingHistory[oldestKey];
            }
        }
        existingHistory[label] = {
            timestamp: Date.now(),
            option: { ...option, isFromHistory: true },
        };
        localStorage.setItem(`${SearchHistory.SEARCH_HISTORY_PREFIX}${searchBarId}-${userId}`, JSON.stringify(existingHistory));
    }
}
