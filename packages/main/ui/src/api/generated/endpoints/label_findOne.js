import gql from "graphql-tag";
import { Organization_nav } from "../fragments/Organization_nav";
import { User_nav } from "../fragments/User_nav";
export const labelFindOne = gql `${Organization_nav}
${User_nav}

query label($input: FindByIdInput!) {
  label(input: $input) {
    apisCount
    focusModesCount
    issuesCount
    meetingsCount
    notesCount
    projectsCount
    routinesCount
    schedulesCount
    smartContractsCount
    standardsCount
    id
    created_at
    updated_at
    color
    label
    owner {
        ... on Organization {
            ...Organization_nav
        }
        ... on User {
            ...User_nav
        }
    }
    you {
        canDelete
        canUpdate
    }
  }
}`;
//# sourceMappingURL=label_findOne.js.map