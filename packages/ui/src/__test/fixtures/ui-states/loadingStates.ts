/**
 * Loading state fixtures for UI components
 * These represent various loading states that components might display
 */

/**
 * Generic loading states
 */
export const loadingStates = {
    /**
     * Initial data loading
     */
    initial: {
        isLoading: true,
        data: null,
        error: null,
        message: "Loading...",
    },

    /**
     * Refreshing existing data
     */
    refreshing: {
        isLoading: true,
        isRefreshing: true,
        data: null, // Could have existing data
        error: null,
        message: "Refreshing...",
    },

    /**
     * Loading more data (pagination)
     */
    loadingMore: {
        isLoading: true,
        isLoadingMore: true,
        data: [], // Has existing data
        error: null,
        message: "Loading more...",
    },

    /**
     * Submitting form data
     */
    submitting: {
        isSubmitting: true,
        isLoading: true,
        data: null,
        error: null,
        message: "Saving...",
    },

    /**
     * Deleting data
     */
    deleting: {
        isDeleting: true,
        isLoading: true,
        data: null,
        error: null,
        message: "Deleting...",
    },

    /**
     * Searching/filtering
     */
    searching: {
        isSearching: true,
        isLoading: true,
        data: null,
        error: null,
        message: "Searching...",
    },

    /**
     * Uploading files
     */
    uploading: {
        isUploading: true,
        isLoading: true,
        uploadProgress: 0,
        data: null,
        error: null,
        message: "Uploading...",
    },

    /**
     * Processing/calculating
     */
    processing: {
        isProcessing: true,
        isLoading: true,
        data: null,
        error: null,
        message: "Processing...",
    },

    /**
     * Authenticating
     */
    authenticating: {
        isAuthenticating: true,
        isLoading: true,
        data: null,
        error: null,
        message: "Authenticating...",
    },

    /**
     * Connecting (e.g., WebSocket)
     */
    connecting: {
        isConnecting: true,
        isLoading: true,
        data: null,
        error: null,
        message: "Connecting...",
    },
} as const;

/**
 * Component-specific loading states
 */
export const componentLoadingStates = {
    /**
     * List/table loading
     */
    list: {
        items: [],
        isLoading: true,
        hasMore: false,
        totalCount: null,
        pageInfo: null,
    },

    /**
     * Form loading
     */
    form: {
        values: {},
        isLoading: true,
        isSubmitting: false,
        errors: {},
        touched: {},
    },

    /**
     * Detail view loading
     */
    detail: {
        data: null,
        isLoading: true,
        notFound: false,
        error: null,
    },

    /**
     * Search results loading
     */
    searchResults: {
        results: [],
        isLoading: true,
        query: "",
        totalResults: null,
        facets: {},
    },

    /**
     * Dashboard loading
     */
    dashboard: {
        stats: null,
        recentItems: [],
        notifications: [],
        isLoading: true,
        lastUpdated: null,
    },

    /**
     * Chat loading
     */
    chat: {
        messages: [],
        participants: [],
        isLoading: true,
        isLoadingHistory: false,
        hasMoreHistory: true,
    },

    /**
     * File upload loading
     */
    fileUpload: {
        files: [],
        isUploading: true,
        uploadProgress: {},
        errors: {},
        completed: [],
    },
} as const;

/**
 * Skeleton loading states for shimmer effects
 */
export const skeletonStates = {
    /**
     * Card skeleton
     */
    card: {
        type: "skeleton",
        lines: 3,
        avatar: true,
        actions: true,
    },

    /**
     * List item skeleton
     */
    listItem: {
        type: "skeleton",
        lines: 2,
        thumbnail: true,
    },

    /**
     * Table row skeleton
     */
    tableRow: {
        type: "skeleton",
        columns: 5,
        height: 48,
    },

    /**
     * Form field skeleton
     */
    formField: {
        type: "skeleton",
        label: true,
        input: true,
        helpText: true,
    },

    /**
     * Profile skeleton
     */
    profile: {
        type: "skeleton",
        avatar: true,
        name: true,
        bio: true,
        stats: true,
    },
} as const;

/**
 * Progressive loading states
 */
export const progressiveLoadingStates = {
    /**
     * Upload with progress
     */
    uploadWithProgress: {
        isUploading: true,
        progress: 45,
        bytesUploaded: 4567890,
        totalBytes: 10234567,
        timeRemaining: 30, // seconds
        speed: "1.2 MB/s",
    },

    /**
     * Multi-step process
     */
    multiStep: {
        isProcessing: true,
        currentStep: 2,
        totalSteps: 5,
        currentStepName: "Validating data",
        stepsCompleted: ["Upload", "Parse"],
        stepsRemaining: ["Validate", "Process", "Complete"],
    },

    /**
     * Batch operation
     */
    batchOperation: {
        isProcessing: true,
        itemsProcessed: 25,
        totalItems: 100,
        currentItem: "file_26.pdf",
        errors: [],
        warnings: [],
    },
} as const;

/**
 * Helper function to create loading state with custom message
 */
export const createLoadingState = (message: string = "Loading...") => ({
    isLoading: true,
    data: null,
    error: null,
    message,
});

/**
 * Helper function to create loading state with timeout
 */
export const createTimedLoadingState = (timeoutMs: number = 30000) => ({
    isLoading: true,
    data: null,
    error: null,
    timeout: timeoutMs,
    startTime: Date.now(),
});

/**
 * Helper function to simulate loading progress
 */
export const simulateProgress = (durationMs: number = 3000) => {
    const steps = 20;
    const interval = durationMs / steps;
    let current = 0;

    return {
        start: () => {
            const timer = setInterval(() => {
                current += 100 / steps;
                if (current >= 100) {
                    clearInterval(timer);
                }
            }, interval);
            return timer;
        },
        getProgress: () => Math.min(current, 100),
    };
};