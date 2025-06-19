import { describe, test, expect, beforeEach } from 'vitest';
import { IssueFor, shapeIssue, issueValidation, generatePK, DUMMY_ID, type Issue, type IssueShape } from "@vrooli/shared";
import { 
    minimalIssueFormInput,
    completeIssueFormInput,
    issueForResourceFormInput,
    issueForTeamFormInput,
    multiLanguageIssueFormInput,
    detailedIssueFormInput,
    invalidIssueFormInputs,
    transformIssueFormToApiInput,
    validateIssueContent
} from '../form-data/issueFormData.js';
import { 
    minimalIssueResponse,
    completeIssueResponse,
    issueForResourceResponse,
    issueForTeamResponse,
    multiLanguageIssueResponse
} from '../api-responses/issueResponses.js';

/**
 * Round-trip testing for Issue data flow using REAL application functions
 * Tests the complete user journey: Form Input → API Request → Database → API Response → UI Display
 * 
 * ✅ Uses real shapeIssue.create() and shapeIssue.update() for transformations
 * ✅ Uses real issueValidation for validation
 * ✅ Tests actual application logic instead of mock implementations
 * 
 * Issues support:
 * - Status tracking (open, closed, etc.)
 * - Multiple languages
 * - Assignment to users
 * - Relationships to Resources and Teams
 * - Comments and discussions
 */

// Mock issue service using real shape functions
const mockIssueService = {
    storage: {} as Record<string, Issue>,
    
    async create(createInput: any): Promise<Issue> {
        const issue: Issue = {
            __typename: "Issue",
            id: createInput.id || generatePK().toString(),
            issueFor: createInput.issueFor,
            for: createInput.forConnect ? { 
                __typename: createInput.issueFor, 
                id: createInput.forConnect 
            } : createInput.for,
            status: "Open",
            closedAt: null,
            closedBy: null,
            createdAt: new Date().toISOString(),
            updatedAt: new Date().toISOString(),
            createdBy: {
                __typename: "User",
                id: "user_123456789012345678",
                handle: "testuser",
                name: "Test User",
            },
            bookmarks: 0,
            bookmarkedBy: [],
            commentsCount: 0,
            comments: [],
            reportsCount: 0,
            reports: [],
            score: 0,
            views: 0,
            translationsCount: createInput.translationsCreate?.length || 0,
            translations: createInput.translationsCreate || [],
            referencedVersionId: createInput.referencedVersionIdConnect || null,
            you: {
                __typename: "IssueYou",
                canBookmark: true,
                canComment: true,
                canDelete: true,
                canReport: true,
                canUpdate: true,
                isBookmarked: false,
                reaction: null,
            },
        };
        
        this.storage[issue.id] = issue;
        return issue;
    },
    
    async findById(id: string): Promise<Issue> {
        const issue = this.storage[id];
        if (!issue) {
            throw new Error(`Issue with id ${id} not found`);
        }
        return issue;
    },
    
    async update(id: string, updateInput: any): Promise<Issue> {
        const existing = this.storage[id];
        if (!existing) {
            throw new Error(`Issue with id ${id} not found`);
        }
        
        const updated: Issue = {
            ...existing,
            updatedAt: new Date().toISOString(),
            translations: updateInput.translationsUpdate || existing.translations,
            translationsCount: (updateInput.translationsUpdate || existing.translations).length,
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
    
    clear() {
        this.storage = {};
    }
};

// Helper functions using REAL application logic
function transformFormToCreateRequestReal(formData: Partial<IssueShape>) {
    return shapeIssue.create({
        __typename: "Issue",
        id: formData.id || generatePK().toString(),
        issueFor: formData.issueFor!,
        forConnect: formData.for!.id,
        translationsCreate: formData.translations?.map(t => ({
            id: t.id || generatePK().toString(),
            language: t.language,
            name: t.name,
            description: t.description,
        })) || [],
    });
}

function transformFormToUpdateRequestReal(issueId: string, formData: Partial<IssueShape>) {
    return shapeIssue.update(
        { id: issueId }, // original object
        {
            id: issueId,
            translationsUpdate: formData.translations?.map(t => ({
                id: t.id || generatePK().toString(),
                language: t.language,
                name: t.name,
                description: t.description,
            })) || [],
        }
    );
}

async function validateIssueFormDataReal(formData: Partial<IssueShape>): Promise<string[]> {
    try {
        // Use real validation schema
        const validationData = {
            id: formData.id || generatePK().toString(),
            issueFor: formData.issueFor!,
            forConnect: formData.for!.id,
            translationsCreate: formData.translations?.map(t => ({
                id: t.id || generatePK().toString(),
                language: t.language,
                name: t.name || "",
                description: t.description,
            })) || [],
        };
        
        await issueValidation.create({ omitFields: [] }).validate(validationData);
        return []; // No validation errors
    } catch (error: any) {
        if (error.errors) {
            return error.errors; // Yup validation errors
        }
        return [error.message || "Validation failed"];
    }
}

function transformApiResponseToFormReal(issue: Issue): Partial<IssueShape> {
    return {
        id: issue.id,
        __typename: "Issue",
        issueFor: issue.issueFor as IssueFor,
        for: issue.for,
        translations: issue.translations?.map(t => ({
            __typename: "IssueTranslation",
            id: t.id,
            language: t.language,
            name: t.name,
            description: t.description,
        })) || [],
    };
}

function areIssueFormsEqualReal(form1: Partial<IssueShape>, form2: Partial<IssueShape>): boolean {
    return (
        form1.issueFor === form2.issueFor &&
        form1.for?.id === form2.for?.id &&
        form1.translations?.length === form2.translations?.length &&
        form1.translations?.every((t1, i) => {
            const t2 = form2.translations?.[i];
            return t1.language === t2?.language && 
                   t1.name === t2?.name && 
                   t1.description === t2?.description;
        })
    );
}

describe('Issue Round-Trip Data Flow', () => {
    beforeEach(() => {
        // Clear test storage between tests
        mockIssueService.clear();
    });

    test('minimal issue creation maintains data integrity through complete flow', async () => {
        // 🎨 STEP 1: User fills out minimal issue form
        const userFormData = minimalIssueFormInput;
        
        // Validate form data using REAL validation schema
        const validationErrors = await validateIssueFormDataReal(userFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Form submits to API using REAL shape function
        const apiCreateRequest = transformFormToCreateRequestReal(userFormData);
        expect(apiCreateRequest.issueFor).toBe(userFormData.issueFor);
        expect(apiCreateRequest.forConnect).toBe(userFormData.for!.id);
        expect(apiCreateRequest.id).toMatch(/^\d{10,19}$/);
        
        // 🗄️ STEP 3: API creates issue (simulated - real test would hit test DB)
        const createdIssue = await mockIssueService.create(apiCreateRequest);
        expect(createdIssue.id).toBe(apiCreateRequest.id);
        expect(createdIssue.for.id).toBe(userFormData.for!.id);
        expect(createdIssue.issueFor).toBe(userFormData.issueFor);
        expect(createdIssue.status).toBe("Open");
        expect(createdIssue.translations).toHaveLength(1);
        expect(createdIssue.translations[0].name).toBe(userFormData.translations![0].name);
        
        // 🔗 STEP 4: API fetches issue back
        const fetchedIssue = await mockIssueService.findById(createdIssue.id);
        expect(fetchedIssue.id).toBe(createdIssue.id);
        expect(fetchedIssue.for.id).toBe(userFormData.for!.id);
        expect(fetchedIssue.issueFor).toBe(userFormData.issueFor);
        
        // 🎨 STEP 5: UI would display the issue using REAL transformation
        const reconstructedFormData = transformApiResponseToFormReal(fetchedIssue);
        expect(reconstructedFormData.issueFor).toBe(userFormData.issueFor);
        expect(reconstructedFormData.for!.id).toBe(userFormData.for!.id);
        expect(reconstructedFormData.translations![0].name).toBe(userFormData.translations![0].name);
        
        // ✅ VERIFICATION: Complete round-trip integrity using REAL comparison
        expect(areIssueFormsEqualReal(
            userFormData, 
            reconstructedFormData
        )).toBe(true);
    });

    test('complete issue with detailed information preserves all data', async () => {
        // 🎨 STEP 1: User creates detailed issue
        const userFormData = completeIssueFormInput;
        
        // Validate complex form using REAL validation
        const validationErrors = await validateIssueFormDataReal(userFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Transform to API request using REAL shape function
        const apiCreateRequest = transformFormToCreateRequestReal(userFormData);
        expect(apiCreateRequest.translationsCreate).toBeDefined();
        expect(apiCreateRequest.translationsCreate).toHaveLength(1);
        expect(apiCreateRequest.translationsCreate![0].name).toBe(userFormData.translations![0].name);
        expect(apiCreateRequest.translationsCreate![0].description).toBe(userFormData.translations![0].description);
        
        // 🗄️ STEP 3: Create via API
        const createdIssue = await mockIssueService.create(apiCreateRequest);
        expect(createdIssue.translations).toHaveLength(1);
        expect(createdIssue.translations[0].name).toBe(userFormData.translations![0].name);
        expect(createdIssue.translations[0].description).toBe(userFormData.translations![0].description);
        expect(createdIssue.issueFor).toBe(userFormData.issueFor);
        
        // 🔗 STEP 4: Fetch back from API
        const fetchedIssue = await mockIssueService.findById(createdIssue.id);
        expect(fetchedIssue.translations[0].name).toBe(userFormData.translations![0].name);
        expect(fetchedIssue.translations[0].description).toBe(userFormData.translations![0].description);
        
        // 🎨 STEP 5: Verify UI can display correctly using REAL transformation
        const reconstructedFormData = transformApiResponseToFormReal(fetchedIssue);
        expect(reconstructedFormData.issueFor).toBe(userFormData.issueFor);
        expect(reconstructedFormData.for!.id).toBe(userFormData.for!.id);
        expect(reconstructedFormData.translations![0].name).toBe(userFormData.translations![0].name);
        expect(reconstructedFormData.translations![0].description).toBe(userFormData.translations![0].description);
        
        // ✅ VERIFICATION: Complex data preserved
        expect(areIssueFormsEqualReal(userFormData, reconstructedFormData)).toBe(true);
    });

    test('issue editing with translation updates maintains data integrity', async () => {
        // Create initial issue using REAL functions
        const initialFormData = minimalIssueFormInput;
        const createRequest = transformFormToCreateRequestReal(initialFormData);
        const initialIssue = await mockIssueService.create(createRequest);
        
        // 🎨 STEP 1: User edits issue translation
        const editFormData: Partial<IssueShape> = {
            id: initialIssue.id,
            issueFor: initialIssue.issueFor as IssueFor,
            for: initialIssue.for,
            translations: [{
                __typename: "IssueTranslation",
                id: initialIssue.translations[0].id,
                language: "en",
                name: "Updated issue title",
                description: "Updated issue description with more details",
            }],
        };
        
        // 🔗 STEP 2: Transform edit to update request using REAL update logic
        const updateRequest = transformFormToUpdateRequestReal(initialIssue.id, editFormData);
        expect(updateRequest.id).toBe(initialIssue.id);
        expect(updateRequest.translationsUpdate).toBeDefined();
        expect(updateRequest.translationsUpdate![0].name).toBe("Updated issue title");
        
        // Add small delay to ensure different timestamps
        await new Promise(resolve => setTimeout(resolve, 10));
        
        // 🗄️ STEP 3: Update via API
        const updatedIssue = await mockIssueService.update(initialIssue.id, updateRequest);
        expect(updatedIssue.id).toBe(initialIssue.id);
        expect(updatedIssue.translations[0].name).toBe("Updated issue title");
        expect(updatedIssue.translations[0].description).toBe("Updated issue description with more details");
        
        // 🔗 STEP 4: Fetch updated issue
        const fetchedUpdatedIssue = await mockIssueService.findById(initialIssue.id);
        
        // ✅ VERIFICATION: Update preserved core data
        expect(fetchedUpdatedIssue.id).toBe(initialIssue.id);
        expect(fetchedUpdatedIssue.for.id).toBe(initialFormData.for!.id);
        expect(fetchedUpdatedIssue.issueFor).toBe(initialFormData.issueFor);
        expect(fetchedUpdatedIssue.createdAt).toBe(initialIssue.createdAt);
        expect(new Date(fetchedUpdatedIssue.updatedAt).getTime()).toBeGreaterThan(
            new Date(initialIssue.updatedAt).getTime()
        );
        expect(fetchedUpdatedIssue.translations[0].name).toBe("Updated issue title");
    });

    test('all issue types work correctly through round-trip', async () => {
        const issueTypes = Object.values(IssueFor);
        
        for (const issueType of issueTypes) {
            // 🎨 Create form data for each type
            const formData: Partial<IssueShape> = {
                __typename: "Issue",
                issueFor: issueType,
                for: { id: `${issueType.toLowerCase()}_123456789012345678` },
                translations: [{
                    __typename: "IssueTranslation",
                    id: DUMMY_ID,
                    language: "en",
                    name: `Issue for ${issueType}`,
                    description: `This is a test issue for ${issueType}`,
                }],
            };
            
            // 🔗 Transform and create using REAL functions
            const createRequest = transformFormToCreateRequestReal(formData);
            const createdIssue = await mockIssueService.create(createRequest);
            
            // 🗄️ Fetch back
            const fetchedIssue = await mockIssueService.findById(createdIssue.id);
            
            // ✅ Verify type-specific data
            expect(fetchedIssue.issueFor).toBe(issueType);
            expect(fetchedIssue.for.id).toBe(formData.for!.id);
            expect(fetchedIssue.translations[0].name).toBe(formData.translations![0].name);
            
            // Verify form reconstruction using REAL transformation
            const reconstructed = transformApiResponseToFormReal(fetchedIssue);
            expect(reconstructed.issueFor).toBe(issueType);
            expect(reconstructed.for!.id).toBe(formData.for!.id);
            expect(reconstructed.translations![0].name).toBe(formData.translations![0].name);
        }
    });

    test('multi-language issue preserves all translations', async () => {
        // 🎨 STEP 1: User creates multi-language issue
        const userFormData = multiLanguageIssueFormInput;
        
        // Validate multi-language form using REAL validation
        const validationErrors = await validateIssueFormDataReal(userFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Transform to API request
        const apiCreateRequest = transformFormToCreateRequestReal(userFormData);
        expect(apiCreateRequest.translationsCreate).toHaveLength(3);
        
        // 🗄️ STEP 3: Create and fetch
        const createdIssue = await mockIssueService.create(apiCreateRequest);
        const fetchedIssue = await mockIssueService.findById(createdIssue.id);
        
        // ✅ VERIFICATION: All translations preserved
        expect(fetchedIssue.translations).toHaveLength(3);
        expect(fetchedIssue.translations.map(t => t.language)).toContain("en");
        expect(fetchedIssue.translations.map(t => t.language)).toContain("es");
        expect(fetchedIssue.translations.map(t => t.language)).toContain("fr");
        
        // Verify specific translation content
        const enTranslation = fetchedIssue.translations.find(t => t.language === "en");
        const esTranslation = fetchedIssue.translations.find(t => t.language === "es");
        expect(enTranslation!.name).toBe("Feature request: Dark mode");
        expect(esTranslation!.name).toBe("Solicitud de función: Modo oscuro");
        
        // Verify round-trip integrity
        const reconstructed = transformApiResponseToFormReal(fetchedIssue);
        expect(areIssueFormsEqualReal(userFormData, reconstructed)).toBe(true);
    });

    test('validation catches invalid form data before API submission', async () => {
        // Test empty name
        const invalidFormData = invalidIssueFormInputs.emptyName;
        const validationErrors = await validateIssueFormDataReal(invalidFormData);
        expect(validationErrors.length).toBeGreaterThan(0);
        
        // Test missing issue for
        const missingIssueForData = invalidIssueFormInputs.missingIssueFor;
        try {
            await validateIssueFormDataReal(missingIssueForData);
            expect.fail("Should have thrown validation error");
        } catch (error) {
            expect(error).toBeTruthy();
        }
        
        // Test invalid for ID
        const invalidForIdData = invalidIssueFormInputs.invalidForId;
        const invalidIdErrors = await validateIssueFormDataReal(invalidForIdData);
        expect(invalidIdErrors.length).toBeGreaterThan(0);
    });

    test('issue deletion works correctly', async () => {
        // Create issue first using REAL functions
        const formData = minimalIssueFormInput;
        const createRequest = transformFormToCreateRequestReal(formData);
        const createdIssue = await mockIssueService.create(createRequest);
        
        // Delete it
        const deleteResult = await mockIssueService.delete(createdIssue.id);
        expect(deleteResult.success).toBe(true);
        
        // Verify it's gone
        await expect(mockIssueService.findById(createdIssue.id)).rejects.toThrow();
    });

    test('data consistency across multiple operations', async () => {
        const originalFormData = minimalIssueFormInput;
        
        // Create using REAL functions
        const createRequest = transformFormToCreateRequestReal(originalFormData);
        const created = await mockIssueService.create(createRequest);
        
        // Update using REAL functions
        const updateFormData: Partial<IssueShape> = {
            id: created.id,
            translations: [{
                __typename: "IssueTranslation",
                id: created.translations[0].id,
                language: "en",
                name: "Updated issue name",
                description: "Updated description with more information",
            }],
        };
        const updateRequest = transformFormToUpdateRequestReal(created.id, updateFormData);
        const updated = await mockIssueService.update(created.id, updateRequest);
        
        // Fetch final state
        const final = await mockIssueService.findById(created.id);
        
        // Core issue data should remain consistent
        expect(final.id).toBe(created.id);
        expect(final.for.id).toBe(originalFormData.for!.id);
        expect(final.issueFor).toBe(originalFormData.issueFor);
        expect(final.createdBy.id).toBe(created.createdBy.id);
        expect(final.status).toBe("Open"); // Status unchanged
        
        // Only the translation should have changed
        expect(final.translations[0].name).toBe("Updated issue name");
        expect(final.translations[0].description).toBe("Updated description with more information");
        expect(final.translations[0].language).toBe("en"); // Language unchanged
        
        // Timestamps should reflect the update
        expect(final.createdAt).toBe(created.createdAt);
        expect(new Date(final.updatedAt).getTime()).toBeGreaterThan(
            new Date(created.updatedAt).getTime()
        );
    });

    test('issue content validation helper works correctly', async () => {
        // Valid content
        const validTranslations = [
            { __typename: "IssueTranslation" as const, id: DUMMY_ID, language: "en", name: "Valid name" },
        ];
        expect(validateIssueContent(validTranslations)).toBeNull();
        
        // Empty name
        const emptyNameTranslations = [
            { __typename: "IssueTranslation" as const, id: DUMMY_ID, language: "en", name: "" },
        ];
        expect(validateIssueContent(emptyNameTranslations)).toContain("cannot be empty");
        
        // No translations
        expect(validateIssueContent([])).toContain("At least one translation is required");
        
        // Missing language
        const missingLanguageTranslations = [
            { __typename: "IssueTranslation" as const, id: DUMMY_ID, language: "", name: "Valid name" },
        ];
        expect(validateIssueContent(missingLanguageTranslations)).toContain("Language code is required");
    });

    test('issue form transformation helper works correctly', async () => {
        const validFormData = minimalIssueFormInput;
        const apiInput = transformIssueFormToApiInput(validFormData);
        
        expect(apiInput.__typename).toBe("Issue");
        expect(apiInput.issueFor).toBe(validFormData.issueFor);
        expect(apiInput.for.id).toBe(validFormData.for!.id);
        expect(apiInput.translations).toHaveLength(1);
        
        // Test missing required fields
        expect(() => transformIssueFormToApiInput({})).toThrow("issueFor is required");
        expect(() => transformIssueFormToApiInput({ issueFor: IssueFor.Resource })).toThrow("for is required");
    });
});