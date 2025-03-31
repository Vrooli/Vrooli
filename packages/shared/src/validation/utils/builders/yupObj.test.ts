/* eslint-disable @typescript-eslint/ban-ts-comment */
import { expect } from "chai";
import * as yup from "yup";
import { uuid } from "../../../id/uuid.js";
import { opt } from "./optionality.js";
import { yupObj } from "./yupObj.js"; // Update with the actual path

describe("yupObj", () => {
    // Helper function to create a dummy YupModel
    function createDummyYupModel(...fields: string[]) {
        return {
            create: (d) => yupObj({
                ...fields.reduce((acc, curr) => ({ ...acc, [curr]: yup.string().required() }), {}),
            }, [], [], d),
            update: (d) => yupObj({
                ...fields.reduce((acc, curr) => ({ ...acc, [curr]: yup.string().required() }), {}),
            }, [], [], d),
        };
    }

    // // Test basic field creation
    it("should create an object schema with provided fields", async () => {
        const schema = yupObj(
            { testField: yup.string().required() },
            [],
            [],
            {},
        );
        const unstripped = { testField: "test", extraField: "test" };
        const stripped = { testField: "test" };
        await expect(schema.validate(stripped)).to.be.fulfilled;
        await expect(schema.validate({})).to.be.rejected;
        await expect(schema.validate({ testField: 1 })).to.be.fulfilled;
        // Extra fields are kept in validation, but stripped in casting
        await expect(schema.validate(unstripped)).to.be.fulfilled;
        expect(schema.cast(unstripped, { stripUnknown: true })).to.deep.equal(stripped);
    });

    // Test relationships with action modifiers
    it("should handle one-to-one required Create relationships correctly", async () => {
        const relSchema = createDummyYupModel("dummyField");
        const schema = yupObj(
            { testField: yup.string().required() },
            // @ts-ignore: Testing runtime scenario
            [["dummyRel", ["Create"], "one", "req", relSchema]],
            [],
            {},
        );
        const unstripped = { testField: "boop", dummyRelCreate: { dummyField: "test", anotherField: "yeet" } };
        const stripped = { testField: "boop", dummyRelCreate: { dummyField: "test" } };
        await expect(schema.validate(unstripped)).to.be.fulfilled;
        await expect(schema.validate({})).to.be.rejected;
        expect(schema.cast(unstripped, { stripUnknown: true })).to.deep.equal(stripped);
    });
    it("should handle one-to-one optional Create relationships correctly", async () => {
        const relSchema = createDummyYupModel("dummyField");
        const schema = yupObj(
            { testField: yup.string().required() },
            // @ts-ignore: Testing runtime scenario
            [["dummyRel", ["Create"], "one", "opt", relSchema]],
            [],
            {},
        );
        const withRel = { testField: "boop", dummyRelCreate: { dummyField: "test" } };
        const withoutRel = { testField: "boop" };
        await expect(schema.validate(withRel)).to.be.fulfilled;
        await expect(schema.validate(withoutRel)).to.be.fulfilled;
        await expect(schema.validate({})).to.be.rejected;
        // Doesn't strip optional relationships
        expect(schema.cast(withRel, { stripUnknown: true })).to.deep.equal(withRel);
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
        const withBothRels = { dummyRelCreate: { dummyField: "test" }, dummyRelUpdate: { dummyField: "test" } };
        const withCreateRel = { dummyRelCreate: { dummyField: "test" } };
        const withUpdateRel = { dummyRelUpdate: { dummyField: "test" } };
        const withoutRel = {};
        await expect(schema.validate(withBothRels)).to.be.fulfilled;
        await expect(schema.validate(withCreateRel)).to.be.fulfilled;
        await expect(schema.validate(withUpdateRel)).to.be.fulfilled;
        await expect(schema.validate(withoutRel)).to.be.fulfilled;
        expect(schema.cast(withBothRels, { stripUnknown: true })).to.deep.equal(withBothRels);
        expect(schema.cast(withCreateRel, { stripUnknown: true })).to.deep.equal(withCreateRel);
    });
    it("should handle one-to-many required Create relationships correctly", async () => {
        const relSchema = createDummyYupModel("dummyField");
        const schema = yupObj(
            { testField: yup.string().required() },
            // @ts-ignore: Testing runtime scenario
            [["dummyRel", ["Create"], "many", "req", relSchema]],
            [],
            {},
        );
        const withoutRel = { testField: "boop" };
        const withRel = { testField: "boop", dummyRelCreate: [{ dummyField: "test" }] };
        await expect(schema.validate(withoutRel)).to.be.rejected;
        await expect(schema.validate(withRel)).to.be.fulfilled;
        expect(schema.cast(withRel, { stripUnknown: true })).to.deep.equal(withRel);
    });

    it("should handle relationship with omitted fields - test1", async () => {
        const relSchema = createDummyYupModel("dummyField");
        const schema = yupObj(
            { testField: yup.string().required() },
            // @ts-ignore: Testing runtime scenario
            [["dummyRel", ["Create"], "one", "req", relSchema, ["dummyField"]]],
            [],
            {},
        );
        const unstripped = { testField: "boop", dummyRelCreate: { dummyField: "test", anotherField: "yeet" } };
        const stripped = { testField: "boop", dummyRelCreate: {} };
        // Validation should pass when extra fields are provided
        await expect(schema.validate(unstripped)).to.be.fulfilled;
        await expect(schema.validate(stripped)).to.be.fulfilled;
        // Casting should strip extra fields
        expect(schema.cast(unstripped, { stripUnknown: true })).to.deep.equal(stripped);
        expect(schema.cast(stripped, { stripUnknown: true })).to.deep.equal(stripped);
    });
    it("should handle relationship with omitted fields - test2", async () => {
        const relSchema = createDummyYupModel("dummyField1", "dummyField2");
        const schema = yupObj(
            { testField: yup.string().required() },
            // @ts-ignore: Testing runtime scenario
            [["dummyRel", ["Create"], "one", "req", relSchema, ["dummyField1"]]],
            [],
            {},
        );
        const unstripped = { testField: "boop", dummyRelCreate: { dummyField1: "test", dummyField2: "yeet" } };
        const strippedCorrectly = { testField: "boop", dummyRelCreate: { dummyField2: "yeet" } };
        const strippedIncorrectly = { testField: "boop", dummyRelCreate: { dummyField1: "test" } };
        // Validation should pass when extra fields are provided
        await expect(schema.validate(unstripped)).to.be.fulfilled;
        await expect(schema.validate(strippedCorrectly)).to.be.fulfilled;
        await expect(schema.validate(strippedIncorrectly)).to.be.rejected;
        // Casting should strip extra fields
        expect(schema.cast(unstripped, { stripUnknown: true })).to.deep.equal(strippedCorrectly);
        expect(schema.cast(strippedCorrectly, { stripUnknown: true })).to.deep.equal(strippedCorrectly);
    });
    it("should handle relationship with omitted fields - test3", async () => {
        const grandchildSchema = createDummyYupModel("dummyField1", "dummyField2");
        const childSchema = {
            create: (d) => yupObj(
                { testField: yup.string().required() },
                // @ts-ignore: Testing runtime scenario
                [["grandchild", ["Create", "Update"], "one", "req", grandchildSchema, ["dummyField1"]]],
                [],
                d,
            ),
            update: (d) => yupObj(
                { testField: yup.string().required() },
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
        const unstripped = { childCreate: { testField: "boop", grandchildCreate: { dummyField1: "test", dummyField2: "yeet" } } };
        const strippedCorrectly = { childCreate: { testField: "boop" } };
        const strippedIncorrectly = { childCreate: { testField: "boop", grandchildCreate: { dummyField1: "test" } } };
        // Validation should pass when extra fields are provided
        await expect(parentSchema.validate(unstripped)).to.be.fulfilled;
        await expect(parentSchema.validate(strippedCorrectly)).to.be.fulfilled;
        await expect(parentSchema.validate(strippedIncorrectly)).to.be.fulfilled;
        // Casting should strip extra fields
        expect(parentSchema.cast(unstripped, { stripUnknown: true })).to.deep.equal(strippedCorrectly);
        expect(parentSchema.cast(strippedCorrectly, { stripUnknown: true })).to.deep.equal(strippedCorrectly);
    });

    it("should enforce exclusion pairs on primitive fields - test 1", async () => {
        const schema = yupObj(
            {
                field1: opt(yup.string()),
                field2: opt(yup.string()),
            },
            [],
            [["field1", "field2", true]],
            {},
        );
        const bothFields = { field1: "test", field2: "test" };
        const field1Only = { field1: "test" };
        const field2Only = { field2: "test" };
        const noFields = {};
        await expect(schema.validate(bothFields)).to.be.rejected;
        await expect(schema.validate(field1Only)).to.be.fulfilled;
        await expect(schema.validate(field2Only)).to.be.fulfilled;
        await expect(schema.validate(noFields)).to.be.rejected;
        expect(schema.cast(field1Only, { stripUnknown: true })).to.deep.equal(field1Only);
        expect(schema.cast(field2Only, { stripUnknown: true })).to.deep.equal(field2Only);
    });
    it("should enforce exclusion pairs on primitive fields - test 2", async () => {
        const schema = yupObj(
            {
                field1: opt(yup.string()),
                field2: opt(yup.string()),
            },
            [],
            [["field1", "field2", false]],
            {},
        );
        const bothFields = { field1: "test", field2: "test" };
        const field1Only = { field1: "test" };
        const field2Only = { field2: "test" };
        const noFields = {};
        await expect(schema.validate(bothFields)).to.be.rejected;
        await expect(schema.validate(field1Only)).to.be.fulfilled;
        await expect(schema.validate(field2Only)).to.be.fulfilled;
        await expect(schema.validate(noFields)).to.be.fulfilled;
        expect(schema.cast(field1Only, { stripUnknown: true })).to.deep.equal(field1Only);
        expect(schema.cast(field2Only, { stripUnknown: true })).to.deep.equal(field2Only);
    });
    it("should enforce exclusion pairs on relationship fields - test1", async () => {
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
        const bothFields = { rel1Create: { dummyField: "test" }, rel2Create: { dummyField: "test" } };
        const field1Only = { rel1Create: { dummyField: "test" } };
        const field2Only = { rel2Create: { dummyField: "test" } };
        const noFields = {};
        await expect(schema.validate(bothFields)).to.be.rejected;
        await expect(schema.validate(field1Only)).to.be.fulfilled;
        await expect(schema.validate(field2Only)).to.be.fulfilled;
        await expect(schema.validate(noFields)).to.be.rejected;
        expect(schema.cast(field1Only, { stripUnknown: true })).to.deep.equal(field1Only);
        expect(schema.cast(field2Only, { stripUnknown: true })).to.deep.equal(field2Only);
    });
    it("should enforce exclusion pairs on relationship fields - test2", async () => {
        const relSchema = createDummyYupModel("dummyField");
        const schema = yupObj(
            {},
            [
                // @ts-ignore: Testing runtime scenario
                ["rel1", ["Create", "Update"], "one", "opt", relSchema],
            ],
            [["rel1Create", "rel1Update", true]],
            {},
        );
        const bothFields = { rel1Create: { dummyField: "test" }, rel1Update: { dummyField: "test" } };
        const field1Only = { rel1Create: { dummyField: "test" } };
        const field2Only = { rel1Update: { dummyField: "test" } };
        const noFields = {};
        await expect(schema.validate(bothFields)).to.be.rejected;
        await expect(schema.validate(field1Only)).to.be.fulfilled;
        await expect(schema.validate(field2Only)).to.be.fulfilled;
        await expect(schema.validate(noFields)).to.be.rejected;
        expect(schema.cast(field1Only, { stripUnknown: true })).to.deep.equal(field1Only);
        expect(schema.cast(field2Only, { stripUnknown: true })).to.deep.equal(field2Only);
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
        const bothFields = { field1: "test", rel1Create: { dummyField: "test" } };
        const field1Only = { field1: "test" };
        const field2Only = { rel1Create: { dummyField: "test" } };
        const noFields = {};
        await expect(schema.validate(bothFields)).to.be.rejected;
        await expect(schema.validate(field1Only)).to.be.fulfilled;
        await expect(schema.validate(field2Only)).to.be.fulfilled;
        await expect(schema.validate(noFields)).to.be.rejected;
        expect(schema.cast(field1Only, { stripUnknown: true })).to.deep.equal(field1Only);
        expect(schema.cast(field2Only, { stripUnknown: true })).to.deep.equal(field2Only);
    });

    // Test omitting relationships with action modifiers
    it("should omit specified relationships", async () => {
        const relSchema = createDummyYupModel("dummyField");
        const schema = yupObj(
            {},
            // @ts-ignore: Testing runtime scenario
            [["dummyRel", ["Create"], "one", "req", relSchema]],
            [],
            { omitFields: ["dummyRelCreate"] },
        );
        const unstripped = { dummyRelCreate: { dummyField: "test", anotherField: "yeet" } };
        const stripped = {};
        await expect(schema.validate(unstripped)).to.be.fulfilled;
        await expect(schema.validate(stripped)).to.be.fulfilled;
        expect(schema.cast(unstripped, { stripUnknown: true })).to.deep.equal(stripped);
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
        const data = { testField: "boop", dummyRelConnect: uuid(), dummyRelCreate: { dummyField: "test" } };
        await expect(schema.validate(data)).to.be.fulfilled;
        expect(schema.cast(data, { stripUnknown: true })).to.deep.equal(data);
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
        const data = { testField: "boop", dummyRelConnect: [uuid()], dummyRelCreate: [{ dummyField: "test" }] };
        await expect(schema.validate(data)).to.be.fulfilled;
        expect(schema.cast(data, { stripUnknown: true })).to.deep.equal(data);
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
        const data = { testField: "boop", dummyRelDelete: true, dummyRelDisconnect: true };
        await expect(schema.validate(data)).to.be.fulfilled;
        expect(schema.cast(data, { stripUnknown: true })).to.deep.equal(data);
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
        const data = { testField: "boop", dummyRelDelete: [uuid()], dummyRelDisconnect: [uuid()] };
        await expect(schema.validate(data)).to.be.fulfilled;
        expect(schema.cast(data, { stripUnknown: true })).to.deep.equal(data);
    });
});
