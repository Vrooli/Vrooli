import { projectOrOrganizationPartial, projectOrRoutinePartial, runProjectOrRunRoutinePartial } from 'api/partial';
import { toQuery, toSearch } from 'api/utils';

export const unionEndpoint = {
    projectOrRoutines: toQuery('projectOrRoutines', 'ProjectOrRoutineSearchInput', ...toSearch(projectOrRoutinePartial)),
    projectOrOrganizations: toQuery('projectOrOrganizations', 'ProjectOrOrganizationSearchInput', ...toSearch(projectOrOrganizationPartial)),
    runProjectOrRunRoutines: toQuery('runProjectOrRunRoutines', 'RunProjectOrRunRoutineSearchInput', ...toSearch(runProjectOrRunRoutinePartial)),
}