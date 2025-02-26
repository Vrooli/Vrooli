import { Session } from "@local/shared";
import { getCurrentUser } from "utils/authentication/session.js";
import { getLocalStorageKeys } from "utils/localStorage.js";
import { PubSub } from "utils/pubsub.js";

/**
 * Clears search history from all search bars
 */
export function clearSearchHistory(session: Session) {
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
    PubSub.get().publish("snack", { messageKey: "SearchHistoryCleared", severity: "Success" });
}
