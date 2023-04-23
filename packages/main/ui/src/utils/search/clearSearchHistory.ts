import { Session } from "@local/consts";
import { getCurrentUser } from "../authentication/session";
import { getLocalStorageKeys } from "../localStorage";
import { PubSub } from "../pubsub";

/**
 * Clears search history from all search bars
 */
export const clearSearchHistory = (session: Session) => {
    const { id } = getCurrentUser(session);
    // Find all search history objects in localStorage
    const searchHistoryKeys = getLocalStorageKeys({
        prefix: "search-history-",
        suffix: id ?? "",
    });
    // Clear them
    searchHistoryKeys.forEach(key => {
        localStorage.removeItem(key);
    });
    PubSub.get().publishSnack({ messageKey: "SearchHistoryCleared", severity: "Success" });
};
