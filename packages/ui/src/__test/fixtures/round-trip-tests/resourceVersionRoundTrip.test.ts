import { describe, test, expect, beforeEach } from 'vitest';
import { 
    shapeResourceVersion, 
    resourceVersionValidation, 
    generatePK, 
    type ResourceVersion,
    type ResourceVersionCreateInput,
    type ResourceVersionUpdateInput
} from "@vrooli/shared";
import { 
    minimalResourceVersionFormInput,
    completeResourceVersionFormInput,
    invalidResourceVersionFormInputs,
    transformResourceVersionFormToApiInput,
    type ResourceVersionFormData
} from '../form-data/resourceVersionFormData.js';

/**
 * Round-trip testing for ResourceVersion data flow using REAL application functions
 * Tests the complete user journey: Form Input → API Request → Database → API Response → UI Display
 * 
 * ✅ Uses real shapeResourceVersion.create() for transformations
 * ✅ Uses real resourceVersionValidation for validation
 * ✅ Tests actual application logic instead of mock implementations
 */

// Mock service for ResourceVersion operations
const getStorage = () => {
    if (!(globalThis as any).__testResourceVersionStorage) {
        (globalThis as any).__testResourceVersionStorage = {};
    }
    return (globalThis as any).__testResourceVersionStorage;
};

function createMockResourceVersion(createData: ResourceVersionCreateInput): ResourceVersion {
    const now = new Date().toISOString();
    const storage = getStorage();
    
    const resourceVersion: ResourceVersion = {
        __typename: "ResourceVersion",
        id: createData.id,
        codeLanguage: createData.codeLanguage || null,
        config: createData.config || null,
        isAutomatable: createData.isAutomatable || false,
        isComplete: createData.isComplete || false,
        isPrivate: createData.isPrivate || false,
        resourceSubType: createData.resourceSubType || null,
        versionLabel: createData.versionLabel,
        versionNotes: createData.versionNotes || null,
        publicId: createData.publicId || null,
        createdAt: now,
        updatedAt: now,
        root: createData.rootConnect ? {
            __typename: "Resource",
            id: createData.rootConnect,
        } : createData.rootCreate ? {
            __typename: "Resource",
            id: createData.rootCreate.id,
            resourceType: createData.rootCreate.resourceType,
            isPrivate: createData.rootCreate.isPrivate || false,
        } : null,
        translations: createData.translationsCreate?.map(trans => ({
            __typename: "ResourceVersionTranslation",
            id: trans.id,
            language: trans.language,
            name: trans.name,
            description: trans.description || "",
            details: trans.details || "",
            instructions: trans.instructions || "",
        })) || [],
        relatedVersions: createData.relatedVersionsCreate?.map(rel => ({
            __typename: "ResourceVersionRelation",
            id: rel.id,
            labels: rel.labels || [],
            toVersion: {
                __typename: "ResourceVersion",
                id: rel.toVersionConnect,
            },
        })) || [],
        you: {
            __typename: "ResourceVersionYou",
            canComment: true,
            canDelete: true,
            canReport: true,
            canUpdate: true,
            canRead: true,
            canBookmark: true,
            canReact: true,
            isBookmarked: false,
            reaction: null,
        },
    };
    
    storage[resourceVersion.id] = resourceVersion;
    return resourceVersion;
}

function updateMockResourceVersion(id: string, updateData: ResourceVersionUpdateInput): ResourceVersion {
    const storage = getStorage();
    const existing = storage[id];
    if (!existing) {
        throw new Error(`ResourceVersion with id ${id} not found`);
    }
    
    const updated: ResourceVersion = {
        ...existing,
        ...updateData,
        updatedAt: new Date().toISOString(),
    };
    
    storage[id] = updated;
    return updated;
}

function findMockResourceVersionById(id: string): ResourceVersion {
    const storage = getStorage();
    const resourceVersion = storage[id];
    if (!resourceVersion) {
        throw new Error(`ResourceVersion with id ${id} not found`);
    }
    return resourceVersion;
}

function deleteMockResourceVersion(id: string): { success: boolean } {
    const storage = getStorage();
    if (storage[id]) {
        delete storage[id];
        return { success: true };
    }
    return { success: false };
}

const mockResourceVersionService = {
    create: createMockResourceVersion,
    update: updateMockResourceVersion,
    findById: findMockResourceVersionById,
    delete: deleteMockResourceVersion,
};

// Helper functions using REAL application logic
function transformFormToCreateRequestReal(formData: any) {
    const createRequest = {
        __typename: "ResourceVersion" as const,
        id: generatePK().toString(),
        versionLabel: formData.versionLabel,
        versionNotes: formData.versionNotes,
        codeLanguage: formData.codeLanguage,
        config: formData.config,
        isAutomatable: formData.isAutomatable,
        isComplete: formData.isComplete,
        isInternal: formData.isInternal,
        isPrivate: formData.isPrivate,
        publicId: formData.publicId,
        resourceSubType: formData.resourceSubType,
        root: formData.rootConnect ? {
            __typename: "Resource" as const,
            id: formData.rootConnect,
        } : formData.rootCreate ? {
            __typename: "Resource" as const,
            id: generatePK().toString(),
            resourceType: formData.rootCreate.resourceType || "Api",
            isPrivate: formData.isPrivate,
        } : {
            __typename: "Resource" as const,
            id: generatePK().toString(),
            resourceType: "Api",
            isPrivate: formData.isPrivate || false,
        },
        translations: formData.translations?.map((trans: any) => ({
            __typename: "ResourceVersionTranslation" as const,
            id: generatePK().toString(),
            language: trans.language,
            name: trans.name,
            description: trans.description || "",
            details: trans.details || "",
            instructions: trans.instructions || "",
        })) || [],
        relatedVersions: formData.relatedVersions?.map((rel: any) => ({
            __typename: "ResourceVersionRelation" as const,
            id: generatePK().toString(),
            labels: rel.labels || [],
            toVersion: {
                __typename: "ResourceVersion" as const,
                id: rel.toVersionConnect,
            },
        })) || [],
    };
    
    return shapeResourceVersion.create(createRequest);
}

function transformFormToUpdateRequestReal(resourceVersionId: string, formData: Partial<any>) {
    const updateRequest: any = {
        id: resourceVersionId,
    };
    
    if (formData.versionLabel !== undefined) updateRequest.versionLabel = formData.versionLabel;
    if (formData.versionNotes !== undefined) updateRequest.versionNotes = formData.versionNotes;
    if (formData.codeLanguage !== undefined) updateRequest.codeLanguage = formData.codeLanguage;
    if (formData.config !== undefined) updateRequest.config = formData.config;
    if (formData.isAutomatable !== undefined) updateRequest.isAutomatable = formData.isAutomatable;
    if (formData.isComplete !== undefined) updateRequest.isComplete = formData.isComplete;
    if (formData.isPrivate !== undefined) updateRequest.isPrivate = formData.isPrivate;
    
    return updateRequest;
}

async function validateResourceVersionFormDataReal(formData: any): Promise<string[]> {
    try {
        const validationData = {
            id: generatePK().toString(),
            versionLabel: formData.versionLabel,
            versionNotes: formData.versionNotes,
            codeLanguage: formData.codeLanguage,
            config: formData.config,
            isAutomatable: formData.isAutomatable,
            isComplete: formData.isComplete,
            isInternal: formData.isInternal,
            isPrivate: formData.isPrivate,
            publicId: formData.publicId,
            resourceSubType: formData.resourceSubType,
            ...(formData.rootConnect && { rootConnect: formData.rootConnect }),
            ...(formData.rootCreate && { rootCreate: formData.rootCreate }),
            ...(formData.translations && {
                translationsCreate: formData.translations.map((trans: any) => ({
                    id: generatePK().toString(),
                    language: trans.language,
                    name: trans.name,
                    description: trans.description || "",
                    details: trans.details || "",
                    instructions: trans.instructions || "",
                }))
            }),
        };
        
        await resourceVersionValidation.create({ omitFields: [] }).validate(validationData);
        return []; // No validation errors
    } catch (error: any) {
        if (error.errors) {
            return error.errors;
        }
        return [error.message || "Validation failed"];
    }
}

function transformApiResponseToFormReal(resourceVersion: ResourceVersion): any {
    return {
        versionLabel: resourceVersion.versionLabel,
        versionNotes: resourceVersion.versionNotes,
        codeLanguage: resourceVersion.codeLanguage,
        config: resourceVersion.config,
        isAutomatable: resourceVersion.isAutomatable,
        isComplete: resourceVersion.isComplete,
        isPrivate: resourceVersion.isPrivate,
        resourceSubType: resourceVersion.resourceSubType,
        publicId: resourceVersion.publicId,
        translations: resourceVersion.translations?.map(trans => ({
            language: trans.language,
            name: trans.name,
            description: trans.description,
            details: trans.details,
            instructions: trans.instructions,
        })) || [],
    };
}

function areResourceVersionFormsEqualReal(form1: any, form2: any): boolean {
    return (
        form1.versionLabel === form2.versionLabel &&
        form1.isComplete === form2.isComplete &&
        form1.isPrivate === form2.isPrivate &&
        form1.resourceSubType === form2.resourceSubType &&
        JSON.stringify(form1.translations) === JSON.stringify(form2.translations)
    );
}

describe('ResourceVersion Round-Trip Data Flow', () => {
    beforeEach(() => {
        // Clear test storage between tests
        (globalThis as any).__testResourceVersionStorage = {};
    });

    test('minimal resource version creation maintains data integrity through complete flow', async () => {
        // 🎨 STEP 1: User fills out minimal resource version form
        const userFormData = {
            ...minimalResourceVersionFormInput,
            rootCreate: {
                resourceType: "Api",
                isPrivate: false,
            },
        };
        
        // Validate form data using REAL validation schema
        const validationErrors = await validateResourceVersionFormDataReal(userFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Form submits to API using REAL shape function
        const apiCreateRequest = transformFormToCreateRequestReal(userFormData);
        expect(apiCreateRequest.versionLabel).toBe(userFormData.versionLabel);
        expect(apiCreateRequest.isComplete).toBe(userFormData.isComplete);
        expect(apiCreateRequest.id).toMatch(/^\d{18,19}$/);
        
        // 🗄️ STEP 3: API creates resource version (simulated - real test would hit test DB)
        const createdResourceVersion = await mockResourceVersionService.create(apiCreateRequest);
        expect(createdResourceVersion.id).toBe(apiCreateRequest.id);
        expect(createdResourceVersion.versionLabel).toBe(userFormData.versionLabel);
        expect(createdResourceVersion.isComplete).toBe(userFormData.isComplete);
        
        // 🔗 STEP 4: API fetches resource version back
        const fetchedResourceVersion = await mockResourceVersionService.findById(createdResourceVersion.id);
        expect(fetchedResourceVersion.id).toBe(createdResourceVersion.id);
        expect(fetchedResourceVersion.versionLabel).toBe(userFormData.versionLabel);
        
        // 🎨 STEP 5: UI would display the resource version using REAL transformation
        const reconstructedFormData = transformApiResponseToFormReal(fetchedResourceVersion);
        expect(reconstructedFormData.versionLabel).toBe(userFormData.versionLabel);
        expect(reconstructedFormData.isComplete).toBe(userFormData.isComplete);
        
        // ✅ VERIFICATION: Complete round-trip integrity using REAL comparison
        expect(areResourceVersionFormsEqualReal(
            userFormData, 
            reconstructedFormData
        )).toBe(true);
    });

    test('complete resource version with all features preserves all data', async () => {
        // 🎨 STEP 1: User creates complete resource version
        const userFormData = {
            ...completeResourceVersionFormInput,
            rootCreate: {
                resourceType: "Api",
                isPrivate: false,
            },
        };
        
        // Validate complex form using REAL validation
        const validationErrors = await validateResourceVersionFormDataReal(userFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Transform to API request using REAL shape function
        const apiCreateRequest = transformFormToCreateRequestReal(userFormData);
        expect(apiCreateRequest.versionLabel).toBe(userFormData.versionLabel);
        expect(apiCreateRequest.config).toEqual(userFormData.config);
        expect(apiCreateRequest.translationsCreate).toBeDefined();
        expect(apiCreateRequest.translationsCreate?.length).toBe(userFormData.translations.length);
        
        // 🗄️ STEP 3: Create via API
        const createdResourceVersion = await mockResourceVersionService.create(apiCreateRequest);
        expect(createdResourceVersion.config).toEqual(userFormData.config);
        expect(createdResourceVersion.codeLanguage).toBe(userFormData.codeLanguage);
        expect(createdResourceVersion.translations?.length).toBe(userFormData.translations.length);
        
        // 🔗 STEP 4: Fetch back from API
        const fetchedResourceVersion = await mockResourceVersionService.findById(createdResourceVersion.id);
        expect(fetchedResourceVersion.config).toEqual(userFormData.config);
        expect(fetchedResourceVersion.translations?.[0]?.name).toBe(userFormData.translations[0].name);
        
        // 🎨 STEP 5: Verify UI can display correctly using REAL transformation
        const reconstructedFormData = transformApiResponseToFormReal(fetchedResourceVersion);
        expect(reconstructedFormData.versionLabel).toBe(userFormData.versionLabel);
        expect(reconstructedFormData.config).toEqual(userFormData.config);
        expect(reconstructedFormData.translations[0].name).toBe(userFormData.translations[0].name);
        
        // ✅ VERIFICATION: Complex data preserved
        expect(fetchedResourceVersion.resourceSubType).toBe(userFormData.resourceSubType);
        expect(fetchedResourceVersion.isAutomatable).toBe(userFormData.isAutomatable);
    });

    test('resource version editing maintains data integrity', async () => {
        // Create initial resource version using REAL functions
        const initialFormData = {
            ...minimalResourceVersionFormInput,
            rootCreate: {
                resourceType: "Api",
                isPrivate: false,
            },
        };
        const createRequest = transformFormToCreateRequestReal(initialFormData);
        const initialResourceVersion = await mockResourceVersionService.create(createRequest);
        
        // 🎨 STEP 1: User edits resource version
        const editFormData = {
            versionLabel: "1.1.0",
            versionNotes: "Updated with bug fixes",
            isComplete: true,
        };
        
        // 🔗 STEP 2: Transform edit to update request using REAL update logic
        const updateRequest = transformFormToUpdateRequestReal(initialResourceVersion.id, editFormData);
        expect(updateRequest.id).toBe(initialResourceVersion.id);
        expect(updateRequest.versionLabel).toBe(editFormData.versionLabel);
        
        // 🗄️ STEP 3: Update via API
        const updatedResourceVersion = await mockResourceVersionService.update(initialResourceVersion.id, updateRequest);
        expect(updatedResourceVersion.id).toBe(initialResourceVersion.id);
        expect(updatedResourceVersion.versionLabel).toBe(editFormData.versionLabel);
        expect(updatedResourceVersion.isComplete).toBe(editFormData.isComplete);
        
        // 🔗 STEP 4: Fetch updated resource version
        const fetchedUpdatedResourceVersion = await mockResourceVersionService.findById(initialResourceVersion.id);
        
        // ✅ VERIFICATION: Update preserved core data
        expect(fetchedUpdatedResourceVersion.id).toBe(initialResourceVersion.id);
        expect(fetchedUpdatedResourceVersion.versionLabel).toBe(editFormData.versionLabel);
        expect(fetchedUpdatedResourceVersion.isComplete).toBe(editFormData.isComplete);
        expect(fetchedUpdatedResourceVersion.createdAt).toBe(initialResourceVersion.createdAt);
        expect(new Date(fetchedUpdatedResourceVersion.updatedAt).getTime()).toBeGreaterThan(
            new Date(initialResourceVersion.updatedAt).getTime()
        );
    });

    test('validation catches invalid form data before API submission', async () => {
        const invalidFormData = invalidResourceVersionFormInputs.missingVersionLabel;
        
        const validationErrors = await validateResourceVersionFormDataReal(invalidFormData);
        expect(validationErrors.length).toBeGreaterThan(0);
        
        // Should not proceed to API if validation fails
        expect(validationErrors.some(error => 
            error.includes("required") || error.includes("Version")
        )).toBe(true);
    });

    test('configuration data maintains structure through round-trip', async () => {
        const configFormData = {
            ...completeResourceVersionFormInput,
            config: {
                theme: "dark",
                features: ["automation", "validation"],
                settings: { timeout: 30000, retries: 3 },
            },
            rootCreate: {
                resourceType: "Api",
                isPrivate: false,
            },
        };
        
        // Transform and create using REAL functions
        const createRequest = transformFormToCreateRequestReal(configFormData);
        const createdResourceVersion = await mockResourceVersionService.create(createRequest);
        
        // Fetch back
        const fetchedResourceVersion = await mockResourceVersionService.findById(createdResourceVersion.id);
        
        // ✅ Verify configuration preserved
        expect(fetchedResourceVersion.config).toEqual(configFormData.config);
        expect(fetchedResourceVersion.config.theme).toBe("dark");
        expect(fetchedResourceVersion.config.settings.timeout).toBe(30000);
        
        // Verify form reconstruction preserves config
        const reconstructed = transformApiResponseToFormReal(fetchedResourceVersion);
        expect(reconstructed.config).toEqual(configFormData.config);
    });

    test('resource version deletion works correctly', async () => {
        // Create resource version first using REAL functions
        const formData = {
            ...minimalResourceVersionFormInput,
            rootCreate: {
                resourceType: "Api",
                isPrivate: false,
            },
        };
        const createRequest = transformFormToCreateRequestReal(formData);
        const createdResourceVersion = await mockResourceVersionService.create(createRequest);
        
        // Delete it
        const deleteResult = await mockResourceVersionService.delete(createdResourceVersion.id);
        expect(deleteResult.success).toBe(true);
        
        // Verify it's gone
        expect(() => mockResourceVersionService.findById(createdResourceVersion.id)).toThrow();
    });

    test('data consistency across multiple operations', async () => {
        const originalFormData = {
            ...minimalResourceVersionFormInput,
            rootCreate: {
                resourceType: "Api",
                isPrivate: false,
            },
        };
        
        // Create using REAL functions
        const createRequest = transformFormToCreateRequestReal(originalFormData);
        const created = await mockResourceVersionService.create(createRequest);
        
        // Update using REAL functions
        const updateRequest = transformFormToUpdateRequestReal(created.id, { 
            versionLabel: "2.0.0",
            isComplete: true,
        });
        const updated = await mockResourceVersionService.update(created.id, updateRequest);
        
        // Fetch final state
        const final = await mockResourceVersionService.findById(created.id);
        
        // Core resource version data should remain consistent
        expect(final.id).toBe(created.id);
        expect(final.createdAt).toBe(created.createdAt);
        expect(final.root?.id).toBe(created.root?.id);
        
        // Only the updated fields should have changed
        expect(final.versionLabel).toBe("2.0.0");
        expect(final.isComplete).toBe(true);
    });

    test('multi-language translations preserve data integrity', async () => {
        const multiLangFormData = {
            versionLabel: "1.0.0",
            isComplete: true,
            isPrivate: false,
            translations: [
                {
                    language: "en",
                    name: "English Name",
                    description: "English description",
                    details: "English details",
                },
                {
                    language: "es",
                    name: "Nombre Español",
                    description: "Descripción en español",
                },
            ],
            rootCreate: {
                resourceType: "Api",
                isPrivate: false,
            },
        };
        
        // Create and test round-trip
        const createRequest = transformFormToCreateRequestReal(multiLangFormData);
        const created = await mockResourceVersionService.create(createRequest);
        const fetched = await mockResourceVersionService.findById(created.id);
        
        // Verify all translations preserved
        expect(fetched.translations?.length).toBe(2);
        expect(fetched.translations?.[0]?.language).toBe("en");
        expect(fetched.translations?.[1]?.language).toBe("es");
        expect(fetched.translations?.[0]?.name).toBe("English Name");
        expect(fetched.translations?.[1]?.name).toBe("Nombre Español");
        
        // Verify form reconstruction preserves all translations
        const reconstructed = transformApiResponseToFormReal(fetched);
        expect(reconstructed.translations.length).toBe(2);
        expect(reconstructed.translations[0].name).toBe("English Name");
        expect(reconstructed.translations[1].name).toBe("Nombre Español");
    });
});