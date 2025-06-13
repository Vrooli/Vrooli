import { describe, expect, it } from "vitest";

// Test the profanity check logic patterns without complex dependencies
describe("profanityCheck logic patterns", () => {
    // Simulate the collectProfanities logic
    const simulateCollectProfanities = (input: any, profanityFields?: string[]) => {
        const result: Record<string, string[]> = {};
        
        // Check specified profanity fields
        if (profanityFields) {
            for (const field of profanityFields) {
                if (input[field] && typeof input[field] === "string") {
                    result[field] = [input[field]];
                }
            }
        }
        
        // Handle translations
        const handleTranslations = (translations: any[]) => {
            for (const translation of translations) {
                for (const field in translation) {
                    // Skip id and language fields
                    if (field === "id" || field === "language" || field.includes("Id")) continue;
                    
                    if (typeof translation[field] === "string") {
                        result[field] = result[field] ? [...result[field], translation[field]] : [translation[field]];
                    }
                }
            }
        };
        
        if (Array.isArray(input.translationsCreate)) {
            handleTranslations(input.translationsCreate);
        }
        if (Array.isArray(input.translationsUpdate)) {
            handleTranslations(input.translationsUpdate);
        }
        
        // Handle tags
        if (Array.isArray(input.tagsConnect)) {
            result.tagsConnect = input.tagsConnect;
        }
        
        return result;
    };

    it("should collect profanity fields from simple input", () => {
        const input = {
            username: "testuser",
            email: "test@email.com",
            other: "ignored"
        };
        
        const result = simulateCollectProfanities(input, ["username", "email"]);
        
        expect(result).toEqual({
            username: ["testuser"],
            email: ["test@email.com"]
        });
        expect(result.other).toBeUndefined();
    });

    it("should collect fields from translationsCreate", () => {
        const input = {
            translationsCreate: [
                { language: "en", name: "English Name", description: "English Desc" },
                { language: "es", name: "Spanish Name", description: "Spanish Desc" }
            ]
        };
        
        const result = simulateCollectProfanities(input);
        
        expect(result).toEqual({
            name: ["English Name", "Spanish Name"],
            description: ["English Desc", "Spanish Desc"]
        });
    });

    it("should skip id and language fields in translations", () => {
        const input = {
            translationsCreate: [
                { id: "123", language: "en", projectId: "456", name: "Test", description: "Desc" }
            ]
        };
        
        const result = simulateCollectProfanities(input);
        
        expect(result.id).toBeUndefined();
        expect(result.language).toBeUndefined();
        expect(result.projectId).toBeUndefined();
        expect(result.name).toEqual(["Test"]);
    });

    it("should handle tagsConnect", () => {
        const input = {
            tagsConnect: ["tag1", "tag2", "tag3"]
        };
        
        const result = simulateCollectProfanities(input);
        
        expect(result).toEqual({
            tagsConnect: ["tag1", "tag2", "tag3"]
        });
    });

    it("should handle both translationsCreate and translationsUpdate", () => {
        const input = {
            translationsCreate: [
                { name: "Created Name" }
            ],
            translationsUpdate: [
                { id: "123", name: "Updated Name" }
            ]
        };
        
        const result = simulateCollectProfanities(input);
        
        expect(result).toEqual({
            name: ["Created Name", "Updated Name"]
        });
    });

    it("should demonstrate profanity check flow", () => {
        // Simulate the main profanity check logic
        const simulateProfanityCheck = (
            inputData: any[],
            hasProfanityFn: (text: string) => boolean,
            isPublicFn: (obj: any) => boolean
        ) => {
            const fieldsToCheck: Record<string, string[]> = {};
            
            for (const item of inputData) {
                // Skip non-Create/Update actions
                if (!["Create", "Update"].includes(item.action)) continue;
                
                // Skip private objects
                if (!isPublicFn(item.input)) continue;
                
                // Collect fields
                const fields = simulateCollectProfanities(item.input, item.profanityFields);
                for (const field in fields) {
                    fieldsToCheck[field] = fieldsToCheck[field] 
                        ? [...fieldsToCheck[field], ...fields[field]] 
                        : fields[field];
                }
            }
            
            // Check for profanity
            for (const field in fieldsToCheck) {
                for (const text of fieldsToCheck[field]) {
                    if (hasProfanityFn(text)) {
                        throw new Error("BannedWord");
                    }
                }
            }
            
            return true;
        };
        
        // Test case 1: Clean content
        const cleanData = [{
            action: "Create",
            input: { username: "cleanuser", email: "clean@email.com" },
            profanityFields: ["username", "email"]
        }];
        
        expect(() => simulateProfanityCheck(
            cleanData,
            () => false, // no profanity
            () => true   // is public
        )).not.toThrow();
        
        // Test case 2: Profane content
        const profaneData = [{
            action: "Create",
            input: { username: "badword" },
            profanityFields: ["username"]
        }];
        
        expect(() => simulateProfanityCheck(
            profaneData,
            (text) => text === "badword", // has profanity
            () => true                     // is public
        )).toThrow("BannedWord");
        
        // Test case 3: Private object (should skip)
        expect(() => simulateProfanityCheck(
            profaneData,
            (text) => text === "badword", // has profanity
            () => false                    // is private
        )).not.toThrow();
    });
});