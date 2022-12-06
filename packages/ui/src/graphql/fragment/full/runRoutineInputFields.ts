import { gql } from 'graphql-tag';

export const runRoutineInputFields = gql`
    fragment runRoutineInputTagFields on Tag {
        tag
        translations {
            id
            language
            description
        }
    }
    fragment runRoutineInputInputItemFields on InputItem {
        id
        isRequired
        name
        translations {
            id
            language
            description
            helpText
        }
        standard {
            id
            default
            isDeleted
            isInternal
            isPrivate
            name
            type
            props
            yup
            tags {
                ...runRoutineTagFields
            }
            translations {
                id
                language
                description
            }
        }
    }
    fragment runRoutineInputFields on RunRoutineInput {
        id
        data
        input {
            ...runRoutineInputInputItemFields
        }
    }
`