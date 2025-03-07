import { handleRegex, hexColorRegex, urlRegex, urlRegexDev, walletAddressRegex } from "./regex.js";

describe("Regular Expressions Tests", () => {
    // Test suite for urlRegex
    describe("urlRegex", () => {
        // Valid URLs
        const validUrls = [
            "http://www.example.com",
            "https://example.com",
            "ftp://example.org",
            "https://sub.example.co.uk",
        ];

        // Invalid URLs
        const invalidUrls = [
            "http://localhost:3000", // local addresses should be invalid
            "http://256.256.256.256", // invalid IP
            "justtext",
        ];

        // Test each valid URL
        test.each(validUrls)("\"%s\" should be a valid URL", (url) => {
            expect(urlRegex.test(url)).to.be.ok;
        });

        // Test each invalid URL
        test.each(invalidUrls)("\"%s\" should be an invalid URL", (url) => {
            expect(urlRegex.test(url)).to.not.be.ok;
        });
    });

    // Test suite for urlRegexDev
    describe("urlRegexDev", () => {
        // Valid URLs for development
        const validDevUrls = [
            "http://localhost:3000",
            "https://127.0.0.1:8080",
            "http://dev.example.com",
            "https://subdomain.example.co.uk",
            "ftp://example.org",
        ];

        // Invalid URLs for development
        const invalidDevUrls = [
            "http://256.256.256.256", // invalid IP
            "justtext",
        ];

        // Test each valid URL for development
        test.each(validDevUrls)("\"%s\" should be a valid development URL", (url) => {
            expect(urlRegexDev.test(url)).to.be.ok;
        });

        // Test each invalid URL for development
        test.each(invalidDevUrls)("\"%s\" should be an invalid development URL", (url) => {
            expect(urlRegexDev.test(url)).to.not.be.ok;
        });

        describe("ReDoS Vulnerability Tests for urlRegexDev", () => {
            // Function to measure execution time
            function measureExecutionTime(regex, input) {
                const start = process.hrtime.bigint();
                const result = regex.test(input);
                const end = process.hrtime.bigint();
                const duration = Number(end - start) / 1e6; // Convert to milliseconds
                return { result, duration };
            }

            // Benign input
            const benignInput = "ftp://example.com";

            // Potentially malicious inputs
            const maliciousInputs = [
                "ftp://0." + "0.".repeat(10) + "0",   // Repeats '0.' 10 times
                "ftp://0." + "0.".repeat(100) + "0",  // Repeats '0.' 100 times
                "ftp://0." + "0.".repeat(1000) + "0", // Repeats '0.' 1000 times
            ];

            it("Benign input should execute quickly", () => {
                const { duration } = measureExecutionTime(urlRegexDev, benignInput);
                expect(duration).to.be.lessThan(1); // Less than 1 millisecond
            });

            test.each(maliciousInputs)("Malicious input of length %i should not cause performance issues", (input) => {
                const { duration } = measureExecutionTime(urlRegexDev, input);
                expect(duration).to.be.lessThan(10); // Execution should be quick
            });
        });
    });

    // Test suite for walletAddressRegex
    describe("walletAddressRegex", () => {
        // Valid Cardano wallet addresses
        const validWalletAddresses = [
            "addr1q9pdxnvnezqpj3hph45ltkr2z3fyllprd2j9cl3zv3j8y5ghl4xyz5hfkdf82jlqm4w8sn3kx2tjy0s8fluxz9flkjqquvvyg9", // 103 characters, starts with addr1
        ];

        // Invalid Cardano wallet addresses
        const invalidWalletAddresses = [
            "addr2v9pdxnvnezqpj3hph45ltkr2z3fyllprd2j9cl3zv3j8y5ghl4xyz5hfkdf82jlqm4w8sn3kx2tjy0s8fluxz9flkjqquvvyg9", // starts with addr2, should be addr1
            "addr1q9pdxnvnezqpj3hph45ltkr2z3fyllprd2j9cl3zv3j8y5ghl4xyz5hfkdf82jlqm4w8sn3kx2tjy0s8fluxz9flkj", // too short
            "randomtext", // not even close
        ];

        // Test each valid Cardano wallet address
        test.each(validWalletAddresses)("\"%s\" should be a valid Cardano wallet address", (address) => {
            expect(walletAddressRegex.test(address)).to.be.ok;
        });

        // Test each invalid Cardano wallet address
        test.each(invalidWalletAddresses)("\"%s\" should be an invalid Cardano wallet address", (address) => {
            expect(walletAddressRegex.test(address)).to.not.be.ok;
        });
    });

    // Test suite for handleRegex
    describe("handleRegex", () => {
        // Valid handles
        const validHandles = [
            "username",
            "user_name",
            "abc123",
            "u_n123456", // mixed letters, underscore, and numbers
            "User_Name1", // mixed case with numbers
            "user______", // multiple underscores
            "a1b2c3d4e",  // exactly 9 characters
            "A2B3C4D5E6F7G8H", // exactly 16 characters, upper case with numbers
        ];

        // Invalid handles
        const invalidHandles = [
            "ab", // too short, less than 3 characters
            "user-name", // contains hyphen, only underscores are allowed
            "user!name", // contains special character, only letters, numbers, and underscores are allowed
            "1", // single character
            "12", // two characters
            "!@#$%^&*()", // only special characters
            "user name", // contains space
            "thisusernameiswaytoolongtobevalid", // more than 16 characters
            "user.name", // contains dot, only underscores are allowed
            "user-name!", // contains hyphen and exclamation mark
            "______", // underscores only, valid length but might not be desirable
            "12345678901234567", // 17 characters, exceeding the maximum
        ];

        // Test each valid handle
        test.each(validHandles)("\"%s\" should be a valid handle", (handle) => {
            expect(handleRegex.test(handle)).to.be.ok;
        });

        // Test each invalid handle
        test.each(invalidHandles)("\"%s\" should be an invalid handle", (handle) => {
            expect(handleRegex.test(handle)).to.not.be.ok;
        });
    });

    // Test suite for hexRegex
    describe("hexColorRegex", () => {
        // Valid hex colors
        const validHexColors = [
            "#FFFFFF",
            "#000000",
            "#123ABC",
            "#abc",
            "#123456",
            "#123ABC",
        ];

        // Invalid hex colors
        const invalidHexColors = [
            "12345", // no # at the start
            "ZZZZZZ", // not a valid hex character
            "#1234567", // too many characters
            "#12", // too few characters
            "GHIJKL", // no # and not a valid hex
            "#12345G", // G is not a valid character in the range 0-9, A-F
            "123456", // no # at the start, valid length
            "  #123456  ", // contains spaces
        ];

        // Test each valid hex color
        test.each(validHexColors)("\"%s\" should be a valid hex color", (hex) => {
            expect(hexColorRegex.test(hex)).to.be.ok;
        });

        // Test each invalid hex color
        test.each(invalidHexColors)("\"%s\" should be an invalid hex color", (hex) => {
            expect(hexColorRegex.test(hex)).to.not.be.ok;
        });
    });
});

