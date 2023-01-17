import { listOrganizationFields, listProjectFields, listRoutineFields, listStandardFields, listUserFields } from 'graphql/partial';
import { toQuery } from 'graphql/utils';

export const feedEndpoint = {
    popular: toQuery('popular', 'PopularInput', [listOrganizationFields, listProjectFields, listRoutineFields, listStandardFields, listUserFields], `
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
    learn: toQuery('learn', null, [listProjectFields, listRoutineFields], `
        learn {
            courses {
                ...listProjectFields
            }
            tutorials {
                ...listRoutineFields
            }
        }
    `),
    research: toQuery('research', null, [listOrganizationFields, listProjectFields, listRoutineFields], `
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
    develop: toQuery('develop', null, [listProjectFields, listRoutineFields], `
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