import { resourceFields } from "./resource";

export const resourceListTranslationFields = ['ResourceListTranslation', `{
    id
    language
    description
    title
}`] as const;
export const resourceListFields = ['ResourceList', `{
    id
    created_at
    index
    usedFor
    translations ${resourceListTranslationFields[1]}
    resources ${resourceFields[1]}
}`] as const;