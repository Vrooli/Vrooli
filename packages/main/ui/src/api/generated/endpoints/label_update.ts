import gql from "graphql-tag";
import { Organization_nav } from "../fragments/Organization_nav";
import { User_nav } from "../fragments/User_nav";

export const labelUpdate = gql`${Organization_nav}
${User_nav}

mutation labelUpdate($input: LabelUpdateInput!) {
  labelUpdate(input: $input) {
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

