import { listOrganizationFields, listProjectFields, listRoutineFields, listStandardFields, listUserFields } from 'graphql/fragment';
import { toGql } from 'graphql/utils';

export const feedEndpoint = {
    popular: toGql('query', 'popular', 'PopularInput', [listOrganizationFields, listProjectFields, listRoutineFields, listStandardFields, listUserFields], `
        organizations {
            ...listOrganizationFields
        }
        projects {
            ...listProjectFields
        }
        routines {
            ...listRoutineFields
        }
        standards {
            ...listStandardFields
        }
        users {
            ...listUserFields
        }
    `),
    learn: toGql('query', 'learn', null, [listProjectFields, listRoutineFields], `
        learn {
            courses {
                ...listProjectFields
            }
            tutorials {
                ...listRoutineFields
            }
        }
    `),
    research: toGql('query', 'research', null, [listOrganizationFields, listProjectFields, listRoutineFields], `
        processes {
            ...listRoutineFields
        }
        newlyCompleted {
            ... on Project {
                ...listProjectFields
            }
            ... on Routine {
                ...listRoutineFields
            }
        }
        needVotes {
            ...listProjectFields
        }
        needInvestments {
            ...listProjectFields
        }
        needMembers {
            ...listOrganizationFields
        }
    `),
    develop: toGql('query', 'develop', null, [listProjectFields, listRoutineFields], `
        completed {
            ... on Project {
                ...listProjectFields
            }
            ... on Routine {
                ...listRoutineFields
            }
        }
        inProgress {
            ... on Project {
                ...listProjectFields
            }
            ... on Routine {
                ...listRoutineFields
            }
        }
        recent {
            ... on Project {
                ...listProjectFields
            }
            ... on Routine {
                ...listRoutineFields
            }
        }
    `),
}