import { describe, it } from "mocha";
import { expect } from "chai";

describe("Simple Test", () => {
    it("should work", () => {
        expect(1 + 1).to.equal(2);
    });
});