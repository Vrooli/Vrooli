import * as yup from "yup";
import { reqArr } from "./reqArr";

describe("reqArr function", () => {
    it("should require an array of strings with required contents", async () => {
        const schema = yup.object({
            tags: reqArr(yup.string().required()),
        });

        await expect(schema.validate({})).rejects.toThrow();
        await expect(schema.validate({ tags: [] })).resolves.toEqual({ tags: [] });
        await expect(schema.validate({ tags: ["tag1", "tag2"] })).resolves.toEqual({ tags: ["tag1", "tag2"] });
        await expect(schema.validate({ tags: ["tag1", "", "tag2"] })).rejects.toThrow();
    });

    it("should require an array of numbers with required contents", async () => {
        const schema = yup.object({
            numbers: reqArr(yup.number().required()),
        });

        await expect(schema.validate({})).rejects.toThrow();
        await expect(schema.validate({ numbers: [] })).resolves.toEqual({ numbers: [] });
        await expect(schema.validate({ numbers: [1, 2, 3] })).resolves.toEqual({ numbers: [1, 2, 3] });
        await expect(schema.validate({ numbers: [1, null, 3] })).rejects.toThrow();
    });

    it("should require an array of booleans with required contents", async () => {
        const schema = yup.object({
            flags: reqArr(yup.boolean().required()),
        });

        await expect(schema.validate({})).rejects.toThrow();
        await expect(schema.validate({ flags: [] })).resolves.toEqual({ flags: [] });
        await expect(schema.validate({ flags: [true, false] })).resolves.toEqual({ flags: [true, false] });
        await expect(schema.validate({ flags: [true, null, false] })).rejects.toThrow();
    });
});
