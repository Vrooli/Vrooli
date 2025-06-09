import { type FormSchema, type GridContainer } from "@vrooli/shared";
import { describe, it, expect, beforeEach, afterEach, vi } from "vitest";
import { normalizeFormContainers } from "./FormView.js";

describe("normalizeFormContainers", () => {
    it("should return one container covering all elements when no containers are provided", () => {
        const formSchema = {
            elements: [{}, {}, {}],
        } as unknown as FormSchema;

        const expected: GridContainer[] = [{
            totalItems: 3,
        }];

        expect(normalizeFormContainers(formSchema)).toEqual(expected);
    });

    it("should return correct ranges when containers are provided with valid ranges", () => {
        const formSchema = {
            elements: [{}, {}, {}, {}, {}],
            containers: [
                { totalItems: 3, disableCollapse: true },
                { totalItems: 2 },
            ],
        } as unknown as FormSchema;

        const expected: GridContainer[] = [
            { totalItems: 3, disableCollapse: true },
            { totalItems: 2, disableCollapse: undefined },
        ];

        expect(normalizeFormContainers(formSchema)).toEqual(expected);
    });

    it("should handle containers exceeding the number of elements", () => {
        const formSchema = {
            elements: [{}, {}, {}, {}],
            containers: [
                { totalItems: 5 },
                { totalItems: 2 },
            ],
        } as unknown as FormSchema;

        const expected: GridContainer[] = [
            { totalItems: 4, disableCollapse: undefined },
        ];

        expect(normalizeFormContainers(formSchema)).toEqual(expected);
    });

    it("should add a final container if necessary to cover all elements", () => {
        const formSchema: FormSchema = {
            elements: [{}, {}, {}, {}, {}],
            containers: [
                { totalItems: 2 },
                { totalItems: 2 },
            ],
        } as unknown as FormSchema;

        const expected: GridContainer[] = [
            { totalItems: 2 },
            { totalItems: 2 },
            { totalItems: 1 },
        ];

        expect(normalizeFormContainers(formSchema)).toEqual(expected);
    });

    it("should handle an empty elements array correctly", () => {
        const formSchema = {
            elements: [], // No elements
            containers: [
                { totalItems: 3 },
                { totalItems: 2 },
            ],
        } as unknown as FormSchema;

        const expected: GridContainer[] = []; // No containers

        expect(normalizeFormContainers(formSchema)).toEqual(expected);
    });

    it("should handle an empty containers array correctly", () => {
        const formSchema = {
            elements: [{}, {}, {}, {}, {}],
            containers: [],
        } as unknown as FormSchema;

        const expected: GridContainer[] = [{
            totalItems: 5,
        }];

        expect(normalizeFormContainers(formSchema)).toEqual(expected);
    });

    it("should handle containers with disableCollapse set correctly", () => {
        const formSchema = {
            elements: [{}, {}, {}, {}, {}],
            containers: [
                { totalItems: 2, disableCollapse: true },
                { totalItems: 3, disableCollapse: false },
            ],
        } as unknown as FormSchema;

        const expected: GridContainer[] = [
            { totalItems: 2, disableCollapse: true },
            { totalItems: 3, disableCollapse: false },
        ];

        expect(normalizeFormContainers(formSchema)).toEqual(expected);
    });
});
