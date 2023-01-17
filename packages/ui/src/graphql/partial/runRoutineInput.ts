import { tagFields } from "./tag";

export const listRunRoutineInputFields = ['RunRoutineInput', `{
    id
}`] as const;
export const runRoutineInputFields = ['RunRoutineInput', `{
    id
    data
    input {
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
            tags ${tagFields[1]}
            translations {
                id
                language
                description
            }
        }
    }
}`] as const;