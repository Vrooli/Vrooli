import { NoteVersionTranslation } from "@shared/consts";
import { GqlPartial } from "types";

export const noteVersionTranslationPartial: GqlPartial<NoteVersionTranslation> = {
    __typename: 'NoteVersionTranslation',
    full: () => ({
        id: true,
        language: true,
        description: true,
        text: true,
    }),
}

export const listNoteVersionFields = ['NoteVersion', `{
    id
}`] as const;
export const noteVersionFields = ['NoteVersion', `{
    id
}`] as const;