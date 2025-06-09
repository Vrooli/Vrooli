/* eslint-disable @typescript-eslint/ban-ts-comment */
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { formatStatusMessages } from "./StatusButton.js";

describe("formatStatusMessages", () => {
    it("returns \"No errors detected.\" when there are no messages", () => {
        expect(formatStatusMessages([])).toEqual("No errors detected.");
    });

    it("returns the single message without bullet points when there is one message", () => {
        expect(formatStatusMessages(["Error loading data"])).toEqual("Error loading data");
    });

    it("returns messages with bullet points when there are multiple messages", () => {
        const messages = ["Error loading data", "Timeout occurred"];
        const expectedOutput = "* Error loading data\n* Timeout occurred";
        expect(formatStatusMessages(messages)).toEqual(expectedOutput);
    });

    it("returns \"No errors detected.\" when input is undefined or null", () => {
        // @ts-ignore: Testing runtime scenario
        expect(formatStatusMessages(undefined)).toEqual("No errors detected.");
        // @ts-ignore: Testing runtime scenario
        expect(formatStatusMessages(null)).toEqual("No errors detected.");
    });

    it("returns \"No errors detected.\" when the input is not an array", () => {
        // @ts-ignore: Testing runtime scenario
        expect(formatStatusMessages("this is a string")).toEqual("No errors detected.");
        // @ts-ignore: Testing runtime scenario
        expect(formatStatusMessages(123)).toEqual("No errors detected.");
        // @ts-ignore: Testing runtime scenario
        expect(formatStatusMessages({})).toEqual("No errors detected.");
        // @ts-ignore: Testing runtime scenario
        expect(formatStatusMessages(true)).toEqual("No errors detected.");
    });

    it("handles a large list of messages", () => {
        const messages = new Array(100).fill("Error").map((val, idx) => `${val} ${idx}`);
        const expectedOutput = messages.map(msg => `* ${msg}`).join("\n");
        expect(formatStatusMessages(messages)).toEqual(expectedOutput);
    });
});
