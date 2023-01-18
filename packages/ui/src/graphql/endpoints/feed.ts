import { listOrganizationFields, listProjectFields, listRoutineFields, listStandardFields, listUserFields } from 'graphql/partial';
import { toQuery } from 'graphql/utils';

export const feedEndpoint = {
    popular: toQuery('popular', 'PopularInput', `{
        organizations {
            ...popular0
        }
        projects {
            ...popular1
        }
        routines {
            ...popular2
        }
        standards {
            ...popular3
        }
        users {
            ...popular4
        }
    }`, [listOrganizationFields, listProjectFields, listRoutineFields, listStandardFields, listUserFields]),
    learn: toQuery('learn', null, `{
        learn {
            courses {
                ...learn0
            }
            tutorials {
                ...learn1
            }
        }
    }`, [listProjectFields, listRoutineFields]),
    research: toQuery('research', null, `{
        processes {
            ...research2
        }
        newlyCompleted {
            ... on Project {
                ...research1
            }
            ... on Routine {
                ...research2
            }
        }
        needVotes {
            ...research1
        }
        needInvestments {
            ...research1
        }
        needMembers {
            ...research0
        }
    }`, [listOrganizationFields, listProjectFields, listRoutineFields]),
    develop: toQuery('develop', null, `{
        completed {
            ... on Project {
                ...develop0
            }
            ... on Routine {
                ...develop1
            }
        }
        inProgress {
            ... on Project {
                ...develop0
            }
            ... on Routine {
                ...develop1
            }
        }
        recent {
            ... on Project {
                ...develop0
            }
            ... on Routine {
                ...develop1
            }
        }
    }`, [listProjectFields, listRoutineFields]),
}