import { describe, test, expect, beforeEach } from 'vitest';
import { shapeMember, memberValidation, generatePK, type Member } from "@vrooli/shared";
import { 
    minimalMemberUpdateFormInput,
    completeMemberUpdateFormInput,
    memberPermissionSets,
    assignDeveloperRoleFormInput,
    assignManagerRoleFormInput,
    transformMemberFormToApiInput,
    validateMemberPermissions
} from '../form-data/memberFormData.js';
import { 
    minimalMemberResponse,
    completeMemberResponse,
    developerMemberResponse,
    memberUpdateResponse
} from '../api-responses/memberResponses.js';

/**
 * Round-trip testing for Member data flow using REAL application functions
 * Tests the complete user journey: Form Input → API Request → Database → API Response → UI Display
 * 
 * ✅ Uses real shapeMember.update() for transformations
 * ✅ Uses real memberValidation for validation
 * ✅ Tests actual application logic instead of mock implementations
 * 
 * Note: Members can only be updated, not created directly (they're created through invites)
 */

// Mock service for testing (in real implementation this would hit test database)
class MockMemberService {
    private storage: Record<string, Member> = {};

    async update(id: string, updateData: any): Promise<Member> {
        const existing = this.storage[id];
        if (!existing) {
            throw new Error(`Member not found: ${id}`);
        }
        
        const updated = {
            ...existing,
            ...updateData,
            id, // Ensure ID stays the same
            updatedAt: new Date().toISOString(),
        };
        
        this.storage[id] = updated;
        return updated;
    }
    
    async findById(id: string): Promise<Member> {
        const member = this.storage[id];
        if (!member) {
            throw new Error(`Member not found: ${id}`);
        }
        return member;
    }
    
    // Helper to seed test data
    seed(member: Member): void {
        this.storage[member.id] = { ...member };
    }
    
    clear(): void {
        this.storage = {};
    }
}

const mockMemberService = new MockMemberService();

// Helper functions using REAL application logic
function transformFormToUpdateRequestReal(memberId: string, formData: any) {
    return shapeMember.update(
        { id: memberId } as Member, // Original member (minimal for update)
        {
            id: memberId,
            isAdmin: formData.isAdmin,
            permissions: formData.permissions,
        }
    );
}

async function validateMemberFormDataReal(formData: any): Promise<string[]> {
    try {
        // Use real validation schema for member updates
        const validationData = {
            id: formData.id || generatePK().toString(),
            isAdmin: formData.isAdmin,
            permissions: formData.permissions,
        };
        
        await memberValidation.update({ omitFields: [] }).validate(validationData);
        return []; // No validation errors
    } catch (error: any) {
        if (error.errors) {
            return error.errors; // Yup validation errors
        }
        return [error.message || "Validation failed"];
    }
}

function transformApiResponseToFormReal(member: Member) {
    return {
        id: member.id,
        isAdmin: member.isAdmin,
        permissions: Array.isArray(member.permissions) ? member.permissions : JSON.parse(member.permissions as string),
        userId: member.user.id,
        teamId: (member as any).team?.id, // Not always present in Member type
    };
}

function areMemberFormsEqualReal(form1: any, form2: any): boolean {
    return (
        form1.isAdmin === form2.isAdmin &&
        JSON.stringify(form1.permissions?.sort()) === JSON.stringify(form2.permissions?.sort())
    );
}

describe('Member Round-Trip Data Flow', () => {
    beforeEach(() => {
        // Clear test storage between tests
        mockMemberService.clear();
    });

    test('minimal member update maintains data integrity through complete flow', async () => {
        // 🗄️ SETUP: Seed existing member (members are created via invites, not directly)
        const existingMember = {
            ...minimalMemberResponse,
            permissions: ["Read"], // Start with minimal permissions
            isAdmin: false,
        };
        mockMemberService.seed(existingMember);
        
        // 🎨 STEP 1: User updates member permissions via form
        const userFormData = {
            ...minimalMemberUpdateFormInput,
            id: existingMember.id,
            permissions: ["Read", "Update"], // Adding Update permission
        };
        
        // Validate form data using REAL validation schema
        const validationErrors = await validateMemberFormDataReal(userFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Form submits to API using REAL shape function
        const apiUpdateRequest = transformFormToUpdateRequestReal(existingMember.id, userFormData);
        expect(apiUpdateRequest.id).toBe(existingMember.id);
        expect(apiUpdateRequest.isAdmin).toBe(userFormData.isAdmin);
        expect(apiUpdateRequest.permissions).toEqual(userFormData.permissions);
        
        // 🗄️ STEP 3: API updates member (simulated - real test would hit test DB)
        const updatedMember = await mockMemberService.update(existingMember.id, apiUpdateRequest);
        expect(updatedMember.id).toBe(existingMember.id);
        expect(updatedMember.isAdmin).toBe(userFormData.isAdmin);
        expect(updatedMember.permissions).toEqual(userFormData.permissions);
        
        // 🔗 STEP 4: API fetches member back
        const fetchedMember = await mockMemberService.findById(existingMember.id);
        expect(fetchedMember.id).toBe(existingMember.id);
        expect(fetchedMember.isAdmin).toBe(userFormData.isAdmin);
        expect(fetchedMember.permissions).toEqual(userFormData.permissions);
        
        // 🎨 STEP 5: UI would display the member using REAL transformation
        const reconstructedFormData = transformApiResponseToFormReal(fetchedMember);
        expect(reconstructedFormData.isAdmin).toBe(userFormData.isAdmin);
        expect(reconstructedFormData.permissions).toEqual(userFormData.permissions);
        
        // ✅ VERIFICATION: Complete round-trip integrity using REAL comparison
        expect(areMemberFormsEqualReal(
            userFormData, 
            reconstructedFormData
        )).toBe(true);
    });

    test('complete admin member update preserves all data', async () => {
        // 🗄️ SETUP: Seed existing non-admin member
        const existingMember = {
            ...minimalMemberResponse,
            permissions: ["Read"],
            isAdmin: false,
        };
        mockMemberService.seed(existingMember);
        
        // 🎨 STEP 1: User promotes member to admin with full permissions
        const userFormData = {
            ...completeMemberUpdateFormInput,
            id: existingMember.id,
        };
        
        // Validate complex form using REAL validation
        const validationErrors = await validateMemberFormDataReal(userFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Transform to API request using REAL shape function
        const apiUpdateRequest = transformFormToUpdateRequestReal(existingMember.id, userFormData);
        expect(apiUpdateRequest.isAdmin).toBe(true);
        expect(apiUpdateRequest.permissions).toContain("Manage");
        expect(apiUpdateRequest.permissions).toContain("Delete");
        expect(apiUpdateRequest.permissions).toContain("UseApi");
        
        // 🗄️ STEP 3: Update via API
        const updatedMember = await mockMemberService.update(existingMember.id, apiUpdateRequest);
        expect(updatedMember.isAdmin).toBe(true);
        expect(updatedMember.permissions).toEqual(userFormData.permissions);
        
        // 🔗 STEP 4: Fetch back from API
        const fetchedMember = await mockMemberService.findById(existingMember.id);
        expect(fetchedMember.isAdmin).toBe(true);
        expect(fetchedMember.permissions).toEqual(userFormData.permissions);
        
        // 🎨 STEP 5: Verify UI can display correctly using REAL transformation
        const reconstructedFormData = transformApiResponseToFormReal(fetchedMember);
        expect(reconstructedFormData.isAdmin).toBe(userFormData.isAdmin);
        expect(reconstructedFormData.permissions).toEqual(userFormData.permissions);
        
        // ✅ VERIFICATION: Admin promotion preserved all permissions
        expect(fetchedMember.isAdmin).toBe(true);
        expect(fetchedMember.permissions).toContain("Manage");
        expect(fetchedMember.user.id).toBe(existingMember.user.id); // User relationship preserved
    });

    test('role-based permission updates work correctly', async () => {
        // Test different role assignments using permission sets
        const roleTests = [
            { name: "Developer", formData: assignDeveloperRoleFormInput },
            { name: "Manager", formData: assignManagerRoleFormInput },
            { name: "Read Only", formData: memberPermissionSets.readOnly },
            { name: "Contributor", formData: memberPermissionSets.contributor },
            { name: "Admin", formData: memberPermissionSets.admin },
        ];

        for (const roleTest of roleTests) {
            // Create fresh member for each test
            const memberId = `member_${roleTest.name.toLowerCase()}_${generatePK()}`;
            const existingMember = {
                ...minimalMemberResponse,
                id: memberId,
                permissions: ["Read"],
                isAdmin: false,
            };
            mockMemberService.seed(existingMember);

            // 🎨 Apply role using REAL functions
            const formData = {
                ...roleTest.formData,
                id: memberId,
            };
            
            const validationErrors = await validateMemberFormDataReal(formData);
            expect(validationErrors).toHaveLength(0);
            
            // 🔗 Transform and update using REAL functions
            const updateRequest = transformFormToUpdateRequestReal(memberId, formData);
            const updated = await mockMemberService.update(memberId, updateRequest);
            
            // 🗄️ Fetch back
            const fetched = await mockMemberService.findById(memberId);
            
            // ✅ Verify role-specific permissions
            expect(fetched.isAdmin).toBe(formData.isAdmin);
            expect(fetched.permissions).toEqual(formData.permissions);
            
            // Verify form reconstruction using REAL transformation
            const reconstructed = transformApiResponseToFormReal(fetched);
            expect(areMemberFormsEqualReal(formData, reconstructed)).toBe(true);
        }
    });

    test('permission validation catches invalid data before API submission', async () => {
        const invalidFormData = {
            id: generatePK().toString(),
            isAdmin: false,
            permissions: [], // Empty permissions should fail validation
        };
        
        // Use helper validation function
        const helperValidationError = validateMemberPermissions(invalidFormData.permissions);
        expect(helperValidationError).toBe("At least one permission must be selected");
        
        // Use real validation schema
        const validationErrors = await validateMemberFormDataReal(invalidFormData);
        expect(validationErrors.length).toBeGreaterThan(0);
    });

    test('invalid permissions are caught by validation', async () => {
        const invalidPermissionsData = {
            id: generatePK().toString(),
            isAdmin: false,
            permissions: ["Read", "InvalidPermission", "AnotherInvalid"], // Invalid permissions
        };
        
        // Use helper validation function
        const helperValidationError = validateMemberPermissions(invalidPermissionsData.permissions);
        expect(helperValidationError).toBe("Invalid permissions: InvalidPermission, AnotherInvalid");
    });

    test('admin flag consistency with permissions', async () => {
        const existingMember = {
            ...minimalMemberResponse,
            permissions: ["Read"],
            isAdmin: false,
        };
        mockMemberService.seed(existingMember);
        
        // Test: Admin flag true but no Manage permission should be allowed (admin can have subset of permissions)
        const formData = {
            id: existingMember.id,
            isAdmin: true,
            permissions: ["Read", "Update"], // Admin but limited permissions
        };
        
        const validationErrors = await validateMemberFormDataReal(formData);
        expect(validationErrors).toHaveLength(0); // Should be valid
        
        const updateRequest = transformFormToUpdateRequestReal(existingMember.id, formData);
        const updated = await mockMemberService.update(existingMember.id, updateRequest);
        
        expect(updated.isAdmin).toBe(true);
        expect(updated.permissions).toEqual(["Read", "Update"]);
    });

    test('member relationship data preserved during updates', async () => {
        // Create member with full relationship data
        const existingMember = {
            ...completeMemberResponse,
            permissions: ["Read"],
            isAdmin: false,
        };
        mockMemberService.seed(existingMember);
        
        // Update only permissions
        const updateFormData = {
            id: existingMember.id,
            isAdmin: true,
            permissions: ["Create", "Read", "Update", "Delete", "UseApi", "Manage"],
        };
        
        const updateRequest = transformFormToUpdateRequestReal(existingMember.id, updateFormData);
        const updated = await mockMemberService.update(existingMember.id, updateRequest);
        
        // Core member data should be updated
        expect(updated.isAdmin).toBe(true);
        expect(updated.permissions).toEqual(updateFormData.permissions);
        
        // Relationship data should be preserved
        expect(updated.user.id).toBe(existingMember.user.id);
        expect(updated.user.handle).toBe(existingMember.user.handle);
        expect(updated.user.name).toBe(existingMember.user.name);
        expect(updated.id).toBe(existingMember.id);
        expect(updated.createdAt).toBe(existingMember.createdAt);
    });

    test('permission arrays are handled consistently', async () => {
        const existingMember = {
            ...minimalMemberResponse,
            permissions: ["Read"], // Array format
            isAdmin: false,
        };
        mockMemberService.seed(existingMember);
        
        // Update with different permission array
        const formData = {
            id: existingMember.id,
            isAdmin: false,
            permissions: ["Create", "Read", "Update", "UseApi"], // New array
        };
        
        const updateRequest = transformFormToUpdateRequestReal(existingMember.id, formData);
        const updated = await mockMemberService.update(existingMember.id, updateRequest);
        
        // Permissions should be arrays (not JSON strings)
        expect(Array.isArray(updated.permissions)).toBe(true);
        expect(updated.permissions).toEqual(formData.permissions);
        
        // Form reconstruction should handle arrays correctly
        const reconstructed = transformApiResponseToFormReal(updated);
        expect(Array.isArray(reconstructed.permissions)).toBe(true);
        expect(reconstructed.permissions).toEqual(formData.permissions);
    });

    test('data consistency across multiple permission updates', async () => {
        const originalMember = {
            ...minimalMemberResponse,
            permissions: ["Read"],
            isAdmin: false,
        };
        mockMemberService.seed(originalMember);
        
        // Series of permission updates
        const updates = [
            { isAdmin: false, permissions: ["Read", "Update"] },
            { isAdmin: false, permissions: ["Create", "Read", "Update"] },
            { isAdmin: true, permissions: ["Create", "Read", "Update", "Delete"] },
            { isAdmin: true, permissions: ["Create", "Read", "Update", "Delete", "UseApi", "Manage"] },
        ];
        
        let currentMember = originalMember;
        
        for (const updateData of updates) {
            const formData = { id: currentMember.id, ...updateData };
            
            // Validate using REAL functions
            const validationErrors = await validateMemberFormDataReal(formData);
            expect(validationErrors).toHaveLength(0);
            
            // Update using REAL functions
            const updateRequest = transformFormToUpdateRequestReal(currentMember.id, formData);
            const updated = await mockMemberService.update(currentMember.id, updateRequest);
            
            // Verify update
            expect(updated.isAdmin).toBe(updateData.isAdmin);
            expect(updated.permissions).toEqual(updateData.permissions);
            
            // Core data should remain consistent
            expect(updated.id).toBe(originalMember.id);
            expect(updated.user.id).toBe(originalMember.user.id);
            expect(updated.createdAt).toBe(originalMember.createdAt);
            
            currentMember = updated;
        }
        
        // Final member should have all permissions
        const final = await mockMemberService.findById(originalMember.id);
        expect(final.isAdmin).toBe(true);
        expect(final.permissions).toContain("Manage");
        expect(final.permissions).toContain("Delete");
        expect(final.permissions).toContain("UseApi");
    });
});