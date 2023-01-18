import { listProjectOrOrganizationFields, listProjectOrRoutineFields, listRunProjectOrRunRoutineFields } from 'graphql/partial';
import { toQuery, toSearch } from 'graphql/utils';

export const unionEndpoint = {
    projectOrRoutines: toQuery('projectOrRoutines', 'ProjectOrRoutineSearchInput', toSearch(listProjectOrRoutineFields)),
    projectOrOrganizations: toQuery('projectOrOrganizations', 'ProjectOrOrganizationSearchInput', toSearch(listProjectOrOrganizationFields)),
    runProjectOrRunRoutines: toQuery('runProjectOrRunRoutines', 'RunProjectOrRunRoutineSearchInput', toSearch(listRunProjectOrRunRoutineFields)),
}