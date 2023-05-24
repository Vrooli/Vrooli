import gql from "graphql-tag";
import { Api_list } from "../fragments/Api_list";
import { Label_list } from "../fragments/Label_list";
import { Note_list } from "../fragments/Note_list";
import { Organization_nav } from "../fragments/Organization_nav";
import { Project_list } from "../fragments/Project_list";
import { Routine_list } from "../fragments/Routine_list";
import { SmartContract_list } from "../fragments/SmartContract_list";
import { Standard_list } from "../fragments/Standard_list";
import { Tag_list } from "../fragments/Tag_list";
import { User_nav } from "../fragments/User_nav";

export const transferTransferUpdate = gql`${Api_list}
${Label_list}
${Note_list}
${Organization_nav}
${Project_list}
${Routine_list}
${SmartContract_list}
${Standard_list}
${Tag_list}
${User_nav}

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
        canUpdate
    }
  }
}`;

