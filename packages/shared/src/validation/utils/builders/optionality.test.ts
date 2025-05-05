import { expect } from "chai";
import * as yup from "yup";
import { opt, optArr, req, reqArr } from "./optionality.js";

describe("opt function", () => {
    it("should make a string field optional, nullable, and with a default of undefined", async () => {
        const schema = yup.object({
            name: opt(yup.string()),
        }).strict();

        expect(await schema.validate({}, { stripUnknown: true, abortEarly: false })).to.deep.equal({});
        expect(await schema.validate({ name: null })).to.deep.equal({ name: null });
        expect(await schema.validate({ name: "John" })).to.deep.equal({ name: "John" });
    });

    it("should make a number field optional, nullable, and with a default of undefined", async () => {
        const schema = yup.object({
            age: opt(yup.number()),
        }).strict();

        expect(await schema.validate({}, { stripUnknown: true, abortEarly: false })).to.deep.equal({});
        expect(await schema.validate({ age: null })).to.deep.equal({ age: null });
        expect(await schema.validate({ age: 30 })).to.deep.equal({ age: 30 });
    });

    it("should make a boolean field optional, nullable, and with a default of undefined", async () => {
        const schema = yup.object({
            active: opt(yup.boolean()),
        }).strict();

        expect(await schema.validate({}, { stripUnknown: true, abortEarly: false })).to.deep.equal({});
        expect(await schema.validate({ active: null })).to.deep.equal({ active: null });
        expect(await schema.validate({ active: true })).to.deep.equal({ active: true });
    });

    it("should work with date fields", async () => {
        const schema = yup.object({
            birthDate: opt(yup.date()),
        }).strict();

        const date = new Date();
        expect(await schema.validate({}, { stripUnknown: true, abortEarly: false })).to.deep.equal({});
        expect(await schema.validate({ birthDate: null })).to.deep.equal({ birthDate: null });
        expect(await schema.validate({ birthDate: date })).to.deep.equal({ birthDate: date });
    });

    // Add more tests for other types of fields if necessary
});

describe("optArr function", () => {
    it("should create an optional array of strings with required contents", async () => {
        const schema = yup.object({
            tags: optArr(yup.string().required()),
        }).strict();

        expect(await schema.validate({}, { stripUnknown: true, abortEarly: false })).to.deep.equal({});
        expect(await schema.validate({ tags: ["tag1", "tag2"] })).to.deep.equal({ tags: ["tag1", "tag2"] });

        try {
            await schema.validate({ tags: ["tag1", "", "tag2"] });
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }
    });

    it("should create an optional array of numbers with required contents", async () => {
        const schema = yup.object({
            numbers: optArr(yup.number().required()),
        }).strict();

        expect(await schema.validate({}, { stripUnknown: true, abortEarly: false })).to.deep.equal({});
        expect(await schema.validate({ numbers: [1, 2, 3] })).to.deep.equal({ numbers: [1, 2, 3] });

        try {
            await schema.validate({ numbers: [1, null, 3] });
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }
    });

    it("should create an optional array of booleans with required contents", async () => {
        const schema = yup.object({
            flags: optArr(yup.boolean().required()),
        }).strict();

        expect(await schema.validate({}, { stripUnknown: true, abortEarly: false })).to.deep.equal({});
        expect(await schema.validate({ flags: [true, false] })).to.deep.equal({ flags: [true, false] });

        try {
            await schema.validate({ flags: [true, null, false] });
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }
    });

});

describe("req function", () => {
    it("should make a string field required with custom error", async () => {
        const schema = yup.object({
            name: req(yup.string()),
        });

        try {
            await schema.validate({});
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }

        expect(await schema.validate({ name: "John" })).to.deep.equal({ name: "John" });
    });

    it("should make a number field required with custom error", async () => {
        const schema = yup.object({
            age: req(yup.number()),
        });

        try {
            await schema.validate({});
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }

        expect(await schema.validate({ age: 30 })).to.deep.equal({ age: 30 });
    });

    it("should make a boolean field required with custom error", async () => {
        const schema = yup.object({
            active: req(yup.boolean()),
        });

        try {
            await schema.validate({});
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }

        expect(await schema.validate({ active: true })).to.deep.equal({ active: true });
    });

    it("should work with date fields", async () => {
        const schema = yup.object({
            birthDate: req(yup.date()),
        });

        const date = new Date();

        try {
            await schema.validate({});
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }

        expect(await schema.validate({ birthDate: date })).to.deep.equal({ birthDate: date });
    });
});

describe("reqArr function", () => {
    it("should require an array of strings with required contents", async () => {
        const schema = yup.object({
            tags: reqArr(yup.string().required()),
        });

        try {
            await schema.validate({});
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }

        expect(await schema.validate({ tags: [] })).to.deep.equal({ tags: [] });
        expect(await schema.validate({ tags: ["tag1", "tag2"] })).to.deep.equal({ tags: ["tag1", "tag2"] });

        try {
            await schema.validate({ tags: ["tag1", "", "tag2"] });
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }
    });

    it("should require an array of numbers with required contents", async () => {
        const schema = yup.object({
            numbers: reqArr(yup.number().required()),
        });

        try {
            await schema.validate({});
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }

        expect(await schema.validate({ numbers: [] })).to.deep.equal({ numbers: [] });
        expect(await schema.validate({ numbers: [1, 2, 3] })).to.deep.equal({ numbers: [1, 2, 3] });

        try {
            await schema.validate({ numbers: [1, null, 3] });
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }
    });

    it("should require an array of booleans with required contents", async () => {
        const schema = yup.object({
            flags: reqArr(yup.boolean().required()),
        });

        try {
            await schema.validate({});
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }

        expect(await schema.validate({ flags: [] })).to.deep.equal({ flags: [] });
        expect(await schema.validate({ flags: [true, false] })).to.deep.equal({ flags: [true, false] });

        try {
            await schema.validate({ flags: [true, null, false] });
            expect.fail("Validation should have failed but passed");
        } catch (error) {
            expect(error).to.be.an.instanceOf(Error);
        }
    });
});
