import { expect } from "chai";
import { NIL, validate as uuidValidateLib } from "uuid";
import { DUMMY_ID, uuid, uuidValidate } from "./uuid.js";

describe("uuidFunctions", () => {
    describe("uuid", () => {
        it("generates a valid v4 UUID", () => {
            const id = uuid();
            expect(uuidValidate(id)).to.equal(true);
            expect(uuidValidateLib(id)).to.equal(true); // using the library's own validate function as a double-check
        });
    });

    describe("uuidValidate", () => {
        it("returns true for a valid v4 UUID", () => {
            const id = uuid();
            expect(uuidValidate(id)).to.equal(true);
        });

        it("returns false for an invalid v4 UUID", () => {
            const invalidUuids = [
                "12345678-1234-1234-1234-123456789abc", // invalid version
                "z2345678-1234-1234-1234-123456789abc", // non-hex character
                "", // empty string
                "12345678123412341234123456789abc", // missing dashes
                null, // null
                undefined, // undefined
                12345, // number
                {}, // object
            ];
            invalidUuids.forEach(invalidUuid => {
                expect(uuidValidate(invalidUuid)).to.equal(false);
            });
        });
    });

    describe("DUMMY_ID", () => {
        it("equals the NIL UUID", () => {
            expect(DUMMY_ID).to.deep.equal(NIL);
        });

        it("is considered a valid UUID", () => {
            expect(uuidValidate(DUMMY_ID)).to.equal(true);
        });
    });
});
