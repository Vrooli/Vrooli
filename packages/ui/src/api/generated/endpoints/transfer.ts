import gql from 'graphql-tag';
import { Api_list } from '../fragments/Api_list';
import { Note_list } from '../fragments/Note_list';
import { Project_list } from '../fragments/Project_list';
import { Routine_list } from '../fragments/Routine_list';
import { SmartContract_list } from '../fragments/SmartContract_list';
import { Standard_list } from '../fragments/Standard_list';

export const transferFindOne = gql`${Api_list}
${Note_list}
${Project_list}
${Routine_list}
${SmartContract_list}
${Standard_list}

query transfer($input: FindByIdInput!) {
  transfer(input: $input) {
    id
    created_at
    updated_at
    mergedOrRejectedAt
    status
    fromOwner {
        id
        name
        handle
    }
    toOwner {
        id
        name
        handle
    }
    object {
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
    you {
        canDelete
        canEdit
    }
  }
}`;

export const transferFindMany = gql`${Api_list}
${Note_list}
${Project_list}
${Routine_list}
${SmartContract_list}
${Standard_list}

query transfers($input: TransferSearchInput!) {
  transfers(input: $input) {
    edges {
        cursor
        node {
            id
            created_at
            updated_at
            mergedOrRejectedAt
            status
            fromOwner {
                id
                name
                handle
            }
            toOwner {
                id
                name
                handle
            }
            object {
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
            you {
                canDelete
                canEdit
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

export const transferRequestSend = gql`${Api_list}
${Note_list}
${Project_list}
${Routine_list}
${SmartContract_list}
${Standard_list}

mutation transferRequestSend($input: TransferRequestSendInput!) {
  transferRequestSend(input: $input) {
    id
    created_at
    updated_at
    mergedOrRejectedAt
    status
    fromOwner {
        id
        name
        handle
    }
    toOwner {
        id
        name
        handle
    }
    object {
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
    you {
        canDelete
        canEdit
    }
  }
}`;

export const transferRequestReceive = gql`${Api_list}
${Note_list}
${Project_list}
${Routine_list}
${SmartContract_list}
${Standard_list}

mutation transferRequestReceive($input: TransferRequestReceiveInput!) {
  transferRequestReceive(input: $input) {
    id
    created_at
    updated_at
    mergedOrRejectedAt
    status
    fromOwner {
        id
        name
        handle
    }
    toOwner {
        id
        name
        handle
    }
    object {
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
    you {
        canDelete
        canEdit
    }
  }
}`;

export const transferTransferUpdate = gql`${Api_list}
${Note_list}
${Project_list}
${Routine_list}
${SmartContract_list}
${Standard_list}

mutation transferUpdate($input: TransferUpdateInput!) {
  transferUpdate(input: $input) {
    id
    created_at
    updated_at
    mergedOrRejectedAt
    status
    fromOwner {
        id
        name
        handle
    }
    toOwner {
        id
        name
        handle
    }
    object {
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
    you {
        canDelete
        canEdit
    }
  }
}`;

export const transferTransferCancel = gql`${Api_list}
${Note_list}
${Project_list}
${Routine_list}
${SmartContract_list}
${Standard_list}

mutation transferCancel($input: FindByIdInput!) {
  transferCancel(input: $input) {
    id
    created_at
    updated_at
    mergedOrRejectedAt
    status
    fromOwner {
        id
        name
        handle
    }
    toOwner {
        id
        name
        handle
    }
    object {
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
    you {
        canDelete
        canEdit
    }
  }
}`;

export const transferTransferAccept = gql`${Api_list}
${Note_list}
${Project_list}
${Routine_list}
${SmartContract_list}
${Standard_list}

mutation transferAccept($input: FindByIdInput!) {
  transferAccept(input: $input) {
    id
    created_at
    updated_at
    mergedOrRejectedAt
    status
    fromOwner {
        id
        name
        handle
    }
    toOwner {
        id
        name
        handle
    }
    object {
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
    you {
        canDelete
        canEdit
    }
  }
}`;

export const transferTransferDeny = gql`${Api_list}
${Note_list}
${Project_list}
${Routine_list}
${SmartContract_list}
${Standard_list}

mutation transferDeny($input: TransferDenyInput!) {
  transferDeny(input: $input) {
    id
    created_at
    updated_at
    mergedOrRejectedAt
    status
    fromOwner {
        id
        name
        handle
    }
    toOwner {
        id
        name
        handle
    }
    object {
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
    you {
        canDelete
        canEdit
    }
  }
}`;

