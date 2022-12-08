import { gql } from 'graphql-tag';
import { listNoteFields } from 'graphql/fragment';

export const notesQuery = gql`
    ${listNoteFields}
    query notes($input: NoteSearchInput!) {
        notes(input: $input) {
            pageInfo {
                endCursor
                hasNextPage
            }
            edges {
                cursor
                node {
                    ...listNoteFields
                }
            }
        }
    }
`