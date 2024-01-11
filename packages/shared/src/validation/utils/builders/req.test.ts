import * as yup from "yup";
import { reqErr } from "../errors";
import { req } from "./req";

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
