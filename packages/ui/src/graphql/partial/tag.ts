const __typename = 'Tag';
export const tagTranslationFields = [__typename, `{
    id
    language
    description
}`] as const;
export const tagFields = [__typename, `{
    id
    tag
    created_at
    stars
    isStarred
    translations ${tagTranslationFields}
}`] as const;