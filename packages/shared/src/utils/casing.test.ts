import { expect } from "chai";
import { lowercaseFirstLetter, pascalCase, snakeCase, uppercaseFirstLetter } from "./casing.js";

describe("lowercaseFirstLetter", () => {
    it("should lowercase the first letter of a string", () => {
        expect(lowercaseFirstLetter("Hello")).to.equal("hello");
        expect(lowercaseFirstLetter("WORLD")).to.equal("wORLD");
    });

    it("should return the same string if the first letter is already lowercase", () => {
        expect(lowercaseFirstLetter("already")).to.equal("already");
    });

    it("should return an empty string if an empty string is provided", () => {
        expect(lowercaseFirstLetter("")).to.equal("");
    });

    it("should handle non-alphabetical characters", () => {
        expect(lowercaseFirstLetter("1hello")).to.equal("1hello");
        expect(lowercaseFirstLetter("@World")).to.equal("@World");
    });
});

describe("uppercaseFirstLetter", () => {
    it("should uppercase the first letter of a string", () => {
        expect(uppercaseFirstLetter("hello")).to.equal("Hello");
        expect(uppercaseFirstLetter("wORLD")).to.equal("WORLD");
    });

    it("should return the same string if the first letter is already uppercase", () => {
        expect(uppercaseFirstLetter("Already")).to.equal("Already");
    });

    it("should return an empty string if an empty string is provided", () => {
        expect(uppercaseFirstLetter("")).to.equal("");
    });

    it("should handle non-alphabetical characters", () => {
        expect(uppercaseFirstLetter("1hello")).to.equal("1hello");
        expect(uppercaseFirstLetter("@world")).to.equal("@world");
    });
});

describe("pascalCase", () => {
    it("should convert a string to pascalCase", () => {
        expect(pascalCase("hello-world")).to.equal("HelloWorld");
        expect(pascalCase("hello_world")).to.equal("HelloWorld");
        expect(pascalCase("hello world")).to.equal("HelloWorld");
        expect(pascalCase("helloWorld")).to.equal("HelloWorld");
        expect(pascalCase("HelloWorld")).to.equal("HelloWorld");
    });

    it("should return an empty string if an empty string is provided", () => {
        expect(pascalCase("")).to.equal("");
    });
});

describe("camelCase", () => {
    it("should convert a string to camelCase", () => {
        expect(pascalCase("hello-world")).to.equal("HelloWorld");
        expect(pascalCase("hello_world")).to.equal("HelloWorld");
        expect(pascalCase("hello world")).to.equal("HelloWorld");
        expect(pascalCase("helloWorld")).to.equal("HelloWorld");
        expect(pascalCase("HelloWorld")).to.equal("HelloWorld");
    });

    it("should return an empty string if an empty string is provided", () => {
        expect(pascalCase("")).to.equal("");
    });
});

describe("snakeCase", () => {
    it("should convert a string to snake_case", () => {
        expect(snakeCase("helloWorld")).to.equal("hello_world");
        expect(snakeCase("HelloWorld")).to.equal("hello_world");
        expect(snakeCase("hello-world")).to.equal("hello_world");
        expect(snakeCase("hello world")).to.equal("hello_world");
        expect(snakeCase("Hello World")).to.equal("hello_world");
    });

    // NOTE: This is why you shouldn't capitalize abbreviations
    it("should handle strings with multiple uppercase letters", () => {
        expect(snakeCase("HTTPServer")).to.equal("h_t_t_p_server");
        expect(snakeCase("URLParser")).to.equal("u_r_l_parser");
        expect(snakeCase("TheHttpServer")).to.equal("the_http_server");
    });

    it("should return the same string if it is already in snake_case", () => {
        expect(snakeCase("already_in_snake_case")).to.equal("already_in_snake_case");
    });

    it("should handle strings with non-alphabetic characters", () => {
        expect(snakeCase("hello1World")).to.equal("hello1_world");
        expect(snakeCase("123helloWorld")).to.equal("123hello_world");
    });

    it("should return an empty string if an empty string is provided", () => {
        expect(snakeCase("")).to.equal("");
    });

    it("should handle strings with special characters and spaces", () => {
        expect(snakeCase("hello-World")).to.equal("hello_world");
        expect(snakeCase("hello_world")).to.equal("hello_world");
        expect(snakeCase("hello world")).to.equal("hello_world");
        expect(snakeCase("hello@world")).to.equal("hello@world");
    });

    it("should not add an underscore at the beginning if the string starts with an uppercase letter", () => {
        expect(snakeCase("HelloWorld")).not.to.match(/^_/);
    });
});
