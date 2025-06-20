/* c8 ignore start */
/**
 * Bookmark fixture factory implementation
 * 
 * This provides complete fixtures for the Bookmark object type following the UserFixtureFactory pattern
 * with full type safety and integration with @vrooli/shared.
 */

// NOTE: These types would normally be imported from @vrooli/shared
// For now, defining them locally based on the shared package structure

export enum BookmarkFor {
    Comment = "Comment",
    Issue = "Issue",
    Resource = "Resource",
    Tag = "Tag",
    Team = "Team",
    User = "User"
}

export interface BookmarkCreateInput {
    id: string;
    bookmarkFor: BookmarkFor;
    forConnect: string;
    listConnect?: string;
    listCreate?: {
        id: string;
        label: string;
    };
}

export interface BookmarkUpdateInput {
    id: string;
    listConnect?: string;
    listUpdate?: {
        id: string;
        label?: string;
    };
}

export interface Bookmark {
    __typename: "Bookmark";
    id: string;
    createdAt: string;
    updatedAt: string;
    by: {
        __typename: "User";
        id: string;
        handle: string;
        name: string;
    };
    to: {
        __typename: string;
        id: string;
    };
    list: {
        __typename: "BookmarkList";
        id: string;
        label: string;
        bookmarksCount: number;
        createdAt: string;
        updatedAt: string;
        bookmarks: Bookmark[];
    };
}

/**
 * Bookmark form data type
 * 
 * This includes UI-specific fields for bookmark creation and management.
 */
export interface BookmarkFormData extends Record<string, unknown> {
    bookmarkFor: BookmarkFor;
    forConnect: string;
    listId?: string;
    createNewList?: boolean; // UI-specific
    newListLabel?: string; // UI-specific
    isPrivate?: boolean;
}

/**
 * Bookmark UI state type
 */
export interface BookmarkUIState {
    isLoading: boolean;
    bookmark: Bookmark | null;
    error: string | null;
    isBookmarked: boolean;
    availableLists: Array<{ id: string; label: string }>;
    showListSelection: boolean;
}

/**
 * Bookmark fixture factory
 * 
 * This provides all the fixtures needed for testing bookmark functionality
 * following the established pattern from UserFixtureFactory.
 */
export class BookmarkFixtureFactory {
    readonly objectType = "bookmark";
    
    /**
     * Create form data for different scenarios
     */
    createFormData(scenario: "minimal" | "complete" | "invalid" | "withNewList" | "withExistingList" | "forProject" | "forRoutine" = "minimal"): BookmarkFormData {
        const generateId = () => Date.now().toString() + Math.random().toString().slice(2, 8);
        
        switch (scenario) {
            case "minimal":
                return {
                    bookmarkFor: BookmarkFor.Resource,
                    forConnect: generateId(),
                };
                
            case "complete":
                return {
                    bookmarkFor: BookmarkFor.Resource,
                    forConnect: generateId(),
                    listId: generateId(),
                    isPrivate: false,
                };
                
            case "invalid":
                return {
                    bookmarkFor: "InvalidType" as any,
                    forConnect: "",
                };
                
            case "withNewList":
                return {
                    bookmarkFor: BookmarkFor.User,
                    forConnect: generateId(),
                    createNewList: true,
                    newListLabel: "My New Bookmarks",
                    isPrivate: true,
                };
                
            case "withExistingList":
                return {
                    bookmarkFor: BookmarkFor.Resource,
                    forConnect: generateId(),
                    listId: generateId(),
                    isPrivate: false,
                };
                
            case "forProject":
                return {
                    bookmarkFor: BookmarkFor.Team,
                    forConnect: generateId(),
                    listId: generateId(),
                };
                
            case "forRoutine":
                return {
                    bookmarkFor: BookmarkFor.Comment,
                    forConnect: generateId(),
                    createNewList: true,
                    newListLabel: "Favorite Comments",
                };
                
            default:
                throw new Error(`Unknown scenario: ${scenario}`);
        }
    }
    
    /**
     * Transform form data to API input
     */
    createAPIInput(formData: BookmarkFormData): BookmarkCreateInput {
        const generateId = () => Date.now().toString() + Math.random().toString().slice(2, 8);
        
        const result: BookmarkCreateInput = {
            id: generateId(),
            bookmarkFor: formData.bookmarkFor,
            forConnect: formData.forConnect,
        };
        
        // Handle list creation or connection
        if (formData.createNewList && formData.newListLabel) {
            result.listCreate = {
                id: generateId(),
                label: formData.newListLabel,
            };
        } else if (formData.listId) {
            result.listConnect = formData.listId;
        }
        
        return result;
    }
    
    /**
     * Create mock bookmark response
     */
    createMockResponse(overrides?: Partial<Bookmark>): Bookmark {
        const generateId = () => Date.now().toString() + Math.random().toString().slice(2, 8);
        
        return {
            __typename: "Bookmark",
            id: generateId(),
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
            by: {
                __typename: "User",
                id: generateId(),
                handle: "testuser",
                name: "Test User",
            },
            to: {
                __typename: "Resource",
                id: generateId(),
            },
            list: {
                __typename: "BookmarkList",
                id: generateId(),
                label: "My Bookmarks",
                bookmarksCount: 1,
                createdAt: new Date().toISOString(),
                updatedAt: new Date().toISOString(),
                bookmarks: [],
            },
            ...overrides
        } as Bookmark;
    }
    
    /**
     * Create bookmark form data for different object types
     */
    createForObjectType(objectType: BookmarkFor, objectId: string): BookmarkFormData {
        return {
            bookmarkFor: objectType,
            forConnect: objectId,
            isPrivate: false,
        };
    }
    
    /**
     * Create bookmark form data with list creation
     */
    createWithNewList(objectType: BookmarkFor, objectId: string, listLabel: string): BookmarkFormData {
        return {
            bookmarkFor: objectType,
            forConnect: objectId,
            createNewList: true,
            newListLabel: listLabel,
            isPrivate: false,
        };
    }
    
    /**
     * Create UI state fixtures
     */
    createUIState(state: "loading" | "error" | "success" | "empty" | "withLists" = "empty", data?: any): BookmarkUIState {
        const generateId = () => Date.now().toString() + Math.random().toString().slice(2, 8);
        
        switch (state) {
            case "loading":
                return {
                    isLoading: true,
                    bookmark: null,
                    error: null,
                    isBookmarked: false,
                    availableLists: [],
                    showListSelection: false
                };
                
            case "error":
                return {
                    isLoading: false,
                    bookmark: null,
                    error: data?.message || "An error occurred",
                    isBookmarked: false,
                    availableLists: [],
                    showListSelection: false
                };
                
            case "success":
                return {
                    isLoading: false,
                    bookmark: data || this.createMockResponse(),
                    error: null,
                    isBookmarked: true,
                    availableLists: [
                        { id: generateId(), label: "My Bookmarks" }
                    ],
                    showListSelection: false
                };
                
            case "withLists":
                return {
                    isLoading: false,
                    bookmark: null,
                    error: null,
                    isBookmarked: false,
                    availableLists: data || [
                        { id: generateId(), label: "Favorites" },
                        { id: generateId(), label: "To Review" }
                    ],
                    showListSelection: true
                };
                
            case "empty":
            default:
                return {
                    isLoading: false,
                    bookmark: null,
                    error: null,
                    isBookmarked: false,
                    availableLists: [],
                    showListSelection: false
                };
        }
    }
    
    /**
     * Validate form data
     */
    async validateFormData(formData: BookmarkFormData): Promise<{ isValid: boolean; errors?: string[] }> {
        const errors: string[] = [];
        
        if (!formData.forConnect) {
            errors.push("forConnect: Target object is required");
        }
        
        if (formData.createNewList && !formData.newListLabel) {
            errors.push("newListLabel: New list label is required when creating a new list");
        }
        
        if (!formData.createNewList && !formData.listId) {
            errors.push("listId: Must either select an existing list or create a new one");
        }
        
        return {
            isValid: errors.length === 0,
            errors: errors.length > 0 ? errors : undefined
        };
    }
    
    /**
     * Test bookmark creation for different object types
     */
    createTestCases() {
        const generateId = () => Date.now().toString() + Math.random().toString().slice(2, 8);
        const objectTypes = Object.values(BookmarkFor);
        
        return objectTypes.map(objectType => ({
            objectType,
            formData: this.createForObjectType(objectType, generateId()),
            expectedResult: this.createMockResponse({
                to: { __typename: objectType, id: generateId() } as any
            })
        }));
    }
    
    /**
     * Create MSW handlers for testing
     */
    createMSWHandlers() {
        // This would return MSW handlers if MSW was available
        // For now, return a placeholder structure
        return {
            createHandlers: () => [],
            toggleHandlers: () => [],
            listHandlers: () => []
        };
    }
    
    /**
     * Test bookmark toggle functionality
     */
    async testToggleBookmark(objectType: BookmarkFor, objectId: string): Promise<{
        success: boolean;
        isBookmarked: boolean;
        bookmark?: Bookmark;
    }> {
        try {
            const formData = this.createForObjectType(objectType, objectId);
            const isValid = await this.validateFormData(formData);
            
            if (!isValid.isValid) {
                return {
                    success: false,
                    isBookmarked: false
                };
            }
            
            const bookmark = this.createMockResponse();
            
            return {
                success: true,
                isBookmarked: true,
                bookmark
            };
        } catch (error) {
            return {
                success: false,
                isBookmarked: false
            };
        }
    }
    
    /**
     * Create fixtures for all bookmark scenarios
     */
    createAllFixtures() {
        const scenarios = ["minimal", "complete", "invalid", "withNewList", "withExistingList", "forProject", "forRoutine"] as const;
        
        return scenarios.reduce((fixtures, scenario) => {
            fixtures[scenario] = {
                formData: this.createFormData(scenario),
                apiInput: scenario !== "invalid" ? this.createAPIInput(this.createFormData(scenario)) : undefined,
                mockResponse: scenario !== "invalid" ? this.createMockResponse() : undefined,
                uiState: this.createUIState(scenario === "invalid" ? "error" : "success")
            };
            return fixtures;
        }, {} as Record<string, any>);
    }
}

/**
 * Default instance for easy importing
 */
export const bookmarkFixtures = new BookmarkFixtureFactory();

/**
 * Specific fixture scenarios for common use cases
 */
export const bookmarkTestScenarios = {
    // Basic bookmark creation
    basicBookmark: () => bookmarkFixtures.createFormData("minimal"),
    
    // Bookmark with new list
    bookmarkWithNewList: () => bookmarkFixtures.createFormData("withNewList"),
    
    // Bookmark different object types
    bookmarkTeam: () => bookmarkFixtures.createFormData("forProject"),
    bookmarkComment: () => bookmarkFixtures.createFormData("forRoutine"),
    
    // UI states
    loadingState: () => bookmarkFixtures.createUIState("loading"),
    errorState: (message?: string) => bookmarkFixtures.createUIState("error", { message }),
    successState: (bookmark?: Bookmark) => bookmarkFixtures.createUIState("success", bookmark),
    
    // Test all object types
    allObjectTypes: () => bookmarkFixtures.createTestCases(),
    
    // Complete fixture set
    allFixtures: () => bookmarkFixtures.createAllFixtures()
};

/* c8 ignore stop */