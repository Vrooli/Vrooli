const __typename = 'Tag';
export const tagTranslationFields = [__typename, `{
    id
    language
    description
}`] as const;
export const tagYouFields = ['TagYou', `{
    isOwn
    isStarred
}`] as const;
export const tagFields = [__typename, `{
    id
    tag
    created_at
    stars
    translations ${tagTranslationFields[1]}
    you ${tagYouFields[1]}
}`] as const;