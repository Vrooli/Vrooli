import { getCurrentUser } from "../authentication/session";
import { getLocalStorageKeys } from "../localStorage";
import { PubSub } from "../pubsub";
export const clearSearchHistory = (session) => {
    const { id } = getCurrentUser(session);
    const searchHistoryKeys = getLocalStorageKeys({
        prefix: "search-history-",
        suffix: id ?? "",
    });
    searchHistoryKeys.forEach(key => {
        localStorage.removeItem(key);
    });
    PubSub.get().publishSnack({ messageKey: "SearchHistoryCleared", severity: "Success" });
};
//# sourceMappingURL=clearSearchHistory.js.map