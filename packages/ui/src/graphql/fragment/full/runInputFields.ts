import { gql } from 'graphql-tag';

export const runInputFields = gql`
    fragment runInputTagFields on Tag {
        tag
        translations {
            id
            language
            description
        }
    }
    fragment runInputInputFields on InputItem {
        id
        isRequired
        name
        translations {
            id
            language
            description
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
                ...runTagFields
            }
            translations {
                id
                language
                description
            }
            version
            versionGroupId
        }
    }
    fragment runInputFields on RunInput {
        id
        data
        input {
            ...runInputInputFields
        }
    }
`