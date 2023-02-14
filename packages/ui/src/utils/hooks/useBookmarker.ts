import { useMutation } from "@apollo/client";
import { Bookmark, BookmarkCreateInput, BookmarkSearchInput, DeleteOneInput, Success } from "@shared/consts";
import { bookmarkCreate, bookmarkFindMany } from "api/generated/endpoints/bookmark";
import { deleteOneOrManyDeleteOne } from "api/generated/endpoints/deleteOneOrMany";

/**
 * Hook for simplifying the use of adding and removing bookmarks on an object
 */
export const useBookmarker = () => {
    const [addBookmark] = useMutation<Bookmark, BookmarkCreateInput, 'bookmarkCreate'>(bookmarkCreate, 'bookmarkCreate');
    const [deleteOne] = useMutation<Success, DeleteOneInput, 'deleteOne'>(deleteOneOrManyDeleteOne, 'deleteOne')
    // In most cases, we must query for bookmarks to remove them, since 
    // we usually only know that an object has a bookmark - not the bookmarks themselves
    const [getData, { data, loading }] = useLazyQuery<Bookmark, BookmarkSearchInput, 'bookmarks'>(bookmarkFindMany, 'bookmarks', { errorPolicy: 'all' });
}