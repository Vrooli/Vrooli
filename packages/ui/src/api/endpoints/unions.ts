import { projectOrOrganizationPartial, projectOrRoutinePartial, runProjectOrRunRoutinePartial } from '../partial';
import { toQuery, toSearch } from '../utils';

export const unionEndpoint = {
    projectOrRoutines: toQuery('projectOrRoutines', 'ProjectOrRoutineSearchInput', ...toSearch(projectOrRoutinePartial)),
    projectOrOrganizations: toQuery('projectOrOrganizations', 'ProjectOrOrganizationSearchInput', ...toSearch(projectOrOrganizationPartial)),
    runProjectOrRunRoutines: toQuery('runProjectOrRunRoutines', 'RunProjectOrRunRoutineSearchInput', ...toSearch(runProjectOrRunRoutinePartial)),
}