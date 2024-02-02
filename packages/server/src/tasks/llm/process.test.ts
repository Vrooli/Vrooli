/* eslint-disable @typescript-eslint/ban-ts-comment */
import { toBotSettings } from "./process";

describe("toBotSettings", () => {
    it("parses valid bot settings correctly", () => {
        const bot = {
            name: "TestBot",
            botSettings: JSON.stringify({
                model: "gpt-3",
                maxTokens: 100,
                translations: {
                    en: {
                        persona: "friendly",
                    },
                },
            }),
        };
        expect(toBotSettings(bot)).toEqual({
            name: "TestBot",
            model: "gpt-3",
            maxTokens: 100,
            translations: {
                en: {
                    persona: "friendly",
                },
            },
        });
    });

    it("handles invalid JSON gracefully", () => {
        const bot = {
            name: "TestBot",
            botSettings: "invalid JSON",
        };
        expect(toBotSettings(bot)).toEqual({ name: "TestBot" });
    });

    it("returns name when botSettings is undefined", () => {
        const bot = {
            name: "TestBot",
            botSettings: undefined,
        };
        expect(toBotSettings(bot)).toEqual({ name: "TestBot" });
    });

    it("ignores non-string botSettings", () => {
        const bot = {
            name: "TestBot",
            botSettings: { some: "object" }, // Intentionally incorrect type
        };
        // @ts-ignore: Testing runtime scenario
        expect(toBotSettings(bot)).toEqual({ name: "TestBot" });
    });

    it("handles empty botSettings object", () => {
        const bot = {
            name: "TestBot",
            botSettings: "{}",
        };
        expect(toBotSettings(bot)).toEqual({ name: "TestBot" });
    });

    it("retains additional properties in botSettings", () => {
        const bot = {
            name: "TestBot",
            botSettings: JSON.stringify({ customProp: "customValue" }),
        };
        expect(toBotSettings(bot)).toEqual({ name: "TestBot", customProp: "customValue" });
    });

    it("handles missing name field", () => {
        const bot = {
            botSettings: JSON.stringify({ model: "gpt-3" }),
        };
        // @ts-ignore: Testing runtime scenario
        expect(toBotSettings(bot)).toEqual({ name: "", model: "gpt-3" });
    });

    it("handles null input gracefully", () => {
        // @ts-ignore: Testing runtime scenario
        expect(toBotSettings(null)).toEqual({ name: "" });
    });
});
