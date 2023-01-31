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

export const smartContractVersionFindOne = gql`...${Api_list}
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

query smartContractVersion($input: FindByIdInput!) {
  smartContractVersion(input: $input) {
    versionNotes
    pullRequest {
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
    resourceList {
        id
        created_at
        translations {
            id
            language
            description
            name
        }
        resources {
            id
            index
            link
            usedFor
            translations {
                id
                language
                description
                name
            }
        }
    }
    root {
        stats {
            id
            created_at
            periodStart
            periodEnd
            periodType
            calls
        }
        id
        created_at
        updated_at
        isPrivate
        issuesCount
        labels {
            ...Label_list
        }
        owner {
            ... on Organization {
                ...Organization_nav
            }
            ... on User {
                ...User_nav
            }
        }
        permissions
        questionsCount
        score
        stars
        tags {
            ...Tag_list
        }
        transfersCount
        views
        you {
            canDelete
            canEdit
            canStar
            canTransfer
            canView
            canVote
            isStarred
            isUpvoted
            isViewed
        }
    }
    translations {
        id
        language
        description
        jsonVariable
    }
    id
    created_at
    updated_at
    isComplete
    isDeleted
    isLatest
    isPrivate
    default
    contractType
    content
    versionIndex
    versionLabel
    commentsCount
    directoryListingsCount
    forksCount
    reportsCount
    you {
        canComment
        canCopy
        canDelete
        canEdit
        canReport
        canUse
        canView
    }
  }
}`;

export const smartContractVersionFindMany = gql`
query smartContractVersions($input: SmartContractVersionSearchInput!) {
  smartContractVersions(input: $input) {
    edges {
        cursor
        node {
            translations {
                id
                language
                description
                jsonVariable
            }
            id
            created_at
            updated_at
            isComplete
            isDeleted
            isLatest
            isPrivate
            default
            contractType
            content
            versionIndex
            versionLabel
            commentsCount
            directoryListingsCount
            forksCount
            reportsCount
            you {
                canComment
                canCopy
                canDelete
                canEdit
                canReport
                canUse
                canView
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const smartContractVersionCreate = gql`...${Api_list}
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

mutation smartContractVersionCreate($input: SmartContractVersionCreateInput!) {
  smartContractVersionCreate(input: $input) {
    versionNotes
    pullRequest {
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
    resourceList {
        id
        created_at
        translations {
            id
            language
            description
            name
        }
        resources {
            id
            index
            link
            usedFor
            translations {
                id
                language
                description
                name
            }
        }
    }
    root {
        stats {
            id
            created_at
            periodStart
            periodEnd
            periodType
            calls
        }
        id
        created_at
        updated_at
        isPrivate
        issuesCount
        labels {
            ...Label_list
        }
        owner {
            ... on Organization {
                ...Organization_nav
            }
            ... on User {
                ...User_nav
            }
        }
        permissions
        questionsCount
        score
        stars
        tags {
            ...Tag_list
        }
        transfersCount
        views
        you {
            canDelete
            canEdit
            canStar
            canTransfer
            canView
            canVote
            isStarred
            isUpvoted
            isViewed
        }
    }
    translations {
        id
        language
        description
        jsonVariable
    }
    id
    created_at
    updated_at
    isComplete
    isDeleted
    isLatest
    isPrivate
    default
    contractType
    content
    versionIndex
    versionLabel
    commentsCount
    directoryListingsCount
    forksCount
    reportsCount
    you {
        canComment
        canCopy
        canDelete
        canEdit
        canReport
        canUse
        canView
    }
  }
}`;

export const smartContractVersionUpdate = gql`...${Api_list}
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

mutation smartContractVersionUpdate($input: SmartContractVersionUpdateInput!) {
  smartContractVersionUpdate(input: $input) {
    versionNotes
    pullRequest {
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
    resourceList {
        id
        created_at
        translations {
            id
            language
            description
            name
        }
        resources {
            id
            index
            link
            usedFor
            translations {
                id
                language
                description
                name
            }
        }
    }
    root {
        stats {
            id
            created_at
            periodStart
            periodEnd
            periodType
            calls
        }
        id
        created_at
        updated_at
        isPrivate
        issuesCount
        labels {
            ...Label_list
        }
        owner {
            ... on Organization {
                ...Organization_nav
            }
            ... on User {
                ...User_nav
            }
        }
        permissions
        questionsCount
        score
        stars
        tags {
            ...Tag_list
        }
        transfersCount
        views
        you {
            canDelete
            canEdit
            canStar
            canTransfer
            canView
            canVote
            isStarred
            isUpvoted
            isViewed
        }
    }
    translations {
        id
        language
        description
        jsonVariable
    }
    id
    created_at
    updated_at
    isComplete
    isDeleted
    isLatest
    isPrivate
    default
    contractType
    content
    versionIndex
    versionLabel
    commentsCount
    directoryListingsCount
    forksCount
    reportsCount
    you {
        canComment
        canCopy
        canDelete
        canEdit
        canReport
        canUse
        canView
    }
  }
}`;

