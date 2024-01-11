import * as yup from "yup";
import { optArr } from "./optArr";

describe("optArr function", () => {
    it("should create an optional array of strings with required contents", async () => {
        const schema = yup.object({
            tags: optArr(yup.string().required()),
        });

        await expect(schema.validate({})).resolves.toEqual({ tags: undefined });
        await expect(schema.validate({ tags: ["tag1", "tag2"] })).resolves.toEqual({ tags: ["tag1", "tag2"] });
        await expect(schema.validate({ tags: ["tag1", "", "tag2"] })).rejects.toThrow();
    });

    it("should create an optional array of numbers with required contents", async () => {
        const schema = yup.object({
            numbers: optArr(yup.number().required()),
        });

        await expect(schema.validate({})).resolves.toEqual({ numbers: undefined });
        await expect(schema.validate({ numbers: [1, 2, 3] })).resolves.toEqual({ numbers: [1, 2, 3] });
        await expect(schema.validate({ numbers: [1, null, 3] })).rejects.toThrow();
    });

    it("should create an optional array of booleans with required contents", async () => {
        const schema = yup.object({
            flags: optArr(yup.boolean().required()),
        });

        await expect(schema.validate({})).resolves.toEqual({ flags: undefined });
        await expect(schema.validate({ flags: [true, false] })).resolves.toEqual({ flags: [true, false] });
        await expect(schema.validate({ flags: [true, null, false] })).rejects.toThrow();
    });

});
