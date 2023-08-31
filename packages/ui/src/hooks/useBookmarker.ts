import { Bookmark, BookmarkCreateInput, BookmarkFor, BookmarkSearchInput, BookmarkSearchResult, DeleteOneInput, DeleteType, endpointGetBookmarks, endpointPostBookmark, endpointPostDeleteOne, exists, GqlModelType, Success, uuid } from "@local/shared";
import { fetchLazyWrapper } from "api";
import { SessionContext } from "contexts/SessionContext";
import { useCallback, useContext, useMemo, useRef, useState } from "react";
import { ObjectActionComplete } from "utils/actions/objectActions";
import { getCurrentUser } from "utils/authentication/session";
import { PubSub } from "utils/pubsub";
import { shapeBookmark } from "utils/shape/models/bookmark";
import { useLazyFetch } from "./useLazyFetch";

type UseBookmarkerProps = {
    objectId: string | null | undefined;
    objectType: `${GqlModelType}`
    onActionComplete: (action: ObjectActionComplete.Bookmark | ObjectActionComplete.BookmarkUndo, data: Bookmark | Success) => unknown;
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
    const [getData] = useLazyFetch<BookmarkSearchInput, BookmarkSearchResult>(endpointGetBookmarks);

    const hasBookmarkingSupport = exists(BookmarkFor[objectType]);

    // Handle dialog for updating a bookmark's lists
    const [isBookmarkDialogOpen, setIsBookmarkDialogOpen] = useState<boolean>(false);
    const closeBookmarkDialog = useCallback(() => { setIsBookmarkDialogOpen(false); }, []);

    const handleAdd = useCallback(() => {
        if (!objectId) {
            PubSub.get().publishSnack({ messageKey: "NotFound", severity: "Error" });
            return;
        }
        const bookmarkListId = bookmarkLists && bookmarkLists.length ? bookmarkLists[0].id : undefined;
        console.log("adding bookmark", bookmarkListId, shapeBookmark.create({
            id: uuid(),
            to: {
                __typename: BookmarkFor[objectType],
                id: objectId,
            },
            list: {
                __typename: "BookmarkList",
                id: bookmarkListId ?? uuid(),
                // Setting label marks this as a create, 
                // which should only be done if there is no bookmarkListId
                label: bookmarkListId ? undefined : "Favorites",
            },
        }));
        fetchLazyWrapper<BookmarkCreateInput, Bookmark>({
            fetch: addBookmark,
            inputs: shapeBookmark.create({
                id: uuid(),
                to: {
                    __typename: BookmarkFor[objectType],
                    id: objectId,
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

    const isRemoveProcessingRef = useRef<boolean>(false);
    const handleRemove = useCallback(async () => {
        if (isRemoveProcessingRef.current || !objectId) return;
        isRemoveProcessingRef.current = true;

        // Fetch bookmarks for the given objectId and objectType.
        const result = await getData({
            [`${objectType[0].toLowerCase()}${objectType.slice(1)}Id`]: objectId,
        });

        // Extract bookmarks from the result.
        const bookmarks = result?.data?.edges.map(edge => edge.node);

        // If no bookmarks are found, display an error.
        if (!bookmarks || bookmarks.length === 0) {
            PubSub.get().publishSnack({ message: "Could not find bookmark", severity: "Error" });
            isRemoveProcessingRef.current = false;
            return;
        }

        // If there's only one bookmark, delete it.
        if (bookmarks.length === 1) {
            const deletionResult = await deleteOne({
                id: bookmarks[0].id,
                objectType: DeleteType.Bookmark,
            });
            if (deletionResult.data?.success) {
                onActionComplete(ObjectActionComplete.BookmarkUndo, deletionResult.data);
            }
            isRemoveProcessingRef.current = false;
            return;
        }

        // If there are multiple bookmarks, open a dialog for the user to select which to delete.
        setIsBookmarkDialogOpen(true);
        isRemoveProcessingRef.current = false;
    }, [getData, deleteOne, objectId, objectType, onActionComplete]);


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
