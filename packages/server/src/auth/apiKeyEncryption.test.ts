import { expect } from "chai";
import { ApiKeyEncryptionService } from "./apiKeyEncryption.js";
import { randomString } from "./codes.js";

// Define test cases
const testCases = [
    {
        test: "should encrypt and decrypt a standard alphanumeric key",
        plaintext: randomString(100, "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789 "),
    },
    {
        test: "should encrypt and decrypt a key with special characters",
        plaintext: randomString(100, "!@#$%^&*()_+-=[]{}|;:,.<>?/\\"),
    },
    {
        test: "should encrypt and decrypt emojis",
        plaintext: "👍🏻👍🏼👍🏽👍🏾👍🏿👎🏻👎🏼👎🏽👎🏾👎🏿👨🏻‍🤝‍👩🏾👨🏼‍🤝‍👨🏽‍🤝‍👨🏾‍🤝‍👨🏿‍🤝‍👩🏻👩🏼👩🏽👩🏾👩🏿✅",
    },
    {
        test: "should encrypt and decrypt accented characters and other languages",
        plaintext: "áéíóúàèéìòùäëïöüâêîôûåæøñçßðżźšžŸÀÁÂÄÇÈÉÊËÌÍÎÏÑÒÓÔÖÙÚÛÜÝÞßðżźšžŸŽŸ",
    },
    {
        test: "should encrypt and decrypt a long (but still valid) key",
        plaintext: randomString(1_024),
    },
    {
        test: "should handle empty strings",
        plaintext: "",
    },
];

describe("ApiKeyEncryptionService", () => {
    const service = ApiKeyEncryptionService.get();

    for (const testCase of testCases) {
        it(testCase.test, () => {
            const encrypted = service.encryptExternal(testCase.plaintext);
            const decrypted = service.decryptExternal(encrypted);
            expect(decrypted).to.equal(testCase.plaintext);
        });
    }

    it("should correctly validate keys", () => {
        expect(ApiKeyEncryptionService.isValidKey("valid")).to.be.true;
        expect(ApiKeyEncryptionService.isValidKey("")).to.be.false;
        expect(ApiKeyEncryptionService.isValidKey("a".repeat(1))).to.be.true;
        expect(ApiKeyEncryptionService.isValidKey("a".repeat(2049))).to.be.false;
        expect(ApiKeyEncryptionService.isValidKey("a".repeat(2048))).to.be.true;
    });
});
