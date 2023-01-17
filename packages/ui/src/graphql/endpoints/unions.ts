import { listProjectOrOrganizationFields, listProjectOrRoutineFields, listRunProjectOrRunRoutineFields } from 'graphql/partial';
import { toQuery, toSearch } from 'graphql/utils';

export const unionEndpoint = {
    projectOrRoutines: toQuery('projectOrRoutines', 'ProjectOrRoutineSearchInput', [listProjectOrRoutineFields], toSearch(listProjectOrRoutineFields)),
    projectOrOrganizations: toQuery('projectOrOrganizations', 'ProjectOrOrganizationSearchInput', [listProjectOrOrganizationFields], toSearch(listProjectOrOrganizationFields)),
    runProjectOrRunRoutines: toQuery('runProjectOrRunRoutines', 'RunProjectOrRunRoutineSearchInput', [listRunProjectOrRunRoutineFields], toSearch(listRunProjectOrRunRoutineFields)),
}