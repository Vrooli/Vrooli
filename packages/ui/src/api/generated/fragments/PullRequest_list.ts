export const PullRequest_list = `fragment PullRequest_list on PullRequest {
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
}`;