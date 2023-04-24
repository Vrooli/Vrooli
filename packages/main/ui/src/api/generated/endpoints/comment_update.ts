import gql from "graphql-tag";
import { Api_nav } from "../fragments/Api_nav";
import { ApiVersion_nav } from "../fragments/ApiVersion_nav";
import { Issue_nav } from "../fragments/Issue_nav";
import { Note_nav } from "../fragments/Note_nav";
import { NoteVersion_nav } from "../fragments/NoteVersion_nav";
import { Organization_nav } from "../fragments/Organization_nav";
import { Post_nav } from "../fragments/Post_nav";
import { Project_nav } from "../fragments/Project_nav";
import { ProjectVersion_nav } from "../fragments/ProjectVersion_nav";
import { PullRequest_nav } from "../fragments/PullRequest_nav";
import { Question_common } from "../fragments/Question_common";
import { QuestionAnswer_common } from "../fragments/QuestionAnswer_common";
import { Routine_nav } from "../fragments/Routine_nav";
import { RoutineVersion_nav } from "../fragments/RoutineVersion_nav";
import { SmartContract_nav } from "../fragments/SmartContract_nav";
import { SmartContractVersion_nav } from "../fragments/SmartContractVersion_nav";
import { Standard_nav } from "../fragments/Standard_nav";
import { StandardVersion_nav } from "../fragments/StandardVersion_nav";
import { Tag_list } from "../fragments/Tag_list";
import { User_nav } from "../fragments/User_nav";

export const commentUpdate = gql`${Api_nav}
${ApiVersion_nav}
${Issue_nav}
${Note_nav}
${NoteVersion_nav}
${Organization_nav}
${Post_nav}
${Project_nav}
${ProjectVersion_nav}
${PullRequest_nav}
${Question_common}
${QuestionAnswer_common}
${Routine_nav}
${RoutineVersion_nav}
${SmartContract_nav}
${SmartContractVersion_nav}
${Standard_nav}
${StandardVersion_nav}
${Tag_list}
${User_nav}

mutation commentUpdate($input: CommentUpdateInput!) {
  commentUpdate(input: $input) {
    commentedOn {
        ... on ApiVersion {
            ...ApiVersion_nav
        }
        ... on Issue {
            ...Issue_nav
        }
        ... on NoteVersion {
            ...NoteVersion_nav
        }
        ... on Post {
            ...Post_nav
        }
        ... on ProjectVersion {
            ...ProjectVersion_nav
        }
        ... on PullRequest {
            ...PullRequest_nav
        }
        ... on Question {
            ...Question_common
        }
        ... on QuestionAnswer {
            ...QuestionAnswer_common
        }
        ... on RoutineVersion {
            ...RoutineVersion_nav
        }
        ... on SmartContractVersion {
            ...SmartContractVersion_nav
        }
        ... on StandardVersion {
            ...StandardVersion_nav
        }
    }
    translations {
        id
        language
        text
    }
    id
    created_at
    updated_at
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    score
    bookmarks
    reportsCount
    you {
        canDelete
        canBookmark
        canReply
        canReport
        canUpdate
        canReact
        isBookmarked
        reaction
    }
  }
}`;
