import gql from 'graphql-tag';
import { Api_list } from '../fragments/Api_list';
import { Organization_nav } from '../fragments/Organization_nav';
import { User_nav } from '../fragments/User_nav';
import { Tag_list } from '../fragments/Tag_list';
import { Label_list } from '../fragments/Label_list';
import { ApiVersion_list } from '../fragments/ApiVersion_list';
import { Note_list } from '../fragments/Note_list';
import { NoteVersion_list } from '../fragments/NoteVersion_list';
import { Project_list } from '../fragments/Project_list';
import { ProjectVersion_list } from '../fragments/ProjectVersion_list';
import { Routine_list } from '../fragments/Routine_list';
import { Label_full } from '../fragments/Label_full';
import { RoutineVersion_list } from '../fragments/RoutineVersion_list';
import { SmartContract_list } from '../fragments/SmartContract_list';
import { SmartContractVersion_list } from '../fragments/SmartContractVersion_list';
import { Standard_list } from '../fragments/Standard_list';
import { StandardVersion_list } from '../fragments/StandardVersion_list';

export const pullRequestFindOne = gql`...${Api_list}
...${Organization_nav}
...${User_nav}
...${Tag_list}
...${Label_list}
...${ApiVersion_list}
...${Note_list}
...${NoteVersion_list}
...${Project_list}
...${ProjectVersion_list}
...${Routine_list}
...${Label_full}
...${RoutineVersion_list}
...${SmartContract_list}
...${SmartContractVersion_list}
...${Standard_list}
...${StandardVersion_list}

query pullRequest($input: FindByIdInput!) {
  pullRequest(input: $input) {
    id
    created_at
    updated_at
    mergedOrRejectedAt
    commentsCount
    status
    from {
        ... on ApiVersion {
            ...ApiVersion_list
        }
        ... on NoteVersion {
            ...NoteVersion_list
        }
        ... on ProjectVersion {
            ...ProjectVersion_list
        }
        ... on RoutineVersion {
            ...RoutineVersion_list
        }
        ... on SmartContractVersion {
            ...SmartContractVersion_list
        }
        ... on StandardVersion {
            ...StandardVersion_list
        }
    }
    to {
        ... on Api {
            ...Api_list
        }
        ... on Note {
            ...Note_list
        }
        ... on Project {
            ...Project_list
        }
        ... on Routine {
            ...Routine_list
        }
        ... on SmartContract {
            ...SmartContract_list
        }
        ... on Standard {
            ...Standard_list
        }
    }
    createdBy {
        id
        name
        handle
    }
    you {
        canComment
        canDelete
        canEdit
        canReport
    }
  }
}`;

export const pullRequestFindMany = gql`...${Api_list}
...${Organization_nav}
...${User_nav}
...${Tag_list}
...${Label_list}
...${ApiVersion_list}
...${Note_list}
...${NoteVersion_list}
...${Project_list}
...${ProjectVersion_list}
...${Routine_list}
...${Label_full}
...${RoutineVersion_list}
...${SmartContract_list}
...${SmartContractVersion_list}
...${Standard_list}
...${StandardVersion_list}

query pullRequests($input: PullRequestSearchInput!) {
  pullRequests(input: $input) {
    edges {
        cursor
        node {
            id
            created_at
            updated_at
            mergedOrRejectedAt
            commentsCount
            status
            from {
                ... on ApiVersion {
                    ...ApiVersion_list
                }
                ... on NoteVersion {
                    ...NoteVersion_list
                }
                ... on ProjectVersion {
                    ...ProjectVersion_list
                }
                ... on RoutineVersion {
                    ...RoutineVersion_list
                }
                ... on SmartContractVersion {
                    ...SmartContractVersion_list
                }
                ... on StandardVersion {
                    ...StandardVersion_list
                }
            }
            to {
                ... on Api {
                    ...Api_list
                }
                ... on Note {
                    ...Note_list
                }
                ... on Project {
                    ...Project_list
                }
                ... on Routine {
                    ...Routine_list
                }
                ... on SmartContract {
                    ...SmartContract_list
                }
                ... on Standard {
                    ...Standard_list
                }
            }
            createdBy {
                id
                name
                handle
            }
            you {
                canComment
                canDelete
                canEdit
                canReport
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const pullRequestCreate = gql`...${Api_list}
...${Organization_nav}
...${User_nav}
...${Tag_list}
...${Label_list}
...${ApiVersion_list}
...${Note_list}
...${NoteVersion_list}
...${Project_list}
...${ProjectVersion_list}
...${Routine_list}
...${Label_full}
...${RoutineVersion_list}
...${SmartContract_list}
...${SmartContractVersion_list}
...${Standard_list}
...${StandardVersion_list}

mutation pullRequestCreate($input: PullRequestCreateInput!) {
  pullRequestCreate(input: $input) {
    id
    created_at
    updated_at
    mergedOrRejectedAt
    commentsCount
    status
    from {
        ... on ApiVersion {
            ...ApiVersion_list
        }
        ... on NoteVersion {
            ...NoteVersion_list
        }
        ... on ProjectVersion {
            ...ProjectVersion_list
        }
        ... on RoutineVersion {
            ...RoutineVersion_list
        }
        ... on SmartContractVersion {
            ...SmartContractVersion_list
        }
        ... on StandardVersion {
            ...StandardVersion_list
        }
    }
    to {
        ... on Api {
            ...Api_list
        }
        ... on Note {
            ...Note_list
        }
        ... on Project {
            ...Project_list
        }
        ... on Routine {
            ...Routine_list
        }
        ... on SmartContract {
            ...SmartContract_list
        }
        ... on Standard {
            ...Standard_list
        }
    }
    createdBy {
        id
        name
        handle
    }
    you {
        canComment
        canDelete
        canEdit
        canReport
    }
  }
}`;

export const pullRequestUpdate = gql`...${Api_list}
...${Organization_nav}
...${User_nav}
...${Tag_list}
...${Label_list}
...${ApiVersion_list}
...${Note_list}
...${NoteVersion_list}
...${Project_list}
...${ProjectVersion_list}
...${Routine_list}
...${Label_full}
...${RoutineVersion_list}
...${SmartContract_list}
...${SmartContractVersion_list}
...${Standard_list}
...${StandardVersion_list}

mutation pullRequestUpdate($input: PullRequestUpdateInput!) {
  pullRequestUpdate(input: $input) {
    id
    created_at
    updated_at
    mergedOrRejectedAt
    commentsCount
    status
    from {
        ... on ApiVersion {
            ...ApiVersion_list
        }
        ... on NoteVersion {
            ...NoteVersion_list
        }
        ... on ProjectVersion {
            ...ProjectVersion_list
        }
        ... on RoutineVersion {
            ...RoutineVersion_list
        }
        ... on SmartContractVersion {
            ...SmartContractVersion_list
        }
        ... on StandardVersion {
            ...StandardVersion_list
        }
    }
    to {
        ... on Api {
            ...Api_list
        }
        ... on Note {
            ...Note_list
        }
        ... on Project {
            ...Project_list
        }
        ... on Routine {
            ...Routine_list
        }
        ... on SmartContract {
            ...SmartContract_list
        }
        ... on Standard {
            ...Standard_list
        }
    }
    createdBy {
        id
        name
        handle
    }
    you {
        canComment
        canDelete
        canEdit
        canReport
    }
  }
}`;

export const pullRequestAccept = gql`...${Api_list}
...${Organization_nav}
...${User_nav}
...${Tag_list}
...${Label_list}
...${ApiVersion_list}
...${Note_list}
...${NoteVersion_list}
...${Project_list}
...${ProjectVersion_list}
...${Routine_list}
...${Label_full}
...${RoutineVersion_list}
...${SmartContract_list}
...${SmartContractVersion_list}
...${Standard_list}
...${StandardVersion_list}

mutation pullRequestAcdept($input: FindByIdInput!) {
  pullRequestAcdept(input: $input) {
    id
    created_at
    updated_at
    mergedOrRejectedAt
    commentsCount
    status
    from {
        ... on ApiVersion {
            ...ApiVersion_list
        }
        ... on NoteVersion {
            ...NoteVersion_list
        }
        ... on ProjectVersion {
            ...ProjectVersion_list
        }
        ... on RoutineVersion {
            ...RoutineVersion_list
        }
        ... on SmartContractVersion {
            ...SmartContractVersion_list
        }
        ... on StandardVersion {
            ...StandardVersion_list
        }
    }
    to {
        ... on Api {
            ...Api_list
        }
        ... on Note {
            ...Note_list
        }
        ... on Project {
            ...Project_list
        }
        ... on Routine {
            ...Routine_list
        }
        ... on SmartContract {
            ...SmartContract_list
        }
        ... on Standard {
            ...Standard_list
        }
    }
    createdBy {
        id
        name
        handle
    }
    you {
        canComment
        canDelete
        canEdit
        canReport
    }
  }
}`;

export const pullRequestReject = gql`...${Api_list}
...${Organization_nav}
...${User_nav}
...${Tag_list}
...${Label_list}
...${ApiVersion_list}
...${Note_list}
...${NoteVersion_list}
...${Project_list}
...${ProjectVersion_list}
...${Routine_list}
...${Label_full}
...${RoutineVersion_list}
...${SmartContract_list}
...${SmartContractVersion_list}
...${Standard_list}
...${StandardVersion_list}

mutation pullRequestReject($input: FindByIdInput!) {
  pullRequestReject(input: $input) {
    id
    created_at
    updated_at
    mergedOrRejectedAt
    commentsCount
    status
    from {
        ... on ApiVersion {
            ...ApiVersion_list
        }
        ... on NoteVersion {
            ...NoteVersion_list
        }
        ... on ProjectVersion {
            ...ProjectVersion_list
        }
        ... on RoutineVersion {
            ...RoutineVersion_list
        }
        ... on SmartContractVersion {
            ...SmartContractVersion_list
        }
        ... on StandardVersion {
            ...StandardVersion_list
        }
    }
    to {
        ... on Api {
            ...Api_list
        }
        ... on Note {
            ...Note_list
        }
        ... on Project {
            ...Project_list
        }
        ... on Routine {
            ...Routine_list
        }
        ... on SmartContract {
            ...SmartContract_list
        }
        ... on Standard {
            ...Standard_list
        }
    }
    createdBy {
        id
        name
        handle
    }
    you {
        canComment
        canDelete
        canEdit
        canReport
    }
  }
}`;

