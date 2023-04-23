import gql from "graphql-tag";
import { Api_nav } from "../fragments/Api_nav";
import { Note_nav } from "../fragments/Note_nav";
import { Organization_nav } from "../fragments/Organization_nav";
import { Project_nav } from "../fragments/Project_nav";
import { Routine_nav } from "../fragments/Routine_nav";
import { SmartContract_nav } from "../fragments/SmartContract_nav";
import { Standard_nav } from "../fragments/Standard_nav";
import { Tag_list } from "../fragments/Tag_list";
import { User_nav } from "../fragments/User_nav";

export const questionFindOne = gql`${Api_nav}
${Note_nav}
${Organization_nav}
${Project_nav}
${Routine_nav}
${SmartContract_nav}
${Standard_nav}
${Tag_list}
${User_nav}

query question($input: FindByIdInput!) {
  question(input: $input) {
    answers {
        comments {
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
        translations {
            id
            language
            text
        }
        id
        created_at
        updated_at
        createdBy {
            id
            name
            handle
        }
        score
        bookmarks
        isAccepted
        commentsCount
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
    createdBy {
        id
        name
        handle
    }
    hasAcceptedAnswer
    isPrivate
    score
    bookmarks
    answersCount
    commentsCount
    reportsCount
    forObject {
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
    tags {
        ...Tag_list
    }
    you {
        reaction
    }
  }
}`;

