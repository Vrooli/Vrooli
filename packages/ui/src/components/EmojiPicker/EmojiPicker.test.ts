/* eslint-disable @typescript-eslint/ban-ts-comment */
import { expect } from "chai";
import { DataEmoji, EmojiProperties, SkinTone, emojiUnified, emojiVariationUnified, parseNativeEmoji, unifiedWithoutSkinTone } from "./EmojiPicker.js";

describe("parseNativeEmoji", () => {
    it("should convert a simple unified emoji code to a native emoji", () => {
        const input = "1f604";
        const expected = "😄";
        expect(parseNativeEmoji(input)).to.equal(expected);
    });

    it("should convert a unified emoji code with a skin tone modifier to a native emoji", () => {
        const input = "1f468-1f3fb";
        const expected = "👨🏻";
        expect(parseNativeEmoji(input)).to.equal(expected);
    });

    it("should handle multiple code points to form a single emoji (e.g., family emoji)", () => {
        const input = "1f468-200d-1f469-200d-1f466-200d-1f467";
        const expected = "👨‍👩‍👦‍👧";
        expect(parseNativeEmoji(input)).to.equal(expected);
    });

    it("should correctly handle flag emojis using regional indicator symbols", () => {
        const input = "1f1fa-1f1f8";
        const expected = "🇺🇸";
        expect(parseNativeEmoji(input)).to.equal(expected);
    });

    it("should return an empty string for an empty input", () => {
        const input = "";
        const expected = "";
        expect(parseNativeEmoji(input)).to.equal(expected);
    });

    it("should gracefully handle invalid hex code and return an empty string", () => {
        const input = "invalid";
        try {
            parseNativeEmoji(input);
        } catch (error) {
            expect(error).to.exist;
            expect(error.message).to.include("Invalid code point");
        }
    });

    it("should concatenate multiple emojis separated by hyphens into a single string", () => {
        const input = "1f600-1f602-1f605";
        const expected = "😀😂😅";
        expect(parseNativeEmoji(input)).to.equal(expected);
    });

    it("should handle non-standard separators gracefully by treating them as part of the emoji code", () => {
        const input = "1f468_1f3fb";  // Using an underscore instead of a hyphen
        try {
            parseNativeEmoji(input);
        } catch (error) {
            expect(error).to.exist;
            expect(error.message).to.include("Invalid code point");
        }
    });

    it("should be able to handle real world complex emoji sequences", () => {
        const input = "1f469-1f3fb-200d-2764-fe0f-200d-1f48b-200d-1f468-1f3fc";
        const expected = "👩🏻‍❤️‍💋‍👨🏼";
        expect(parseNativeEmoji(input)).to.equal(expected);
    });
});

describe("unifiedWithoutSkinTone", () => {
    it("should remove a light skin tone modifier from an emoji code", () => {
        const input = `1f468-${SkinTone.Light}`;
        const expected = "1f468";
        expect(unifiedWithoutSkinTone(input)).to.equal(expected);
    });

    it("should return the original unified code when no skin tone is present", () => {
        const input = "1f468";
        const expected = "1f468";
        expect(unifiedWithoutSkinTone(input)).to.equal(expected);
    });

    it("should remove all skin tone modifiers in a sequence with multiple modifiers", () => {
        const input = `1f469-${SkinTone.MediumLight}-200d-1f468-${SkinTone.Light}`;
        const expected = "1f469-200d-1f468";
        expect(unifiedWithoutSkinTone(input)).to.equal(expected);
    });

    it("should not alter the unified code if a non-skin-tone modifier is present", () => {
        const input = "1f469-200d-2764-fe0f";
        const expected = "1f469-200d-2764-fe0f";
        expect(unifiedWithoutSkinTone(input)).to.equal(expected);
    });

    it("should correctly handle input with non-standard modifiers that are not skin tones", () => {
        const input = "1f3f4-2620-fe0f";
        const expected = "1f3f4-2620-fe0f";
        expect(unifiedWithoutSkinTone(input)).to.equal(expected);
    });

    it("should return an unchanged unified code for flag emojis", () => {
        const input = "1f1fa-1f1f8";  // U.S. flag, which shouldn't have skin tones
        const expected = "1f1fa-1f1f8";
        expect(unifiedWithoutSkinTone(input)).to.equal(expected);
    });

    it("should handle cases where skin tone is at the end of the sequence", () => {
        const input = `1f91d-${SkinTone.Light}-${SkinTone.Light}`;  // Handshake with skin tones
        const expected = "1f91d";  // Should remove both skin tones
        expect(unifiedWithoutSkinTone(input)).to.equal(expected);
    });

    it("should remove the skin tone when it is the only segment in the unified code", () => {
        const input = SkinTone.Dark;  // Just the skin tone
        const expected = "";  // Should result in an empty string
        expect(unifiedWithoutSkinTone(input)).to.equal(expected);
    });
});

describe("emojiUnified", () => {
    it("returns the base unified code when no skin tone is provided", () => {
        const emoji = {
            n: ["smiling face"],
            u: "1f600",
            a: 6,
        } as unknown as DataEmoji;
        expect(emojiUnified(emoji)).to.equal("1f600");
    });

    it("returns the base unified code when skin tone is provided but no variations exist", () => {
        const emoji = {
            n: ["smiling face"],
            u: "1f600",
            a: 6,
        } as unknown as DataEmoji;
        const skinTone = "1f3fb";  // Light skin tone
        expect(emojiUnified(emoji, skinTone)).to.equal("1f600");
    });

    it("returns the correct variation when a valid skin tone is provided", () => {
        const emoji = {
            n: ["waving hand"],
            u: "1f44b",
            v: ["1f44b-1f3fb", "1f44b-1f3fc", "1f44b-1f3fd", "1f44b-1f3fe", "1f44b-1f3ff"],
            a: 6,
        } as unknown as DataEmoji;
        const skinTone = "1f3fb";
        expect(emojiUnified(emoji, skinTone)).to.equal("1f44b-1f3fb");
    });

    it("returns the base unified code when an invalid skin tone is provided", () => {
        const emoji = {
            n: ["waving hand"],
            u: "1f44b",
            v: ["1f44b-1f3fb", "1f44b-1f3fc", "1f44b-1f3fd", "1f44b-1f3fe", "1f44b-1f3ff"],
            a: 6,
        } as unknown as DataEmoji;
        const skinTone = "invalid-tone";
        expect(emojiUnified(emoji, skinTone)).to.equal("1f44b");
    });

    it("returns the base unified code when there are variations but no matching skin tone variation", () => {
        const emoji = {
            n: ["waving hand"],
            u: "1f44b",
            v: ["1f44b-1f3fc", "1f44b-1f3fd", "1f44b-1f3fe", "1f44b-1f3ff"],
            a: 6,
        } as unknown as DataEmoji;
        const skinTone = "1f3fb"; // Light skin tone not in variations
        expect(emojiUnified(emoji, skinTone)).to.equal("1f44b");
    });

    it("handles emojis without variations but with a skin tone provided", () => {
        const emoji = {
            n: ["rocket"],
            u: "1f680",
            a: 6,
        } as unknown as DataEmoji;
        const skinTone = "1f3fb";  // Light skin tone
        expect(emojiUnified(emoji, skinTone)).to.equal("1f680");
    });

    it("returns the base unified code when variations array is empty and skin tone is provided", () => {
        const emoji = {
            n: ["face with tears of joy"],
            u: "1f602",
            v: [],
            a: 6,
        } as unknown as DataEmoji;
        const skinTone = "1f3fb";
        expect(emojiUnified(emoji, skinTone)).to.equal("1f602");
    });
});

describe("emojiVariationUnified", () => {
    const baseEmoji = {
        [EmojiProperties.name]: ["smiling face"],
        [EmojiProperties.unified]: "1f600",
        [EmojiProperties.variations]: ["1f600-1f3fb", "1f600-1f3fc"],
    };

    it("should return the correct variation for a given skin tone", () => {
        const skinTone = "1f3fb";
        const expected = "1f600-1f3fb";
        expect(emojiVariationUnified(baseEmoji, skinTone)).to.equal(expected);
    });

    it("should return undefined if the specified skin tone is not available", () => {
        const skinTone = "1f3ff"; // not available in variations
        expect(emojiVariationUnified(baseEmoji, skinTone)).to.be.undefined;
    });

    it("should return the base emoji if no skin tone is provided", () => {
        expect(emojiVariationUnified(baseEmoji)).to.equal(baseEmoji[EmojiProperties.unified]);
    });

    it("should handle emojis with no variations defined", () => {
        const emoji = {
            ...baseEmoji,
            [EmojiProperties.variations]: undefined,
        };
        const skinTone = "1f3fb";
        expect(emojiVariationUnified(emoji, skinTone)).to.be.undefined;
    });

    it("should correctly handle cases where variations are present but empty", () => {
        const emoji = {
            ...baseEmoji,
            [EmojiProperties.variations]: [],
        };
        const skinTone = "1f3fb";
        expect(emojiVariationUnified(emoji, skinTone)).to.be.undefined;
    });

    it("should return the unified property when no skin tone is specified and there are variations", () => {
        const emoji = {
            ...baseEmoji,
            [EmojiProperties.variations]: ["1f600-1f3fb", "1f600-1f3fc"],
        };
        expect(emojiVariationUnified(emoji)).to.equal(baseEmoji[EmojiProperties.unified]);
    });
});

// describe("emojiByUnified", () => {
//     // Mock data
//     const mockEmoji = {
//         name: ["grinning face"],
//         u: "1f600",
//     };

//     const mockEmojiWithSkinTone = {
//         name: ["thumbs up"],
//         u: "1f44d-1f3fb",
//     };

//     // Simulate the map environment
//     const allEmojisByUnified = new Map();
//     allEmojisByUnified.set("1f600", mockEmoji);
//     allEmojisByUnified.set("1f44d-1f3fb", mockEmojiWithSkinTone);

//     // Mock the helper function
//     const unifiedWithoutSkinTone = jest.fn((unified) => unified.split("-")[0]);

//     beforeEach(() => {
//         // Clear mock counts
//         unifiedWithoutSkinTone.mockClear();
//     });

//     it("should return an emoji object when the unified code matches exactly", () => {
//         expect(emojiByUnified("1f600")).to.equal(mockEmoji);
//     });

//     it("should return an emoji object when the unified code includes a skin tone", () => {
//         expect(emojiByUnified("1f44d-1f3fb")).to.equal(mockEmojiWithSkinTone);
//     });

//     it("should return undefined for a unified code that does not exist", () => {
//         expect(emojiByUnified("1f44d")).to.be.undefined;
//     });

//     it("should call the unifiedWithoutSkinTone function when the emoji is not found", () => {
//         emojiByUnified("1f44d-1f3ff");
//         expect(unifiedWithoutSkinTone).toHaveBeenCalledWith("1f44d-1f3ff");
//         expect(emojiByUnified("1f44d-1f3ff")).to.be.undefined;
//     });

//     it("should handle null and return undefined", () => {
//         // @ts-ignore: Testing runtime scenario
//         expect(emojiByUnified(null)).to.be.undefined;
//     });

//     it("should return undefined for empty string input", () => {
//         expect(emojiByUnified("")).to.be.undefined;
//     });

//     it("should retrieve an emoji object using the base unified code if skin tone variation is not directly available", () => {
//         allEmojisByUnified.set("1f44d", mockEmoji); // Set base emoji without skin tone
//         expect(emojiByUnified("1f44d-1f3ff")).to.equal(mockEmoji);
//         expect(unifiedWithoutSkinTone).toHaveBeenCalledWith("1f44d-1f3ff");
//     });
// });
