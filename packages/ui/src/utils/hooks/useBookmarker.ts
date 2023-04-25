import { Bookmark, BookmarkCreateInput, BookmarkFor, BookmarkSearchInput, BookmarkSearchResult, DeleteOneInput, DeleteType, exists, Success, uuid } from "@local/shared";
import { mutationWrapper, useCustomLazyQuery, useCustomMutation } from "api";
import { bookmarkCreate } from "api/generated/endpoints/bookmark_create";
import { bookmarkFindMany } from "api/generated/endpoints/bookmark_findMany";
import { deleteOneOrManyDeleteOne } from "api/generated/endpoints/deleteOneOrMany_deleteOne";
import { useCallback, useContext, useEffect, useMemo } from "react";
import { ObjectActionComplete } from "utils/actions/objectActions";
import { getCurrentUser } from "utils/authentication/session";
import { PubSub } from "utils/pubsub";
import { SessionContext } from "utils/SessionContext";

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

    const [addBookmark] = useCustomMutation<Bookmark, BookmarkCreateInput>(bookmarkCreate);
    const [deleteOne] = useCustomMutation<Success, DeleteOneInput>(deleteOneOrManyDeleteOne)
    // In most cases, we must query for bookmarks to remove them, since 
    // we usually only know that an object has a bookmark - not the bookmarks themselves
    const [getData, { data, loading }] = useCustomLazyQuery<BookmarkSearchResult, BookmarkSearchInput>(bookmarkFindMany, { errorPolicy: 'all' });

    const hasBookmarkingSupport = exists(BookmarkFor[objectType]);

    const handleAdd = useCallback(() => {
        const bookmarkListId = bookmarkLists && bookmarkLists.length ? bookmarkLists[0].id : undefined;
        mutationWrapper<Bookmark, BookmarkCreateInput>({
            mutation: addBookmark,
            input: {
                id: uuid(),
                bookmarkFor: BookmarkFor[objectType],
                forConnect: objectId!,
                listConnect: bookmarkListId,
                listCreate: bookmarkListId ? undefined : {
                    id: uuid(),
                    label: 'Favorites',
                },
            },
            onSuccess: (data) => { onActionComplete(ObjectActionComplete.Bookmark, data) },
        })
    }, [bookmarkLists, addBookmark, objectType, objectId, onActionComplete]);

    const handleRemove = useCallback(async () => {
        // First we must query for bookmarks on this object. There may be more than one.
        await getData({
            variables: {
                //Lowercase first letter of objectType and add "Id"
                [`${objectType[0].toLowerCase()}${objectType.slice(1)}Id`]: objectId,
            }
        });
    }, [getData, objectId, objectType]);
    useEffect(() => {
        if (!data || loading) return;
        const bookmarks = data?.edges.map(edge => edge.node);
        // If there are no bookmarks, show error snack
        if (bookmarks === undefined || bookmarks.length === 0) {
            PubSub.get().publishSnack({ message: 'Could not find bookmark', severity: 'Error' });
        }
        // If there is exactly one bookmark, delete it
        else if (bookmarks.length === 1) {
            mutationWrapper<Success, DeleteOneInput>({
                mutation: deleteOne,
                input: { id: bookmarks[0].id, objectType: DeleteType.Bookmark },
                onSuccess: (data) => { onActionComplete(ObjectActionComplete.BookmarkUndo, data) },
            });
            return;
        }
        // If there is more than one bookmark, open dialog to select which one to delete
        else {
            console.warn('TODO: Open dialog to select which bookmark to delete')
            //TODO
        }
    }, [data, loading, deleteOne, onActionComplete]);


    const handleBookmark = useCallback((isAdding: boolean) => {
        // Validate objectId and objectType
        if (!objectId) {
            PubSub.get().publishSnack({ messageKey: `CouldNotReadObject`, severity: 'Error' });
            return;
        }
        if (!hasBookmarkingSupport) {
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