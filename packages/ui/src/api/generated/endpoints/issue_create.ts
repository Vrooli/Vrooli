import gql from 'graphql-tag';
import { Api_nav } from '../fragments/Api_nav';
import { Label_common } from '../fragments/Label_common';
import { Note_nav } from '../fragments/Note_nav';
import { Organization_nav } from '../fragments/Organization_nav';
import { Project_nav } from '../fragments/Project_nav';
import { Routine_nav } from '../fragments/Routine_nav';
import { SmartContract_nav } from '../fragments/SmartContract_nav';
import { Standard_nav } from '../fragments/Standard_nav';
import { User_nav } from '../fragments/User_nav';

export const issueCreate = gql`${Api_nav}
${Label_common}
${Note_nav}
${Organization_nav}
${Project_nav}
${Routine_nav}
${SmartContract_nav}
${Standard_nav}
${User_nav}

mutation issueCreate($input: IssueCreateInput!) {
  issueCreate(input: $input) {
    closedBy {
        id
        name
        handle
    }
    createdBy {
        id
        name
        handle
    }
    translations {
        id
        language
        description
        name
    }
    id
    created_at
    updated_at
    closedAt
    referencedVersionId
    status
    to {
        ... on Api {
            ...Api_nav
        }
        ... on Note {
            ...Note_nav
        }
        ... on Organization {
            ...Organization_nav
        }
        ... on Project {
            ...Project_nav
        }
        ... on Routine {
            ...Routine_nav
        }
        ... on SmartContract {
            ...SmartContract_nav
        }
        ... on Standard {
            ...Standard_nav
        }
    }
    commentsCount
    reportsCount
    score
    bookmarks
    views
    labels {
        ...Label_common
    }
    you {
        canComment
        canDelete
        canBookmark
        canReport
        canUpdate
        canRead
        canReact
        isBookmarked
        reaction
    }
  }
}`;

