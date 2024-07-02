import { FormSchema } from "../../../../forms/types";
import { NormalizedGridContainer, normalizeFormContainers } from "./GeneratedGrid";

describe("normalizeFormContainers", () => {
    it("should return one container covering all elements when no containers are provided", () => {
        const formSchema = {
            elements: [{}, {}, {}],
        } as unknown as FormSchema;

        const expected: NormalizedGridContainer[] = [{
            startIndex: 0,
            endIndex: 2,
            disableCollapse: false,
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

        const expected: NormalizedGridContainer[] = [
            { startIndex: 0, endIndex: 2, disableCollapse: true },
            { startIndex: 3, endIndex: 4, disableCollapse: undefined },
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

        const expected: NormalizedGridContainer[] = [
            { startIndex: 0, endIndex: 3, disableCollapse: undefined },
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

        const expected: NormalizedGridContainer[] = [
            { startIndex: 0, endIndex: 1, disableCollapse: undefined },
            { startIndex: 2, endIndex: 3, disableCollapse: undefined },
            { startIndex: 4, endIndex: 4, disableCollapse: false },
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

        const expected: NormalizedGridContainer[] = []; // No containers

        expect(normalizeFormContainers(formSchema)).toEqual(expected);
    });

    it("should handle an empty containers array correctly", () => {
        const formSchema = {
            elements: [{}, {}, {}, {}, {}],
            containers: [],
        } as unknown as FormSchema;

        const expected: NormalizedGridContainer[] = [{
            startIndex: 0,
            endIndex: 4,
            disableCollapse: false,
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

        const expected: NormalizedGridContainer[] = [
            { startIndex: 0, endIndex: 1, disableCollapse: true },
            { startIndex: 2, endIndex: 4, disableCollapse: false },
        ];

        expect(normalizeFormContainers(formSchema)).toEqual(expected);
    });
});
