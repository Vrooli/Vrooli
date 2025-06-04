
import { type BookmarkList, type BookmarkListSearchInput, type BookmarkListSearchResult, endpointsBookmarkList } from "@local/shared";
import { create } from "zustand";
import { fetchData } from "../api/fetchData.js";
import { ServerResponseParser } from "../api/responseParser.js";

interface BookmarkListsState {
    bookmarkLists: BookmarkList[];
    isLoading: boolean;
    fetchBookmarkLists: () => Promise<BookmarkList[]>;
}

export const useBookmarkListsStore = create<BookmarkListsState>()((set, get) => ({
    bookmarkLists: [],
    isLoading: false,
    fetchBookmarkLists: async () => {
        if (get().bookmarkLists.length > 0 || get().isLoading) {
            // Already fetched or currently fetching
            return get().bookmarkLists;
        }

        set({ isLoading: true });

        try {
            const response = await fetchData<BookmarkListSearchInput, BookmarkListSearchResult>({
                ...endpointsBookmarkList.findMany,
                inputs: {},
            });

            if (response.errors) {
                ServerResponseParser.displayErrors(response.errors);
                throw new Error("Failed to fetch bookmark lists");
            }

            const bookmarkLists = response.data?.edges.map(edge => edge.node) ?? [];

            set({ bookmarkLists, isLoading: false });
            return bookmarkLists;
        } catch (error) {
            console.error("Error fetching bookmark lists:", error);
            set({ isLoading: false });
            return [];
        }
    },
}));
