import { expect } from "chai";
import * as yup from "yup";
import { reqErr } from "../errors.js";
import { opt, optArr, req, reqArr } from "./optionality.js";

describe("opt function", () => {
    it("should make a string field optional, nullable, and with a default of undefined", async () => {
        const schema = yup.object({
            name: opt(yup.string()),
        });

        await expect(schema.validate({})).resolves.toEqual({ name: undefined });
        await expect(schema.validate({ name: null })).resolves.toEqual({ name: null });
        await expect(schema.validate({ name: "John" })).resolves.toEqual({ name: "John" });
    });

    it("should make a number field optional, nullable, and with a default of undefined", async () => {
        const schema = yup.object({
            age: opt(yup.number()),
        });

        await expect(schema.validate({})).resolves.toEqual({ age: undefined });
        await expect(schema.validate({ age: null })).resolves.toEqual({ age: null });
        await expect(schema.validate({ age: 30 })).resolves.toEqual({ age: 30 });
    });

    it("should make a boolean field optional, nullable, and with a default of undefined", async () => {
        const schema = yup.object({
            active: opt(yup.boolean()),
        });

        await expect(schema.validate({})).resolves.toEqual({ active: undefined });
        await expect(schema.validate({ active: null })).resolves.toEqual({ active: null });
        await expect(schema.validate({ active: true })).resolves.toEqual({ active: true });
    });

    it("should work with date fields", async () => {
        const schema = yup.object({
            birthDate: opt(yup.date()),
        });

        const date = new Date();
        await expect(schema.validate({})).resolves.toEqual({ birthDate: undefined });
        await expect(schema.validate({ birthDate: null })).resolves.toEqual({ birthDate: null });
        await expect(schema.validate({ birthDate: date })).resolves.toEqual({ birthDate: date });
    });

    // Add more tests for other types of fields if necessary
});

describe("optArr function", () => {
    it("should create an optional array of strings with required contents", async () => {
        const schema = yup.object({
            tags: optArr(yup.string().required()),
        });

        await expect(schema.validate({})).resolves.toEqual({ tags: undefined });
        await expect(schema.validate({ tags: ["tag1", "tag2"] })).resolves.toEqual({ tags: ["tag1", "tag2"] });
        await expect(schema.validate({ tags: ["tag1", "", "tag2"] })).rejects.to.throw();
    });

    it("should create an optional array of numbers with required contents", async () => {
        const schema = yup.object({
            numbers: optArr(yup.number().required()),
        });

        await expect(schema.validate({})).resolves.toEqual({ numbers: undefined });
        await expect(schema.validate({ numbers: [1, 2, 3] })).resolves.toEqual({ numbers: [1, 2, 3] });
        await expect(schema.validate({ numbers: [1, null, 3] })).rejects.to.throw();
    });

    it("should create an optional array of booleans with required contents", async () => {
        const schema = yup.object({
            flags: optArr(yup.boolean().required()),
        });

        await expect(schema.validate({})).resolves.toEqual({ flags: undefined });
        await expect(schema.validate({ flags: [true, false] })).resolves.toEqual({ flags: [true, false] });
        await expect(schema.validate({ flags: [true, null, false] })).rejects.to.throw();
    });

});

describe("req function", () => {
    it("should make a string field required with custom error", async () => {
        const schema = yup.object({
            name: req(yup.string()),
        });

        await expect(schema.validate({})).rejects.toThrow(reqErr());
        await expect(schema.validate({ name: "John" })).resolves.toEqual({ name: "John" });
    });

    it("should make a number field required with custom error", async () => {
        const schema = yup.object({
            age: req(yup.number()),
        });

        await expect(schema.validate({})).rejects.toThrow(reqErr());
        await expect(schema.validate({ age: 30 })).resolves.toEqual({ age: 30 });
    });

    it("should make a boolean field required with custom error", async () => {
        const schema = yup.object({
            active: req(yup.boolean()),
        });

        await expect(schema.validate({})).rejects.toThrow(reqErr());
        await expect(schema.validate({ active: true })).resolves.toEqual({ active: true });
    });

    it("should work with date fields", async () => {
        const schema = yup.object({
            birthDate: req(yup.date()),
        });

        const date = new Date();
        await expect(schema.validate({})).rejects.toThrow(reqErr());
        await expect(schema.validate({ birthDate: date })).resolves.toEqual({ birthDate: date });
    });
});

describe("reqArr function", () => {
    it("should require an array of strings with required contents", async () => {
        const schema = yup.object({
            tags: reqArr(yup.string().required()),
        });

        await expect(schema.validate({})).rejects.to.throw();
        await expect(schema.validate({ tags: [] })).resolves.toEqual({ tags: [] });
        await expect(schema.validate({ tags: ["tag1", "tag2"] })).resolves.toEqual({ tags: ["tag1", "tag2"] });
        await expect(schema.validate({ tags: ["tag1", "", "tag2"] })).rejects.to.throw();
    });

    it("should require an array of numbers with required contents", async () => {
        const schema = yup.object({
            numbers: reqArr(yup.number().required()),
        });

        await expect(schema.validate({})).rejects.to.throw();
        await expect(schema.validate({ numbers: [] })).resolves.toEqual({ numbers: [] });
        await expect(schema.validate({ numbers: [1, 2, 3] })).resolves.toEqual({ numbers: [1, 2, 3] });
        await expect(schema.validate({ numbers: [1, null, 3] })).rejects.to.throw();
    });

    it("should require an array of booleans with required contents", async () => {
        const schema = yup.object({
            flags: reqArr(yup.boolean().required()),
        });

        await expect(schema.validate({})).rejects.to.throw();
        await expect(schema.validate({ flags: [] })).resolves.toEqual({ flags: [] });
        await expect(schema.validate({ flags: [true, false] })).resolves.toEqual({ flags: [true, false] });
        await expect(schema.validate({ flags: [true, null, false] })).rejects.to.throw();
    });
});
