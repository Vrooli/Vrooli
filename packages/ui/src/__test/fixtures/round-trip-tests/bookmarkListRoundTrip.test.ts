import { describe, test, expect, beforeEach } from 'vitest';
import { shapeBookmarkList, bookmarkListValidation, generatePK, type BookmarkList } from "@vrooli/shared";
import { 
    minimalBookmarkListFormInput,
    completeBookmarkListFormInput,
    bookmarkListWithBookmarksFormInput,
    type BookmarkListFormData
} from '../form-data/bookmarkListFormData.js';
import { 
    minimalBookmarkListResponse,
    completeBookmarkListResponse,
    bookmarkListResponseVariants
} from '../api-responses/bookmarkListResponses.js';

/**
 * Round-trip testing for BookmarkList data flow using REAL application functions
 * Tests the complete user journey: Form Input → API Request → Database → API Response → UI Display
 * 
 * ✅ Uses real shapeBookmarkList.create() for transformations
 * ✅ Uses real bookmarkListValidation for validation
 * ✅ Tests actual application logic instead of mock implementations
 */

// Mock bookmark list service for testing (simulates API calls)
const mockBookmarkListService = {
    storage: {} as Record<string, BookmarkList>,

    async create(data: any): Promise<BookmarkList> {
        const id = data.id || generatePK().toString();
        const now = new Date().toISOString();
        
        const bookmarkList: BookmarkList = {
            __typename: "BookmarkList",
            id,
            label: data.label,
            bookmarks: data.bookmarksCreate ? data.bookmarksCreate.map((bookmark: any) => ({
                __typename: "Bookmark",
                id: bookmark.id || generatePK().toString(),
                to: {
                    __typename: bookmark.bookmarkFor,
                    id: bookmark.forConnect,
                },
                by: {
                    __typename: "User",
                    id: "user_test_123456789012345678",
                    handle: "testuser",
                    name: "Test User",
                },
                list: {
                    __typename: "BookmarkList",
                    id,
                    label: data.label,
                },
                createdAt: now,
                updatedAt: now,
            })) : [],
            bookmarksCount: data.bookmarksCreate ? data.bookmarksCreate.length : 0,
            createdAt: now,
            updatedAt: now,
        };

        this.storage[id] = bookmarkList;
        return bookmarkList;
    },

    async findById(id: string): Promise<BookmarkList> {
        const bookmarkList = this.storage[id];
        if (!bookmarkList) {
            throw new Error(`BookmarkList with id ${id} not found`);
        }
        return bookmarkList;
    },

    async update(id: string, data: any): Promise<BookmarkList> {
        const existing = await this.findById(id);
        const now = new Date().toISOString();
        
        const updated: BookmarkList = {
            ...existing,
            label: data.label || existing.label,
            updatedAt: now,
        };

        this.storage[id] = updated;
        return updated;
    },

    async delete(id: string): Promise<{ success: boolean }> {
        if (this.storage[id]) {
            delete this.storage[id];
            return { success: true };
        }
        return { success: false };
    },
};

// Type for bookmark list form data
type BookmarkListFormData = {
    label: string;
    description?: string;
    initialBookmarks?: Array<{
        objectType: string;
        objectId: string;
    }>;
};

// Helper functions using REAL application logic
function transformFormToCreateRequestReal(formData: BookmarkListFormData) {
    return shapeBookmarkList.create({
        __typename: "BookmarkList",
        id: generatePK().toString(),
        label: formData.label,
        bookmarks: formData.initialBookmarks ? formData.initialBookmarks.map(bookmark => ({
            __typename: "Bookmark",
            id: generatePK().toString(),
            to: {
                __typename: bookmark.objectType as any,
                id: bookmark.objectId,
            },
        })) : undefined,
    });
}

function transformFormToUpdateRequestReal(bookmarkListId: string, formData: Partial<BookmarkListFormData>) {
    const updateRequest: { id: string; label?: string } = {
        id: bookmarkListId,
    };
    
    if (formData.label) {
        updateRequest.label = formData.label;
    }
    
    return updateRequest;
}

async function validateBookmarkListFormDataReal(formData: BookmarkListFormData): Promise<string[]> {
    try {
        // Use real validation schema - construct the request object first
        const validationData = {
            id: generatePK().toString(),
            label: formData.label,
            ...(formData.initialBookmarks && {
                bookmarksCreate: formData.initialBookmarks.map(bookmark => ({
                    id: generatePK().toString(),
                    bookmarkFor: bookmark.objectType,
                    forConnect: bookmark.objectId,
                })),
            }),
        };
        
        await bookmarkListValidation.create({ omitFields: [] }).validate(validationData);
        return []; // No validation errors
    } catch (error: any) {
        if (error.errors) {
            return error.errors; // Yup validation errors
        }
        return [error.message || "Validation failed"];
    }
}

function transformApiResponseToFormReal(bookmarkList: BookmarkList): BookmarkListFormData {
    return {
        label: bookmarkList.label,
        initialBookmarks: bookmarkList.bookmarks?.map(bookmark => ({
            objectType: bookmark.to.__typename,
            objectId: bookmark.to.id,
        })),
    };
}

function areBookmarkListFormsEqualReal(form1: BookmarkListFormData, form2: BookmarkListFormData): boolean {
    return (
        form1.label === form2.label &&
        JSON.stringify(form1.initialBookmarks || []) === JSON.stringify(form2.initialBookmarks || [])
    );
}

describe('BookmarkList Round-Trip Data Flow', () => {
    beforeEach(() => {
        // Clear test storage between tests
        (globalThis as any).__testBookmarkListStorage = {};
        mockBookmarkListService.storage = {};
    });

    test('minimal bookmark list creation maintains data integrity through complete flow', async () => {
        // 🎨 STEP 1: User fills out minimal bookmark list form
        const userFormData: BookmarkListFormData = {
            label: "My Bookmarks",
        };
        
        // Validate form data using REAL validation schema
        const validationErrors = await validateBookmarkListFormDataReal(userFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Form submits to API using REAL shape function
        const apiCreateRequest = transformFormToCreateRequestReal(userFormData);
        expect(apiCreateRequest.label).toBe(userFormData.label);
        expect(apiCreateRequest.id).toMatch(/^\d{10,19}$/); // Valid ID (generatePK might be shorter in test env)
        
        // 🗄️ STEP 3: API creates bookmark list (simulated - real test would hit test DB)
        const createdBookmarkList = await mockBookmarkListService.create(apiCreateRequest);
        expect(createdBookmarkList.id).toBe(apiCreateRequest.id);
        expect(createdBookmarkList.label).toBe(userFormData.label);
        expect(createdBookmarkList.bookmarks).toEqual([]);
        expect(createdBookmarkList.bookmarksCount).toBe(0);
        
        // 🔗 STEP 4: API fetches bookmark list back
        const fetchedBookmarkList = await mockBookmarkListService.findById(createdBookmarkList.id);
        expect(fetchedBookmarkList.id).toBe(createdBookmarkList.id);
        expect(fetchedBookmarkList.label).toBe(userFormData.label);
        expect(fetchedBookmarkList.bookmarks).toEqual([]);
        
        // 🎨 STEP 5: UI would display the bookmark list using REAL transformation
        // Verify that form data can be reconstructed from API response
        const reconstructedFormData = transformApiResponseToFormReal(fetchedBookmarkList);
        expect(reconstructedFormData.label).toBe(userFormData.label);
        expect(reconstructedFormData.initialBookmarks).toEqual([]);
        
        // ✅ VERIFICATION: Complete round-trip integrity using REAL comparison
        expect(areBookmarkListFormsEqualReal(
            userFormData, 
            reconstructedFormData
        )).toBe(true);
    });

    test('complete bookmark list with initial bookmarks preserves all data', async () => {
        // 🎨 STEP 1: User creates bookmark list with initial bookmarks
        const userFormData: BookmarkListFormData = {
            label: "Project Resources",
            initialBookmarks: [
                {
                    objectType: "Resource",
                    objectId: "123456789012345678",
                },
                {
                    objectType: "Tag",
                    objectId: "234567890123456789",
                },
                {
                    objectType: "User",
                    objectId: "345678901234567890",
                },
            ],
        };
        
        // Validate complex form using REAL validation
        const validationErrors = await validateBookmarkListFormDataReal(userFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Transform to API request using REAL shape function
        const apiCreateRequest = transformFormToCreateRequestReal(userFormData);
        expect(apiCreateRequest.bookmarksCreate).toBeDefined();
        expect(apiCreateRequest.bookmarksCreate).toHaveLength(3);
        expect(apiCreateRequest.bookmarksCreate![0].bookmarkFor).toBe("Resource");
        expect(apiCreateRequest.bookmarksCreate![0].forConnect).toBe("123456789012345678");
        
        // 🗄️ STEP 3: Create via API
        const createdBookmarkList = await mockBookmarkListService.create(apiCreateRequest);
        expect(createdBookmarkList.bookmarks).toHaveLength(3);
        expect(createdBookmarkList.bookmarksCount).toBe(3);
        expect(createdBookmarkList.bookmarks![0].to.__typename).toBe("Resource");
        expect(createdBookmarkList.bookmarks![0].to.id).toBe("123456789012345678");
        
        // 🔗 STEP 4: Fetch back from API
        const fetchedBookmarkList = await mockBookmarkListService.findById(createdBookmarkList.id);
        expect(fetchedBookmarkList.bookmarks).toHaveLength(3);
        expect(fetchedBookmarkList.bookmarksCount).toBe(3);
        
        // 🎨 STEP 5: Verify UI can display correctly using REAL transformation
        const reconstructedFormData = transformApiResponseToFormReal(fetchedBookmarkList);
        expect(reconstructedFormData.label).toBe(userFormData.label);
        expect(reconstructedFormData.initialBookmarks).toHaveLength(3);
        expect(reconstructedFormData.initialBookmarks![0].objectType).toBe("Resource");
        expect(reconstructedFormData.initialBookmarks![0].objectId).toBe("123456789012345678");
        
        // ✅ VERIFICATION: All bookmark data preserved
        expect(areBookmarkListFormsEqualReal(
            userFormData, 
            reconstructedFormData
        )).toBe(true);
    });

    test('bookmark list editing maintains data integrity', async () => {
        // Create initial bookmark list using REAL functions
        const initialFormData: BookmarkListFormData = {
            label: "Initial List",
        };
        const createRequest = transformFormToCreateRequestReal(initialFormData);
        const initialBookmarkList = await mockBookmarkListService.create(createRequest);
        
        // 🎨 STEP 1: User edits bookmark list
        const editFormData: Partial<BookmarkListFormData> = {
            label: "Updated List Name",
        };
        
        // 🔗 STEP 2: Transform edit to update request using REAL update logic
        const updateRequest = transformFormToUpdateRequestReal(initialBookmarkList.id, editFormData);
        expect(updateRequest.id).toBe(initialBookmarkList.id);
        expect(updateRequest.label).toBe(editFormData.label);
        
        // Add small delay to ensure different timestamps
        await new Promise(resolve => setTimeout(resolve, 10));
        
        // 🗄️ STEP 3: Update via API
        const updatedBookmarkList = await mockBookmarkListService.update(initialBookmarkList.id, updateRequest);
        expect(updatedBookmarkList.id).toBe(initialBookmarkList.id);
        expect(updatedBookmarkList.label).toBe(editFormData.label);
        
        // 🔗 STEP 4: Fetch updated bookmark list
        const fetchedUpdatedBookmarkList = await mockBookmarkListService.findById(initialBookmarkList.id);
        
        // ✅ VERIFICATION: Update preserved core data
        expect(fetchedUpdatedBookmarkList.id).toBe(initialBookmarkList.id);
        expect(fetchedUpdatedBookmarkList.label).toBe(editFormData.label);
        expect(fetchedUpdatedBookmarkList.createdAt).toBe(initialBookmarkList.createdAt); // Created date unchanged
        // Updated date should be different (new Date() creates different timestamps)
        expect(new Date(fetchedUpdatedBookmarkList.updatedAt).getTime()).toBeGreaterThan(
            new Date(initialBookmarkList.updatedAt).getTime()
        );
    });

    test('validation catches invalid form data before API submission', async () => {
        const invalidFormData: BookmarkListFormData = {
            label: "", // Empty label should fail validation
        };
        
        const validationErrors = await validateBookmarkListFormDataReal(invalidFormData);
        expect(validationErrors.length).toBeGreaterThan(0);
        
        // Should not proceed to API if validation fails
        expect(validationErrors.some(error => 
            error.includes("label") || error.includes("required")
        )).toBe(true);
    });

    test('bookmark list deletion works correctly', async () => {
        // Create bookmark list first using REAL functions
        const formData: BookmarkListFormData = {
            label: "Test List",
        };
        const createRequest = transformFormToCreateRequestReal(formData);
        const createdBookmarkList = await mockBookmarkListService.create(createRequest);
        
        // Delete it
        const deleteResult = await mockBookmarkListService.delete(createdBookmarkList.id);
        expect(deleteResult.success).toBe(true);
        
        // Verify it's deleted
        await expect(mockBookmarkListService.findById(createdBookmarkList.id)).rejects.toThrow();
    });

    test('data consistency across multiple operations', async () => {
        const originalFormData: BookmarkListFormData = {
            label: "Original List",
            initialBookmarks: [
                {
                    objectType: "Resource",
                    objectId: "123456789012345678",
                },
            ],
        };
        
        // Create using REAL functions
        const createRequest = transformFormToCreateRequestReal(originalFormData);
        const created = await mockBookmarkListService.create(createRequest);
        
        // Update using REAL functions
        const updateRequest = transformFormToUpdateRequestReal(created.id, { 
            label: "Updated List Name" 
        });
        const updated = await mockBookmarkListService.update(created.id, updateRequest);
        
        // Fetch final state
        const final = await mockBookmarkListService.findById(created.id);
        
        // Core bookmark list data should remain consistent
        expect(final.id).toBe(created.id);
        expect(final.label).toBe("Updated List Name");
        expect(final.bookmarks).toHaveLength(1);
        expect(final.bookmarks![0].to.id).toBe("123456789012345678");
        expect(final.bookmarksCount).toBe(1);
        
        // Timestamps should be properly managed
        expect(final.createdAt).toBe(created.createdAt); // Created date unchanged
        expect(new Date(final.updatedAt).getTime()).toBeGreaterThan(
            new Date(created.updatedAt).getTime()
        ); // Updated date changed
    });

    test('empty bookmark list creation and manipulation', async () => {
        // Create empty list
        const emptyFormData: BookmarkListFormData = {
            label: "Empty List",
        };
        
        const createRequest = transformFormToCreateRequestReal(emptyFormData);
        const created = await mockBookmarkListService.create(createRequest);
        
        // Verify empty state
        expect(created.bookmarks).toEqual([]);
        expect(created.bookmarksCount).toBe(0);
        
        // Fetch and verify
        const fetched = await mockBookmarkListService.findById(created.id);
        expect(fetched.bookmarks).toEqual([]);
        expect(fetched.bookmarksCount).toBe(0);
        
        // Verify form reconstruction
        const reconstructed = transformApiResponseToFormReal(fetched);
        expect(reconstructed.label).toBe(emptyFormData.label);
        expect(reconstructed.initialBookmarks).toEqual([]);
        
        // Verify round-trip
        expect(areBookmarkListFormsEqualReal(emptyFormData, reconstructed)).toBe(true);
    });

    test('bookmark list with various object types maintains type integrity', async () => {
        const mixedFormData: BookmarkListFormData = {
            label: "Mixed Content List",
            initialBookmarks: [
                { objectType: "Resource", objectId: "resource_123456789012345678" },
                { objectType: "User", objectId: "user_234567890123456789" },
                { objectType: "Team", objectId: "team_345678901234567890" },
                { objectType: "Tag", objectId: "tag_456789012345678901" },
            ],
        };
        
        // Validate and create
        const validationErrors = await validateBookmarkListFormDataReal(mixedFormData);
        expect(validationErrors).toHaveLength(0);
        
        const createRequest = transformFormToCreateRequestReal(mixedFormData);
        const created = await mockBookmarkListService.create(createRequest);
        
        // Verify all types preserved
        expect(created.bookmarks).toHaveLength(4);
        expect(created.bookmarks![0].to.__typename).toBe("Resource");
        expect(created.bookmarks![1].to.__typename).toBe("User");
        expect(created.bookmarks![2].to.__typename).toBe("Team");
        expect(created.bookmarks![3].to.__typename).toBe("Tag");
        
        // Fetch and verify types maintained
        const fetched = await mockBookmarkListService.findById(created.id);
        const reconstructed = transformApiResponseToFormReal(fetched);
        
        expect(reconstructed.initialBookmarks).toHaveLength(4);
        expect(reconstructed.initialBookmarks![0].objectType).toBe("Resource");
        expect(reconstructed.initialBookmarks![1].objectType).toBe("User");
        expect(reconstructed.initialBookmarks![2].objectType).toBe("Team");
        expect(reconstructed.initialBookmarks![3].objectType).toBe("Tag");
        
        // Verify complete round-trip
        expect(areBookmarkListFormsEqualReal(mixedFormData, reconstructed)).toBe(true);
    });
});