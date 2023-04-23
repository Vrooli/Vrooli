import { BookmarkFor, DeleteType } from "@local/consts";
import { exists } from "@local/utils";
import { uuid } from "@local/uuid";
import { useCallback, useContext, useEffect, useMemo } from "react";
import { mutationWrapper, useCustomLazyQuery, useCustomMutation } from "../../api";
import { bookmarkCreate } from "../../api/generated/endpoints/bookmark_create";
import { bookmarkFindMany } from "../../api/generated/endpoints/bookmark_findMany";
import { deleteOneOrManyDeleteOne } from "../../api/generated/endpoints/deleteOneOrMany_deleteOne";
import { ObjectActionComplete } from "../actions/objectActions";
import { getCurrentUser } from "../authentication/session";
import { PubSub } from "../pubsub";
import { SessionContext } from "../SessionContext";
export const useBookmarker = ({ objectId, objectType, onActionComplete, }) => {
    const session = useContext(SessionContext);
    const { bookmarkLists } = useMemo(() => getCurrentUser(session), [session]);
    const [addBookmark] = useCustomMutation(bookmarkCreate);
    const [deleteOne] = useCustomMutation(deleteOneOrManyDeleteOne);
    const [getData, { data, loading }] = useCustomLazyQuery(bookmarkFindMany, { errorPolicy: "all" });
    const hasBookmarkingSupport = exists(BookmarkFor[objectType]);
    const handleAdd = useCallback(() => {
        const bookmarkListId = bookmarkLists && bookmarkLists.length ? bookmarkLists[0].id : undefined;
        mutationWrapper({
            mutation: addBookmark,
            input: {
                id: uuid(),
                bookmarkFor: BookmarkFor[objectType],
                forConnect: objectId,
                listConnect: bookmarkListId,
                listCreate: bookmarkListId ? undefined : {
                    id: uuid(),
                    label: "Favorites",
                },
            },
            onSuccess: (data) => { onActionComplete(ObjectActionComplete.Bookmark, data); },
        });
    }, [bookmarkLists, addBookmark, objectType, objectId, onActionComplete]);
    const handleRemove = useCallback(async () => {
        await getData({
            variables: {
                [`${objectType[0].toLowerCase()}${objectType.slice(1)}Id`]: objectId,
            },
        });
    }, [getData, objectId, objectType]);
    useEffect(() => {
        if (!data || loading)
            return;
        const bookmarks = data?.edges.map(edge => edge.node);
        if (bookmarks === undefined || bookmarks.length === 0) {
            PubSub.get().publishSnack({ message: "Could not find bookmark", severity: "Error" });
        }
        else if (bookmarks.length === 1) {
            mutationWrapper({
                mutation: deleteOne,
                input: { id: bookmarks[0].id, objectType: DeleteType.Bookmark },
                onSuccess: (data) => { onActionComplete(ObjectActionComplete.BookmarkUndo, data); },
            });
            return;
        }
        else {
            console.warn("TODO: Open dialog to select which bookmark to delete");
        }
    }, [data, loading, deleteOne, onActionComplete]);
    const handleBookmark = useCallback((isAdding) => {
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
        }
        else {
            handleRemove();
        }
    }, [handleAdd, handleRemove, hasBookmarkingSupport, objectId]);
    return { handleBookmark, hasBookmarkingSupport };
};
//# sourceMappingURL=useBookmarker.js.map