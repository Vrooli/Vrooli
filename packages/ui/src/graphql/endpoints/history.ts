import { listRunRoutineFields, listStarFields, listViewFields } from 'graphql/partial';
import { toQuery } from 'graphql/utils';

export const historyEndpoint = {
    history: toQuery('history', 'HistoryInput', [listRunRoutineFields, listStarFields, listViewFields], `
        activeRuns {
            ...listRunRoutineFields
        }
        completedRuns {
            ...listRunRoutineFields
        }
        recentlyViewed {
            ...listViewFields
        }
        recentlyStarred {
            ...listStarFields
        }
    `),
}