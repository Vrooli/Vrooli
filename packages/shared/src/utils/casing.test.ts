import { lowercaseFirstLetter, pascalCase, snakeCase, uppercaseFirstLetter } from "./casing";

describe("lowercaseFirstLetter", () => {
    it("should lowercase the first letter of a string", () => {
        expect(lowercaseFirstLetter("Hello")).toBe("hello");
        expect(lowercaseFirstLetter("WORLD")).toBe("wORLD");
    });

    it("should return the same string if the first letter is already lowercase", () => {
        expect(lowercaseFirstLetter("already")).toBe("already");
    });

    it("should return an empty string if an empty string is provided", () => {
        expect(lowercaseFirstLetter("")).toBe("");
    });

    it("should handle non-alphabetical characters", () => {
        expect(lowercaseFirstLetter("1hello")).toBe("1hello");
        expect(lowercaseFirstLetter("@World")).toBe("@World");
    });
});

describe("uppercaseFirstLetter", () => {
    it("should uppercase the first letter of a string", () => {
        expect(uppercaseFirstLetter("hello")).toBe("Hello");
        expect(uppercaseFirstLetter("wORLD")).toBe("WORLD");
    });

    it("should return the same string if the first letter is already uppercase", () => {
        expect(uppercaseFirstLetter("Already")).toBe("Already");
    });

    it("should return an empty string if an empty string is provided", () => {
        expect(uppercaseFirstLetter("")).toBe("");
    });

    it("should handle non-alphabetical characters", () => {
        expect(uppercaseFirstLetter("1hello")).toBe("1hello");
        expect(uppercaseFirstLetter("@world")).toBe("@world");
    });
});

describe("pascalCase", () => {
    it("should convert a string to pascalCase", () => {
        expect(pascalCase("hello-world")).toBe("HelloWorld");
        expect(pascalCase("hello_world")).toBe("HelloWorld");
        expect(pascalCase("hello world")).toBe("HelloWorld");
        expect(pascalCase("helloWorld")).toBe("HelloWorld");
        expect(pascalCase("HelloWorld")).toBe("HelloWorld");
    });

    it("should return an empty string if an empty string is provided", () => {
        expect(pascalCase("")).toBe("");
    });
});

describe("snakeCase", () => {
    it("should convert a string to snake_case", () => {
        expect(snakeCase("helloWorld")).toBe("hello_world");
        expect(snakeCase("HelloWorld")).toBe("hello_world");
        expect(snakeCase("hello-world")).toBe("hello_world");
        expect(snakeCase("hello world")).toBe("hello_world");
        expect(snakeCase("Hello World")).toBe("hello_world");
    });

    // NOTE: This is why you shouldn't capitalize abbreviations
    it("should handle strings with multiple uppercase letters", () => {
        expect(snakeCase("HTTPServer")).toBe("h_t_t_p_server");
        expect(snakeCase("URLParser")).toBe("u_r_l_parser");
        expect(snakeCase("TheHttpServer")).toBe("the_http_server");
    });

    it("should return the same string if it is already in snake_case", () => {
        expect(snakeCase("already_in_snake_case")).toBe("already_in_snake_case");
    });

    it("should handle strings with non-alphabetic characters", () => {
        expect(snakeCase("hello1World")).toBe("hello1_world");
        expect(snakeCase("123helloWorld")).toBe("123hello_world");
    });

    it("should return an empty string if an empty string is provided", () => {
        expect(snakeCase("")).toBe("");
    });

    it("should handle strings with special characters and spaces", () => {
        expect(snakeCase("hello-World")).toBe("hello_world");
        expect(snakeCase("hello_world")).toBe("hello_world");
        expect(snakeCase("hello world")).toBe("hello_world");
        expect(snakeCase("hello@world")).toBe("hello@world");
    });

    it("should not add an underscore at the beginning if the string starts with an uppercase letter", () => {
        expect(snakeCase("HelloWorld")).not.toMatch(/^_/);
    });
});