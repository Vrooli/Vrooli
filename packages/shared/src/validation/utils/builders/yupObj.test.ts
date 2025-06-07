/* eslint-disable @typescript-eslint/ban-ts-comment */
import { describe, it, expect } from "vitest";
import * as yup from "yup";
import { description } from "../index.js";
import { opt, req } from "./optionality.js";
import { yupObj } from "./yupObj.js"; // Update with the actual path

async function assertValid(schema: yup.ObjectSchema<any>, unstripped: any, stripped: any) {
    try {
        await schema.validate(unstripped);
    } catch (error) {
        console.error(`[yupObj.assertValid] Failed to validate unstripped object: ${JSON.stringify(unstripped)}`);
        console.error(`[yupObj.assertValid] Validation error: ${error.message}`);
        console.error(`[yupObj.assertValid] Error path: ${error.path}`);
        console.error(`[yupObj.assertValid] Error value: ${JSON.stringify(error.value)}`);
        throw error;
    }

    try {
        await schema.validate(stripped);
    } catch (error) {
        console.error(`[yupObj.assertValid] Failed to validate stripped object: ${JSON.stringify(stripped)}`);
        console.error(`[yupObj.assertValid] Validation error: ${error.message}`);
        console.error(`[yupObj.assertValid] Error path: ${error.path}`);
        console.error(`[yupObj.assertValid] Error value: ${JSON.stringify(error.value)}`);
        throw error;
    }

    try {
        const result = schema.cast(unstripped, { stripUnknown: true });
        expect(result).to.deep.equal(stripped, `Cast result doesn't match expected stripped object. 
        Got: ${JSON.stringify(result)} 
        Expected: ${JSON.stringify(stripped)}`);
    } catch (error) {
        console.error("[yupObj.assertValid] Failed to cast or compare objects");
        console.error(`[yupObj.assertValid] Unstripped: ${JSON.stringify(unstripped)}`);
        console.error(`[yupObj.assertValid] Stripped: ${JSON.stringify(stripped)}`);
        console.error(`[yupObj.assertValid] Cast result: ${JSON.stringify(schema.cast(unstripped, { stripUnknown: true }))}`);
        throw error;
    }
}

async function assertInvalid(schema: yup.ObjectSchema<any>, unstripped: any) {
    try {
        await schema.validate(unstripped);
        // If we get here, the validation succeeded when it should have failed
        throw new Error(`Expected validation to fail for object: ${JSON.stringify(unstripped)}, but it succeeded`);
    } catch (error) {
        if (error.name !== "ValidationError") {
            // This is not a ValidationError, so it's not what we expected
            console.error(`[yupObj.assertInvalid] Got unexpected error type: ${error.name}`);
            console.error(`[yupObj.assertInvalid] Error message: ${error.message}`);
            throw error;
        }
    }
}

describe("yupObj", () => {
    // Helper function to create a dummy YupModel
    function createDummyYupModel(...fields: string[]) {
        return {
            create: (d) => yupObj({
                ...fields.reduce((acc, curr) => ({ ...acc, [curr]: req(description) }), {}),
            }, [], [], d),
            update: (d) => yupObj({
                ...fields.reduce((acc, curr) => ({ ...acc, [curr]: req(description) }), {}),
            }, [], [], d),
        };
    }

    // Test relationships with action modifiers
    it("should handle one-to-one required Create relationships correctly", async () => {
        const relSchema = createDummyYupModel("dummyField");
        const schema = yupObj(
            { testField: req(yup.string()) },
            // @ts-ignore: Testing runtime scenario
            [["dummyRel", ["Create"], "one", "req", relSchema]],
            [],
            {},
        );
        // Should pass and strip unknown fields
        await assertValid(
            schema,
            { testField: "boop", dummyRelCreate: { dummyField: "test", anotherField: "yeet" } },
            { testField: "boop", dummyRelCreate: { dummyField: "test" } },
        );
        // No relationship fields provided - should fail
        await assertInvalid(schema, { testField: "boop" });
    });
    it("should handle one-to-one optional Create relationships correctly", async () => {
        const relSchema = createDummyYupModel("dummyField");
        const schema = yupObj(
            { testField: req(yup.string()) },
            // @ts-ignore: Testing runtime scenario
            [["dummyRel", ["Create"], "one", "opt", relSchema]],
            [],
            {},
        );
        // Should pass and strip unknown fields
        await assertValid(
            schema,
            { testField: "boop", dummyRelCreate: { dummyField: "test" } },
            { testField: "boop", dummyRelCreate: { dummyField: "test" } },
        );
        await assertValid(
            schema,
            { testField: "boop", dummyRelCreate: { dummyField: "test", anotherField: "yeet" } },
            { testField: "boop", dummyRelCreate: { dummyField: "test" } },
        );
        // No relationship fields provided - should pass because relationship is optional
        await assertValid(
            schema,
            { testField: "boop" },
            { testField: "boop" },
        );
    });
    it("should handle one-to-one optional Create/Update relationships correctly", async () => {
        const relSchema = createDummyYupModel("dummyField");
        const schema = yupObj(
            {},
            // @ts-ignore: Testing runtime scenario
            [["dummyRel", ["Create", "Update"], "one", "opt", relSchema]],
            [],
            {},
        );
        // No relationship fields provided - should pass because relationship is optional
        await assertValid(schema, {}, {});
        // Only Create provided - should pass and strip unknown fields
        await assertValid(
            schema,
            { dummyRelCreate: { dummyField: "test" }, anotherField: "yeet" },
            { dummyRelCreate: { dummyField: "test" } },
        );
        // Only Update provided - should pass and strip unknown fields
        await assertValid(
            schema,
            { dummyRelUpdate: { dummyField: "test" }, anotherField: "yeet" },
            { dummyRelUpdate: { dummyField: "test" } },
        );
        // Both Create and Update provided - should fail
        await assertInvalid(
            schema,
            { dummyRelCreate: { dummyField: "test" }, dummyRelUpdate: { dummyField: "test", anotherField: "yeet" } },
        );
    });
    it("should handle one-to-many required Create relationships correctly", async () => {
        const relSchema = createDummyYupModel("dummyField");
        const schema = yupObj(
            { testField: req(yup.string()) },
            // @ts-ignore: Testing runtime scenario
            [["dummyRel", ["Create"], "many", "req", relSchema]],
            [],
            {},
        );
        // Should pass and strip unknown fields
        await assertValid(
            schema,
            { testField: "boop", dummyRelCreate: [{ dummyField: "test", anotherField: "yeet" }] },
            { testField: "boop", dummyRelCreate: [{ dummyField: "test" }] },
        );
        // Empty array provided - should pass
        await assertValid(
            schema,
            { testField: "boop", dummyRelCreate: [] },
            { testField: "boop", dummyRelCreate: [] },
        );
        // Only Create is allowed - should fail
        await assertInvalid(
            schema,
            { testField: "boop", dummyRelUpdate: [{ dummyField: "test" }] },
        );
        // Create has wrong shape - should fail
        await assertInvalid(
            schema,
            { testField: "boop", dummyRelCreate: [{ wrongField: "yeet" }] },
        );
        // Create is null - should fail
        await assertInvalid(
            schema,
            { testField: "boop", dummyRelCreate: null },
        );
        // Create is undefined - should fail
        await assertInvalid(
            schema,
            { testField: "boop", dummyRelCreate: undefined },
        );
        // Create is not an array - should fail
        await assertInvalid(
            schema,
            { testField: "boop", dummyRelCreate: "not an array" },
        );
    });

    describe("should handle relationship with omitted fields", () => {
        it("test1", async () => {
            const relSchema = createDummyYupModel("dummyField");
            const schema = yupObj(
                { testField: req(yup.string()) },
                // @ts-ignore: Testing runtime scenario
                [["dummyRel", ["Create"], "one", "req", relSchema, ["dummyField"]]],
                [],
                {},
            );
            // Should pass and strip unknown fields
            await assertValid(
                schema,
                { testField: "boop", dummyRelCreate: { dummyField: "test", anotherField: "yeet" } },
                { testField: "boop", dummyRelCreate: {} },
            );
            // Not providing the omitted required field should pass because it's converted to an optional field
            await assertValid(
                schema,
                { testField: "boop", dummyRelCreate: { anotherField: "yeet" } },
                { testField: "boop", dummyRelCreate: {} },
            );
        });
        it("test2", async () => {
            const relSchema = createDummyYupModel("dummyField1", "dummyField2");
            const schema = yupObj(
                { testField: req(yup.string()) },
                // @ts-ignore: Testing runtime scenario
                [["dummyRel", ["Create"], "one", "req", relSchema, ["dummyField1"]]],
                [],
                {},
            );
            // Should pass and strip unknown fields
            await assertValid(
                schema,
                { testField: "boop", dummyRelCreate: { dummyField1: "test", dummyField2: "yeet" } },
                { testField: "boop", dummyRelCreate: { dummyField2: "yeet" } },
            );
            // Not providing the omitted required field should pass because it's converted to an optional field
            await assertValid(
                schema,
                { testField: "boop", dummyRelCreate: { dummyField2: "test" } },
                { testField: "boop", dummyRelCreate: { dummyField2: "test" } },
            );
            // Not providing the non-omitted required field should fail
            await assertInvalid(
                schema,
                { testField: "boop", dummyRelCreate: { dummyField1: "test" } },
            );
        });
        it("test3", async () => {
            const grandchildSchema = createDummyYupModel("dummyField1", "dummyField2");
            const childSchema = {
                create: (d) => yupObj(
                    { testField: req(yup.string()) },
                    // @ts-ignore: Testing runtime scenario
                    [["grandchild", ["Create", "Update"], "one", "req", grandchildSchema, ["dummyField1"]]],
                    [],
                    d,
                ),
                update: (d) => yupObj(
                    { testField: req(yup.string()) },
                    // @ts-ignore: Testing runtime scenario
                    [["grandchild", ["Create", "Update"], "one", "req", grandchildSchema, ["dummyField1"]]],
                    [],
                    d,
                ),
            };
            const parentSchema = yupObj(
                {},
                // @ts-ignore: Testing runtime scenario
                [["child", ["Create", "Update", "Delete"], "one", "req", childSchema, ["grandchild"]]], // Should exclude all grandchild fields, not just grandchildCreate, grandchildUpdate, etc.
                [],
                {},
            );
            // The whole grandchild object should be excluded
            await assertValid(
                parentSchema,
                { childCreate: { testField: "boop", grandchildCreate: { dummyField1: "test", dummyField2: "yeet" } } },
                { childCreate: { testField: "boop" } },
            );
            // Shouldn't need to provide grandchild because it's excluded
            await assertValid(
                parentSchema,
                { childCreate: { testField: "boop" } },
                { childCreate: { testField: "boop" } },
            );
        });
    });

    describe("should enforce exclusion pairs on primitive fields", () => {
        it("test 1", async () => {
            const schema = yupObj(
                {
                    field1: opt(yup.string()),
                    field2: opt(yup.string()),
                },
                [],
                [["field1", "field2", true]],
                {},
            );

            // One field provided - should pass
            await assertValid(schema, { field1: "test" }, { field1: "test" });
            await assertValid(schema, { field2: "test" }, { field2: "test" });
            // Both fields provided - should fail
            await assertInvalid(schema, { field1: "test", field2: "test" });
            // No fields provided - should fail because we set the pair to "true" (required)
            await assertInvalid(schema, {});
        });

        it("test 2", async () => {
            const schema = yupObj(
                {
                    field1: opt(yup.string()),
                    field2: opt(yup.string()),
                },
                [],
                [["field1", "field2", false]],
                {},
            );

            // One field provided - should pass
            await assertValid(schema, { field1: "test" }, { field1: "test" });
            await assertValid(schema, { field2: "test" }, { field2: "test" });
            // Both fields provided - should fail 
            await assertInvalid(schema, { field1: "test", field2: "test" });
            // No fields provided - should pass because we set the pair to "false" (optional)
            await assertValid(schema, {}, {});
        });
    });

    describe("should enforce exclusion pairs on relationship fields", () => {
        it("test1", async () => {
            const relSchema = createDummyYupModel("dummyField");
            const schema = yupObj(
                {},
                [
                    // @ts-ignore: Testing runtime scenario
                    ["rel1", ["Create"], "one", "opt", relSchema],
                    // @ts-ignore: Testing runtime scenario
                    ["rel2", ["Create"], "one", "opt", relSchema],
                ],
                [["rel1Create", "rel2Create", true]],
                {},
            );

            // One field provided - should pass
            await assertValid(
                schema,
                { rel1Create: { dummyField: "test" } },
                { rel1Create: { dummyField: "test" } },
            );
            await assertValid(
                schema,
                { rel2Create: { dummyField: "test" } },
                { rel2Create: { dummyField: "test" } },
            );
            // Both fields provided - should fail
            await assertInvalid(schema, { rel1Create: { dummyField: "test" }, rel2Create: { dummyField: "test" } });
            // No fields provided - should fail because we set the pair to "true" (required)
            await assertInvalid(schema, {});
        });

        it("test2", async () => {
            const relSchema = createDummyYupModel("dummyField");
            const schema = yupObj(
                {},
                [
                    // @ts-ignore: Testing runtime scenario
                    ["rel1", ["Create", "Update"], "one", "opt", relSchema],
                ],
                [["rel1Create", "rel1Update", false]],
                {},
            );

            // One field provided - should pass
            await assertValid(
                schema,
                { rel1Create: { dummyField: "test" } },
                { rel1Create: { dummyField: "test" } },
            );
            await assertValid(
                schema,
                { rel1Update: { dummyField: "test" } },
                { rel1Update: { dummyField: "test" } },
            );
            // Both fields provided - should fail
            await assertInvalid(schema, { rel1Create: { dummyField: "test" }, rel1Update: { dummyField: "test" } });
            // No fields provided - should pass because we set the pair to "false" (optional)
            await assertValid(schema, {}, {});
        });
    });

    it("should enforce exclusion pairs on relationship/primitive field pairs", async () => {
        const relSchema = createDummyYupModel("dummyField");
        const schema = yupObj(
            { field1: opt(yup.string()) },
            [
                // @ts-ignore: Testing runtime scenario
                ["rel1", ["Create"], "one", "opt", relSchema],
            ],
            [["field1", "rel1Create", true]],
            {},
        );

        await assertValid(schema, { field1: "test" }, { field1: "test" });
        await assertValid(
            schema,
            { rel1Create: { dummyField: "test" } },
            { rel1Create: { dummyField: "test" } },
        );
        await assertInvalid(schema, { field1: "test", rel1Create: { dummyField: "test" } });
        await assertInvalid(schema, {});
    });

    it("should omit specified relationships", async () => {
        const relSchema = createDummyYupModel("dummyField");
        const schema = yupObj(
            {},
            // @ts-ignore: Testing runtime scenario
            [["dummyRel", ["Create"], "one", "req", relSchema]],
            [],
            { omitFields: ["dummyRelCreate"] },
        );

        await assertValid(
            schema,
            { dummyRelCreate: { dummyField: "test", anotherField: "yeet" } },
            {},
        );
        await assertValid(schema, {}, {});
    });

    it("should treat one-to-one Connects as an ID", async () => {
        const relSchema = createDummyYupModel("dummyField");
        const schema = yupObj(
            { testField: yup.string().required() },
            // @ts-ignore: Testing runtime scenario
            [["dummyRel", ["Create", "Connect"], "one", "opt", relSchema]],
            [],
            {},
        );

        // Connect should accept an ID
        const id = "123456789012345678"; // Changed to Snowflake-like ID
        await assertValid(
            schema,
            { testField: "boop", dummyRelConnect: id },
            { testField: "boop", dummyRelConnect: id },
        );
        await assertInvalid(schema, { testField: "boop", dummyRelConnect: "not an id" });
        // Create should accept an object
        await assertValid(
            schema,
            { testField: "boop", dummyRelCreate: { dummyField: "test" } },
            { testField: "boop", dummyRelCreate: { dummyField: "test" } },
        );
        await assertInvalid(schema, { testField: "boop", dummyRelCreate: id });
        // Should only accept one of the two fields
        await assertInvalid(schema, { testField: "boop", dummyRelCreate: { dummyField: "test" }, dummyRelConnect: id });
        // Since it's optional, no fields provided - should pass
        await assertValid(schema, { testField: "boop" }, { testField: "boop" });
    });

    it("should treat one-to-many Connects as an ID array", async () => {
        const relSchema = createDummyYupModel("dummyField");
        const schema = yupObj(
            { testField: yup.string().required() },
            // @ts-ignore: Testing runtime scenario
            [["dummyRel", ["Create", "Connect"], "many", "opt", relSchema]],
            [],
            {},
        );

        // Connect should accept an ID array
        const ids = ["123456789012345678", "123456789012345679"]; // Changed to Snowflake-like IDs
        await assertValid(
            schema,
            { testField: "boop", dummyRelConnect: ids },
            { testField: "boop", dummyRelConnect: ids },
        );
        await assertInvalid(schema, { testField: "boop", dummyRelConnect: { dummyField: "test" } });
        await assertInvalid(schema, { testField: "boop", dummyRelConnect: "not an id" });
        // Create should accept an object array
        await assertValid(
            schema,
            { testField: "boop", dummyRelCreate: [{ dummyField: "test" }] },
            { testField: "boop", dummyRelCreate: [{ dummyField: "test" }] },
        );
        await assertInvalid(schema, { testField: "boop", dummyRelCreate: { dummyField: "test" } });
        await assertInvalid(schema, { testField: "boop", dummyRelCreate: ids });
        // Should only accept one of the two fields
        await assertInvalid(schema, { testField: "boop", dummyRelCreate: { dummyField: "test" }, dummyRelConnect: ids });
        // Since it's optional, no fields provided - should pass
        await assertValid(schema, { testField: "boop" }, { testField: "boop" });
    });

    it("should treat one-to-one Deletes and Disconnects as a boolean", async () => {
        const relSchema = createDummyYupModel("dummyField");
        const schema = yupObj(
            { testField: yup.string().required() },
            // @ts-ignore: Testing runtime scenario
            [["dummyRel", ["Delete", "Disconnect"], "one", "opt", relSchema]],
            [],
            {},
        );

        // Delete should accept a boolean
        await assertValid(schema, { testField: "boop", dummyRelDelete: true }, { testField: "boop", dummyRelDelete: true });
        await assertInvalid(schema, { testField: "boop", dummyRelDelete: "not a boolean" });
        // Disconnect should accept a boolean
        await assertValid(schema, { testField: "boop", dummyRelDisconnect: true }, { testField: "boop", dummyRelDisconnect: true });
        await assertInvalid(schema, { testField: "boop", dummyRelDisconnect: "not a boolean" });
        // Should only accept one of the two fields
        await assertInvalid(schema, { testField: "boop", dummyRelDelete: true, dummyRelDisconnect: true });
        // Since it's optional, no fields provided - should pass
        await assertValid(schema, { testField: "boop" }, { testField: "boop" });
    });

    it("should treat one-to-many Deletes and Disconnects as an ID array", async () => {
        const relSchema = createDummyYupModel("dummyField");
        const schema = yupObj(
            { testField: yup.string().required() },
            // @ts-ignore: Testing runtime scenario
            [["dummyRel", ["Delete", "Disconnect"], "many", "opt", relSchema]],
            [],
            {},
        );

        // Delete should accept an ID array
        const ids = ["123456789012345678", "123456789012345679"]; // Changed to Snowflake-like IDs
        await assertValid(
            schema,
            { testField: "boop", dummyRelDelete: ids },
            { testField: "boop", dummyRelDelete: ids },
        );
        await assertInvalid(schema, { testField: "boop", dummyRelDelete: { dummyField: "test" } });
        await assertInvalid(schema, { testField: "boop", dummyRelDelete: "not an id" });
        // Disconnect should accept an ID array
        await assertValid(
            schema,
            { testField: "boop", dummyRelDisconnect: ids },
            { testField: "boop", dummyRelDisconnect: ids },
        );
        await assertInvalid(schema, { testField: "boop", dummyRelDisconnect: { dummyField: "test" } });
        await assertInvalid(schema, { testField: "boop", dummyRelDisconnect: "not an id" });
        // Should accept both fields because it's many-to-many
        await assertValid(
            schema,
            { testField: "boop", dummyRelDelete: ids, dummyRelDisconnect: ids },
            { testField: "boop", dummyRelDelete: ids, dummyRelDisconnect: ids },
        );
        // Since it's optional, no fields provided - should pass
        await assertValid(schema, { testField: "boop" }, { testField: "boop" });
    });
});
