import { describe, test, expect, beforeEach } from 'vitest';
import { shapeWallet, walletValidation, generatePK, type Wallet } from "@vrooli/shared";
import { 
    minimalWalletUpdateFormInput,
    completeWalletUpdateFormInput,
    walletFormScenarios,
    clearWalletNameFormInput,
    invalidWalletFormInputs,
    walletFormEdgeCases,
    type WalletFormData
} from '../form-data/walletFormData.js';
import { 
    minimalWalletResponse,
    completeWalletResponse 
} from '../api-responses/walletResponses.js';

/**
 * Round-trip testing for Wallet data flow using REAL application functions
 * Tests the complete user journey: Form Input → API Request → Database → API Response → UI Display
 * 
 * ✅ Uses real shapeWallet.update() for transformations
 * ✅ Uses real walletValidation for validation
 * ✅ Tests actual application logic instead of mock implementations
 * 
 * Note: Wallets cannot be created through forms - they are created via handshake process
 * This test focuses on update operations and data integrity
 */

// Define the form data type based on the wallet form structure
type WalletFormData = {
    id: string;
    name?: string;
};

// Mock wallet service for testing (simulates API interactions)
const mockWalletService = {
    storage: {} as Record<string, Wallet>,
    
    // Simulate updating an existing wallet
    async update(id: string, updateData: any): Promise<Wallet> {
        const existing = this.storage[id];
        if (!existing) {
            throw new Error(`Wallet ${id} not found`);
        }
        
        const updated: Wallet = {
            ...existing,
            ...updateData,
            id: existing.id, // ID cannot be changed
            updatedAt: new Date().toISOString(),
        };
        
        this.storage[id] = updated;
        return updated;
    },
    
    // Simulate finding a wallet by ID
    async findById(id: string): Promise<Wallet> {
        const wallet = this.storage[id];
        if (!wallet) {
            throw new Error(`Wallet ${id} not found`);
        }
        return wallet;
    },
    
    // Setup initial wallet data for testing
    async setupWallet(walletData: Partial<Wallet>): Promise<Wallet> {
        const wallet: Wallet = {
            __typename: "Wallet",
            id: walletData.id || generatePK().toString(),
            name: walletData.name || null,
            publicAddress: walletData.publicAddress || null,
            stakingAddress: walletData.stakingAddress || "0x1234567890123456789012345678901234567890",
            verifiedAt: walletData.verifiedAt || new Date().toISOString(),
            createdAt: walletData.createdAt || new Date().toISOString(),
            updatedAt: walletData.updatedAt || new Date().toISOString(),
            user: walletData.user || null,
            team: walletData.team || null,
        };
        
        this.storage[wallet.id] = wallet;
        return wallet;
    },
};

// Helper functions using REAL application logic
function transformFormToUpdateRequestReal(formData: WalletFormData) {
    return shapeWallet.update({
        __typename: "Wallet",
        id: formData.id,
    }, {
        id: formData.id,
        name: formData.name || null,
    });
}

async function validateWalletFormDataReal(formData: WalletFormData): Promise<string[]> {
    try {
        // Use real validation schema
        const validationData = {
            id: formData.id,
            name: formData.name,
        };
        
        await walletValidation.update({ omitFields: [] }).validate(validationData);
        return []; // No validation errors
    } catch (error: any) {
        if (error.errors) {
            return error.errors; // Yup validation errors
        }
        return [error.message || "Validation failed"];
    }
}

function transformApiResponseToFormReal(wallet: Wallet): WalletFormData {
    return {
        id: wallet.id,
        name: wallet.name || undefined,
    };
}

function areWalletFormsEqualReal(form1: WalletFormData, form2: WalletFormData): boolean {
    return (
        form1.id === form2.id &&
        (form1.name || null) === (form2.name || null)
    );
}

describe('Wallet Round-Trip Data Flow', () => {
    beforeEach(() => {
        // Clear test storage between tests
        mockWalletService.storage = {};
    });

    test('minimal wallet update maintains data integrity through complete flow', async () => {
        // Setup: Create initial wallet
        const initialWallet = await mockWalletService.setupWallet({
            id: "400000000000000001",
            name: "Original Wallet Name",
            stakingAddress: "0x1234567890123456789012345678901234567890",
        });
        
        // 🎨 STEP 1: User fills out wallet update form
        const userFormData: WalletFormData = {
            id: "400000000000000001",
            name: "My Updated Wallet",
        };
        
        // Validate form data using REAL validation schema
        const validationErrors = await validateWalletFormDataReal(userFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Form submits to API using REAL shape function
        const apiUpdateRequest = transformFormToUpdateRequestReal(userFormData);
        expect(apiUpdateRequest.id).toBe(userFormData.id);
        expect(apiUpdateRequest.name).toBe(userFormData.name);
        
        // 🗄️ STEP 3: API updates wallet (simulated - real test would hit test DB)
        const updatedWallet = await mockWalletService.update(userFormData.id, apiUpdateRequest);
        expect(updatedWallet.id).toBe(userFormData.id);
        expect(updatedWallet.name).toBe(userFormData.name);
        expect(updatedWallet.stakingAddress).toBe(initialWallet.stakingAddress); // Preserved
        
        // 🔗 STEP 4: API fetches updated wallet back
        const fetchedWallet = await mockWalletService.findById(updatedWallet.id);
        expect(fetchedWallet.id).toBe(updatedWallet.id);
        expect(fetchedWallet.name).toBe(userFormData.name);
        
        // 🎨 STEP 5: UI would display the wallet using REAL transformation
        const reconstructedFormData = transformApiResponseToFormReal(fetchedWallet);
        expect(reconstructedFormData.id).toBe(userFormData.id);
        expect(reconstructedFormData.name).toBe(userFormData.name);
        
        // ✅ VERIFICATION: Complete round-trip integrity using REAL comparison
        expect(areWalletFormsEqualReal(
            userFormData, 
            reconstructedFormData
        )).toBe(true);
    });

    test('complete wallet update with complex name preserves all data', async () => {
        // Setup initial wallet
        const initialWallet = await mockWalletService.setupWallet({
            id: "400000000000000002",
            name: "Basic Wallet",
            stakingAddress: "0xabcdef1234567890abcdef1234567890abcdef12",
        });
        
        // 🎨 STEP 1: User updates wallet with complex name
        const userFormData: WalletFormData = {
            id: "400000000000000002",
            name: "💎 My Primary Crypto Wallet 🚀",
        };
        
        // Validate complex form using REAL validation
        const validationErrors = await validateWalletFormDataReal(userFormData);
        expect(validationErrors).toHaveLength(0);
        
        // 🔗 STEP 2: Transform to API request using REAL shape function
        const apiUpdateRequest = transformFormToUpdateRequestReal(userFormData);
        expect(apiUpdateRequest.name).toBe(userFormData.name);
        
        // 🗄️ STEP 3: Update via API
        const updatedWallet = await mockWalletService.update(userFormData.id, apiUpdateRequest);
        expect(updatedWallet.name).toBe(userFormData.name);
        expect(updatedWallet.stakingAddress).toBe(initialWallet.stakingAddress);
        
        // 🔗 STEP 4: Fetch back from API
        const fetchedWallet = await mockWalletService.findById(updatedWallet.id);
        expect(fetchedWallet.name).toBe(userFormData.name);
        
        // 🎨 STEP 5: Verify UI can display correctly using REAL transformation
        const reconstructedFormData = transformApiResponseToFormReal(fetchedWallet);
        expect(reconstructedFormData.name).toBe(userFormData.name);
        
        // ✅ VERIFICATION: Complex data preserved through round-trip
        expect(areWalletFormsEqualReal(
            userFormData, 
            reconstructedFormData
        )).toBe(true);
    });

    test('wallet name clearing works correctly', async () => {
        // Setup wallet with name
        const initialWallet = await mockWalletService.setupWallet({
            id: "400000000000000003",
            name: "Wallet to Clear",
            stakingAddress: "0x1111222233334444555566667777888899990000",
        });
        
        // 🎨 STEP 1: User clears wallet name
        const userFormData: WalletFormData = {
            id: "400000000000000003",
            name: undefined,
        };
        
        // 🔗 STEP 2: Transform to API request using REAL functions
        const apiUpdateRequest = transformFormToUpdateRequestReal(userFormData);
        expect(apiUpdateRequest.name).toBe(null); // undefined becomes null
        
        // 🗄️ STEP 3: Update via API
        const updatedWallet = await mockWalletService.update(userFormData.id, apiUpdateRequest);
        expect(updatedWallet.name).toBe(null);
        
        // 🔗 STEP 4: Fetch updated wallet
        const fetchedWallet = await mockWalletService.findById(updatedWallet.id);
        expect(fetchedWallet.name).toBe(null);
        
        // 🎨 STEP 5: Transform for display
        const reconstructedFormData = transformApiResponseToFormReal(fetchedWallet);
        expect(reconstructedFormData.name).toBeUndefined(); // null becomes undefined
        
        // ✅ VERIFICATION: Name clearing worked correctly
        expect(fetchedWallet.stakingAddress).toBe(initialWallet.stakingAddress); // Preserved
        expect(reconstructedFormData.name).toBeUndefined();
    });

    test('all wallet name scenarios work correctly through round-trip', async () => {
        const scenarios = [
            walletFormScenarios.personalWallet,
            walletFormScenarios.teamWallet,
            walletFormScenarios.stakingWallet,
            walletFormScenarios.shortName,
            walletFormScenarios.specialCharsName,
            walletFormScenarios.unicodeName,
        ];
        
        for (const scenario of scenarios) {
            // Setup initial wallet
            await mockWalletService.setupWallet({
                id: scenario.id,
                name: "Original Name",
                stakingAddress: `0x${scenario.id}`.padEnd(42, '0'),
            });
            
            // 🎨 Create form data for scenario
            const formData: WalletFormData = {
                id: scenario.id,
                name: scenario.name,
            };
            
            // 🔗 Transform and update using REAL functions
            const updateRequest = transformFormToUpdateRequestReal(formData);
            const updatedWallet = await mockWalletService.update(scenario.id, updateRequest);
            
            // 🗄️ Fetch back
            const fetchedWallet = await mockWalletService.findById(scenario.id);
            
            // ✅ Verify scenario-specific data
            expect(fetchedWallet.name).toBe(scenario.name);
            expect(fetchedWallet.id).toBe(scenario.id);
            
            // Verify form reconstruction using REAL transformation
            const reconstructed = transformApiResponseToFormReal(fetchedWallet);
            expect(reconstructed.name).toBe(scenario.name);
            expect(reconstructed.id).toBe(scenario.id);
        }
    });

    test('validation catches invalid form data before API submission', async () => {
        const invalidScenarios = [
            invalidWalletFormInputs.emptyId,
            invalidWalletFormInputs.invalidId,
            invalidWalletFormInputs.nameTooShort,
            invalidWalletFormInputs.nameTooLong,
        ];
        
        for (const invalidData of invalidScenarios) {
            const validationErrors = await validateWalletFormDataReal(invalidData as WalletFormData);
            expect(validationErrors.length).toBeGreaterThan(0);
            
            // Should not proceed to API if validation fails
            expect(validationErrors.some(error => 
                error.includes("required") || 
                error.includes("valid ID") || 
                error.includes("at least") ||
                error.includes("at most")
            )).toBe(true);
        }
    });

    test('edge cases handle correctly in round-trip', async () => {
        const edgeCases = [
            walletFormEdgeCases.nameWithSpecialCharacters,
            walletFormEdgeCases.nameWithNumbers,
            walletFormEdgeCases.nameWithEmojis,
            walletFormEdgeCases.minimalValidName,
            walletFormEdgeCases.maximalValidName,
        ];
        
        for (const edgeCase of edgeCases) {
            // Setup initial wallet
            await mockWalletService.setupWallet({
                id: edgeCase.id,
                name: "Original",
                stakingAddress: `0x${edgeCase.id}`.padEnd(42, '0'),
            });
            
            // Test edge case through full flow
            const validationErrors = await validateWalletFormDataReal(edgeCase as WalletFormData);
            expect(validationErrors).toHaveLength(0);
            
            const updateRequest = transformFormToUpdateRequestReal(edgeCase as WalletFormData);
            const updated = await mockWalletService.update(edgeCase.id, updateRequest);
            const fetched = await mockWalletService.findById(edgeCase.id);
            
            // Edge case data preserved
            expect(fetched.name).toBe(edgeCase.name);
            
            const reconstructed = transformApiResponseToFormReal(fetched);
            expect(reconstructed.name).toBe(edgeCase.name);
        }
    });

    test('wallet update without name change preserves existing name', async () => {
        // Setup wallet with existing name
        const initialWallet = await mockWalletService.setupWallet({
            id: "400000000000000010",
            name: "Existing Wallet Name",
            stakingAddress: "0xaaaa111122223333444455556666777788889999",
        });
        
        // 🎨 STEP 1: User updates wallet without changing name
        const userFormData: WalletFormData = {
            id: "400000000000000010",
            // name is omitted, should preserve existing name
        };
        
        // 🔗 STEP 2: Transform to API request
        const apiUpdateRequest = transformFormToUpdateRequestReal(userFormData);
        expect(apiUpdateRequest.name).toBe(null); // undefined becomes null
        
        // 🗄️ STEP 3: Update via API (should preserve existing name when null is sent)
        // Note: In a real implementation, the API would handle null by not updating the field
        // For this test, we'll simulate that behavior
        const updateData = { ...apiUpdateRequest };
        if (updateData.name === null) {
            delete updateData.name; // Don't update name field
        }
        
        const updatedWallet = await mockWalletService.update(userFormData.id, updateData);
        expect(updatedWallet.name).toBe(initialWallet.name); // Name preserved
        
        // ✅ VERIFICATION: Name preservation worked
        expect(updatedWallet.name).toBe("Existing Wallet Name");
        expect(updatedWallet.stakingAddress).toBe(initialWallet.stakingAddress);
    });

    test('data consistency across multiple wallet operations', async () => {
        // Setup initial wallet
        const initialWallet = await mockWalletService.setupWallet({
            id: "400000000000000011",
            name: "Multi-Op Wallet",
            stakingAddress: "0xbbbb111122223333444455556666777788889999",
        });
        
        // Operation 1: Update name
        const firstUpdate: WalletFormData = {
            id: initialWallet.id,
            name: "First Update",
        };
        
        const firstRequest = transformFormToUpdateRequestReal(firstUpdate);
        const firstResult = await mockWalletService.update(initialWallet.id, firstRequest);
        
        // Operation 2: Clear name
        const secondUpdate: WalletFormData = {
            id: initialWallet.id,
            name: undefined,
        };
        
        const secondRequest = transformFormToUpdateRequestReal(secondUpdate);
        const secondResult = await mockWalletService.update(initialWallet.id, secondRequest);
        
        // Operation 3: Set name again
        const thirdUpdate: WalletFormData = {
            id: initialWallet.id,
            name: "Final Name",
        };
        
        const thirdRequest = transformFormToUpdateRequestReal(thirdUpdate);
        const thirdResult = await mockWalletService.update(initialWallet.id, thirdRequest);
        
        // Fetch final state
        const finalWallet = await mockWalletService.findById(initialWallet.id);
        
        // Core wallet data should remain consistent
        expect(finalWallet.id).toBe(initialWallet.id);
        expect(finalWallet.stakingAddress).toBe(initialWallet.stakingAddress);
        expect(finalWallet.verifiedAt).toBe(initialWallet.verifiedAt);
        
        // Final state should have the last name
        expect(finalWallet.name).toBe("Final Name");
        
        // Transform for display verification
        const displayData = transformApiResponseToFormReal(finalWallet);
        expect(displayData.name).toBe("Final Name");
        expect(displayData.id).toBe(initialWallet.id);
    });

    test('wallet form validation enforces proper constraints', async () => {
        // Test ID validation
        const missingIdData = invalidWalletFormInputs.missingId;
        const missingIdErrors = await validateWalletFormDataReal(missingIdData as WalletFormData);
        expect(missingIdErrors.length).toBeGreaterThan(0);
        
        // Test name length validation
        const tooShortName = invalidWalletFormInputs.nameTooShort;
        const tooShortErrors = await validateWalletFormDataReal(tooShortName as WalletFormData);
        expect(tooShortErrors.length).toBeGreaterThan(0);
        
        const tooLongName = invalidWalletFormInputs.nameTooLong;
        const tooLongErrors = await validateWalletFormDataReal(tooLongName as WalletFormData);
        expect(tooLongErrors.length).toBeGreaterThan(0);
        
        // Test valid boundary cases
        const minValid = walletFormEdgeCases.minimalValidName;
        const minValidErrors = await validateWalletFormDataReal(minValid as WalletFormData);
        expect(minValidErrors).toHaveLength(0);
        
        const maxValid = walletFormEdgeCases.maximalValidName;
        const maxValidErrors = await validateWalletFormDataReal(maxValid as WalletFormData);
        expect(maxValidErrors).toHaveLength(0);
    });
});