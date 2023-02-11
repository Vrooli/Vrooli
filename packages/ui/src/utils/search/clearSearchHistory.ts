import { Session } from '@shared/consts';
import { SnackSeverity } from 'components';
import { getCurrentUser } from 'utils/authentication';
import { getLocalStorageKeys } from 'utils/localStorage';
import { PubSub } from 'utils/pubsub';

/**
 * Clears search history from all search bars
 */
export const clearSearchHistory = (session: Session) => {
    const { id } = getCurrentUser(session);
    // Find all search history objects in localStorage
    const searchHistoryKeys = getLocalStorageKeys({
        prefix: 'search-history-',
        suffix: id ?? '',
    });
    // Clear them
    searchHistoryKeys.forEach(key => {
        localStorage.removeItem(key);
    });
    PubSub.get().publishSnack({ messageKey: 'SearchHistoryCleared', severity: SnackSeverity.Success });
}