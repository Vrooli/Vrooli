/* eslint-disable @typescript-eslint/ban-ts-comment */
import * as yup from "yup";
import { uuid } from "../../../id";
import { opt } from "./opt";
import { yupObj } from "./yupObj"; // Update with the actual path

describe("yupObj", () => {
    // Helper function to create a dummy YupModel
    const createDummyYupModel = (...fields: string[]) => ({
        create: (d) => yupObj({
            ...fields.reduce((acc, curr) => ({ ...acc, [curr]: yup.string().required() }), {}),
        }, [], [], d),
        update: (d) => yupObj({
            ...fields.reduce((acc, curr) => ({ ...acc, [curr]: yup.string().required() }), {}),
        }, [], [], d),
    });

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
        await expect(schema.validate(stripped)).resolves.toEqual(stripped);
        await expect(schema.validate({})).rejects.toThrow();
        await expect(schema.validate({ testField: 1 })).resolves.toEqual({ testField: "1" });
        // Extra fields are kept in validation, but stripped in casting
        await expect(schema.validate(unstripped)).resolves.toEqual(unstripped);
        expect(schema.cast(unstripped, { stripUnknown: true })).toEqual(stripped);
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
        await expect(schema.validate(unstripped)).resolves.toEqual(unstripped);
        await expect(schema.validate({})).rejects.toThrow();
        expect(schema.cast(unstripped, { stripUnknown: true })).toEqual(stripped);
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
        await expect(schema.validate(withRel)).resolves.toEqual(withRel);
        await expect(schema.validate(withoutRel)).resolves.toEqual(withoutRel);
        await expect(schema.validate({})).rejects.toThrow();
        // Doesn't strip optional relationships
        expect(schema.cast(withRel, { stripUnknown: true })).toEqual(withRel);
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
        await expect(schema.validate(withBothRels)).resolves.toEqual(withBothRels);
        await expect(schema.validate(withCreateRel)).resolves.toEqual(withCreateRel);
        await expect(schema.validate(withUpdateRel)).resolves.toEqual(withUpdateRel);
        await expect(schema.validate(withoutRel)).resolves.toEqual(withoutRel);
        expect(schema.cast(withBothRels, { stripUnknown: true })).toEqual(withBothRels);
        expect(schema.cast(withCreateRel, { stripUnknown: true })).toEqual(withCreateRel);
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
        await expect(schema.validate(withoutRel)).rejects.toThrow();
        await expect(schema.validate(withRel)).resolves.toEqual(withRel);
        expect(schema.cast(withRel, { stripUnknown: true })).toEqual(withRel);
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
        await expect(schema.validate(unstripped)).resolves.toEqual(unstripped);
        await expect(schema.validate(stripped)).resolves.toEqual(stripped);
        // Casting should strip extra fields
        expect(schema.cast(unstripped, { stripUnknown: true })).toEqual(stripped);
        expect(schema.cast(stripped, { stripUnknown: true })).toEqual(stripped);
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
        await expect(schema.validate(unstripped)).resolves.toEqual(unstripped);
        await expect(schema.validate(strippedCorrectly)).resolves.toEqual(strippedCorrectly);
        await expect(schema.validate(strippedIncorrectly)).rejects.toThrow();
        // Casting should strip extra fields
        expect(schema.cast(unstripped, { stripUnknown: true })).toEqual(strippedCorrectly);
        expect(schema.cast(strippedCorrectly, { stripUnknown: true })).toEqual(strippedCorrectly);
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
        await expect(parentSchema.validate(unstripped)).resolves.toEqual(unstripped);
        await expect(parentSchema.validate(strippedCorrectly)).resolves.toEqual(strippedCorrectly);
        await expect(parentSchema.validate(strippedIncorrectly)).resolves.toEqual(strippedIncorrectly);
        // Casting should strip extra fields
        expect(parentSchema.cast(unstripped, { stripUnknown: true })).toEqual(strippedCorrectly);
        expect(parentSchema.cast(strippedCorrectly, { stripUnknown: true })).toEqual(strippedCorrectly);
    });

    it("should enforce exclusion pairs on primitive fields", async () => {
        const schema = yupObj(
            {
                field1: opt(yup.string()),
                field2: opt(yup.string()),
            },
            [],
            [["field1", "field2"]],
            {},
        );
        const bothFields = { field1: "test", field2: "test" };
        const field1Only = { field1: "test" };
        const field2Only = { field2: "test" };
        const noFields = {};
        await expect(schema.validate(bothFields)).rejects.toThrow();
        await expect(schema.validate(field1Only)).resolves.toEqual(field1Only);
        await expect(schema.validate(field2Only)).resolves.toEqual(field2Only);
        await expect(schema.validate(noFields)).rejects.toThrow();
        expect(schema.cast(field1Only, { stripUnknown: true })).toEqual(field1Only);
        expect(schema.cast(field2Only, { stripUnknown: true })).toEqual(field2Only);
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
            [["rel1Create", "rel2Create"]],
            {},
        );
        const bothFields = { rel1Create: { dummyField: "test" }, rel2Create: { dummyField: "test" } };
        const field1Only = { rel1Create: { dummyField: "test" } };
        const field2Only = { rel2Create: { dummyField: "test" } };
        const noFields = {};
        await expect(schema.validate(bothFields)).rejects.toThrow();
        await expect(schema.validate(field1Only)).resolves.toEqual(field1Only);
        await expect(schema.validate(field2Only)).resolves.toEqual(field2Only);
        await expect(schema.validate(noFields)).rejects.toThrow();
        expect(schema.cast(field1Only, { stripUnknown: true })).toEqual(field1Only);
        expect(schema.cast(field2Only, { stripUnknown: true })).toEqual(field2Only);
    });
    it("should enforce exclusion pairs on relationship fields - test2", async () => {
        const relSchema = createDummyYupModel("dummyField");
        const schema = yupObj(
            {},
            [
                // @ts-ignore: Testing runtime scenario
                ["rel1", ["Create", "Update"], "one", "opt", relSchema],
            ],
            [["rel1Create", "rel1Update"]],
            {},
        );
        const bothFields = { rel1Create: { dummyField: "test" }, rel1Update: { dummyField: "test" } };
        const field1Only = { rel1Create: { dummyField: "test" } };
        const field2Only = { rel1Update: { dummyField: "test" } };
        const noFields = {};
        await expect(schema.validate(bothFields)).rejects.toThrow();
        await expect(schema.validate(field1Only)).resolves.toEqual(field1Only);
        await expect(schema.validate(field2Only)).resolves.toEqual(field2Only);
        await expect(schema.validate(noFields)).rejects.toThrow();
        expect(schema.cast(field1Only, { stripUnknown: true })).toEqual(field1Only);
        expect(schema.cast(field2Only, { stripUnknown: true })).toEqual(field2Only);
    });
    it("should enforce exclusion pairs on relationship/primitive field pairs", async () => {
        const relSchema = createDummyYupModel("dummyField");
        const schema = yupObj(
            { field1: opt(yup.string()) },
            [
                // @ts-ignore: Testing runtime scenario
                ["rel1", ["Create"], "one", "opt", relSchema],
            ],
            [["field1", "rel1Create"]],
            {},
        );
        const bothFields = { field1: "test", rel1Create: { dummyField: "test" } };
        const field1Only = { field1: "test" };
        const field2Only = { rel1Create: { dummyField: "test" } };
        const noFields = {};
        await expect(schema.validate(bothFields)).rejects.toThrow();
        await expect(schema.validate(field1Only)).resolves.toEqual(field1Only);
        await expect(schema.validate(field2Only)).resolves.toEqual(field2Only);
        await expect(schema.validate(noFields)).rejects.toThrow();
        expect(schema.cast(field1Only, { stripUnknown: true })).toEqual(field1Only);
        expect(schema.cast(field2Only, { stripUnknown: true })).toEqual(field2Only);
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
        await expect(schema.validate(unstripped)).resolves.toEqual(unstripped);
        await expect(schema.validate(stripped)).resolves.toEqual(stripped);
        expect(schema.cast(unstripped, { stripUnknown: true })).toEqual(stripped);
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
        await expect(schema.validate(data)).resolves.toEqual(data);
        expect(schema.cast(data, { stripUnknown: true })).toEqual(data);
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
        await expect(schema.validate(data)).resolves.toEqual(data);
        expect(schema.cast(data, { stripUnknown: true })).toEqual(data);
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
        await expect(schema.validate(data)).resolves.toEqual(data);
        expect(schema.cast(data, { stripUnknown: true })).toEqual(data);
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
        await expect(schema.validate(data)).resolves.toEqual(data);
        expect(schema.cast(data, { stripUnknown: true })).toEqual(data);
    });
});
