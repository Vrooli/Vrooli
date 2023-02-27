import { Bookmark, BookmarkCreateInput, BookmarkFor, BookmarkSearchInput, DeleteOneInput, Success } from "@shared/consts";
import { exists } from "@shared/utils";
import { mutationWrapper, useCustomLazyQuery, useCustomMutation } from "api";
import { bookmarkCreate } from "api/generated/endpoints/bookmark_create";
import { bookmarkFindMany } from "api/generated/endpoints/bookmark_findMany";
import { deleteOneOrManyDeleteOne } from "api/generated/endpoints/deleteOneOrMany_deleteOne";
import { useCallback } from "react";
import { ObjectActionComplete } from "utils/actions";
import { PubSub } from "utils/pubsub";

type UseBookmarkerProps = {
    objectId: string | null | undefined;
    objectType: `${BookmarkFor}`
    onActionComplete: (action: ObjectActionComplete.Bookmark | ObjectActionComplete.BookmarkUndo, data: Bookmark | Success) => void;
}

/**
 * Hook for simplifying the use of adding and removing bookmarks on an object
 */
export const useBookmarker = ({
    objectId,
    objectType,
    onActionComplete,
}: UseBookmarkerProps) => {
    const [addBookmark] = useCustomMutation<Bookmark, BookmarkCreateInput>(bookmarkCreate);
    const [deleteOne] = useCustomMutation<Success, DeleteOneInput>(deleteOneOrManyDeleteOne)
    // In most cases, we must query for bookmarks to remove them, since 
    // we usually only know that an object has a bookmark - not the bookmarks themselves
    const [getData, { data, loading }] = useCustomLazyQuery<Bookmark, BookmarkSearchInput>(bookmarkFindMany, { errorPolicy: 'all' });

    const hasBookmarkingSupport = exists(BookmarkFor[objectType]);

    const handleAdd = useCallback(() => {
        mutationWrapper<Bookmark, BookmarkCreateInput>({
            mutation: addBookmark,
            // input: { isUpvote, forConnect: objectId, voteFor: VoteFor[objectType] },
            // onSuccess: (data) => { onActionComplete(isUpvote ? ObjectActionComplete.VoteUp : ObjectActionComplete.VoteDown, data) },
        })
    }, [objectId, objectType, onActionComplete, addBookmark]);

    const handleRemove = useCallback(async () => {
        // First we must query for bookmarks on this object. There may be more than one.
    }, [objectId, objectType, onActionComplete, deleteOne]);

    const handleBookmark = useCallback((isAdding: boolean) => {
        // Validate objectId and objectType
        if (!objectId) {
            PubSub.get().publishSnack({ messageKey: `CouldNotReadObject`, severity: 'Error' });
            return;
        }
        if(!hasBookmarkingSupport) {
            PubSub.get().publishSnack({ messageKey: 'BookmarkNotSupported', severity: 'Error' });
            return;
        }
        if (isAdding) {
            handleAdd();
        } else {
            handleRemove();
        }
    }, [handleAdd, handleRemove, hasBookmarkingSupport, objectId]);

    return { handleBookmark, hasBookmarkingSupport };
}