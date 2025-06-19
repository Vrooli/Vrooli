import { type ReportResponseCreateInput, type ReportResponse, generatePK, ReportSuggestedAction, ReportStatus } from "@vrooli/shared";
import { type ReportResponseShape } from "../form-data/reportResponseFormData.js";

/**
 * Transformation utilities for converting between different report response data formats
 * These functions handle the data flow between UI forms, API requests, and responses
 */

/**
 * Transform form data to API create request
 * This represents what happens when a moderator submits a report response form
 */
export function transformFormToCreateRequest(formData: Partial<ReportResponseShape>): ReportResponseCreateInput {
    const request: ReportResponseCreateInput = {
        id: generatePK().toString(),
        actionSuggested: formData.actionSuggested!,
        reportConnect: formData.reportConnect!,
    };

    // Add optional fields if provided
    if (formData.details) {
        request.details = formData.details;
    }
    
    if (formData.language) {
        request.language = formData.language;
    }

    return request;
}

/**
 * Transform API response back to form data
 * This would be used when editing an existing report response
 */
export function transformApiResponseToForm(reportResponse: ReportResponse): Partial<ReportResponseShape> {
    return {
        actionSuggested: reportResponse.actionSuggested,
        details: reportResponse.details || undefined,
        language: reportResponse.language || undefined,
        reportConnect: reportResponse.report.id,
    };
}

/**
 * Transform form data to update request
 * Used when updating report response details or action
 */
export function transformFormToUpdateRequest(
    responseId: string, 
    formData: Partial<ReportResponseShape>
): { id: string; actionSuggested?: ReportSuggestedAction; details?: string; language?: string } {
    const request: { id: string; actionSuggested?: ReportSuggestedAction; details?: string; language?: string } = {
        id: responseId,
    };

    if (formData.actionSuggested !== undefined) {
        request.actionSuggested = formData.actionSuggested;
    }
    
    if (formData.details !== undefined) {
        request.details = formData.details;
    }
    
    if (formData.language !== undefined) {
        request.language = formData.language;
    }

    return request;
}

/**
 * Validate form data before submission
 * Returns array of validation errors, empty if valid
 */
export function validateReportResponseFormData(formData: Partial<ReportResponseShape>): string[] {
    const errors: string[] = [];

    if (!formData.actionSuggested) {
        errors.push("Action suggestion is required");
    }

    if (!formData.reportConnect || formData.reportConnect.trim() === "") {
        errors.push("Report ID is required");
    }

    // Validate ID format (should be snowflake-like)
    if (formData.reportConnect && !/^\d{10,19}$/.test(formData.reportConnect)) {
        errors.push("Invalid report ID format");
    }

    // Validate details length if provided
    if (formData.details && formData.details.length > 8192) {
        errors.push("Details are too long (max 8192 characters)");
    }

    // Validate language format if provided
    if (formData.language && !/^[a-z]{2}(-[A-Z]{2})?$/.test(formData.language)) {
        errors.push("Invalid language code format");
    }

    return errors;
}

/**
 * Helper to determine if two report response form data objects represent the same response
 * Useful for testing data consistency
 */
export function areReportResponseFormsEqual(
    form1: Partial<ReportResponseShape>, 
    form2: Partial<ReportResponseShape>
): boolean {
    return (
        form1.actionSuggested === form2.actionSuggested &&
        form1.details === form2.details &&
        form1.language === form2.language &&
        form1.reportConnect === form2.reportConnect
    );
}

/**
 * Mock API service functions for testing
 * These simulate the actual API calls that would be made
 */
export const mockReportResponseService = {
    /**
     * Simulate creating a report response via API
     */
    async create(request: ReportResponseCreateInput): Promise<ReportResponse> {
        // Simulate API delay
        await new Promise(resolve => setTimeout(resolve, 100));
        
        // Create the report response
        const reportResponse: ReportResponse = {
            __typename: "ReportResponse",
            id: request.id,
            actionSuggested: request.actionSuggested,
            details: request.details || null,
            language: request.language || null,
            report: {
                __typename: "Report",
                id: request.reportConnect,
                createdFor: "Comment" as any, // Simplified for testing
                createdAt: "2024-01-01T00:00:00Z",
                details: "Original report details",
                language: "en",
                publicId: "RPT-001",
                reason: "spam",
                responses: [],
                responsesCount: 1,
                status: ReportStatus.Open,
                updatedAt: "2024-01-01T00:00:00Z",
                you: {
                    __typename: "ReportYou",
                    canDelete: false,
                    canRespond: true,
                    canUpdate: false,
                    isOwn: false,
                },
            },
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
            you: {
                __typename: "ReportResponseYou",
                canDelete: true,
                canUpdate: true,
            },
        };

        // Store in global test storage for retrieval by findById
        const storage = (globalThis as any).__testReportResponseStorage || {};
        storage[request.id] = JSON.parse(JSON.stringify(reportResponse)); // Store a deep copy
        (globalThis as any).__testReportResponseStorage = storage;
        
        return reportResponse;
    },

    /**
     * Simulate fetching a report response by ID
     * In a real implementation, this would fetch from database by ID
     */
    async findById(id: string): Promise<ReportResponse> {
        await new Promise(resolve => setTimeout(resolve, 50));
        
        // For testing, we'll return the same report response that was created
        const storedResponses = (globalThis as any).__testReportResponseStorage || {};
        if (storedResponses[id]) {
            // Return a deep copy to prevent mutations
            return JSON.parse(JSON.stringify(storedResponses[id]));
        }
        
        // Fallback for testing - create a minimal response
        return {
            __typename: "ReportResponse",
            id,
            actionSuggested: ReportSuggestedAction.NonIssue,
            details: "Fallback response details",
            language: "en",
            report: {
                __typename: "Report",
                id: "fallback_report_123456789012345678",
                createdFor: "Comment" as any,
                createdAt: "2024-01-01T00:00:00Z",
                details: "Fallback report details",
                language: "en",
                publicId: "RPT-FALLBACK",
                reason: "spam",
                responses: [],
                responsesCount: 1,
                status: ReportStatus.Open,
                updatedAt: "2024-01-01T00:00:00Z",
                you: {
                    __typename: "ReportYou",
                    canDelete: false,
                    canRespond: true,
                    canUpdate: false,
                    isOwn: false,
                },
            },
            createdAt: "2024-01-01T00:00:00Z",
            updatedAt: "2024-01-01T00:00:00Z",
            you: {
                __typename: "ReportResponseYou",
                canDelete: true,
                canUpdate: true,
            },
        };
    },

    /**
     * Simulate updating a report response
     */
    async update(id: string, updates: { actionSuggested?: ReportSuggestedAction; details?: string; language?: string }): Promise<ReportResponse> {
        await new Promise(resolve => setTimeout(resolve, 75));
        
        const reportResponse = await this.findById(id);
        
        // Create a deep copy to avoid mutating the original object
        const updatedReportResponse = JSON.parse(JSON.stringify(reportResponse));
        
        if (updates.actionSuggested !== undefined) {
            updatedReportResponse.actionSuggested = updates.actionSuggested;
        }
        
        if (updates.details !== undefined) {
            updatedReportResponse.details = updates.details;
        }
        
        if (updates.language !== undefined) {
            updatedReportResponse.language = updates.language;
        }
        
        updatedReportResponse.updatedAt = new Date().toISOString();
        
        // Update in storage
        const storage = (globalThis as any).__testReportResponseStorage || {};
        storage[id] = updatedReportResponse;
        (globalThis as any).__testReportResponseStorage = storage;
        
        return updatedReportResponse;
    },

    /**
     * Simulate deleting a report response
     */
    async delete(id: string): Promise<{ success: boolean }> {
        await new Promise(resolve => setTimeout(resolve, 25));
        
        // Remove from storage
        const storage = (globalThis as any).__testReportResponseStorage || {};
        delete storage[id];
        (globalThis as any).__testReportResponseStorage = storage;
        
        return { success: true };
    },
};