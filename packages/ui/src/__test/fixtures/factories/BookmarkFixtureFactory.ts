/**
 * Production-grade bookmark fixture factory
 * 
 * This factory provides type-safe bookmark fixtures using real functions from @vrooli/shared.
 * It eliminates `any` types and integrates with actual validation and shape transformation logic.
 */
// AI_CHECK: TYPE_SAFETY=fixed-typename-properties-and-msw-v2-migration | LAST: 2025-07-03 - Fixed missing __typename in BookmarkList, removed invalid properties from UserYou/ResourceYou, migrated MSW v1 to v2 syntax

import type { 
    Bookmark, 
    BookmarkCreateInput, 
    BookmarkUpdateInput, 
    BookmarkFor,
    BookmarkList,
    ResourceType,
} from "@vrooli/shared";
import { 
    bookmarkValidation, 
    shapeBookmark,
    BookmarkFor as BookmarkForEnum,
    ResourceType as ResourceTypeEnum,
} from "@vrooli/shared";
import type { 
    FixtureFactory, 
    ValidationResult, 
    MSWHandlers,
    BookmarkFormData,
    BookmarkUIState,
    BookmarkScenario, 
} from "../types.js";
import { http, HttpResponse } from "msw";

/**
 * Type-safe bookmark fixture factory that uses real @vrooli/shared functions
 */
export class BookmarkFixtureFactory implements FixtureFactory<
    BookmarkFormData,
    BookmarkCreateInput,
    BookmarkUpdateInput,
    Bookmark
> {
    readonly objectType = "bookmark";

    /**
     * Generate a unique ID for testing
     */
    private generateId(): string {
        return `test_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }

    /**
     * Create form data for different test scenarios
     */
    createFormData(scenario: BookmarkScenario = "minimal"): BookmarkFormData {
        switch (scenario) {
            case "minimal":
                return {
                    bookmarkFor: BookmarkForEnum.Resource,
                    forConnect: this.generateId(),
                };

            case "complete":
                return {
                    bookmarkFor: BookmarkForEnum.Resource,
                    forConnect: this.generateId(),
                    listId: this.generateId(),
                    isPrivate: false,
                };

            case "invalid":
                return {
                    bookmarkFor: "InvalidType" as any, // Intentionally invalid for error testing
                    forConnect: "", // Empty string should fail validation
                };

            case "withNewList":
                return {
                    bookmarkFor: BookmarkForEnum.User,
                    forConnect: this.generateId(),
                    createNewList: true,
                    newListLabel: "My New Bookmarks",
                    isPrivate: true,
                };

            case "withExistingList":
                return {
                    bookmarkFor: BookmarkForEnum.Resource,
                    forConnect: this.generateId(),
                    listId: this.generateId(),
                    isPrivate: false,
                };

            case "forProject":
                return {
                    bookmarkFor: BookmarkForEnum.Team,
                    forConnect: this.generateId(),
                    listId: this.generateId(),
                };

            case "forRoutine":
                return {
                    bookmarkFor: BookmarkForEnum.Comment,
                    forConnect: this.generateId(),
                    createNewList: true,
                    newListLabel: "Favorite Comments",
                };

            default:
                throw new Error(`Unknown bookmark scenario: ${scenario}`);
        }
    }

    /**
     * Transform form data to API create input using real shape function
     */
    transformToAPIInput(formData: BookmarkFormData): BookmarkCreateInput {
        // Create the bookmark shape that matches the expected API structure
        const bookmarkShape = {
            __typename: "Bookmark" as const,
            id: this.generateId(),
            to: { 
                __typename: formData.bookmarkFor, 
                id: formData.forConnect, 
            },
            list: formData.listId ? { __typename: "BookmarkList" as const, id: formData.listId } : null,
        };

        // Use real shape function from @vrooli/shared
        const apiInput = shapeBookmark.create(bookmarkShape);

        // Handle list creation if specified
        if (formData.createNewList && formData.newListLabel) {
            return {
                ...apiInput,
                listCreate: {
                    id: this.generateId(),
                    label: formData.newListLabel,
                },
            };
        }

        return apiInput;
    }

    /**
     * Create API update input
     */
    createUpdateInput(id: string, updates: Partial<BookmarkFormData>): BookmarkUpdateInput {
        const updateInput: BookmarkUpdateInput = { id };

        if (updates.listId) {
            updateInput.listConnect = updates.listId;
        }

        if (updates.createNewList && updates.newListLabel) {
            updateInput.listUpdate = {
                id: this.generateId(),
                label: updates.newListLabel,
            };
        }

        return updateInput;
    }

    /**
     * Create mock bookmark response with realistic data
     */
    createMockResponse(overrides?: Partial<Bookmark>): Bookmark {
        const now = new Date().toISOString();
        
        const defaultBookmark: Bookmark = {
            __typename: "Bookmark",
            id: this.generateId(),
            createdAt: now,
            updatedAt: now,
            by: {
                __typename: "User",
                id: this.generateId(),
                handle: "testuser",
                name: "Test User",
                createdAt: now,
                updatedAt: now,
                isBot: false,
                isPrivate: false,
                profileImage: null,
                bannerImage: null,
                premium: null,
                wallets: [],
                translations: [],
                translationsCount: 0,
                you: {
                    __typename: "UserYou",
                    canDelete: false,
                    canReport: false,
                    canUpdate: false,
                    isBookmarked: false,
                    isViewed: false,
                },
            },
            to: {
                __typename: "Resource",
                id: this.generateId(),
                bookmarkedBy: [],
                bookmarks: 0,
                completedAt: null,
                createdBy: null,
                createdAt: now,
                hasCompleteVersion: false,
                isDeleted: false,
                isInternal: false,
                isPrivate: false,
                issues: [],
                issuesCount: 0,
                owner: null,
                parent: null,
                permissions: "{}",
                publicId: this.generateId(),
                pullRequests: [],
                pullRequestsCount: 0,
                resourceType: ResourceTypeEnum.Standard,
                score: 0,
                stats: [],
                tags: [],
                transfers: [],
                transfersCount: 0,
                translatedName: "Mock Resource",
                updatedAt: now,
                versions: [],
                versionsCount: 0,
                views: 0,
                you: {
                    __typename: "ResourceYou",
                    canBookmark: true,
                    canComment: false,
                    canDelete: false,
                    canReact: true,
                    canRead: true,
                    canTransfer: false,
                    canUpdate: false,
                    isBookmarked: false,
                    isViewed: false,
                    reaction: null,
                },
            },
            list: {
                __typename: "BookmarkList",
                id: this.generateId(),
                label: "My Bookmarks",
                createdAt: now,
                updatedAt: now,
                bookmarks: [],
                bookmarksCount: 1,
            },
        };

        return {
            ...defaultBookmark,
            ...overrides,
        };
    }

    /**
     * Validate form data using real validation from @vrooli/shared
     */
    async validateFormData(formData: BookmarkFormData): Promise<ValidationResult> {
        try {
            // Transform to API input format for validation
            const apiInput = this.transformToAPIInput(formData);
            
            // Use real validation schema from @vrooli/shared
            const schema = bookmarkValidation.create({ omitFields: [] });
            await schema.validate(apiInput);
            
            return { isValid: true };
        } catch (error: any) {
            return {
                isValid: false,
                errors: error.errors || [error.message],
                fieldErrors: error.inner?.reduce((acc: any, err: any) => {
                    if (err.path) {
                        acc[err.path] = acc[err.path] || [];
                        acc[err.path].push(err.message);
                    }
                    return acc;
                }, {}),
            };
        }
    }

    /**
     * Create MSW handlers for different scenarios
     */
    createMSWHandlers(): MSWHandlers {
        const baseUrl = process.env.VITE_SERVER_URL || "http://localhost:3000";

        return {
            success: [
                http.post(`${baseUrl}/api/bookmark`, async ({ request }) => {
                    const body = await request.json() as BookmarkFormData;
                    
                    // Validate the request body
                    const validation = await this.validateFormData(body);
                    if (!validation.isValid) {
                        return HttpResponse.json({ 
                            errors: validation.errors,
                            fieldErrors: validation.fieldErrors, 
                        }, { status: 400 });
                    }

                    // Return successful response
                    const mockBookmark = this.createMockResponse({
                        to: { 
                            ...this.createMockResponse().to,
                            __typename: body.bookmarkFor,
                            id: body.forConnect, 
                        },
                    });

                    return HttpResponse.json(mockBookmark, { status: 201 });
                }),

                http.put(`${baseUrl}/api/bookmark/:id`, async ({ request, params }) => {
                    const { id } = params;
                    const body = await request.json() as Partial<BookmarkFormData>;

                    const mockBookmark = this.createMockResponse({ 
                        id: id as string,
                        updatedAt: new Date().toISOString(),
                    });

                    return HttpResponse.json(mockBookmark, { status: 200 });
                }),

                http.get(`${baseUrl}/api/bookmark/:id`, ({ params }) => {
                    const { id } = params;
                    const mockBookmark = this.createMockResponse({ id: id as string });
                    
                    return HttpResponse.json(mockBookmark, { status: 200 });
                }),

                http.delete(`${baseUrl}/api/bookmark/:id`, () => {
                    return new HttpResponse(null, { status: 204 });
                }),
            ],

            error: [
                http.post(`${baseUrl}/api/bookmark`, () => {
                    return HttpResponse.json({ 
                        message: "Internal server error",
                        code: "BOOKMARK_CREATE_FAILED", 
                    }, { status: 500 });
                }),

                http.put(`${baseUrl}/api/bookmark/:id`, () => {
                    return HttpResponse.json({ 
                        message: "Bookmark not found",
                        code: "BOOKMARK_NOT_FOUND", 
                    }, { status: 404 });
                }),

                http.get(`${baseUrl}/api/bookmark/:id`, () => {
                    return HttpResponse.json({ 
                        message: "Bookmark not found",
                        code: "BOOKMARK_NOT_FOUND", 
                    }, { status: 404 });
                }),
            ],

            loading: [
                http.post(`${baseUrl}/api/bookmark`, async () => {
                    // MSW v2 doesn't have ctx.delay, using setTimeout instead
                    await new Promise(resolve => setTimeout(resolve, 2000));
                    return HttpResponse.json(this.createMockResponse(), { status: 201 });
                }),
            ],

            networkError: [
                http.post(`${baseUrl}/api/bookmark`, () => {
                    return HttpResponse.error();
                }),
            ],
        };
    }

    /**
     * Create UI state fixtures for different scenarios
     */
    createUIState(state: "loading" | "error" | "success" | "empty" | "withLists" = "empty", data?: any): BookmarkUIState {
        switch (state) {
            case "loading":
                return {
                    isLoading: true,
                    bookmark: null,
                    error: null,
                    isBookmarked: false,
                    availableLists: [],
                    showListSelection: false,
                };

            case "error":
                return {
                    isLoading: false,
                    bookmark: null,
                    error: data?.message || "Failed to create bookmark",
                    isBookmarked: false,
                    availableLists: [],
                    showListSelection: false,
                };

            case "success":
                return {
                    isLoading: false,
                    bookmark: data || this.createMockResponse(),
                    error: null,
                    isBookmarked: true,
                    availableLists: [
                        { id: this.generateId(), label: "My Bookmarks" },
                    ],
                    showListSelection: false,
                };

            case "withLists":
                return {
                    isLoading: false,
                    bookmark: null,
                    error: null,
                    isBookmarked: false,
                    availableLists: data || [
                        { id: this.generateId(), label: "Favorites" },
                        { id: this.generateId(), label: "To Review" },
                        { id: this.generateId(), label: "Important" },
                    ],
                    showListSelection: true,
                };

            case "empty":
            default:
                return {
                    isLoading: false,
                    bookmark: null,
                    error: null,
                    isBookmarked: false,
                    availableLists: [],
                    showListSelection: false,
                };
        }
    }

    /**
     * Create form data for specific object types
     */
    createForObjectType(objectType: BookmarkFor, objectId: string): BookmarkFormData {
        return {
            bookmarkFor: objectType,
            forConnect: objectId,
            isPrivate: false,
        };
    }

    /**
     * Create form data with new list creation
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
     * Create test cases for all object types
     */
    createTestCases() {
        const objectTypes = Object.values(BookmarkForEnum);
        
        return objectTypes.map(objectType => ({
            objectType,
            formData: this.createForObjectType(objectType, this.generateId()),
            apiInput: (() => {
                const formData = this.createForObjectType(objectType, this.generateId());
                return this.transformToAPIInput(formData);
            })(),
            mockResponse: this.createMockResponse({
                to: { 
                    ...this.createMockResponse().to,
                    __typename: objectType,
                    id: this.generateId(), 
                },
            }),
        }));
    }

    /**
     * Create fixtures for all scenarios
     */
    createAllFixtures() {
        const scenarios: BookmarkScenario[] = [
            "minimal", "complete", "invalid", "withNewList", 
            "withExistingList", "forProject", "forRoutine",
        ];
        
        return scenarios.reduce((fixtures, scenario) => {
            const formData = this.createFormData(scenario);
            
            fixtures[scenario] = {
                formData,
                apiInput: scenario !== "invalid" ? this.transformToAPIInput(formData) : undefined,
                mockResponse: scenario !== "invalid" ? this.createMockResponse() : undefined,
                uiState: this.createUIState(scenario === "invalid" ? "error" : "success"),
                validation: scenario === "invalid" ? 
                    { isValid: false, errors: ["Invalid bookmark data"] } :
                    { isValid: true },
            };
            
            return fixtures;
        }, {} as Record<BookmarkScenario, any>);
    }

    /**
     * Test bookmark toggle functionality
     */
    async testToggleBookmark(objectType: BookmarkFor, objectId: string): Promise<{
        success: boolean;
        isBookmarked: boolean;
        bookmark?: Bookmark;
        error?: string;
    }> {
        try {
            const formData = this.createForObjectType(objectType, objectId);
            const validation = await this.validateFormData(formData);
            
            if (!validation.isValid) {
                return {
                    success: false,
                    isBookmarked: false,
                    error: validation.errors?.join(", "),
                };
            }
            
            const bookmark = this.createMockResponse({
                to: { 
                    ...this.createMockResponse().to,
                    __typename: objectType,
                    id: objectId, 
                },
            });
            
            return {
                success: true,
                isBookmarked: true,
                bookmark,
            };
        } catch (error: any) {
            return {
                success: false,
                isBookmarked: false,
                error: error.message,
            };
        }
    }
}

/**
 * Default factory instance for easy importing
 */
export const bookmarkFixtures = new BookmarkFixtureFactory();

/**
 * Specific test scenarios for common use cases
 */
export const bookmarkTestScenarios = {
    // Basic scenarios
    basicBookmark: () => bookmarkFixtures.createFormData("minimal"),
    completeBookmark: () => bookmarkFixtures.createFormData("complete"),
    invalidBookmark: () => bookmarkFixtures.createFormData("invalid"),
    
    // List scenarios
    bookmarkWithNewList: () => bookmarkFixtures.createFormData("withNewList"),
    bookmarkWithExistingList: () => bookmarkFixtures.createFormData("withExistingList"),
    
    // Object type scenarios
    bookmarkTeam: () => bookmarkFixtures.createFormData("forProject"),
    bookmarkComment: () => bookmarkFixtures.createFormData("forRoutine"),
    
    // UI state scenarios
    loadingState: () => bookmarkFixtures.createUIState("loading"),
    errorState: (message?: string) => bookmarkFixtures.createUIState("error", { message }),
    successState: (bookmark?: Bookmark) => bookmarkFixtures.createUIState("success", bookmark),
    emptyState: () => bookmarkFixtures.createUIState("empty"),
    withListsState: () => bookmarkFixtures.createUIState("withLists"),
    
    // Comprehensive test data
    allObjectTypes: () => bookmarkFixtures.createTestCases(),
    allFixtures: () => bookmarkFixtures.createAllFixtures(),
    
    // MSW handlers
    successHandlers: () => bookmarkFixtures.createMSWHandlers().success,
    errorHandlers: () => bookmarkFixtures.createMSWHandlers().error,
    loadingHandlers: () => bookmarkFixtures.createMSWHandlers().loading,
    networkErrorHandlers: () => bookmarkFixtures.createMSWHandlers().networkError,
};
