/**
 * Success state fixtures for UI components
 * These represent various success states that components might display
 */

/**
 * Generic success responses
 */
export const successStates = {
    /**
     * Create success
     */
    created: {
        success: true,
        message: "Successfully created",
        data: {
            id: "new_123456789",
            createdAt: "2024-01-20T10:00:00Z",
        },
        nextAction: {
            label: "View",
            url: "/items/new_123456789",
        },
    },

    /**
     * Update success
     */
    updated: {
        success: true,
        message: "Changes saved successfully",
        data: {
            id: "item_123456789",
            updatedAt: "2024-01-20T10:00:00Z",
            changes: ["name", "description"],
        },
    },

    /**
     * Delete success
     */
    deleted: {
        success: true,
        message: "Successfully deleted",
        data: {
            deletedId: "item_123456789",
            deletedAt: "2024-01-20T10:00:00Z",
        },
        nextAction: {
            label: "Go to list",
            url: "/items",
        },
    },

    /**
     * Upload success
     */
    uploaded: {
        success: true,
        message: "File uploaded successfully",
        data: {
            fileId: "file_123456789",
            fileName: "document.pdf",
            fileSize: 1048576,
            uploadedAt: "2024-01-20T10:00:00Z",
        },
    },

    /**
     * Send/Submit success
     */
    sent: {
        success: true,
        message: "Message sent successfully",
        data: {
            messageId: "msg_123456789",
            recipientCount: 5,
            sentAt: "2024-01-20T10:00:00Z",
        },
    },

    /**
     * Copy success
     */
    copied: {
        success: true,
        message: "Copied to clipboard",
        data: {
            content: "https://example.com/share/123",
        },
        duration: 2000, // Auto-dismiss after 2 seconds
    },

    /**
     * Save success
     */
    saved: {
        success: true,
        message: "Saved successfully",
        data: {
            savedAt: "2024-01-20T10:00:00Z",
            autoSave: false,
        },
    },
} as const;

/**
 * Operation-specific success states
 */
export const operationSuccessStates = {
    /**
     * Authentication success
     */
    auth: {
        login: {
            success: true,
            message: "Welcome back!",
            data: {
                userId: "user_123456789",
                lastLogin: "2024-01-19T15:00:00Z",
                sessionExpiry: "2024-01-21T10:00:00Z",
            },
            redirect: "/dashboard",
        },
        logout: {
            success: true,
            message: "You've been logged out successfully",
            redirect: "/",
        },
        register: {
            success: true,
            message: "Account created! Please check your email to verify.",
            data: {
                userId: "user_new_123456",
                email: "user@example.com",
            },
        },
        passwordReset: {
            success: true,
            message: "Password reset successfully",
            redirect: "/login",
        },
    },

    /**
     * Data operations success
     */
    data: {
        import: {
            success: true,
            message: "Data imported successfully",
            data: {
                totalRecords: 1500,
                successCount: 1485,
                errorCount: 15,
                duration: 3456, // milliseconds
            },
            details: {
                errors: [
                    { row: 145, reason: "Invalid email format" },
                    { row: 892, reason: "Duplicate entry" },
                ],
            },
        },
        export: {
            success: true,
            message: "Export completed",
            data: {
                fileName: "export_2024-01-20.csv",
                records: 2500,
                fileSize: 2457600, // bytes
                downloadUrl: "/downloads/export_123456.csv",
            },
        },
        sync: {
            success: true,
            message: "Sync completed",
            data: {
                itemsSynced: 45,
                lastSyncTime: "2024-01-20T10:00:00Z",
                nextSyncTime: "2024-01-20T11:00:00Z",
            },
        },
    },

    /**
     * Collaboration success
     */
    collaboration: {
        invite: {
            success: true,
            message: "Invitation sent",
            data: {
                inviteId: "invite_123456789",
                recipientEmail: "colleague@example.com",
                expiresAt: "2024-01-27T10:00:00Z",
            },
        },
        share: {
            success: true,
            message: "Shared successfully",
            data: {
                shareUrl: "https://app.vrooli.com/share/abc123",
                permissions: ["view", "comment"],
                expiresAt: null, // No expiration
            },
        },
        permission: {
            success: true,
            message: "Permissions updated",
            data: {
                userId: "user_123456789",
                oldPermissions: ["view"],
                newPermissions: ["view", "edit", "delete"],
            },
        },
    },

    /**
     * Payment success
     */
    payment: {
        purchase: {
            success: true,
            message: "Payment successful",
            data: {
                orderId: "order_123456789",
                amount: 4999, // cents
                currency: "USD",
                receiptUrl: "/receipts/order_123456789",
            },
        },
        subscription: {
            success: true,
            message: "Subscription activated",
            data: {
                plan: "Premium",
                billingCycle: "monthly",
                nextBillingDate: "2024-02-20",
                features: ["unlimited_storage", "priority_support"],
            },
        },
        refund: {
            success: true,
            message: "Refund processed",
            data: {
                refundId: "refund_123456789",
                amount: 4999,
                reason: "Customer request",
                processedAt: "2024-01-20T10:00:00Z",
            },
        },
    },
} as const;

/**
 * Component success states
 */
export const componentSuccessStates = {
    /**
     * Form submission success
     */
    form: {
        isSubmitting: false,
        isSubmitted: true,
        submitSuccess: true,
        submitResult: successStates.created,
        values: {}, // Reset or keep based on UX
        errors: {},
        touched: {},
    },

    /**
     * List operation success
     */
    list: {
        items: [], // Updated list
        isLoading: false,
        lastOperation: {
            type: "delete",
            success: true,
            affectedItems: ["item_123"],
            message: "1 item deleted",
        },
    },

    /**
     * Upload component success
     */
    upload: {
        files: [],
        isUploading: false,
        uploadComplete: true,
        successfulUploads: [
            {
                fileName: "document.pdf",
                fileSize: 1048576,
                uploadedAt: "2024-01-20T10:00:00Z",
            },
        ],
        failedUploads: [],
    },

    /**
     * Notification success
     */
    notification: {
        show: true,
        type: "success",
        message: "Operation completed successfully",
        duration: 5000,
        action: {
            label: "Undo",
            handler: () => console.log("Undo clicked"),
        },
    },
} as const;

/**
 * Progress completion states
 */
export const completionStates = {
    /**
     * Task completion
     */
    taskComplete: {
        success: true,
        progress: 100,
        message: "All tasks completed!",
        summary: {
            total: 10,
            completed: 10,
            duration: 12345, // milliseconds
        },
    },

    /**
     * Multi-step completion
     */
    wizardComplete: {
        success: true,
        currentStep: 5,
        totalSteps: 5,
        message: "Setup completed successfully!",
        data: {
            configId: "config_123456789",
            steps: ["account", "profile", "preferences", "team", "review"],
        },
    },

    /**
     * Achievement unlocked
     */
    achievement: {
        success: true,
        type: "achievement",
        title: "First Project Created!",
        description: "You've created your first project",
        icon: "🎉",
        points: 100,
    },
} as const;

/**
 * Feedback messages for success states
 */
export const successFeedback = {
    /**
     * Encouraging messages
     */
    encouraging: [
        "Great job!",
        "Awesome work!",
        "You're doing great!",
        "Keep it up!",
        "Excellent!",
    ],

    /**
     * Informative messages
     */
    informative: [
        "Your changes have been saved",
        "Operation completed successfully",
        "Process finished without errors",
        "All items processed successfully",
    ],

    /**
     * Next steps
     */
    nextSteps: [
        "What would you like to do next?",
        "Ready for the next task",
        "You can now proceed to the next step",
        "Feel free to explore more features",
    ],
} as const;

/**
 * Helper function to create success state with custom data
 */
export const createSuccessState = (
    message: string,
    data?: Record<string, any>,
    nextAction?: { label: string; url: string },
) => ({
    success: true,
    message,
    data: data || {},
    timestamp: new Date().toISOString(),
    nextAction,
});

/**
 * Helper function to create toast notification
 */
export const createSuccessToast = (
    message: string,
    duration: number = 3000,
    action?: { label: string; handler: () => void },
) => ({
    id: `toast_${Date.now()}`,
    type: "success" as const,
    message,
    duration,
    action,
    createdAt: new Date().toISOString(),
});

/**
 * Helper function to format success message with count
 */
export const formatSuccessCount = (count: number, singular: string, plural: string) => {
    if (count === 0) return `No ${plural} affected`;
    if (count === 1) return `1 ${singular} successfully processed`;
    return `${count} ${plural} successfully processed`;
};