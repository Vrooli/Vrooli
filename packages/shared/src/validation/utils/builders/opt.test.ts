import * as yup from "yup";
import { opt } from "./opt";

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
