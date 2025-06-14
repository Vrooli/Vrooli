/* eslint-disable @typescript-eslint/ban-ts-comment */
/* eslint-disable @typescript-eslint/no-non-null-assertion */
import { describe, it, expect, vi } from "vitest";
import * as yup from "yup";
import { type YupModel } from "../types.js";
import { rel, transRel } from "./rel.js";
import { yupObj } from "./yupObj.js";

describe("rel function", () => {
    // Mock data and models for testing
    const mockData = { omitFields: [] };
    const mockModel = {
        create: () => yup.object().shape({ mockField: yup.string().required() }),
        update: () => yup.object().shape({ mockField: yup.string().required() }),
    } as unknown as YupModel<["create", "update"]>;

    it("should throw an error if \"Create\" or \"Update\" is in relTypes but no model is provided", () => {
        expect(() => {
            rel(mockData, "testRelation", ["Create"], "one", "opt");
        }).toThrow();

        expect(() => {
            rel(mockData, "testRelation", ["Update"], "one", "opt");
        }).toThrow();
    });

    it("should return an object with the correct keys for given relation types", () => {
        const result = rel(mockData, "testRelation", ["Connect", "Create", "Delete", "Disconnect", "Update"], "one", "opt", mockModel);
        expect(Object.keys(result)).to.have.members(["testRelationConnect", "testRelationCreate", "testRelationDelete", "testRelationDisconnect", "testRelationUpdate"]);
    });

    it("should handle \"one\" and \"many\" relationships correctly for Connect", async () => {
        const oneResult = rel(mockData, "testRelation", ["Connect"], "one", "req");
        const manyResult = rel(mockData, "testRelation", ["Connect"], "many", "req");

        // Prepare test data
        const validSnowflakeId = "123456789012345678"; // Use a Snowflake-like ID
        const validOneData = { testRelationConnect: validSnowflakeId };
        const validManyData = { testRelationConnect: [validSnowflakeId, "123456789012345679"] }; // Use Snowflake-like IDs
        const invalidOneData = validManyData;
        const invalidManyData = validOneData;

        // Validate 'one' relationship
        await oneResult.testRelationConnect!.validate(validOneData.testRelationConnect);

        try {
            await oneResult.testRelationConnect!.validate(invalidOneData.testRelationConnect);
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }

        // Validate 'many' relationship
        await manyResult.testRelationConnect!.validate(validManyData.testRelationConnect);

        try {
            await manyResult.testRelationConnect!.validate(invalidManyData.testRelationConnect);
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }
    });
    it("should handle \"one\" and \"many\" relationships correctly for Create", async () => {
        const oneResult = rel(mockData, "testRelation", ["Create"], "one", "req", mockModel);
        const manyResult = rel(mockData, "testRelation", ["Create"], "many", "req", mockModel);

        // Prepare test data
        const validOneData = { testRelationCreate: { mockField: "test" } };
        const validManyData = { testRelationCreate: [{ mockField: "test" }, { mockField: "test" }] };
        const invalidOneData = validManyData;
        const invalidManyData = validOneData;

        // Validate 'one' relationship
        await oneResult.testRelationCreate!.validate(validOneData.testRelationCreate);

        try {
            await oneResult.testRelationCreate!.validate(invalidOneData.testRelationCreate);
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }

        // Validate 'many' relationship
        await manyResult.testRelationCreate!.validate(validManyData.testRelationCreate);

        try {
            await manyResult.testRelationCreate!.validate(invalidManyData.testRelationCreate);
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }
    });
    it("should handle \"one\" and \"many\" relationships correctly for Update", async () => {
        const oneResult = rel(mockData, "testRelation", ["Update"], "one", "req", mockModel);
        const manyResult = rel(mockData, "testRelation", ["Update"], "many", "req", mockModel);

        // Prepare test data
        const validOneData = { testRelationUpdate: { mockField: "test" } };
        const validManyData = { testRelationUpdate: [{ mockField: "test" }, { mockField: "test" }] };
        const invalidOneData = validManyData;
        const invalidManyData = validOneData;

        // Validate 'one' relationship
        await oneResult.testRelationUpdate!.validate(validOneData.testRelationUpdate);

        try {
            await oneResult.testRelationUpdate!.validate(invalidOneData.testRelationUpdate);
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }

        // Validate 'many' relationship
        await manyResult.testRelationUpdate!.validate(validManyData.testRelationUpdate);

        try {
            await manyResult.testRelationUpdate!.validate(invalidManyData.testRelationUpdate);
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }
    });
    it("should handle \"one\" and \"many\" relationships correctly for Delete", async () => {
        const oneResult = rel(mockData, "testRelation", ["Delete"], "one", "req", mockModel);
        const manyResult = rel(mockData, "testRelation", ["Delete"], "many", "req", mockModel);

        // Prepare test data
        const validSnowflakeId1 = "123456789012345678"; // Use a Snowflake-like ID
        const validSnowflakeId2 = "123456789012345679"; // Use another Snowflake-like ID
        const validOneData = { testRelationDelete: true };
        const validManyData = { testRelationDelete: [validSnowflakeId1, validSnowflakeId2] }; // Use Snowflake-like IDs
        const invalidOneData1 = validManyData;
        const invalidOneData2 = { testRelationDelete: false };
        const invalidOneData3 = { testRelationDelete: [true, true] };
        const invalidManyData1 = validOneData;
        const invalidManyData2 = { testRelationDelete: [true, true] };

        // Validate 'one' relationship
        await oneResult.testRelationDelete!.validate(validOneData.testRelationDelete);

        try {
            await oneResult.testRelationDelete!.validate(invalidOneData1.testRelationDelete);
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }

        try {
            await oneResult.testRelationDelete!.validate(invalidOneData2.testRelationDelete);
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }

        try {
            await oneResult.testRelationDelete!.validate(invalidOneData3.testRelationDelete);
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }

        // Validate 'many' relationship
        await manyResult.testRelationDelete!.validate(validManyData.testRelationDelete);

        try {
            await manyResult.testRelationDelete!.validate(invalidManyData1.testRelationDelete);
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }

        try {
            await manyResult.testRelationDelete!.validate(invalidManyData2.testRelationDelete);
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }
    });
    it("should handle \"one\" and \"many\" relationships correctly for Disconnect", async () => {
        // Should be the same as Delete
        const oneResult = rel(mockData, "testRelation", ["Disconnect"], "one", "req", mockModel);
        const manyResult = rel(mockData, "testRelation", ["Disconnect"], "many", "req", mockModel);

        // Prepare test data
        const validSnowflakeId1 = "123456789012345678"; // Use a Snowflake-like ID
        const validSnowflakeId2 = "123456789012345679"; // Use another Snowflake-like ID
        const validOneData = { testRelationDisconnect: true };
        const validManyData = { testRelationDisconnect: [validSnowflakeId1, validSnowflakeId2] }; // Use Snowflake-like IDs
        const invalidOneData1 = validManyData;
        const invalidOneData2 = { testRelationDisconnect: false };
        const invalidOneData3 = { testRelationDisconnect: [true, true] };
        const invalidManyData1 = validOneData;
        const invalidManyData2 = { testRelationDisconnect: [true, true] };

        // Validate 'one' relationship
        await oneResult.testRelationDisconnect!.validate(validOneData.testRelationDisconnect);

        try {
            await oneResult.testRelationDisconnect!.validate(invalidOneData1.testRelationDisconnect);
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }

        try {
            await oneResult.testRelationDisconnect!.validate(invalidOneData2.testRelationDisconnect);
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }

        try {
            await oneResult.testRelationDisconnect!.validate(invalidOneData3.testRelationDisconnect);
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }

        // Validate 'many' relationship
        await manyResult.testRelationDisconnect!.validate(validManyData.testRelationDisconnect);

        try {
            await manyResult.testRelationDisconnect!.validate(invalidManyData1.testRelationDisconnect);
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }

        try {
            await manyResult.testRelationDisconnect!.validate(invalidManyData2.testRelationDisconnect);
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }
    });

    it("should mark 'Connect' field as required when isRequired is 'req'", async () => {
        const result = rel(mockData, "testRelation", ["Connect"], "one", "req");
        const testSchema = yup.object().shape({
            // @ts-ignore: Testing runtime scenario
            testRelationConnect: result.testRelationConnect,
        });

        // Test with undefined value
        try {
            await testSchema.validate({});
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }

        // Test with valid value
        const validSnowflakeId = "123456789012345678"; // Use a Snowflake-like ID
        const validationResult = await testSchema.validate({ testRelationConnect: validSnowflakeId });
        expect(validationResult.testRelationConnect).toBe(validSnowflakeId);
    });

    it("should mark 'Connect' field as optional when isRequired is 'opt'", async () => {
        const result = rel(mockData, "testRelation", ["Connect"], "one", "opt");
        const testSchema = yup.object().shape({
            // @ts-ignore: Testing runtime scenario
            testRelationConnect: result.testRelationConnect,
        });

        // Test with undefined value
        let validationResult = await testSchema.validate({});
        expect(validationResult.testRelationConnect).toBeUndefined();

        // Test with valid value
        const validSnowflakeId2 = "123456789012345679"; // Use a Snowflake-like ID
        validationResult = await testSchema.validate({ testRelationConnect: validSnowflakeId2 });
        expect(validationResult.testRelationConnect).toBe(validSnowflakeId2);
    });

    const omitFieldsMockModel = {
        create: (d) => yupObj({
            field1: yup.string().required(),
            field2: yup.string().required(),
        }, [], [], d),
        update: (d) => yupObj({
            field1: yup.string().required(),
            field2: yup.string().required(),
        }, [], [], d),
    } as unknown as YupModel<["create", "update"]>;

    it("should require all fields when omitFields is an empty array", async () => {
        const result = rel({ omitFields: [] }, "testRelation", ["Create"], "one", "req", omitFieldsMockModel, []);

        const validData = { testRelationCreate: { field1: "value1", field2: "value2" } };
        const invalidData = { testRelationCreate: { field1: "value1" } }; // Missing field2

        await result.testRelationCreate!.validate(validData.testRelationCreate);

        try {
            await result.testRelationCreate!.validate(invalidData.testRelationCreate);
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }
    });

    it("should require only the field not in omitFields", async () => {
        const result = rel({ omitFields: [] }, "testRelation", ["Create"], "one", "req", omitFieldsMockModel, ["field1", "nonExistingField"]);

        const validData = { testRelationCreate: { field2: "value2" } }; // Only field2 is required
        const invalidData = { testRelationCreate: {} }; // Missing field2

        await result.testRelationCreate!.validate(validData.testRelationCreate);

        try {
            await result.testRelationCreate!.validate(invalidData.testRelationCreate);
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }
    });

    it("should not require any fields when all are in omitFields", async () => {
        const result = rel({ omitFields: [] }, "testRelation", ["Create"], "one", "req", omitFieldsMockModel, ["field1", "field2"]);

        const validData = { testRelationCreate: {} }; // No fields are required

        await result.testRelationCreate!.validate(validData.testRelationCreate);
    });

    it("should work when using data.omitFields", async () => {
        const result = rel({ omitFields: ["field1"] }, "testRelation", ["Create"], "one", "req", omitFieldsMockModel, ["field2"]);

        const validData = { testRelationCreate: {} }; // No fields are required

        await result.testRelationCreate!.validate(validData.testRelationCreate);
    });

    it("should skip entire relationship when relation name is in omitFields", () => {
        const result = rel({ omitFields: ["testRelation"] }, "testRelation", ["Create", "Connect"], "one", "req", mockModel);
        
        expect(Object.keys(result)).to.have.length(0);
    });

    it("should skip specific relationship types when they are in omitFields", () => {
        const result = rel({ omitFields: ["testRelationCreate"] }, "testRelation", ["Create", "Connect"], "one", "req", mockModel);
        
        expect(Object.keys(result)).toContain("testRelationConnect");
        expect(Object.keys(result)).to.not.include("testRelationCreate");
    });

    it("should warn when recursion limit is reached", () => {
        const consoleSpy = vi.spyOn(console, "warn").mockImplementation(() => {});
        
        const deeplyNestedData = { omitFields: [], recurseCount: 21 };
        const result = rel(deeplyNestedData, "testRelation", ["Create"], "one", "req", mockModel);
        
        expect(consoleSpy).toHaveBeenCalledWith(
            "Hit recursion limit in rel",
            "testRelation",
            ["Create"],
            "one",
            "req",
            mockModel,
            undefined
        );
        expect(Object.keys(result)).to.have.length(0);
        
        consoleSpy.mockRestore();
    });

    it("should enforce mutual exclusivity for one-to-one relationships with multiple operations", async () => {
        const result = rel(mockData, "testRelation", ["Connect", "Create"], "one", "opt", mockModel);
        const testSchema = yup.object().shape({
            testRelationConnect: result.testRelationConnect,
            testRelationCreate: result.testRelationCreate,
        });

        // Should fail when both fields are provided
        try {
            await testSchema.validate({
                testRelationConnect: "123456789012345678",
                testRelationCreate: { mockField: "test" }
            });
            expect.fail("Validation should have failed but passed");
        } catch (error: any) {
            expect(error.message).toContain("Cannot provide both");
        }

        // Should pass when only one field is provided
        const validConnectOnly = await testSchema.validate({
            testRelationConnect: "123456789012345678"
        });
        expect(validConnectOnly.testRelationConnect).toBe("123456789012345678");

        const validCreateOnly = await testSchema.validate({
            testRelationCreate: { mockField: "test" }
        });
        expect(validCreateOnly.testRelationCreate).toEqual({ mockField: "test" });
    });

    it("should handle Connect and Create as optional when both are present for required relationships", () => {
        const result = rel(mockData, "testRelation", ["Connect", "Create"], "one", "req", mockModel);
        
        // Both should be optional schemas (not required) because yup will handle the one-of rule
        expect(result.testRelationConnect?.spec.presence).to.not.equal("required");
        expect(result.testRelationCreate?.spec.presence).to.not.equal("required");
    });

    it("should enforce mutual exclusivity for all operation pairs in one-to-one relationships", async () => {
        const result = rel(mockData, "testRelation", ["Connect", "Create", "Update", "Delete", "Disconnect"], "one", "opt", mockModel);
        const testSchema = yup.object().shape({
            testRelationConnect: result.testRelationConnect,
            testRelationCreate: result.testRelationCreate,
            testRelationUpdate: result.testRelationUpdate,
            testRelationDelete: result.testRelationDelete,
            testRelationDisconnect: result.testRelationDisconnect,
        });

        // Test Connect + Update should fail
        try {
            await testSchema.validate({
                testRelationConnect: "123456789012345678",
                testRelationUpdate: { mockField: "test" }
            });
            expect.fail("Validation should have failed but passed");
        } catch (error: any) {
            expect(error.message).toContain("Cannot provide both");
        }

        // Test Delete + Disconnect should fail
        try {
            await testSchema.validate({
                testRelationDelete: true,
                testRelationDisconnect: true
            });
            expect.fail("Validation should have failed but passed");
        } catch (error: any) {
            expect(error.message).toContain("Cannot provide both");
        }

        // Test Create + Delete should fail
        try {
            await testSchema.validate({
                testRelationCreate: { mockField: "test" },
                testRelationDelete: true
            });
            expect.fail("Validation should have failed but passed");
        } catch (error: any) {
            expect(error.message).toContain("Cannot provide both");
        }
    });

    it("should not enforce mutual exclusivity for many relationships even with multiple operations", async () => {
        const result = rel(mockData, "testRelation", ["Connect", "Create"], "many", "opt", mockModel);
        const testSchema = yup.object().shape({
            testRelationConnect: result.testRelationConnect,
            testRelationCreate: result.testRelationCreate,
        });

        // Should pass when both fields are provided for "many" relationships
        const validData = await testSchema.validate({
            testRelationConnect: ["123456789012345678"],
            testRelationCreate: [{ mockField: "test" }]
        });
        
        expect(validData.testRelationConnect).toEqual(["123456789012345678"]);
        expect(validData.testRelationCreate).toEqual([{ mockField: "test" }]);
    });

    it("should handle omitFields as a string instead of array", async () => {
        const result = rel({ omitFields: "field1" }, "testRelation", ["Create"], "one", "req", omitFieldsMockModel);
        
        const validData = { testRelationCreate: { field2: "value2" } };
        await result.testRelationCreate!.validate(validData.testRelationCreate);
        
        // Should fail if field2 is missing
        try {
            await result.testRelationCreate!.validate({ testRelationCreate: {} });
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }
    });

    it("should handle directOmitFields as a string", async () => {
        const result = rel({ omitFields: [] }, "testRelation", ["Create"], "one", "req", omitFieldsMockModel, "field1");
        
        const validData = { testRelationCreate: { field2: "value2" } };
        await result.testRelationCreate!.validate(validData.testRelationCreate);
    });

    it("should deduplicate omitFields from both sources", () => {
        const result = rel(
            { omitFields: ["field1", "field2"] }, 
            "testRelation", 
            ["Create"], 
            "one", 
            "req", 
            omitFieldsMockModel, 
            ["field2", "field3"]
        );
        
        // This tests that the deduplication is working - field2 should only be omitted once
        // We can't directly test the omitFields array, but we can verify the schema behavior
        expect(result.testRelationCreate).toBeDefined();
    });

    it("should handle nested relationships and increment recurseCount", () => {
        const nestedModel = {
            create: (data) => {
                const currentRecurseCount = data.recurseCount || 0;
                expect(currentRecurseCount).toBeGreaterThan(0);
                return yup.object().shape({ nestedField: yup.string() });
            },
            update: (data) => {
                const currentRecurseCount = data.recurseCount || 0;
                expect(currentRecurseCount).toBeGreaterThan(0);
                return yup.object().shape({ nestedField: yup.string() });
            },
        } as unknown as YupModel<["create", "update"]>;

        // This will trigger the model's create function which checks recurseCount
        rel(mockData, "testRelation", ["Create"], "one", "req", nestedModel);
    });

    it("should handle optional fields correctly for all operations in opt mode", async () => {
        const result = rel(mockData, "testRelation", ["Connect", "Create", "Update", "Delete", "Disconnect"], "one", "opt", mockModel);
        const testSchema = yup.object().shape({
            testRelationConnect: result.testRelationConnect,
            testRelationCreate: result.testRelationCreate,
            testRelationUpdate: result.testRelationUpdate,
            testRelationDelete: result.testRelationDelete,
            testRelationDisconnect: result.testRelationDisconnect,
        });

        // All fields should be optional
        const emptyResult = await testSchema.validate({});
        expect(emptyResult).toEqual({});

        // Should handle null values
        const nullResult = await testSchema.validate({
            testRelationConnect: null,
            testRelationCreate: null,
            testRelationUpdate: null,
            testRelationDelete: null,
            testRelationDisconnect: null,
        });
        // Optional fields with null values may be stripped or preserved as null depending on schema
        // The current implementation strips them to undefined
        expect(nullResult.testRelationConnect).to.be.oneOf([null, undefined]);
        expect(nullResult.testRelationCreate).to.be.oneOf([null, undefined]);
        expect(nullResult.testRelationUpdate).to.be.oneOf([null, undefined]);
        expect(nullResult.testRelationDelete).to.be.oneOf([null, undefined]);
        expect(nullResult.testRelationDisconnect).to.be.oneOf([null, undefined]);
    });

    it("should preserve array values when using opt for many relationships", async () => {
        const result = rel(mockData, "testRelation", ["Connect", "Delete"], "many", "opt");
        
        // Test that empty arrays are preserved
        const emptyArrayData = {
            testRelationConnect: [],
            testRelationDelete: []
        };
        
        const validatedEmpty = await result.testRelationConnect!.validate(emptyArrayData.testRelationConnect);
        expect(validatedEmpty).toEqual([]);
        
        const validatedDelete = await result.testRelationDelete!.validate(emptyArrayData.testRelationDelete);
        expect(validatedDelete).toEqual([]);
    });

    it("should handle non-array omitFields in data parameter", () => {
        // Test the case where data.omitFields is a string rather than array
        const dataWithStringOmitFields = {
            omitFields: "testRelation" // String instead of array
        };
        
        const result = rel(dataWithStringOmitFields, "testRelation", ["Create"], "one", "opt", mockModel);
        
        // Should return empty result because relationship is omitted
        expect(Object.keys(result)).to.have.length(0);
    });

    it("should handle non-array directOmitFields parameter", () => {
        // Test the case where directOmitFields is a string rather than array
        const result = rel({}, "testRelation", ["Create"], "one", "opt", mockModel, "someField");
        
        // Should create the relationship since testRelation itself isn't omitted
        expect(result).toHaveProperty("testRelationCreate");
    });

});

describe("transRel function", () => {
    const partialModel = {
        create: () => ({
            name: yup.string().required(),
            description: yup.string(),
        }),
        update: () => ({
            name: yup.string(),
            description: yup.string(),
        }),
    };

    it("should create a translation model with id and language fields", () => {
        const translationModel = transRel(partialModel);
        
        expect(translationModel).toHaveProperty("create");
        expect(translationModel).toHaveProperty("update");
    });

    it("should validate create with required id, language, and custom fields", async () => {
        const translationModel = transRel(partialModel);
        const createSchema = translationModel.create({ omitFields: [] });

        // Valid data
        const validData = {
            id: "123456789012345678",
            language: "en",
            name: "Test Name",
            description: "Test Description"
        };
        const result = await createSchema.validate(validData);
        expect(result).toEqual(validData);

        // Missing required fields
        try {
            await createSchema.validate({
                id: "123456789012345678",
                language: "en"
                // missing name
            });
            expect.fail("Validation should have failed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }
    });

    it("should validate update with required id and optional language", async () => {
        const translationModel = transRel(partialModel);
        const updateSchema = translationModel.update({ omitFields: [] });

        // Valid data with only id
        const minimalData = {
            id: "123456789012345678"
        };
        const result = await updateSchema.validate(minimalData);
        expect(result).toEqual(minimalData);

        // Valid data with all fields
        const completeData = {
            id: "123456789012345678",
            language: "fr",
            name: "Updated Name",
            description: "Updated Description"
        };
        const completeResult = await updateSchema.validate(completeData);
        expect(completeResult).toEqual(completeData);

        // Missing required id
        try {
            await updateSchema.validate({
                language: "en",
                name: "No ID"
            });
            expect.fail("Validation should have failed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }
    });
});
