import { listRunRoutineFields, listStarFields, listViewFields } from 'graphql/fragment';
import { toGql } from 'graphql/utils';

export const historyEndpoint = {
    history: toGql('query', 'history', 'HistoryInput', [listRunRoutineFields, listStarFields, listViewFields], `
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