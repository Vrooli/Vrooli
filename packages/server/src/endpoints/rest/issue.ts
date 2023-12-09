import { issue_close, issue_create, issue_findMany, issue_findOne, issue_update } from "../generated";
import { IssueEndpoints } from "../logic/issue";
import { setupRoutes } from "./base";

export const IssueRest = setupRoutes({
    "/issue/:id": {
        get: [IssueEndpoints.Query.issue, issue_findOne],
        put: [IssueEndpoints.Mutation.issueUpdate, issue_update],
    },
    "/issues": {
        get: [IssueEndpoints.Query.issues, issue_findMany],
    },
    "/issue": {
        post: [IssueEndpoints.Mutation.issueCreate, issue_create],
    },
    "/issue/:id/close": {
        put: [IssueEndpoints.Mutation.issueClose, issue_close],
    },
});
