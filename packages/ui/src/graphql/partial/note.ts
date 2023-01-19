import { NoteYou } from "@shared/consts";
import { GqlPartial } from "types";

export const noteYouPartial: GqlPartial<NoteYou> = {
    __typename: 'NoteYou',
    full: {
        canDelete: true,
        canEdit: true,
        canStar: true,
        canTransfer: true,
        canView: true,
        canVote: true,
        isStarred: true,
        isUpvoted: true,
        isViewed: true,
    },
}

export const listNoteFields = ['Note', `{
    id
}`] as const;
export const noteFields = ['Note', `{
    id
}`] as const;