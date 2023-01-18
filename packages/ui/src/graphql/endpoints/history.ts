import { listRunRoutineFields, listStarFields, listViewFields } from 'graphql/partial';
import { toQuery } from 'graphql/utils';

export const historyEndpoint = {
    history: toQuery('history', 'HistoryInput', `{
        activeRuns {
            ...history0
        }
        completedRuns {
            ...history0
        }
        recentlyStarred {
            ...history1
        }
        recentlyViewed {
            ...history2
        }
    }`, [listRunRoutineFields, listStarFields, listViewFields]),
}