import { type SessionUser } from "@vrooli/shared";
import { expect } from "chai";
import { ModelMap } from "../models/base/index.js";
import { type PreMap } from "../models/types.js";
import { type InputsById } from "../utils/types.js";
import { calculatePreShapeData } from "./cuds.js";

describe("calculatePreShapeData", () => {
    let originalModelMap;

    beforeEach(() => {
        vi.clearAllMocks();
        originalModelMap = { ...ModelMap };
    });

    afterEach(() => {
        Object.assign(ModelMap, originalModelMap);
    });

    it("should handle types without a pre function", async () => {
        const inputsByType = {
            TypeA: {
                Create: [{ node: "TypeA", input: { /* ... */ } }],
                Update: [],
                Delete: [],
            },
            TypeB: {
                Create: [],
                Update: [{ node: "TypeB", input: { /* ... */ } }, { node: "TypeB", input: { /* ... */ } }],
                Delete: [{ node: "TypeB", input: { /* ... */ } }],
            },
        };
        const userData = {} as unknown as SessionUser;
        const inputsById: InputsById = {};
        const preMap: PreMap = {};

        Object.assign(ModelMap, {
            getLogic: jest.fn().mockImplementation(() => ({
                mutate: {
                    shape: {
                        // No pre function
                    },
                },
            })),
        });

        await calculatePreShapeData(inputsByType, userData, inputsById, preMap);

        expect(preMap).to.deep.equal({
            TypeA: {},
            TypeB: {},
        });
    });

    it("should handle types with a pre function", async () => {
        const inputsByType = {
            TypeA: {
                Create: [],
                Update: [],
                Delete: [{ node: "TypeA", input: { /* ... */ } }],
            },
        };
        const userData = {} as unknown as SessionUser;
        const inputsById: InputsById = {};
        const preMap: PreMap = {};

        Object.assign(ModelMap, {
            getLogic: jest.fn().mockImplementation(() => ({
                mutate: {
                    shape: {
                        pre: jest.fn().mockReturnValue({ beep: "boop" }),
                    },
                },
            })),
        });

        await calculatePreShapeData(inputsByType, userData, inputsById, preMap);

        expect(preMap).to.deep.equal({
            TypeA: { beep: "boop" },
        });
    });

    it("pre function should be able to modify `Create`, `Update`, and `Delete` arrays", async () => {
        const inputsByType = {
            TypeA: {
                Create: [{ node: "TypeA", input: { /* ... */ } }],
                Update: [],
                Delete: [],
            },
        };
        const userData = {} as unknown as SessionUser;
        const inputsById: InputsById = {};
        const preMap: PreMap = {};

        Object.assign(ModelMap, {
            getLogic: jest.fn().mockImplementation(() => ({
                mutate: {
                    shape: {
                        pre: jest.fn().mockImplementation(({ Create, Update, Delete }) => {
                            Create.push({ node: "TypeA", input: { just: "added" } });
                            Update.push({ node: "TypeA", input: { hello: "world" } });
                            Delete.push({ node: "TypeA", input: { chicken: "nugget" } });
                            return {};
                        }),
                    },
                },
            })),
        });

        await calculatePreShapeData(inputsByType, userData, inputsById, preMap);

        expect(preMap).to.deep.equal({
            TypeA: {},
        });
        expect(inputsByType).to.deep.equal({
            TypeA: {
                Create: [{ node: "TypeA", input: { /* ... */ } }, { node: "TypeA", input: { just: "added" } }],
                Update: [{ node: "TypeA", input: { hello: "world" } }],
                Delete: [{ node: "TypeA", input: { chicken: "nugget" } }],
            },
        });
    });
});
