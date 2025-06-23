/**
 * Bookmark UI State Fixtures
 * 
 * This file provides comprehensive UI state fixtures for bookmark-related components.
 * It includes loading, error, success, and empty states with realistic scenarios.
 */

import type { 
    Bookmark, 
    BookmarkList,
    BookmarkFor, 
} from "@vrooli/shared";
import type { 
    BookmarkUIState,
    UIStateFactory,
    LoadingContext,
    AppError,
    BookmarkFormData, 
} from "../types.js";

/**
 * Extended bookmark UI state with additional context
 */
export interface ExtendedBookmarkUIState extends BookmarkUIState {
    // Form-related state
    formState?: {
        isSubmitting: boolean;
        isDirty: boolean;
        errors: Record<string, string>;
        touched: Record<string, boolean>;
        values: BookmarkFormData;
    };
    
    // Interaction state
    interactionState?: {
        isHovered: boolean;
        isFocused: boolean;
        isPressed: boolean;
        lastAction?: "created" | "updated" | "deleted" | "toggled";
        actionTimestamp?: number;
    };
    
    // List management state
    listState?: {
        isCreatingList: boolean;
        isSelectingList: boolean;
        searchQuery: string;
        filteredLists: Array<{ id: string; label: string }>;
    };
    
    // Async operation state
    operationState?: {
        currentOperation?: "create" | "update" | "delete" | "toggle";
        operationId?: string;
        progress?: number;
        cancellable?: boolean;
        onCancel?: () => void;
    };
    
    // Error recovery state
    recoveryState?: {
        retryCount: number;
        maxRetries: number;
        nextRetryDelay: number;
        canRetry: boolean;
        lastError?: AppError;
    };
}

/**
 * Bookmark-specific loading contexts
 */
export type BookmarkLoadingContext = LoadingContext & {
    operation: "creating" | "updating" | "deleting" | "fetching" | "toggling" | "list-creating";
    bookmarkType?: BookmarkFor;
    objectId?: string;
    listId?: string;
};

/**
 * Bookmark-specific error types
 */
export interface BookmarkError extends AppError {
    code: 
        | "BOOKMARK_CREATE_FAILED"
        | "BOOKMARK_UPDATE_FAILED" 
        | "BOOKMARK_DELETE_FAILED"
        | "BOOKMARK_NOT_FOUND"
        | "LIST_CREATE_FAILED"
        | "LIST_NOT_FOUND"
        | "VALIDATION_ERROR"
        | "PERMISSION_DENIED"
        | "NETWORK_ERROR"
        | "SERVER_ERROR";
    bookmarkContext?: {
        bookmarkFor?: BookmarkFor;
        objectId?: string;
        listId?: string;
        operation?: string;
    };
}

/**
 * Production-grade bookmark UI state factory
 */
export class BookmarkStateFactory implements UIStateFactory<ExtendedBookmarkUIState> {
    /**
     * Generate unique ID for testing
     */
    private generateId(): string {
        return `state_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    }
    
    /**
     * Create loading state with context
     */
    createLoadingState(context?: BookmarkLoadingContext): ExtendedBookmarkUIState {
        return {
            isLoading: true,
            bookmark: null,
            error: null,
            isBookmarked: false,
            availableLists: [],
            showListSelection: false,
            
            operationState: {
                currentOperation: context?.operation || "creating",
                operationId: this.generateId(),
                progress: context?.progress,
                cancellable: context?.cancellable || false,
                onCancel: context?.cancellable ? () => console.log("Operation cancelled") : undefined,
            },
            
            formState: {
                isSubmitting: true,
                isDirty: false,
                errors: {},
                touched: {},
                values: {
                    bookmarkFor: context?.bookmarkType || "Resource",
                    forConnect: context?.objectId || this.generateId(),
                },
            },
        };
    }
    
    /**
     * Create error state with detailed error information
     */
    createErrorState(error: BookmarkError): ExtendedBookmarkUIState {
        return {
            isLoading: false,
            bookmark: null,
            error: error.message,
            isBookmarked: false,
            availableLists: [],
            showListSelection: false,
            
            recoveryState: {
                retryCount: 0,
                maxRetries: 3,
                nextRetryDelay: 1000,
                canRetry: error.recoverable || false,
                lastError: error,
            },
            
            formState: error.code === "VALIDATION_ERROR" ? {
                isSubmitting: false,
                isDirty: true,
                errors: error.details?.fieldErrors || {},
                touched: Object.keys(error.details?.fieldErrors || {}).reduce((acc, key) => {
                    acc[key] = true;
                    return acc;
                }, {} as Record<string, boolean>),
                values: error.details?.formData || {
                    bookmarkFor: "Resource",
                    forConnect: "",
                },
            } : undefined,
        };
    }
    
    /**
     * Create success state after successful operation
     */
    createSuccessState(
        data: Bookmark | { bookmark: Bookmark; message?: string }, 
        message?: string,
    ): ExtendedBookmarkUIState {
        const bookmark = "bookmark" in data ? data.bookmark : data;
        const successMessage = "message" in data ? data.message : message;
        
        return {
            isLoading: false,
            bookmark,
            error: null,
            isBookmarked: true,
            availableLists: [
                { id: bookmark.list.id, label: bookmark.list.label },
            ],
            showListSelection: false,
            
            interactionState: {
                isHovered: false,
                isFocused: false,
                isPressed: false,
                lastAction: "created",
                actionTimestamp: Date.now(),
            },
            
            formState: {
                isSubmitting: false,
                isDirty: false,
                errors: {},
                touched: {},
                values: {
                    bookmarkFor: bookmark.to.__typename as BookmarkFor,
                    forConnect: bookmark.to.id,
                    listId: bookmark.list.id,
                },
            },
        };
    }
    
    /**
     * Create empty state when no bookmarks exist
     */
    createEmptyState(): ExtendedBookmarkUIState {
        return {
            isLoading: false,
            bookmark: null,
            error: null,
            isBookmarked: false,
            availableLists: [],
            showListSelection: false,
            
            formState: {
                isSubmitting: false,
                isDirty: false,
                errors: {},
                touched: {},
                values: {
                    bookmarkFor: "Resource",
                    forConnect: "",
                },
            },
        };
    }
    
    /**
     * Create state with available lists for selection
     */
    createWithListsState(lists?: Array<{ id: string; label: string }>): ExtendedBookmarkUIState {
        const defaultLists = lists || [
            { id: this.generateId(), label: "Favorites" },
            { id: this.generateId(), label: "To Review" },
            { id: this.generateId(), label: "Important" },
            { id: this.generateId(), label: "Resources" },
        ];
        
        return {
            isLoading: false,
            bookmark: null,
            error: null,
            isBookmarked: false,
            availableLists: defaultLists,
            showListSelection: true,
            
            listState: {
                isCreatingList: false,
                isSelectingList: true,
                searchQuery: "",
                filteredLists: defaultLists,
            },
            
            formState: {
                isSubmitting: false,
                isDirty: false,
                errors: {},
                touched: {},
                values: {
                    bookmarkFor: "Resource",
                    forConnect: this.generateId(),
                },
            },
        };
    }
    
    /**
     * Create state for list creation flow
     */
    createListCreationState(listLabel?: string): ExtendedBookmarkUIState {
        return {
            isLoading: false,
            bookmark: null,
            error: null,
            isBookmarked: false,
            availableLists: [],
            showListSelection: true,
            
            listState: {
                isCreatingList: true,
                isSelectingList: false,
                searchQuery: listLabel || "",
                filteredLists: [],
            },
            
            formState: {
                isSubmitting: false,
                isDirty: true,
                errors: {},
                touched: { newListLabel: true },
                values: {
                    bookmarkFor: "Resource",
                    forConnect: this.generateId(),
                    createNewList: true,
                    newListLabel: listLabel || "My New List",
                },
            },
        };
    }
    
    /**
     * Create state with form validation errors
     */
    createValidationErrorState(fieldErrors: Record<string, string>): ExtendedBookmarkUIState {
        return {
            isLoading: false,
            bookmark: null,
            error: "Please fix the validation errors below",
            isBookmarked: false,
            availableLists: [],
            showListSelection: false,
            
            formState: {
                isSubmitting: false,
                isDirty: true,
                errors: fieldErrors,
                touched: Object.keys(fieldErrors).reduce((acc, key) => {
                    acc[key] = true;
                    return acc;
                }, {} as Record<string, boolean>),
                values: {
                    bookmarkFor: "Resource",
                    forConnect: "",
                },
            },
        };
    }
    
    /**
     * Transition to loading state from current state
     */
    transitionToLoading(
        currentState: ExtendedBookmarkUIState,
        context?: BookmarkLoadingContext,
    ): ExtendedBookmarkUIState {
        return {
            ...currentState,
            isLoading: true,
            error: null,
            
            operationState: {
                currentOperation: context?.operation || "creating",
                operationId: this.generateId(),
                progress: context?.progress,
                cancellable: context?.cancellable || false,
            },
            
            formState: currentState.formState ? {
                ...currentState.formState,
                isSubmitting: true,
                errors: {}, // Clear errors on new attempt
            } : undefined,
        };
    }
    
    /**
     * Transition to success state from current state
     */
    transitionToSuccess(
        currentState: ExtendedBookmarkUIState,
        data: Bookmark,
    ): ExtendedBookmarkUIState {
        return {
            ...currentState,
            isLoading: false,
            bookmark: data,
            error: null,
            isBookmarked: true,
            
            interactionState: {
                ...currentState.interactionState,
                lastAction: "created",
                actionTimestamp: Date.now(),
            },
            
            formState: currentState.formState ? {
                ...currentState.formState,
                isSubmitting: false,
                isDirty: false,
                errors: {},
            } : undefined,
            
            operationState: undefined, // Clear operation state on success
            recoveryState: undefined, // Clear recovery state on success
        };
    }
    
    /**
     * Transition to error state from current state
     */
    transitionToError(
        currentState: ExtendedBookmarkUIState,
        error: BookmarkError,
    ): ExtendedBookmarkUIState {
        const currentRetryCount = currentState.recoveryState?.retryCount || 0;
        
        return {
            ...currentState,
            isLoading: false,
            error: error.message,
            
            recoveryState: {
                retryCount: currentRetryCount + 1,
                maxRetries: 3,
                nextRetryDelay: Math.min(1000 * Math.pow(2, currentRetryCount), 10000), // Exponential backoff
                canRetry: (currentRetryCount + 1) < 3 && (error.recoverable || false),
                lastError: error,
            },
            
            formState: currentState.formState ? {
                ...currentState.formState,
                isSubmitting: false,
                errors: error.code === "VALIDATION_ERROR" ? error.details?.fieldErrors || {} : {},
            } : undefined,
            
            operationState: undefined, // Clear operation state on error
        };
    }
    
    /**
     * Create state for bookmark toggle operation
     */
    createToggleState(isCurrentlyBookmarked: boolean, targetObject: {
        type: BookmarkFor;
        id: string;
        name?: string;
    }): ExtendedBookmarkUIState {
        return {
            isLoading: false,
            bookmark: null,
            error: null,
            isBookmarked: isCurrentlyBookmarked,
            availableLists: [],
            showListSelection: !isCurrentlyBookmarked, // Show list selection when bookmarking
            
            formState: {
                isSubmitting: false,
                isDirty: false,
                errors: {},
                touched: {},
                values: {
                    bookmarkFor: targetObject.type,
                    forConnect: targetObject.id,
                },
            },
            
            interactionState: {
                isHovered: false,
                isFocused: false,
                isPressed: false,
                lastAction: undefined,
                actionTimestamp: undefined,
            },
        };
    }
}

/**
 * State transition validator for bookmark states
 */
export class BookmarkStateTransitionValidator {
    /**
     * Validate state transitions
     */
    validateTransition(
        from: ExtendedBookmarkUIState,
        to: ExtendedBookmarkUIState,
    ): { valid: boolean; reason?: string } {
        // Cannot transition from loading to loading
        if (from.isLoading && to.isLoading) {
            return { 
                valid: false, 
                reason: "Cannot transition from loading to loading state", 
            };
        }
        
        // Cannot have bookmark and error at the same time
        if (to.bookmark && to.error) {
            return { 
                valid: false, 
                reason: "Cannot have both bookmark data and error in the same state", 
            };
        }
        
        // If bookmark exists, isBookmarked should be true
        if (to.bookmark && !to.isBookmarked) {
            return { 
                valid: false, 
                reason: "State inconsistency: bookmark exists but isBookmarked is false", 
            };
        }
        
        // If isBookmarked is true, bookmark should exist (unless loading)
        if (to.isBookmarked && !to.bookmark && !to.isLoading) {
            return { 
                valid: false, 
                reason: "State inconsistency: isBookmarked is true but no bookmark data exists", 
            };
        }
        
        return { valid: true };
    }
}

/**
 * Pre-configured bookmark state scenarios
 */
export const bookmarkStateScenarios = {
    // Loading scenarios
    creatingBookmark: () => new BookmarkStateFactory().createLoadingState({
        operation: "creating",
        message: "Creating bookmark...",
        progress: 50,
    }),
    
    updatingBookmark: () => new BookmarkStateFactory().createLoadingState({
        operation: "updating",
        message: "Updating bookmark...",
    }),
    
    deletingBookmark: () => new BookmarkStateFactory().createLoadingState({
        operation: "deleting",
        message: "Removing bookmark...",
    }),
    
    // Error scenarios
    networkError: () => new BookmarkStateFactory().createErrorState({
        code: "NETWORK_ERROR",
        message: "Unable to connect to server",
        recoverable: true,
        details: {},
    }),
    
    validationError: () => new BookmarkStateFactory().createErrorState({
        code: "VALIDATION_ERROR",
        message: "Please fix the errors below",
        recoverable: true,
        details: {
            fieldErrors: {
                forConnect: "Target object is required",
                newListLabel: "List name is required when creating a new list",
            },
        },
    }),
    
    permissionError: () => new BookmarkStateFactory().createErrorState({
        code: "PERMISSION_DENIED",
        message: "You do not have permission to bookmark this item",
        recoverable: false,
        details: {},
    }),
    
    // Success scenarios
    bookmarkCreated: (bookmark?: Bookmark) => {
        const factory = new BookmarkStateFactory();
        return factory.createSuccessState(
            bookmark || createMockBookmark(),
            "Bookmark created successfully!",
        );
    },
    
    bookmarkUpdated: (bookmark?: Bookmark) => {
        const factory = new BookmarkStateFactory();
        return factory.createSuccessState(
            bookmark || createMockBookmark(),
            "Bookmark updated successfully!",
        );
    },
    
    // Interactive scenarios
    selectingList: () => new BookmarkStateFactory().createWithListsState(),
    
    creatingNewList: () => new BookmarkStateFactory().createListCreationState("My Custom List"),
    
    toggleBookmark: (isBookmarked: boolean) => new BookmarkStateFactory().createToggleState(
        isBookmarked,
        { type: "Resource", id: "test-resource-123", name: "Test Resource" },
    ),
    
    // Empty scenarios
    noBookmarks: () => new BookmarkStateFactory().createEmptyState(),
    
    noListsAvailable: () => new BookmarkStateFactory().createWithListsState([]),
};

/**
 * Helper function to create mock bookmark data
 */
function createMockBookmark(): Bookmark {
    const now = new Date().toISOString();
    const id = `mock_${Date.now()}_${Math.random().toString(36).substr(2, 9)}`;
    
    return {
        __typename: "Bookmark",
        id,
        createdAt: now,
        updatedAt: now,
        by: {
            __typename: "User",
            id: `user_${id}`,
            handle: "testuser",
            name: "Test User",
            createdAt: now,
            updatedAt: now,
            isBot: false,
            isPrivate: false,
            profileImage: null,
            bannerImage: null,
            premium: false,
            premiumExpiration: null,
            roles: [],
            wallets: [],
            translations: [],
            translationsCount: 0,
            you: {
                __typename: "UserYou",
                isBlocked: false,
                isBlockedByYou: false,
                canDelete: false,
                canReport: false,
                canUpdate: false,
                isBookmarked: false,
                isReacted: false,
                reactionSummary: {
                    __typename: "ReactionSummary",
                    emotion: null,
                    count: 0,
                },
            },
        },
        to: {
            __typename: "Resource",
            id: `resource_${id}`,
            createdAt: now,
            updatedAt: now,
            isInternal: false,
            isPrivate: false,
            usedBy: [],
            usedByCount: 0,
            versions: [],
            versionsCount: 0,
            you: {
                __typename: "ResourceYou",
                canDelete: false,
                canUpdate: false,
                canReport: false,
                isBookmarked: true,
                isReacted: false,
                reaction: null,
            },
        },
        list: {
            __typename: "BookmarkList",
            id: `list_${id}`,
            label: "My Bookmarks",
            createdAt: now,
            updatedAt: now,
            bookmarks: [],
            bookmarksCount: 1,
            you: {
                __typename: "BookmarkListYou",
                canDelete: true,
                canUpdate: true,
            },
        },
    };
}

// Export default factory instance
export const bookmarkStates = new BookmarkStateFactory();
export const bookmarkStateValidator = new BookmarkStateTransitionValidator();
