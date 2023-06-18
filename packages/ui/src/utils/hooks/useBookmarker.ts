import { Bookmark, BookmarkCreateInput, BookmarkFor, BookmarkSearchInput, BookmarkSearchResult, DeleteOneInput, DeleteType, endpointGetBookmarks, endpointPostBookmark, endpointPostDeleteOne, exists, Success, uuid } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { useCallback, useContext, useEffect, useMemo, useState } from "react";
import { ObjectActionComplete } from "utils/actions/objectActions";
import { getCurrentUser } from "utils/authentication/session";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";
import { shapeBookmark } from "utils/shape/models/bookmark";
import { useLazyFetch } from "./useLazyFetch";

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
    const session = useContext(SessionContext);
    const { bookmarkLists } = useMemo(() => getCurrentUser(session), [session]);

    const [addBookmark] = useLazyFetch<BookmarkCreateInput, Bookmark>(endpointPostBookmark);
    const [deleteOne] = useLazyFetch<DeleteOneInput, Success>(endpointPostDeleteOne);
    // In most cases, we must query for bookmarks to remove them, since 
    // we usually only know that an object has a bookmark - not the bookmarks themselves
    const [getData, { data, loading }] = useLazyFetch<BookmarkSearchInput, BookmarkSearchResult>(endpointGetBookmarks);

    const hasBookmarkingSupport = exists(BookmarkFor[objectType]);

    // Handle dialog for updating a bookmark's lists
    const [isBookmarkDialogOpen, setIsBookmarkDialogOpen] = useState<boolean>(false);
    const closeBookmarkDialog = useCallback(() => { setIsBookmarkDialogOpen(false); }, []);

    const handleAdd = useCallback(() => {
        const bookmarkListId = bookmarkLists && bookmarkLists.length ? bookmarkLists[0].id : undefined;
        fetchLazyWrapper<BookmarkCreateInput, Bookmark>({
            fetch: addBookmark,
            inputs: shapeBookmark.create({
                id: uuid(),
                to: {
                    __typename: BookmarkFor[objectType],
                    id: objectId!,
                },
                list: {
                    __typename: "BookmarkList",
                    id: bookmarkListId ?? uuid(),
                    // Setting label marks this as a create, 
                    // which should only be done if there is no bookmarkListId
                    label: bookmarkListId ? undefined : "Favorites",
                },
            }),
            onSuccess: (data) => { onActionComplete(ObjectActionComplete.Bookmark, data); },
        });
    }, [bookmarkLists, addBookmark, objectType, objectId, onActionComplete]);

    const handleRemove = useCallback(async () => {
        // First we must query for bookmarks on this object. There may be more than one.
        getData({
            //Lowercase first letter of objectType and add "Id"
            [`${objectType[0].toLowerCase()}${objectType.slice(1)}Id`]: objectId,
        });
    }, [getData, objectId, objectType]);
    useEffect(() => {
        if (!data || loading) return;
        const bookmarks = data?.edges.map(edge => edge.node);
        // If there are no bookmarks, show error snack
        if (bookmarks === undefined || bookmarks.length === 0) {
            PubSub.get().publishSnack({ message: "Could not find bookmark", severity: "Error" });
        }
        // If there is exactly one bookmark, delete it
        else if (bookmarks.length === 1) {
            fetchLazyWrapper<DeleteOneInput, Success>({
                fetch: deleteOne,
                inputs: { id: bookmarks[0].id, objectType: DeleteType.Bookmark },
                onSuccess: (data) => { onActionComplete(ObjectActionComplete.BookmarkUndo, data); },
            });
            return;
        }
        // If there is more than one bookmark, open dialog to select which one to delete
        else {
            setIsBookmarkDialogOpen(true);
        }
    }, [data, loading, deleteOne, onActionComplete]);


    const handleBookmark = useCallback((isAdding: boolean) => {
        // Validate objectId and objectType
        if (!objectId) {
            PubSub.get().publishSnack({ messageKey: "CouldNotReadObject", severity: "Error" });
            return;
        }
        if (!hasBookmarkingSupport) {
            PubSub.get().publishSnack({ messageKey: "BookmarkNotSupported", severity: "Error" });
            return;
        }
        if (isAdding) {
            handleAdd();
        } else {
            handleRemove();
        }
    }, [handleAdd, handleRemove, hasBookmarkingSupport, objectId]);

    return {
        isBookmarkDialogOpen,
        handleBookmark,
        closeBookmarkDialog,
        hasBookmarkingSupport,
    };
};
