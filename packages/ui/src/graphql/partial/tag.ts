const type = 'Tag';
export const tagTranslationFields = [type, `{
    id
    language
    description
}`] as const;
export const tagFields = [type, `{
    id
    tag
    created_at
    stars
    isStarred
    translations ${tagTranslationFields}
}`] as const;