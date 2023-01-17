export const resourceTranslationFields = ['ResourceTranslation', `{
    id
    language
    description
    name
}`] as const;
export const resourceFields = ['Resource', `{
    id
    index
    link
    usedFor
    translations ${resourceTranslationFields[1]}
}`] as const;