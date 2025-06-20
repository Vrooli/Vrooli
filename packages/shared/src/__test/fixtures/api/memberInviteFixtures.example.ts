/**
 * Example usage of typed memberInviteFixtures
 * This file demonstrates the type safety and autocomplete benefits
 */

import { 
    memberInviteFixtures, 
    memberInviteTestDataFactory,
    typedMemberInviteFixtures 
} from "./memberInviteFixtures.js";
import type { MemberInviteCreateInput, MemberInviteUpdateInput } from "../../../api/types.js";

// Example 1: Using fixtures directly with full type safety
function testDirectFixtureUsage() {
    // TypeScript knows the exact shape of the data
    const minimalCreate: MemberInviteCreateInput = memberInviteFixtures.minimal.create;
    const minimalUpdate: MemberInviteUpdateInput = memberInviteFixtures.minimal.update;
    
    // Autocomplete works for all properties
    console.log(minimalCreate.id);
    console.log(minimalCreate.teamConnect);
    console.log(minimalCreate.userConnect);
    console.log(minimalCreate.message); // optional
    console.log(minimalCreate.willBeAdmin); // optional
    console.log(minimalCreate.willHavePermissions); // optional
}

// Example 2: Using the factory to create custom test data
function testFactoryUsage() {
    // Create minimal invite with overrides
    const customInvite = memberInviteTestDataFactory.createMinimal({
        message: "Welcome to our team!",
        willBeAdmin: true
    });
    
    // Create complete invite with overrides
    const completeInvite = memberInviteTestDataFactory.createComplete({
        willHavePermissions: JSON.stringify(["read", "write", "delete"])
    });
    
    // TypeScript ensures the overrides match the expected types
    // This would cause a type error:
    // const badInvite = memberInviteTestDataFactory.createMinimal({
    //     willBeAdmin: "not-a-boolean" // Type error!
    // });
}

// Example 3: Using validated factory methods
async function testValidatedFactoryUsage() {
    try {
        // Create and validate minimal invite
        const validatedInvite = await memberInviteTestDataFactory.createMinimalValidated({
            message: "Validated invitation"
        });
        
        // The returned data is guaranteed to pass validation
        console.log("Valid invite:", validatedInvite);
        
    } catch (error) {
        // Validation errors are caught here
        console.error("Validation failed:", error);
    }
}

// Example 4: Using typed fixtures with validation
async function testTypedFixtures() {
    // Typed fixtures include validation methods
    if (typedMemberInviteFixtures.validateCreate) {
        const validated = await typedMemberInviteFixtures.validateCreate(
            memberInviteFixtures.minimal.create
        );
        console.log("Validated:", validated);
    }
}

// Example 5: Type checking prevents common mistakes
function demonstrateTypeSafety() {
    // TypeScript catches these errors at compile time:
    
    // Wrong property types
    // const invite: MemberInviteCreateInput = {
    //     id: 123, // Error: should be string
    //     teamConnect: true, // Error: should be string
    //     userConnect: null // Error: should be string
    // };
    
    // Missing required properties
    // const incomplete: MemberInviteCreateInput = {
    //     id: "123456789012345678"
    //     // Error: missing teamConnect and userConnect
    // };
    
    // Unknown properties
    // const extra = memberInviteTestDataFactory.createMinimal({
    //     unknownProp: "value" // Error: property doesn't exist
    // });
}

export {
    testDirectFixtureUsage,
    testFactoryUsage,
    testValidatedFactoryUsage,
    testTypedFixtures,
    demonstrateTypeSafety
};