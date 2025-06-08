import { generatePK, type AutocompleteOption, type Session, type SessionUser } from "@vrooli/shared";
import { describe, it, expect, afterAll, beforeEach } from "vitest";
import { SearchHistory } from "./searchHistory.js";

const userId = generatePK().toString();
const validSession: Session = {
    __typename: "Session",
    isLoggedIn: true,
    users: [{
        __typename: "SessionUser",
        id: userId,
        credits: "0",
        hasPremium: false,
        languages: ["en"],
        publicId: userId,
    }] as SessionUser[],
};
const invalidSession: Partial<Session> = {};

const SEARCH_HISTORY_PREFIX = SearchHistory["SEARCH_HISTORY_PREFIX"];
const MAX_HISTORY_LENGTH = SearchHistory["MAX_HISTORY_LENGTH"];

// Helper function to create a history with a specified number of items
function createHistory(count: number): { [label: string]: { timestamp: number; option: AutocompleteOption } } {
    const history: { [label: string]: { timestamp: number; option: AutocompleteOption } } = {};
    for (let i = 1; i <= count; i++) {
        const label = `option${i}`;
        history[label] = { timestamp: 100 + i, option: { label, __typename: "SomeType" } as unknown as AutocompleteOption };
    }
    return history;
}

describe("SearchHistory", () => {
    beforeEach(() => {
        localStorage.clear();
    });

    afterAll(() => {
        localStorage.clear();
    });

    describe("getSearchHistory", () => {
        it("should retrieve search history from local storage", () => {
            const searchBarId = "bar1";
            const history = { "option1": { timestamp: 123, option: { label: "option1", __typename: "SomeType" } } };
            localStorage.setItem(`${SEARCH_HISTORY_PREFIX}${searchBarId}-${userId}`, JSON.stringify(history));
            const result = SearchHistory.getSearchHistory(searchBarId, userId);
            expect(result).toEqual(history);
        });

        it("should return empty object if no history exists", () => {
            const searchBarId = "bar2";
            const result = SearchHistory.getSearchHistory(searchBarId, userId);
            expect(result).toEqual({});
        });

        it("should return empty object if history is invalid JSON", () => {
            const searchBarId = "bar3";
            localStorage.setItem(`${SEARCH_HISTORY_PREFIX}${searchBarId}-${userId}`, "invalid json");
            const result = SearchHistory.getSearchHistory(searchBarId, userId);
            expect(result).toEqual({});
        });

        it("should return empty object if stored data is not an object", () => {
            const searchBarId = "bar4";
            localStorage.setItem(`${SEARCH_HISTORY_PREFIX}${searchBarId}-${userId}`, JSON.stringify("not an object"));
            const result = SearchHistory.getSearchHistory(searchBarId, userId);
            expect(result).toEqual({});
        });
    });

    describe("updateHistoryItems", () => {
        it("should update bookmarks and isBookmarked across multiple search bars", () => {
            const searchBarId = "bar1"; // Not directly used in updates, but set for consistency
            const initialHistoryBar1 = {
                "optionA": { timestamp: 100, option: { label: "optionA", __typename: "SomeType", bookmarks: 5, isBookmarked: true, description: "old" } },
                "shortcut1": { timestamp: 150, option: { label: "shortcut1", __typename: "Shortcut", someProp: "value" } },
            };
            const initialHistoryBar2 = {
                "optionB": { timestamp: 200, option: { label: "optionB", __typename: "AnotherType", bookmarks: 10, isBookmarked: false } },
                "action1": { timestamp: 250, option: { label: "action1", __typename: "Action", anotherProp: "value" } },
            };
            // Set up localStorage with multiple search bar histories
            localStorage.setItem(`${SEARCH_HISTORY_PREFIX}bar1-${userId}`, JSON.stringify(initialHistoryBar1));
            localStorage.setItem(`${SEARCH_HISTORY_PREFIX}bar2-${userId}`, JSON.stringify(initialHistoryBar2));
            const options = [
                { label: "optionA", __typename: "SomeType", bookmarks: 6, isBookmarked: true, description: "new" },
                { label: "optionB", __typename: "AnotherType", bookmarks: 10, isBookmarked: true },
                { label: "optionC", __typename: "SomeType", bookmarks: 0, isBookmarked: false },
                { label: "shortcut1", __typename: "Shortcut", someProp: "newValue" },
                { label: "action1", __typename: "Action", anotherProp: "newValue" },
            ] as unknown as AutocompleteOption[];
            SearchHistory.updateHistoryItems(searchBarId, userId, options);
            // Check updated history for bar1
            const updatedHistoryBar1 = JSON.parse(localStorage.getItem(`${SEARCH_HISTORY_PREFIX}bar1-${userId}`) as string);
            expect(updatedHistoryBar1["optionA"].option).toEqual({
                label: "optionA", __typename: "SomeType", bookmarks: 6, isBookmarked: true, description: "old",
            });
            expect(updatedHistoryBar1["optionA"].timestamp).toBe(100);
            expect(updatedHistoryBar1["shortcut1"]).toEqual(initialHistoryBar1["shortcut1"]);
            // Check updated history for bar2
            const updatedHistoryBar2 = JSON.parse(localStorage.getItem(`${SEARCH_HISTORY_PREFIX}bar2-${userId}`) as string);
            expect(updatedHistoryBar2["optionB"].option).toEqual({
                label: "optionB", __typename: "AnotherType", bookmarks: 10, isBookmarked: true,
            });
            expect(updatedHistoryBar2["optionB"].timestamp).toBe(200);
            expect(updatedHistoryBar2["action1"]).toEqual(initialHistoryBar2["action1"]);
            // Ensure optionC wasn't added
            expect(updatedHistoryBar1).not.toHaveProperty("optionC");
            expect(updatedHistoryBar2).not.toHaveProperty("optionC");
        });

        it("should not update if bookmarks and isBookmarked are unchanged", () => {
            const searchBarId = "bar1";
            const initialHistory = {
                "optionA": { timestamp: 100, option: { label: "optionA", __typename: "SomeType", bookmarks: 5, isBookmarked: true } },
            };
            localStorage.setItem(`${SEARCH_HISTORY_PREFIX}${searchBarId}-${userId}`, JSON.stringify(initialHistory));
            const options = [
                { label: "optionA", __typename: "SomeType", bookmarks: 5, isBookmarked: true },
            ] as unknown as AutocompleteOption[];
            SearchHistory.updateHistoryItems(searchBarId, userId, options);
            const updatedHistory = JSON.parse(localStorage.getItem(`${SEARCH_HISTORY_PREFIX}${searchBarId}-${userId}`) as string);
            expect(updatedHistory["optionA"]).toEqual(initialHistory["optionA"]);
        });

        it("should do nothing if options is undefined", () => {
            const searchBarId = "bar1";
            const initialHistory = {
                "optionA": { timestamp: 100, option: { label: "optionA", __typename: "SomeType", bookmarks: 5, isBookmarked: true } },
            };
            localStorage.setItem(`${SEARCH_HISTORY_PREFIX}${searchBarId}-${userId}`, JSON.stringify(initialHistory));
            SearchHistory.updateHistoryItems(searchBarId, userId, undefined);
            const historyAfter = JSON.parse(localStorage.getItem(`${SEARCH_HISTORY_PREFIX}${searchBarId}-${userId}`) as string);
            expect(historyAfter).toEqual(initialHistory);
        });
    });

    describe("clearSearchHistory", () => {
        it("should clear all search history for the current user and publish success", () => {
            const keys = [`${SEARCH_HISTORY_PREFIX}bar1-${userId}`, `${SEARCH_HISTORY_PREFIX}bar2-${userId}`];
            // Set up localStorage with some keys
            keys.forEach(key => localStorage.setItem(key, JSON.stringify({})));
            SearchHistory.clearSearchHistory(validSession);
            // Verify all keys are removed
            keys.forEach(key => {
                expect(localStorage.getItem(key)).toBeNull();
            });
        });

        it("should handle empty key list gracefully", () => {
            // No keys set in localStorage
            SearchHistory.clearSearchHistory(validSession as Session);
            // localStorage should remain empty
            expect(localStorage.length).toBe(0);
        });

        it("should do nothing if session is invalid", () => {
            const key = `${SEARCH_HISTORY_PREFIX}bar1-${userId}`;
            localStorage.setItem(key, JSON.stringify({}));
            SearchHistory.clearSearchHistory(invalidSession as Session);
            // Verify the key is still present
            expect(localStorage.getItem(key)).toBe(JSON.stringify({}));
        });
    });

    describe("removeSearchHistoryItem", () => {
        const searchBarId = "bar1";
        const labelToRemove = "option1";
        const otherLabel = "option2";
        const history = {
            [labelToRemove]: { timestamp: 100, option: { label: labelToRemove, __typename: "SomeType" } },
            [otherLabel]: { timestamp: 200, option: { label: otherLabel, __typename: "AnotherType" } },
        };

        it("should remove an existing item from history", () => {
            localStorage.setItem(`${SEARCH_HISTORY_PREFIX}${searchBarId}-${userId}`, JSON.stringify(history));
            SearchHistory.removeSearchHistoryItem(searchBarId, userId, labelToRemove);
            const updatedHistory = JSON.parse(localStorage.getItem(`${SEARCH_HISTORY_PREFIX}${searchBarId}-${userId}`) as string);
            expect(updatedHistory).not.toHaveProperty(labelToRemove);
            expect(updatedHistory).toHaveProperty(otherLabel);
        });

        it("should do nothing when removing a non-existing item", () => {
            localStorage.setItem(`${SEARCH_HISTORY_PREFIX}${searchBarId}-${userId}`, JSON.stringify(history));
            SearchHistory.removeSearchHistoryItem(searchBarId, userId, "nonExisting");
            const updatedHistory = JSON.parse(localStorage.getItem(`${SEARCH_HISTORY_PREFIX}${searchBarId}-${userId}`) as string);
            expect(updatedHistory).toEqual(history);
        });

        it("should do nothing when history is empty", () => {
            SearchHistory.removeSearchHistoryItem(searchBarId, userId, labelToRemove);
            const updatedHistory = localStorage.getItem(`${SEARCH_HISTORY_PREFIX}${searchBarId}-${userId}`);
            expect(updatedHistory).toBeNull();
        });

        it("should clear out the entry if it's the only one", () => {
            localStorage.setItem(`${SEARCH_HISTORY_PREFIX}${searchBarId}-${userId}`, JSON.stringify({ [labelToRemove]: { timestamp: 100, option: { label: labelToRemove, __typename: "SomeType" } } }));
            SearchHistory.removeSearchHistoryItem(searchBarId, userId, labelToRemove);
            const updatedHistory = localStorage.getItem(`${SEARCH_HISTORY_PREFIX}${searchBarId}-${userId}`);
            expect(updatedHistory).toBeNull();
        });
    });

    describe("addSearchHistoryItem", () => {
        const searchBarId = "bar1";

        it("should add a new item to history", () => {
            const option = { label: "newOption", __typename: "SomeType" } as unknown as AutocompleteOption;
            SearchHistory.addSearchHistoryItem(searchBarId, userId, option);
            const history = JSON.parse(localStorage.getItem(`${SEARCH_HISTORY_PREFIX}${searchBarId}-${userId}`) as string);
            expect(history).toHaveProperty("newOption");
            expect(history["newOption"].option).toEqual({ ...option, isFromHistory: true });
            expect(typeof history["newOption"].timestamp).toBe("number");
        });

        it("should remove the oldest item when history exceeds max length", () => {
            const history = createHistory(MAX_HISTORY_LENGTH);
            localStorage.setItem(`${SEARCH_HISTORY_PREFIX}${searchBarId}-${userId}`, JSON.stringify(history));
            const newOption = { label: "newOption", __typename: "SomeType" } as unknown as AutocompleteOption;
            SearchHistory.addSearchHistoryItem(searchBarId, userId, newOption);
            const updatedHistory = JSON.parse(localStorage.getItem(`${SEARCH_HISTORY_PREFIX}${searchBarId}-${userId}`) as string);
            expect(updatedHistory).not.toHaveProperty("option1");
            expect(updatedHistory).toHaveProperty("newOption");
            expect(updatedHistory["newOption"].option).toEqual({ ...newOption, isFromHistory: true });
            expect(Object.keys(updatedHistory).length).toBe(MAX_HISTORY_LENGTH);
        });

        it("should update timestamp when adding an existing item", async () => {
            const option = { label: "existingOption", __typename: "SomeType" } as unknown as AutocompleteOption;
            SearchHistory.addSearchHistoryItem(searchBarId, userId, option);
            const firstHistory = JSON.parse(localStorage.getItem(`${SEARCH_HISTORY_PREFIX}${searchBarId}-${userId}`) as string);
            const firstTimestamp = firstHistory["existingOption"].timestamp;
            await new Promise(resolve => setTimeout(resolve, 10)); // Wait 10ms
            SearchHistory.addSearchHistoryItem(searchBarId, userId, option);
            const secondHistory = JSON.parse(localStorage.getItem(`${SEARCH_HISTORY_PREFIX}${searchBarId}-${userId}`) as string);
            const secondTimestamp = secondHistory["existingOption"].timestamp;
            expect(secondTimestamp).toBeGreaterThan(firstTimestamp);
            expect(secondHistory["existingOption"].option).toEqual({ ...option, isFromHistory: true });
        });

        it("should add item to empty history", () => {
            const option = { label: "firstOption", __typename: "SomeType" } as unknown as AutocompleteOption;
            SearchHistory.addSearchHistoryItem(searchBarId, userId, option);
            const history = JSON.parse(localStorage.getItem(`${SEARCH_HISTORY_PREFIX}${searchBarId}-${userId}`) as string);
            expect(history).toHaveProperty("firstOption");
            expect(history["firstOption"].option).toEqual({ ...option, isFromHistory: true });
        });
    });

    it("should add multiple items and remove one correctly", () => {
        const searchBarId = "bar1";
        const option1 = { label: "option1", __typename: "SomeType" } as unknown as AutocompleteOption;
        const option2 = { label: "option2", __typename: "SomeType" } as unknown as AutocompleteOption;
        const option3 = { label: "option3", __typename: "SomeType" } as unknown as AutocompleteOption;

        // Add three options
        SearchHistory.addSearchHistoryItem(searchBarId, userId, option1);
        SearchHistory.addSearchHistoryItem(searchBarId, userId, option2);
        SearchHistory.addSearchHistoryItem(searchBarId, userId, option3);

        // Verify all three are present
        let history = SearchHistory.getSearchHistory(searchBarId, userId);
        expect(Object.keys(history)).toEqual(expect.arrayContaining(["option1", "option2", "option3"]));

        // Remove option2
        SearchHistory.removeSearchHistoryItem(searchBarId, userId, "option2");

        // Verify option2 is removed, others remain
        history = SearchHistory.getSearchHistory(searchBarId, userId);
        expect(Object.keys(history)).toEqual(expect.arrayContaining(["option1", "option3"]));
        expect(history).not.toHaveProperty("option2");
    });

    it("should maintain max length and add new item when exceeding limit", () => {
        const searchBarId = "bar1";
        const maxLength = SearchHistory["MAX_HISTORY_LENGTH"];
        const userId = "user1";

        // Add 500 items
        for (let i = 1; i <= maxLength; i++) {
            const option = { label: `option${i}`, __typename: "SomeType" } as unknown as AutocompleteOption;
            SearchHistory.addSearchHistoryItem(searchBarId, userId, option);
        }

        // Capture initial keys
        const initialKeys = Object.keys(SearchHistory.getSearchHistory(searchBarId, userId));

        // Add 501st item
        const newOption = { label: "newOption", __typename: "SomeType" } as unknown as AutocompleteOption;
        SearchHistory.addSearchHistoryItem(searchBarId, userId, newOption);

        // Verify history
        const history = SearchHistory.getSearchHistory(searchBarId, userId);
        const updatedKeys = Object.keys(history);

        expect(updatedKeys.length).toBe(maxLength);           // Length stays at 500
        expect(updatedKeys).to.include("newOption");             // New item is added
        expect(initialKeys.filter(k => !updatedKeys.includes(k)).length).toBe(1); // One item removed
    });

    it("should add items and then update their bookmarks correctly", () => {
        const searchBarId = "bar1";
        const option1 = { label: "option1", __typename: "SomeType", bookmarks: 5, isBookmarked: true } as unknown as AutocompleteOption;
        const option2 = { label: "option2", __typename: "SomeType", bookmarks: 10, isBookmarked: false } as unknown as AutocompleteOption;

        // Add options
        SearchHistory.addSearchHistoryItem(searchBarId, userId, option1);
        SearchHistory.addSearchHistoryItem(searchBarId, userId, option2);

        // Update options
        const updatedOption1 = { label: "option1", __typename: "SomeType", bookmarks: 6, isBookmarked: false } as unknown as AutocompleteOption;
        const updatedOption2 = { label: "option2", __typename: "SomeType", bookmarks: 11, isBookmarked: true } as unknown as AutocompleteOption;
        SearchHistory.updateHistoryItems(searchBarId, userId, [updatedOption1, updatedOption2]);

        // Verify updates
        const history = SearchHistory.getSearchHistory(searchBarId, userId);
        expect((history["option1"].option as any).bookmarks).toBe(6);
        expect((history["option1"].option as any).isBookmarked).to.be.false;
        expect((history["option2"].option as any).bookmarks).toBe(11);
        expect((history["option2"].option as any).isBookmarked).to.be.true;
    });

    it("should add items to multiple search bars and clear all history", () => {
        const searchBarId1 = "bar1";
        const searchBarId2 = "bar2";
        const option1 = { label: "option1", __typename: "SomeType" } as unknown as AutocompleteOption;
        const option2 = { label: "option2", __typename: "SomeType" } as unknown as AutocompleteOption;

        // Add items to both search bars
        SearchHistory.addSearchHistoryItem(searchBarId1, userId, option1);
        SearchHistory.addSearchHistoryItem(searchBarId2, userId, option2);

        // Verify history exists
        expect(localStorage.getItem(`${SEARCH_HISTORY_PREFIX}${searchBarId1}-${userId}`)).to.not.be.null;
        expect(localStorage.getItem(`${SEARCH_HISTORY_PREFIX}${searchBarId2}-${userId}`)).to.not.be.null;

        // Clear all history
        SearchHistory.clearSearchHistory(validSession as Session);

        // Verify history is cleared
        expect(localStorage.getItem(`${SEARCH_HISTORY_PREFIX}${searchBarId1}-${userId}`)).toBeNull();
        expect(localStorage.getItem(`${SEARCH_HISTORY_PREFIX}${searchBarId2}-${userId}`)).toBeNull();
    });

    it("should maintain separate histories for different users", () => {
        const searchBarId = "bar1";
        const userId1 = generatePK().toString();
        const userId2 = generatePK().toString();
        const option1 = { label: "option1", __typename: "SomeType" } as unknown as AutocompleteOption;
        const option2 = { label: "option2", __typename: "SomeType" } as unknown as AutocompleteOption;

        // Add items for different users
        SearchHistory.addSearchHistoryItem(searchBarId, userId1, option1);
        SearchHistory.addSearchHistoryItem(searchBarId, userId2, option2);

        // Verify isolation
        const historyUser1 = SearchHistory.getSearchHistory(searchBarId, userId1);
        expect(historyUser1).toHaveProperty("option1");
        expect(historyUser1).not.toHaveProperty("option2");

        const historyUser2 = SearchHistory.getSearchHistory(searchBarId, userId2);
        expect(historyUser2).toHaveProperty("option2");
        expect(historyUser2).not.toHaveProperty("option1");
    });
});
