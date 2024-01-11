/* eslint-disable @typescript-eslint/ban-ts-comment */
/* eslint-disable @typescript-eslint/no-non-null-assertion */
import * as yup from "yup";
import { uuid } from "../../../id";
import { YupModel } from "../types";
import { rel } from "./rel";
import { yupObj } from "./yupObj";

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
        }).toThrow();

        expect(() => {
            rel(mockData, "testRelation", ["Update"], "one", "opt");
        }).toThrow();
    });

    it("should return an object with the correct keys for given relation types", () => {
        const result = rel(mockData, "testRelation", ["Connect", "Create", "Delete", "Disconnect", "Update"], "one", "opt", mockModel);
        expect(Object.keys(result)).toEqual(expect.arrayContaining(["testRelationConnect", "testRelationCreate", "testRelationDelete", "testRelationDisconnect", "testRelationUpdate"]));
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
        await expect(oneResult.testRelationConnect!.validate(validOneData.testRelationConnect)).resolves.toBeTruthy();
        await expect(oneResult.testRelationConnect!.validate(invalidOneData.testRelationConnect)).rejects.toThrow();

        // Validate 'many' relationship
        await expect(manyResult.testRelationConnect!.validate(validManyData.testRelationConnect)).resolves.toBeTruthy();
        await expect(manyResult.testRelationConnect!.validate(invalidManyData.testRelationConnect)).rejects.toThrow();
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
        await expect(oneResult.testRelationCreate!.validate(validOneData.testRelationCreate)).resolves.toBeTruthy();
        await expect(oneResult.testRelationCreate!.validate(invalidOneData.testRelationCreate)).rejects.toThrow();

        // Validate 'many' relationship
        await expect(manyResult.testRelationCreate!.validate(validManyData.testRelationCreate)).resolves.toBeTruthy();
        await expect(manyResult.testRelationCreate!.validate(invalidManyData.testRelationCreate)).rejects.toThrow();
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
        await expect(oneResult.testRelationUpdate!.validate(validOneData.testRelationUpdate)).resolves.toBeTruthy();
        await expect(oneResult.testRelationUpdate!.validate(invalidOneData.testRelationUpdate)).rejects.toThrow();

        // Validate 'many' relationship
        await expect(manyResult.testRelationUpdate!.validate(validManyData.testRelationUpdate)).resolves.toBeTruthy();
        await expect(manyResult.testRelationUpdate!.validate(invalidManyData.testRelationUpdate)).rejects.toThrow();
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
        await expect(oneResult.testRelationDelete!.validate(validOneData.testRelationDelete)).resolves.toBeTruthy();
        await expect(oneResult.testRelationDelete!.validate(invalidOneData1.testRelationDelete)).rejects.toThrow();
        await expect(oneResult.testRelationDelete!.validate(invalidOneData2.testRelationDelete)).rejects.toThrow();
        await expect(oneResult.testRelationDelete!.validate(invalidOneData3.testRelationDelete)).rejects.toThrow();

        // Validate 'many' relationship
        await expect(manyResult.testRelationDelete!.validate(validManyData.testRelationDelete)).resolves.toBeTruthy();
        await expect(manyResult.testRelationDelete!.validate(invalidManyData1.testRelationDelete)).rejects.toThrow();
        await expect(manyResult.testRelationDelete!.validate(invalidManyData2.testRelationDelete)).rejects.toThrow();
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
        await expect(oneResult.testRelationDisconnect!.validate(validOneData.testRelationDisconnect)).resolves.toBeTruthy();
        await expect(oneResult.testRelationDisconnect!.validate(invalidOneData1.testRelationDisconnect)).rejects.toThrow();
        await expect(oneResult.testRelationDisconnect!.validate(invalidOneData2.testRelationDisconnect)).rejects.toThrow();
        await expect(oneResult.testRelationDisconnect!.validate(invalidOneData3.testRelationDisconnect)).rejects.toThrow();

        // Validate 'many' relationship
        await expect(manyResult.testRelationDisconnect!.validate(validManyData.testRelationDisconnect)).resolves.toBeTruthy();
        await expect(manyResult.testRelationDisconnect!.validate(invalidManyData1.testRelationDisconnect)).rejects.toThrow();
        await expect(manyResult.testRelationDisconnect!.validate(invalidManyData2.testRelationDisconnect)).rejects.toThrow();
    });

    it("should mark 'Connect' field as required when isRequired is 'req'", async () => {
        const result = rel(mockData, "testRelation", ["Connect"], "one", "req");
        const testSchema = yup.object().shape({
            // @ts-ignore: Testing runtime scenario
            testRelationConnect: result.testRelationConnect,
        });

        // Test with undefined value
        await expect(testSchema.validate({})).rejects.toThrow();

        // Test with valid value
        const validUuid = uuid(); // Assuming uuid() returns a valid UUID
        await expect(testSchema.validate({ testRelationConnect: validUuid })).resolves.toEqual({ testRelationConnect: validUuid });
    });

    it("should mark 'Connect' field as optional when isRequired is 'opt'", async () => {
        const result = rel(mockData, "testRelation", ["Connect"], "one", "opt");
        const testSchema = yup.object().shape({
            // @ts-ignore: Testing runtime scenario
            testRelationConnect: result.testRelationConnect,
        });

        // Test with undefined value
        await expect(testSchema.validate({})).resolves.toEqual({});

        // Test with valid value
        const validUuid = uuid();
        await expect(testSchema.validate({ testRelationConnect: validUuid })).resolves.toEqual({ testRelationConnect: validUuid });
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

        await expect(result.testRelationCreate!.validate(validData.testRelationCreate)).resolves.toBeTruthy();
        await expect(result.testRelationCreate!.validate(invalidData.testRelationCreate)).rejects.toThrow();
    });

    it("should require only the field not in omitFields", async () => {
        const result = rel({ omitFields: [] }, "testRelation", ["Create"], "one", "req", omitFieldsMockModel, ["field1", "nonExistingField"]);

        const validData = { testRelationCreate: { field2: "value2" } }; // Only field2 is required
        const invalidData = { testRelationCreate: {} }; // Missing field2

        await expect(result.testRelationCreate!.validate(validData.testRelationCreate)).resolves.toBeTruthy();
        await expect(result.testRelationCreate!.validate(invalidData.testRelationCreate)).rejects.toThrow();
    });

    it("should not require any fields when all are in omitFields", async () => {
        const result = rel({ omitFields: [] }, "testRelation", ["Create"], "one", "req", omitFieldsMockModel, ["field1", "field2"]);

        const validData = { testRelationCreate: {} }; // No fields are required

        await expect(result.testRelationCreate!.validate(validData.testRelationCreate)).resolves.toBeTruthy();
    });

    it("should work when using data.omitFields", async () => {
        const result = rel({ omitFields: ["field1"] }, "testRelation", ["Create"], "one", "req", omitFieldsMockModel, ["field2"]);

        const validData = { testRelationCreate: {} }; // No fields are required

        await expect(result.testRelationCreate!.validate(validData.testRelationCreate)).resolves.toBeTruthy();
    });

});
