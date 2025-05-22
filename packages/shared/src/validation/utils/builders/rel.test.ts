/* eslint-disable @typescript-eslint/ban-ts-comment */
/* eslint-disable @typescript-eslint/no-non-null-assertion */
import { expect } from "chai";
import * as yup from "yup";
import { uuid } from "../../../id/uuid.js";
import { type YupModel } from "../types.js";
import { rel } from "./rel.js";
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
        }).to.throw();

        expect(() => {
            rel(mockData, "testRelation", ["Update"], "one", "opt");
        }).to.throw();
    });

    it("should return an object with the correct keys for given relation types", () => {
        const result = rel(mockData, "testRelation", ["Connect", "Create", "Delete", "Disconnect", "Update"], "one", "opt", mockModel);
        expect(Object.keys(result)).to.have.members(["testRelationConnect", "testRelationCreate", "testRelationDelete", "testRelationDisconnect", "testRelationUpdate"]);
    });

    it("should handle \"one\" and \"many\" relationships correctly for Connect", async () => {
        const oneResult = rel(mockData, "testRelation", ["Connect"], "one", "req");
        const manyResult = rel(mockData, "testRelation", ["Connect"], "many", "req");

        // Prepare test data
        const validOneData = { testRelationConnect: uuid() };
        const validManyData = { testRelationConnect: [uuid(), uuid()] };
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
        const validOneData = { testRelationDelete: true };
        const validManyData = { testRelationDelete: [uuid(), uuid()] };
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
        const validOneData = { testRelationDisconnect: true };
        const validManyData = { testRelationDisconnect: [uuid(), uuid()] };
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
        const validUuid = uuid();
        const validationResult = await testSchema.validate({ testRelationConnect: validUuid });
        expect(validationResult).to.deep.equal({ testRelationConnect: validUuid });
    });

    it("should mark 'Connect' field as optional when isRequired is 'opt'", async () => {
        const result = rel(mockData, "testRelation", ["Connect"], "one", "opt");
        const testSchema = yup.object().shape({
            // @ts-ignore: Testing runtime scenario
            testRelationConnect: result.testRelationConnect,
        });

        // Test with undefined value
        const emptyValidationResult = await testSchema.validate({});
        expect(emptyValidationResult).to.deep.equal({});

        // Test with valid value
        const validUuid = uuid();
        const validationResult = await testSchema.validate({ testRelationConnect: validUuid });
        expect(validationResult).to.deep.equal({ testRelationConnect: validUuid });
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

});
