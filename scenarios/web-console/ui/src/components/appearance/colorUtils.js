/**
 * @vrooliComponentSource react-component-library:ColorPicker
 * @vrooliComponentVersion 1.0.0
 * @vrooliComponentAdoption 623f00f8-2a74-40ec-83bc-67c4575b6cb6
 * @vrooliComponentAppliedAt 2026-07-22T16:50:28Z
 * @vrooliComponentSourceSha256 1abf09f40c8fef0dddc2e47827ab5778f5a44b57fc09ac284863f59707424ec7
 * @vrooliComponentDriftHash 1abf09f40c8fef0dddc2e47827ab5778f5a44b57fc09ac284863f59707424ec7
 *
 * This file was copied from React Component Library. Local edits are allowed;
 * run "react-component-library adoptions refresh" to inspect drift.
 */
const HEX_COLOR = /^#([0-9a-fA-F]{3}|[0-9a-fA-F]{6})$/;
export function isHexColor(value) {
    return typeof value === "string" && HEX_COLOR.test(value);
}
export function isLightColor(value) {
    if (!isHexColor(value))
        return false;
    const hex = value.length === 4 ? value.slice(1).split("").map((part) => part + part).join("") : value.slice(1);
    const rgb = [0, 2, 4].map((offset) => Number.parseInt(hex.slice(offset, offset + 2), 16) / 255);
    const [red = 0, green = 0, blue = 0] = rgb.map((channel) => channel <= 0.03928 ? channel / 12.92 : ((channel + 0.055) / 1.055) ** 2);
    return 0.2126 * red + 0.7152 * green + 0.0722 * blue > 0.5;
}
export function parseColorValue(value) {
    const colors = (value ?? "").split("|").map((part) => part.trim()).filter(isHexColor).slice(0, 2);
    return { colors, transparent: colors.length === 0 };
}
export function serializeColorValue(colors) {
    const valid = colors.filter(isHexColor).slice(0, 2);
    return valid.length ? valid.join("|") : "transparent";
}
