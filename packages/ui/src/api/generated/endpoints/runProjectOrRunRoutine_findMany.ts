import gql from "graphql-tag";
import { Label_full } from "../fragments/Label_full";
import { Organization_nav } from "../fragments/Organization_nav";
import { RunProject_list } from "../fragments/RunProject_list";
import { RunRoutine_list } from "../fragments/RunRoutine_list";
import { User_nav } from "../fragments/User_nav";

export const runProjectOrRunRoutineFindMany = gql`${Label_full}
${Organization_nav}
${RunProject_list}
${RunRoutine_list}
${User_nav}

query runProjectOrRunRoutines($input: RunProjectOrRunRoutineSearchInput!) {
  runProjectOrRunRoutines(input: $input) {
    edges {
        cursor
        node {
            ... on RunProject {
                ...RunProject_list
            }
            ... on RunRoutine {
                ...RunRoutine_list
            }
        }
    }
    pageInfo {
        endCursor
        hasNextPage
    }
  }
}`;

