/* eslint-disable @typescript-eslint/ban-ts-comment */
import { Node, NodeLink, RoutineType, RoutineVersion, uuid } from "@local/shared";
import { DirectoryStep, EndStep, MultiRoutineStep, RoutineListStep, RunStep, SingleRoutineStep, StartStep } from "../../../types";
import { RunStepType } from "../../../utils/consts";
import { getNextLocation, getPreviousLocation, getStepComplexity, insertStep, parseChildOrder, sortStepsAndAddDecisions, stepFromLocation, stepNeedsQuerying } from "./RunView";

describe("insertStep", () => {
    beforeAll(() => {
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        jest.spyOn(console, "warn").mockImplementation(() => { });
    });
    afterAll(() => {
        jest.restoreAllMocks();
    });

    it("adds project data to a project", () => {
        const mockStepData: Omit<DirectoryStep, "location"> = {
            __type: RunStepType.Directory,
            description: "a",
            directoryId: null,
            hasBeenQueried: true,
            isOrdered: false,
            isRoot: true,
            name: "a",
            projectVersionId: "projectVersion123",
            steps: [],
        };
        const mockRootStep: DirectoryStep = {
            __type: RunStepType.Directory,
            description: "root",
            directoryId: null,
            hasBeenQueried: true,
            isOrdered: false,
            isRoot: true,
            location: [1],
            name: "root",
            projectVersionId: "projectVersion234",
            steps: [{
                __type: RunStepType.Directory,
                description: "b",
                directoryId: null,
                hasBeenQueried: false,
                isOrdered: false,
                isRoot: true,
                location: [1, 1],
                name: "b",
                projectVersionId: "projectVersion123", // Matches mockStepData, so it should be replaced by mockStepData
                steps: [],
            }],
        };
        const result = insertStep(mockStepData, mockRootStep);
        expect(result.steps[0]).toEqual({ ...mockStepData, location: [1, 1] });
    });

    it("adds multi-step routine data to a multi-step routine", () => {
        const mockStepData: Omit<MultiRoutineStep, "location"> = {
            __type: RunStepType.MultiRoutine,
            description: "newBoi",
            name: "hello",
            nodes: [],
            nodeLinks: [],
            routineVersionId: "routineVersion999",
        };
        const mockRootStep: MultiRoutineStep = {
            __type: RunStepType.MultiRoutine,
            description: "root",
            name: "root",
            location: [1],
            nodes: [{
                __type: RunStepType.Start,
                description: "",
                name: "",
                location: [1, 1],
                nodeId: "node123",
            }, {
                __type: RunStepType.RoutineList,
                description: "boop",
                name: "beep",
                location: [1, 2],
                nodeId: "node234",
                steps: [{
                    __type: RunStepType.SingleRoutine,
                    description: "",
                    name: "",
                    location: [1, 2, 1],
                    routineVersion: { id: "routineVersion999" } as RoutineVersion, // Matches mockStepData, so it should be replaced by mockStepData
                }],
            }, {
                __type: RunStepType.End,
                description: "",
                name: "",
                location: [1, 3],
                nodeId: "node345",
            }],
            nodeLinks: [],
            routineVersionId: "routineVersion123",
        };
        const result = insertStep(mockStepData, mockRootStep);
        expect((result.nodes[1] as RoutineListStep).steps[0]).toEqual({ ...mockStepData, location: [1, 2, 1] });
    });

    it("does not add data to a root that's a subroutine", () => {
        const mockStepData: Omit<MultiRoutineStep, "location"> = {
            __type: RunStepType.MultiRoutine,
            description: "newBoi",
            name: "hello",
            nodes: [],
            nodeLinks: [],
            routineVersionId: "routineVersion999",
        };
        const mockRootStep: SingleRoutineStep = {
            __type: RunStepType.SingleRoutine,
            description: "root",
            location: [1],
            name: "root",
            routineVersion: { id: "routineVersion529" } as RoutineVersion,
        };
        const result = insertStep(mockStepData, mockRootStep);
        expect(result).toEqual({ ...mockRootStep, location: [1] });
    });

    it("adds project data to a deeply-nested project", () => {
        const mockStepData: Omit<DirectoryStep, "location"> = {
            __type: RunStepType.Directory,
            description: "new data",
            directoryId: null,
            hasBeenQueried: false,
            isOrdered: false,
            isRoot: true,
            name: "new data",
            projectVersionId: "projectVersionNested",
            steps: [],
        };

        const mockRootStep: DirectoryStep = {
            __type: RunStepType.Directory,
            description: "root",
            directoryId: null,
            hasBeenQueried: true,
            isOrdered: false,
            isRoot: true,
            location: [1],
            name: "root",
            projectVersionId: "rootProject",
            steps: [{
                __type: RunStepType.Directory,
                description: "nested level 1",
                directoryId: null,
                hasBeenQueried: true,
                isOrdered: false,
                isRoot: false,
                location: [1, 1],
                name: "nested level 1",
                projectVersionId: "level1Project",
                steps: [{
                    __type: RunStepType.Directory,
                    description: "nested level 2",
                    directoryId: null,
                    hasBeenQueried: true,
                    isOrdered: false,
                    isRoot: false,
                    location: [1, 1, 1],
                    name: "nested level 2",
                    projectVersionId: "level2Project",
                    steps: [{
                        __type: RunStepType.Directory,
                        description: "nested level 3",
                        directoryId: null,
                        hasBeenQueried: true,
                        isOrdered: false,
                        isRoot: false,
                        location: [1, 1, 1, 1],
                        name: "nested level 3",
                        projectVersionId: "projectVersionNested", // Matches mockStepData
                        steps: [],
                    }],
                }],
            }],
        };

        const result = insertStep(mockStepData, mockRootStep);

        // Traversing down to the deeply-nested step
        expect(result.steps[0].steps[0].steps[0]).toEqual({ ...mockStepData, location: [1, 1, 1, 1] });
    });

    it("adds routine data to a deeply-nested routine", () => {
        const mockStepData: Omit<MultiRoutineStep, "location"> = {
            __type: RunStepType.MultiRoutine,
            description: "deep routine",
            name: "deep routine",
            nodes: [],
            nodeLinks: [],
            routineVersionId: "deepRoutineVersion",
        };

        const mockRootStep: MultiRoutineStep = {
            __type: RunStepType.MultiRoutine,
            description: "root",
            location: [1],
            name: "root",
            nodes: [{
                __type: RunStepType.Start,
                description: "start",
                location: [1, 1],
                name: "start",
                nodeId: "startNode",
            }, {
                __type: RunStepType.RoutineList,
                description: "routine list level 1",
                location: [1, 2],
                name: "routine list level 1",
                nodeId: "routineListNode1",
                steps: [{
                    __type: RunStepType.MultiRoutine,
                    description: "multi routine level 2",
                    location: [1, 2, 1],
                    name: "multi routine level 2",
                    nodes: [{
                        __type: RunStepType.RoutineList,
                        description: "routine list level 3",
                        name: "routine list level 3",
                        location: [1, 2, 1, 1],
                        nodeId: "routineListNode3",
                        steps: [{
                            __type: RunStepType.SingleRoutine,
                            description: "subroutine",
                            location: [1, 2, 1, 1, 1],
                            name: "subroutine",
                            routineVersion: { id: "deepRoutineVersion" } as any, // Mock routineVersion
                        }],
                    }],
                    nodeLinks: [],
                    routineVersionId: "level2Routine",
                }],
            }, {
                __type: RunStepType.End,
                description: "end",
                location: [1, 3],
                name: "end",
                nodeId: "endNode",
            }],
            nodeLinks: [],
            routineVersionId: "rootRoutine",
        };

        const result = insertStep(mockStepData, mockRootStep);

        // Traversing down to the deeply-nested step
        expect((((result.nodes[1] as RoutineListStep).steps[0] as MultiRoutineStep).nodes[0] as RoutineListStep).steps[0]).toEqual({ ...mockStepData, location: [1, 2, 1, 1, 1] });
    });

    it("does not change the root if the data doesn't match", () => {
        const mockStepData: Omit<MultiRoutineStep, "location"> = {
            __type: RunStepType.MultiRoutine,
            description: "non-matching data",
            name: "non-matching data",
            nodes: [],
            nodeLinks: [],
            routineVersionId: "nonMatchingRoutineVersion",
        };

        const mockRootStep: MultiRoutineStep = {
            __type: RunStepType.MultiRoutine,
            description: "root",
            location: [1],
            name: "root",
            nodes: [{
                __type: RunStepType.Start,
                description: "start",
                location: [1, 1],
                name: "start",
                nodeId: "startNode",
            }, {
                __type: RunStepType.RoutineList,
                description: "routine list",
                location: [1, 2],
                name: "routine list",
                nodeId: "routineListNode",
                steps: [{
                    __type: RunStepType.SingleRoutine,
                    description: "subroutine",
                    location: [1, 2, 1],
                    name: "subroutine",
                    routineVersion: { id: "routine333" } as RoutineVersion,
                }],
            }],
            nodeLinks: [],
            routineVersionId: "rootRoutine",
        };

        const result = insertStep(mockStepData, mockRootStep);
        expect(result).toEqual(mockRootStep);
    });
});

describe("stepFromLocation", () => {
    beforeAll(() => {
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        jest.spyOn(console, "warn").mockImplementation(() => { });
    });
    afterAll(() => {
        jest.restoreAllMocks();
    });

    it("should navigate through a Directory step correctly", () => {
        const rootStep: DirectoryStep = {
            __type: RunStepType.Directory,
            description: "boop",
            directoryId: "directory123",
            hasBeenQueried: true,
            isOrdered: false,
            isRoot: true,
            location: [1],
            name: "hello",
            projectVersionId: "projectVersion123",
            steps: [
                {
                    __type: RunStepType.SingleRoutine,
                    description: "a",
                    location: [1, 1],
                    name: "Step 1",
                    routineVersion: { id: "routineVersion123" } as RoutineVersion,
                },
                {
                    __type: RunStepType.SingleRoutine,
                    description: "b",
                    location: [1, 3], // Shouldn't matter that the location is incorrect, as it should rely on the structure of the steps rather than the defined locations
                    name: "Step 2",
                    routineVersion: { id: "routineVersion234" } as RoutineVersion,
                },
            ],
        };
        const result1 = stepFromLocation([1], rootStep);
        expect(result1).toEqual(rootStep);
        const result2 = stepFromLocation([1, 2], rootStep);
        expect(result2).toEqual(rootStep.steps[1]);
        const result3 = stepFromLocation([1, 3], rootStep);
        expect(result3).toBeNull();
        const result4 = stepFromLocation([2], rootStep);
        expect(result4).toBeNull();
        const result5 = stepFromLocation([2, 1], rootStep);
        expect(result5).toBeNull();
    });

    it("should navigate through a multi-step routine correctly", () => {
        const rootStep: MultiRoutineStep = {
            __type: RunStepType.MultiRoutine,
            description: "root",
            location: [1],
            name: "root",
            nodes: [
                {
                    __type: RunStepType.Start,
                    description: "start",
                    location: [1, 1],
                    name: "start",
                    nodeId: "startNode",
                },
                {
                    __type: RunStepType.RoutineList,
                    description: "routine list",
                    location: [1, 2],
                    name: "routine list",
                    nodeId: "routineListNode1",
                    steps: [
                        {
                            __type: RunStepType.SingleRoutine,
                            description: "subroutine",
                            location: [1, 2, 1],
                            name: "subroutine",
                            routineVersion: { id: "routineVersion123" } as RoutineVersion,
                        },
                    ],
                },
                {
                    __type: RunStepType.RoutineList,
                    description: "routine list",
                    location: [1, 3],
                    name: "routine list",
                    nodeId: "routineListNode2",
                    steps: [
                        {
                            __type: RunStepType.SingleRoutine,
                            description: "subroutine",
                            location: [1, 3, 1],
                            name: "subroutine",
                            routineVersion: { id: "routineVersion234" } as RoutineVersion,
                        },
                        {
                            __type: RunStepType.MultiRoutine,
                            description: "multi routine",
                            location: [1, 3, 2],
                            name: "multi routine",
                            nodes: [
                                {
                                    __type: RunStepType.RoutineList,
                                    description: "routine list",
                                    location: [1, 3, 2, 1],
                                    name: "routine list",
                                    nodeId: "routineListNode3",
                                    steps: [
                                        {
                                            __type: RunStepType.SingleRoutine,
                                            description: "subroutine",
                                            location: [1, 3, 2, 1, 1],
                                            name: "subroutine",
                                            routineVersion: { id: "routineVersion345" } as RoutineVersion,
                                        },
                                    ],
                                },
                            ],
                            nodeLinks: [],
                            routineVersionId: "multiRoutine123",
                        },
                    ],
                },
                {
                    __type: RunStepType.End,
                    description: "end",
                    location: [1, 3],
                    name: "end",
                    nodeId: "endNode",
                },
            ],
            nodeLinks: [],
            routineVersionId: "rootRoutine",
        };
        const result1 = stepFromLocation([1], rootStep);
        expect(result1).toEqual(rootStep);
        const result2 = stepFromLocation([1, 1], rootStep);
        expect(result2).toEqual(rootStep.nodes[0]);
        const result3 = stepFromLocation([1, 2], rootStep);
        expect(result3).toEqual(rootStep.nodes[1]);
        const result4 = stepFromLocation([1, 3, 2], rootStep);
        expect(result4).toEqual((rootStep.nodes[2] as RoutineListStep).steps[1]);
        const result5 = stepFromLocation([1, 3, 2, 1], rootStep);
        expect(result5).toEqual(((rootStep.nodes[2] as RoutineListStep).steps[1] as MultiRoutineStep).nodes[0]);
        const result6 = stepFromLocation([1, 3, 2, 1, 1], rootStep);
        expect(result6).toEqual((((rootStep.nodes[2] as RoutineListStep).steps[1] as MultiRoutineStep).nodes[0] as RoutineListStep).steps[0]);
    });

    // Should return null if the location is invalid
    it("should return null for an invalid location", () => {
        const rootStep: DirectoryStep = {
            __type: RunStepType.Directory,
            description: "root",
            directoryId: "directory123",
            hasBeenQueried: false,
            isOrdered: false,
            isRoot: true,
            location: [1],
            name: "root",
            projectVersionId: "projectVersion123",
            steps: [],
        };
        const result1 = stepFromLocation([2, 1], rootStep);
        expect(result1).toBeNull();
        const result2 = stepFromLocation([], rootStep);
        expect(result2).toBeNull();
    });
});

describe("getPreviousLocation", () => {
    beforeAll(() => {
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        jest.spyOn(console, "warn").mockImplementation(() => { });
    });
    afterAll(() => {
        jest.restoreAllMocks();
    });

    it("returns null if already at the root", () => {
        expect(getPreviousLocation([])).toBeNull();
    });

    it("returns the previous location when decremented", () => {
        expect(getPreviousLocation([1, 2, 3])).toEqual([1, 2, 2]);
        expect(getPreviousLocation([1, 2, 1])).toEqual([1, 2]);
    });

    it("returns the previous location and handles multiple decrements", () => {
        expect(getPreviousLocation([1, 1])).toEqual([1]);
        expect(getPreviousLocation([1])).toBeNull();
        expect(getPreviousLocation([2])).toEqual([1]);
    });

    it("handles deeply nested steps", () => {
        expect(getPreviousLocation([1, 2, 3, 4, 5])).toEqual([1, 2, 3, 4, 4]);
    });
});

describe("getNextLocation", () => {
    beforeAll(() => {
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        jest.spyOn(console, "warn").mockImplementation(() => { });
    });
    afterAll(() => {
        jest.restoreAllMocks();
    });

    it("returns null if the root step is invalid", () => {
        expect(getNextLocation([1], null)).toBeNull();
    });

    it("works with directories", () => {
        const rootStep: DirectoryStep = {
            __type: RunStepType.Directory,
            description: "root",
            directoryId: "directory123",
            hasBeenQueried: false,
            isOrdered: false,
            isRoot: true,
            location: [1],
            name: "root",
            projectVersionId: "projectVersion123",
            steps: [
                {
                    __type: RunStepType.SingleRoutine,
                    description: "a",
                    location: [1, 1],
                    name: "Step 1",
                    routineVersion: { id: "routineVersion123" } as RoutineVersion,
                },
                {
                    __type: RunStepType.SingleRoutine,
                    description: "b",
                    location: [1, 2],
                    name: "Step 2",
                    routineVersion: { id: "routineVersion234" } as RoutineVersion,
                },
            ],
        };
        expect(getNextLocation([1], rootStep)).toEqual([1, 1]);
        expect(getNextLocation([1, 1], rootStep)).toEqual([1, 2]);
        expect(getNextLocation([1, 2], rootStep)).toBeNull();
        expect(getNextLocation([2], rootStep)).toBeNull();
        expect(getNextLocation([], rootStep)).toEqual([1]);
    });

    it("works with multi-step routines", () => {
        const rootStep: MultiRoutineStep = {
            __type: RunStepType.MultiRoutine,
            description: "root",
            location: [1],
            name: "root",
            nodes: [
                {
                    __type: RunStepType.Start,
                    description: "start",
                    location: [1, 1],
                    name: "start",
                    nodeId: "startNode",
                },
                {
                    __type: RunStepType.RoutineList,
                    description: "routine list",
                    location: [1, 2],
                    name: "routine list",
                    nodeId: "routineListNode1",
                    steps: [
                        {
                            __type: RunStepType.SingleRoutine,
                            description: "subroutine",
                            location: [1, 2, 1],
                            name: "subroutine",
                            routineVersion: { id: "routineVersion123" } as RoutineVersion,
                        },
                    ],
                },
                {
                    __type: RunStepType.RoutineList,
                    description: "routine list",
                    location: [1, 3],
                    name: "routine list",
                    nodeId: "routineListNode2",
                    steps: [
                        {
                            __type: RunStepType.SingleRoutine,
                            description: "subroutine",
                            location: [1, 3, 1],
                            name: "subroutine",
                            routineVersion: { id: "routineVersion234" } as RoutineVersion,
                        },
                        {
                            __type: RunStepType.MultiRoutine,
                            description: "multi routine",
                            location: [1, 3, 2],
                            name: "multi routine",
                            nodes: [
                                {
                                    __type: RunStepType.RoutineList,
                                    description: "routine list",
                                    location: [1, 3, 2, 1],
                                    name: "routine list",
                                    nodeId: "routineListNode3",
                                    steps: [
                                        {
                                            __type: RunStepType.SingleRoutine,
                                            description: "subroutine",
                                            location: [1, 3, 2, 1, 1],
                                            name: "subroutine",
                                            routineVersion: { id: "routineVersion345" } as RoutineVersion,
                                        },
                                    ],
                                },
                            ],
                            nodeLinks: [],
                            routineVersionId: "multiRoutine123",
                        },
                    ],
                },
                {
                    __type: RunStepType.End,
                    description: "end",
                    location: [1, 4],
                    name: "end",
                    nodeId: "endNode",
                },
            ],
            nodeLinks: [],
            routineVersionId: "rootRoutine",
        };
        expect(getNextLocation([1], rootStep)).toEqual([1, 1]);
        expect(getNextLocation([1, 1], rootStep)).toEqual([1, 2]);
        expect(getNextLocation([1, 2], rootStep)).toEqual([1, 2, 1]);
        expect(getNextLocation([1, 2, 1], rootStep)).toEqual([1, 3]);
        expect(getNextLocation([1, 3], rootStep)).toEqual([1, 3, 1]);
        expect(getNextLocation([1, 3, 1], rootStep)).toEqual([1, 3, 2]);
        expect(getNextLocation([1, 3, 2], rootStep)).toEqual([1, 3, 2, 1]);
        expect(getNextLocation([1, 3, 2, 1], rootStep)).toEqual([1, 3, 2, 1, 1]);
        expect(getNextLocation([1, 3, 2, 1, 1], rootStep)).toEqual([1, 4]);
    });

    it("returns null if we're in a decision step", () => {
        const rootStep: MultiRoutineStep = {
            __type: RunStepType.MultiRoutine,
            description: "root",
            location: [1],
            name: "root",
            nodes: [
                {
                    __type: RunStepType.Start,
                    description: "start",
                    location: [1, 1],
                    name: "start",
                    nodeId: "startNode",
                },
                {
                    __type: RunStepType.Decision,
                    description: "decision",
                    location: [1, 2],
                    name: "Decision Step",
                    options: [
                        {
                            link: {} as NodeLink,
                            step: {
                                __type: RunStepType.RoutineList,
                                description: "routine list 1",
                                isOrdered: false,
                                location: [1, 3, 2, 1],
                                name: "routine list a",
                                nodeId: "routineListNode3",
                                steps: [
                                    {
                                        __type: RunStepType.SingleRoutine,
                                        description: "subroutine 1",
                                        location: [1, 3, 2, 1, 1],
                                        name: "subroutine a",
                                        routineVersion: { id: "routineVersion345" } as RoutineVersion,
                                    },
                                ],
                            } as RoutineListStep,
                        },
                        {
                            link: {} as NodeLink,
                            step: {
                                __type: RunStepType.RoutineList,
                                description: "routine list 2",
                                isOrdered: true,
                                location: [1, 4, 2, 1, 2, 2, 2],
                                name: "routine list b",
                                nodeId: "routineListNode3",
                                steps: [
                                    {
                                        __type: RunStepType.SingleRoutine,
                                        description: "subroutine 2",
                                        location: [1, 3, 2, 1, 1],
                                        name: "subroutine b",
                                        routineVersion: { id: "routineVersion345" } as RoutineVersion,
                                    },
                                ],
                            } as RoutineListStep,
                        },
                    ],
                },
                {
                    __type: RunStepType.RoutineList,
                    description: "routine list",
                    location: [1, 3],
                    name: "routine list",
                    nodeId: "routineListNode2",
                    steps: [
                        {
                            __type: RunStepType.SingleRoutine,
                            description: "subroutine",
                            location: [1, 3, 1],
                            name: "subroutine",
                            routineVersion: { id: "routineVersion234" } as RoutineVersion,
                        },
                        {
                            __type: RunStepType.MultiRoutine,
                            description: "multi routine",
                            location: [1, 3, 2],
                            name: "multi routine",
                            nodes: [
                                {
                                    __type: RunStepType.RoutineList,
                                    description: "routine list",
                                    location: [1, 3, 2, 1],
                                    name: "routine list",
                                    nodeId: "routineListNode3",
                                    steps: [
                                        {
                                            __type: RunStepType.SingleRoutine,
                                            description: "subroutine",
                                            location: [1, 3, 2, 1, 1],
                                            name: "subroutine",
                                            routineVersion: { id: "routineVersion345" } as RoutineVersion,
                                        },
                                    ],
                                },
                            ],
                            nodeLinks: [],
                            routineVersionId: "multiRoutine123",
                        },
                    ],
                },
                {
                    __type: RunStepType.End,
                    description: "end",
                    location: [1, 4],
                    name: "end",
                    nodeId: "endNode",
                },
            ],
            nodeLinks: [],
            routineVersionId: "rootRoutine",
        };
        // Points to a decision, so it should be fine
        expect(getNextLocation([1, 1], rootStep)).toEqual([1, 2]);
        // Start location is a decision, so it should return null
        expect(getNextLocation([1, 2], rootStep)).toBeNull();
    });
});

describe("stepNeedsQuerying", () => {
    beforeAll(() => {
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        jest.spyOn(console, "warn").mockImplementation(() => { });
    });
    afterAll(() => {
        jest.restoreAllMocks();
    });

    it("returns false if the step is null", () => {
        expect(stepNeedsQuerying(null)).toBe(false);
    });

    it("returns false if the step is undefined", () => {
        expect(stepNeedsQuerying(undefined)).toBe(false);
    });

    it("returns false for an EndStep", () => {
        const endStep: EndStep = {
            __type: RunStepType.End,
            description: "hi",
            name: "End",
            nodeId: "node123",
            location: [1, 2],
            wasSuccessful: true,
        };
        expect(stepNeedsQuerying(endStep)).toBe(false);
    });

    it("returns false for a StartStep", () => {
        const startStep: StartStep = {
            __type: RunStepType.Start,
            description: "hey",
            name: "Start",
            nodeId: "node123",
            location: [1, 1],
        };
        expect(stepNeedsQuerying(startStep)).toBe(false);
    });

    describe("SingleRoutineStep", () => {
        it("returns true if the subroutine has subroutines", () => {
            const singleRoutineStep: SingleRoutineStep = {
                __type: RunStepType.SingleRoutine,
                description: "boop",
                name: "Subroutine Step",
                location: [1],
                routineVersion: {
                    id: "routine123",
                    routineType: RoutineType.MultiStep,
                    nodes: [{ id: "node123" }] as Node[],
                    nodeLinks: [{ id: "link123" }] as NodeLink[],
                } as RoutineVersion,
            };
            expect(stepNeedsQuerying(singleRoutineStep)).toBe(true);
        });

        it("returns false if the subroutine does not have subroutines", () => {
            const singleRoutineStep: SingleRoutineStep = {
                __type: RunStepType.SingleRoutine,
                description: "boopies",
                name: "Subroutine Step",
                location: [1],
                routineVersion: {
                    id: "routine123",
                    routineType: RoutineType.Informational,
                } as RoutineVersion,
            };
            expect(stepNeedsQuerying(singleRoutineStep)).toBe(false);
        });
    });

    describe("DirectoryStep", () => {
        it("returns true if the directory has not been queried yet", () => {
            const directoryStep: DirectoryStep = {
                __type: RunStepType.Directory,
                description: "boop",
                directoryId: "directory123",
                hasBeenQueried: false, // Marked as not queried
                isOrdered: false,
                isRoot: false,
                name: "Directory",
                location: [1],
                projectVersionId: "projectVersion123",
                steps: [{ name: "Child Step", __type: RunStepType.SingleRoutine }] as DirectoryStep["steps"],
            };
            expect(stepNeedsQuerying(directoryStep)).toBe(true);
        });

        it("returns false if the directory has already been queried", () => {
            const directoryStep: DirectoryStep = {
                __type: RunStepType.Directory,
                description: "boop",
                directoryId: "directory123",
                hasBeenQueried: true, // Marked as already queried
                isOrdered: false,
                isRoot: false,
                name: "Directory",
                location: [1],
                projectVersionId: "projectVersion123",
                steps: [{ name: "Child Step", __type: RunStepType.SingleRoutine }] as DirectoryStep["steps"],
            };
            expect(stepNeedsQuerying(directoryStep)).toBe(false);
        });
    });
});

describe("getStepComplexity", () => {
    beforeAll(() => {
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        jest.spyOn(console, "warn").mockImplementation(() => { });
    });
    afterAll(() => {
        jest.restoreAllMocks();
    });

    it("should return 0 for End step", () => {
        const endStep: RunStep = { __type: RunStepType.End, description: "", name: "", nodeId: "1", location: [1] };
        expect(getStepComplexity(endStep)).toBe(0);
    });

    it("should return 0 for Start step", () => {
        const startStep: RunStep = { __type: RunStepType.Start, description: "", name: "", nodeId: "1", location: [1] };
        expect(getStepComplexity(startStep)).toBe(0);
    });

    it("should return 1 for Decision step", () => {
        const decisionStep: RunStep = { __type: RunStepType.Decision, description: "", name: "", nodeId: "1", location: [1], options: [] };
        expect(getStepComplexity(decisionStep)).toBe(1);
    });

    it("should return the complexity of SingleRoutine step", () => {
        const singleRoutineStep: SingleRoutineStep = {
            __type: RunStepType.SingleRoutine,
            description: "",
            name: "",
            location: [1],
            routineVersion: { complexity: 3, id: "1", routineType: RoutineType.MultiStep } as RoutineVersion,
        };
        expect(getStepComplexity(singleRoutineStep)).toBe(3); // Complexity of the routine version
    });

    it("should calculate complexity for MultiRoutine step", () => {
        const multiRoutineStep: MultiRoutineStep = {
            __type: RunStepType.MultiRoutine,
            description: "",
            name: "",
            location: [1],
            nodes: [
                { __type: RunStepType.Start, description: "", name: "", nodeId: "1", location: [1, 1] },
                { __type: RunStepType.Decision, description: "", name: "", nodeId: "2", location: [1, 2], options: [] },
                {
                    __type: RunStepType.RoutineList,
                    description: "",
                    name: "",
                    nodeId: "1",
                    location: [1],
                    steps: [
                        {
                            __type: RunStepType.SingleRoutine,
                            description: "",
                            name: "",
                            location: [1, 1],
                            routineVersion: { complexity: 1, id: "1", routineType: RoutineType.Informational } as RoutineVersion,
                        },
                        {
                            __type: RunStepType.SingleRoutine,
                            description: "",
                            name: "",
                            location: [1, 2],
                            routineVersion: { complexity: 9, id: "2", routineType: RoutineType.MultiStep } as RoutineVersion,
                        },
                    ],
                },
                { __type: RunStepType.End, description: "", name: "", nodeId: "3", location: [1, 3] },
            ],
            nodeLinks: [],
            routineVersionId: "1",
        };
        expect(getStepComplexity(multiRoutineStep)).toBe(11);
    });

    it("should calculate complexity for RoutineList step", () => {
        const routineListStep: RoutineListStep = {
            __type: RunStepType.RoutineList,
            description: "",
            isOrdered: false,
            name: "",
            nodeId: "1",
            location: [1],
            parentRoutineVersionId: "420",
            steps: [
                {
                    __type: RunStepType.SingleRoutine,
                    description: "",
                    name: "",
                    location: [1, 1],
                    routineVersion: { complexity: 2, id: "1", routineType: RoutineType.MultiStep } as RoutineVersion,
                },
                {
                    __type: RunStepType.MultiRoutine,
                    description: "root",
                    location: [1],
                    name: "root",
                    nodes: [
                        {
                            __type: RunStepType.Start,
                            description: "start",
                            location: [1, 1],
                            name: "start",
                            nodeId: "startNode",
                        },
                        {
                            __type: RunStepType.Decision,
                            description: "decision",
                            location: [1, 2],
                            name: "Decision Step",
                            options: [
                                {
                                    link: {} as NodeLink,
                                    step: {
                                        __type: RunStepType.RoutineList,
                                        description: "routine list 1",
                                        isOrdered: false,
                                        location: [1, 3, 2, 1],
                                        name: "routine list a",
                                        nodeId: "routineListNode3",
                                        steps: [
                                            {
                                                __type: RunStepType.SingleRoutine,
                                                description: "subroutine 1",
                                                location: [1, 3, 2, 1, 1],
                                                name: "subroutine a",
                                                routineVersion: { id: "routineVersion345", complexity: 100000 } as RoutineVersion,
                                            },
                                        ],
                                    } as RoutineListStep,
                                },
                                {
                                    link: {} as NodeLink,
                                    step: {
                                        __type: RunStepType.RoutineList,
                                        description: "routine list 2",
                                        isOrdered: true,
                                        location: [1, 4, 2, 1, 2, 2, 2],
                                        name: "routine list b",
                                        nodeId: "routineListNode3",
                                        steps: [
                                            {
                                                __type: RunStepType.SingleRoutine,
                                                description: "subroutine 2",
                                                location: [1, 3, 2, 1, 1],
                                                name: "subroutine b",
                                                routineVersion: { id: "routineVersion345", complexity: 9999999 } as RoutineVersion,
                                            },
                                        ],
                                    } as RoutineListStep,
                                },
                            ],
                        },
                        {
                            __type: RunStepType.RoutineList,
                            description: "routine list",
                            location: [1, 3],
                            name: "routine list",
                            nodeId: "routineListNode2",
                            steps: [
                                {
                                    __type: RunStepType.SingleRoutine,
                                    description: "subroutine",
                                    location: [1, 3, 1],
                                    name: "subroutine",
                                    routineVersion: { id: "routineVersion234", complexity: 2 } as RoutineVersion,
                                },
                                {
                                    __type: RunStepType.MultiRoutine,
                                    description: "multi routine",
                                    location: [1, 3, 2],
                                    name: "multi routine",
                                    nodes: [
                                        {
                                            __type: RunStepType.RoutineList,
                                            description: "routine list",
                                            location: [1, 3, 2, 1],
                                            name: "routine list",
                                            nodeId: "routineListNode3",
                                            steps: [
                                                {
                                                    __type: RunStepType.SingleRoutine,
                                                    description: "subroutine",
                                                    location: [1, 3, 2, 1, 1],
                                                    name: "subroutine",
                                                    routineVersion: { id: "routineVersion345", complexity: 10 } as RoutineVersion,
                                                },
                                            ],
                                        },
                                    ],
                                    nodeLinks: [],
                                    routineVersionId: "multiRoutine123",
                                },
                            ],
                        },
                        {
                            __type: RunStepType.End,
                            description: "end",
                            location: [1, 4],
                            name: "end",
                            nodeId: "endNode",
                        },
                    ],
                    nodeLinks: [],
                    routineVersionId: "rootRoutine",
                },
            ],
        };
        expect(getStepComplexity(routineListStep)).toBe(15);
    });

    it("should calculate complexity for Directory step", () => {
        const directoryStep: DirectoryStep = {
            __type: RunStepType.Directory,
            description: "",
            name: "",
            location: [1],
            directoryId: "1",
            hasBeenQueried: true,
            isOrdered: false,
            isRoot: false,
            projectVersionId: "1",
            steps: [
                {
                    __type: RunStepType.SingleRoutine,
                    description: "",
                    name: "",
                    location: [1, 1],
                    routineVersion: { complexity: 2, id: "1", routineType: RoutineType.MultiStep } as RoutineVersion,
                },
                {
                    __type: RunStepType.Directory,
                    description: "b",
                    directoryId: null,
                    hasBeenQueried: false,
                    isOrdered: false,
                    isRoot: true,
                    location: [1, 1],
                    name: "b",
                    projectVersionId: "projectVersion123", // Matches mockStepData, so it should be replaced by mockStepData
                    steps: [{
                        __type: RunStepType.SingleRoutine,
                        description: "",
                        name: "",
                        location: [1, 1],
                        routineVersion: { complexity: 12, id: "1", routineType: RoutineType.MultiStep } as RoutineVersion,
                    }],
                },
            ],
        };
        expect(getStepComplexity(directoryStep)).toBe(14);
    });
});

describe("parseChildOrder", () => {
    beforeAll(() => {
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        jest.spyOn(console, "warn").mockImplementation(() => { });
    });
    afterAll(() => {
        jest.restoreAllMocks();
    });

    // It should work with both uuids and alphanumeric strings
    const id1 = uuid();
    const id2 = "123";
    const id3 = uuid();
    const id4 = "9ahqqh9AGZ";
    const id5 = uuid();
    const id6 = "88201376AB1IUO96883667654";

    describe("without sides", () => {
        it("comma-separated string", () => {
            const input = `${id1},${id2},${id3},${id4}`;
            const expected = [id1, id2, id3, id4];
            expect(parseChildOrder(input)).toEqual(expected);
        });
        it("space and comma-separated string", () => {
            const input = `${id1},${id2} ${id3}, ${id4},,${id5}`;
            const expected = [id1, id2, id3, id4, id5];
            expect(parseChildOrder(input)).toEqual(expected);
        });
        it("single UUID", () => {
            const input = uuid();
            const expected = [input];
            expect(parseChildOrder(input)).toEqual(expected);
        });
        it("single alphanumeric string", () => {
            const input = "123abCD";
            const expected = [input];
            expect(parseChildOrder(input)).toEqual(expected);
        });
    });

    describe("with left and right sides", () => {
        it("comma delimiter", () => {
            const input = `l(${id1},${id2},${id3}),r(${id4},${id5},${id6})`;
            const expected = [id1, id2, id3, id4, id5, id6];
            expect(parseChildOrder(input)).toEqual(expected);
        });
        it("comma and space delimiter", () => {
            const input = `l(${id1},${id2},${id3}), r(${id4},${id5},${id6})`;
            const expected = [id1, id2, id3, id4, id5, id6];
            expect(parseChildOrder(input)).toEqual(expected);
        });
        it("spaces delimiter", () => {
            const input = `l(${id1},${id2},${id3})   r(${id4},${id5},${id6})`;
            const expected = [id1, id2, id3, id4, id5, id6];
            expect(parseChildOrder(input)).toEqual(expected);
        });
        it("no delimiter", () => {
            const input = `l(${id1},${id2},${id3})r(${id4},${id5},${id6})`;
            const expected = [id1, id2, id3, id4, id5, id6];
            expect(parseChildOrder(input)).toEqual(expected);
        });
    });

    it("should return an empty array for an empty string", () => {
        const input = "";
        const expected: string[] = [];
        expect(parseChildOrder(input)).toEqual(expected);
    });

    it("should parse root format with empty left order correctly", () => {
        const input = `l(),r(${id4},${id2},${id6})`;
        const expected = [id4, id2, id6];
        expect(parseChildOrder(input)).toEqual(expected);
    });

    it("should parse root format with empty right order correctly", () => {
        const input = `l(${id1},${id2},${id6}),r()`;
        const expected = [id1, id2, id6];
        expect(parseChildOrder(input)).toEqual(expected);
    });

    it("should return an empty array for root format with both orders empty", () => {
        const input = "l(),r()";
        const expected: string[] = [];
        expect(parseChildOrder(input)).toEqual(expected);
    });

    describe("should handle malformed input string", () => {
        it("no closing parenthesis", () => {
            const input = "l(123,456,789";
            const expected = [];
            expect(parseChildOrder(input)).toEqual(expected);
        });
        it("invalid character", () => {
            const input = "23423_3242";
            const expected = [];
            expect(parseChildOrder(input)).toEqual(expected);
        });
        it("characters after closing parenthesis", () => {
            const input = "l(123)2342";
            const expected = [];
            expect(parseChildOrder(input)).toEqual(expected);
        });
        it("nested parentheses", () => {
            const input = "l(333,(222,111),555),r(888,123,321)";
            const expected = [];
            expect(parseChildOrder(input)).toEqual(expected);
        });
    });
});

/**
 * Jest helper function to validate the sequence of steps in a routine.
 * It ensures that steps appear in a logical order based on the node links.
 */
function expectValidStepSequence(
    steps: MultiRoutineStep["nodes"],
    nodeLinks: NodeLink[],
) {
    if (steps.length === 0) return;
    expect(steps[0].__type).toBe(RunStepType.Start);

    // Check that the next step logically follows the current step
    for (let i = 0; i < steps.length - 1; i++) {
        const currentStep = steps[i];
        const nextStep = steps[i + 1];

        // Ignore end steps, as they are the last step in the sequence
        if (currentStep.__type === RunStepType.End) {
            continue;
        }

        // Ignore decision steps, as they jump to a different step instead of following the sequence
        if (currentStep.__type === RunStepType.Decision) {
            continue;
        }

        // Multiple outgoing links indicate a decision step next
        const outgoingLinks = nodeLinks.filter(link => link.from?.id === (currentStep as RoutineListStep | StartStep).nodeId);
        if (outgoingLinks.length > 1) {
            expect(nextStep.__type).toBe(
                RunStepType.Decision,
                // @ts-ignore: expect-message
                `Expected a Decision step after a step with multiple outgoing links. Step index: ${i}`,
            );
            continue;
        } else {
            expect(nextStep.__type).not.toBe(
                RunStepType.Decision,
                // @ts-ignore: expect-message
                `Expected a non-Decision step after a step with a single outgoing link. Step index: ${i}`,
            );
        }

        // Needs at least 1 outgoing link to the next step
        const hasLinkToNextStep = outgoingLinks.some(link => link.to?.id === (nextStep as EndStep | RoutineListStep | StartStep).nodeId);
        expect(hasLinkToNextStep).toBe(
            true,
            // @ts-ignore: expect-message
            `Expected a link from step ${currentStep.nodeId} to ${nextStep.nodeId}. Step index: ${i}`,
        );
    }

    // Validate the last step is either:
    // - An EndStep
    // - A DecisionStep (as it'll move you to another part of the list)
    // - A RoutineListStep with a `redirect` property (indicates cycle, and will move you to another part of the list)
    const lastStep = steps[steps.length - 1];
    if (![RunStepType.End, RunStepType.Decision, RunStepType.RoutineList].includes(lastStep.__type)) {
        throw new Error(`Unexpected last step type: ${lastStep.__type}`);
    }
    if (lastStep.__type === RunStepType.RoutineList) {
        const redirect = Object.prototype.hasOwnProperty.call(lastStep, "redirectId") ? lastStep.redirectId : undefined;
        if (!redirect) {
            throw new Error("Cannot end on a RoutineList without a redirect property");
        }
    }
    expect(
        lastStep.__type === RunStepType.End ||
        lastStep.__type === RunStepType.Decision ||
        (lastStep.__type === RunStepType.RoutineList && "redirect" in lastStep && lastStep.redirect !== undefined),
        // @ts-ignore: expect-message
        `Expected the last step to be an EndStep, DecisionStep, or RoutineListStep with a redirect property. Got ${lastStep.__type}`,
    ).toBe(true);

    // If the last step is a RoutineListStep with a redirect, validate the redirect
    if (lastStep.__type === RunStepType.RoutineList && "redirect" in lastStep && lastStep.redirect !== undefined) {
        const redirectTarget = steps.find(step =>
            (step as RoutineListStep | StartStep).nodeId === lastStep.redirect,
        );
        expect(redirectTarget).toBeDefined(
            // @ts-ignore: expect-message
            `Redirect target ${lastStep.redirect} not found in steps`,
        );
    }
}

describe("sortStepsAndAddDecisions", () => {
    it("doesn't sort if start step missing", () => {
        const steps = [
            { __type: RunStepType.End, description: "", name: "", nodeId: "1", location: [1, 3] },
        ];
        const nodeLinks = [] as NodeLink[];
        const result = sortStepsAndAddDecisions(steps, nodeLinks);
        expect(result).toEqual(steps); // Result should be unchanged
    });

    //TODO test unlinked steps removed

    it("should sort steps in correct order for a linear path", () => {
        const steps = [
            { __type: RunStepType.Start, description: "", name: "", nodeId: "1", location: [1, 1] },
            { __type: RunStepType.RoutineList, description: "", name: "", nodeId: "2", location: [1, 2], steps: [] },
            { __type: RunStepType.End, description: "", name: "", nodeId: "3", location: [1, 3] },
        ];
        const nodeLinks = [
            { from: { id: "1" }, to: { id: "2" } },
            { from: { id: "2" }, to: { id: "3" } },
        ] as NodeLink[];
        const result = sortStepsAndAddDecisions(steps, nodeLinks);
        expect(result.map(step => step.nodeId)).toEqual(["1", "2", "3"]);
    });

    it("should add a decision step when there are multiple outgoing links", () => {
        const steps = [
            { __type: RunStepType.Start, description: "", name: "", nodeId: "1", location: [1, 1] },
            { __type: RunStepType.RoutineList, description: "", name: "", nodeId: "2", location: [1, 2], steps: [] },
            { __type: RunStepType.RoutineList, description: "", name: "", nodeId: "3", location: [1, 3], steps: [] },
            { __type: RunStepType.End, description: "", name: "", nodeId: "4", location: [1, 4] },
        ];
        const nodeLinks = [
            // Start to both RoutineLists
            { from: { id: "1" }, to: { id: "2" } },
            { from: { id: "1" }, to: { id: "3" } },
            { from: { id: "2" }, to: { id: "4" } },
            // RoutineList to End
            { from: { id: "3" }, to: { id: "4" } },
        ] as NodeLink[];
        const result = sortStepsAndAddDecisions(steps, nodeLinks);
        expectValidStepSequence(result, nodeLinks);
        expect(result.length).toBe(steps.length + 1); // One decision step
    });

    it("should handle cycles in the graph", () => {
        const steps = [
            { __type: RunStepType.Start, description: "", name: "", nodeId: "1", location: [1, 1] },
            { __type: RunStepType.RoutineList, description: "", name: "", nodeId: "2", location: [1, 2], steps: [] },
            { __type: RunStepType.RoutineList, description: "", name: "", nodeId: "3", location: [1, 3], steps: [] },
            { __type: RunStepType.RoutineList, description: "", name: "", nodeId: "4", location: [1, 2], steps: [] },
            { __type: RunStepType.End, description: "", name: "", nodeId: "5", location: [1, 3] },
        ];
        const nodeLinks = [
            // Start to first RoutineList
            { from: { id: "1" }, to: { id: "2" } },
            // First RoutineList to next two RoutineLists
            { from: { id: "2" }, to: { id: "3" } },
            { from: { id: "2" }, to: { id: "4" } },
            // Second RoutineList to End
            { from: { id: "3" }, to: { id: "5" } },
            // Third RoutineList to first RoutineList
            { from: { id: "4" }, to: { id: "2" } },
        ] as NodeLink[];
        console.log("yeet before");
        const result = sortStepsAndAddDecisions(steps, nodeLinks);
        console.log("yeet after", JSON.stringify(result));
        expectValidStepSequence(result, nodeLinks);
        expect(result.length).toBe(steps.length + 1); // One decision step
    });

    // it("should set correct locations for all steps", () => {
    //     const steps = [
    //         { __type: RunStepType.Start, description: "", name: "", nodeId: "1", location: [1, 1] },
    //         { __type: RunStepType.RoutineList, description: "", name: "", nodeId: "2", location: [1, 2], steps: [] },
    //         { __type: RunStepType.End, description: "", name: "", nodeId: "3", location: [1, 3] },
    //     ];
    //     const nodeLinks = [
    //         { from: { id: "1" }, to: { id: "2" } },
    //         { from: { id: "2" }, to: { id: "3" } },
    //     ] as NodeLink[];
    //     const result = sortStepsAndAddDecisions(steps, nodeLinks);
    //     expect(result[0].location).toEqual([1, 1]);
    //     expect(result[1].location).toEqual([1, 2]);
    //     expect(result[2].location).toEqual([1, 3]);
    // });

    // it("should not add decision step for single outgoing link", () => {
    //     const steps = [
    //         { __type: RunStepType.Start, description: "", name: "", nodeId: "1", location: [1, 1] },
    //         { __type: RunStepType.RoutineList, description: "", name: "", nodeId: "2", location: [1, 2], steps: [] },
    //         { __type: RunStepType.End, description: "", name: "", nodeId: "3", location: [1, 3] },
    //     ];
    //     const nodeLinks = [
    //         { from: { id: "1" }, to: { id: "2" } },
    //         { from: { id: "2" }, to: { id: "3" } },
    //     ] as NodeLink[];
    //     const result = sortStepsAndAddDecisions(steps, nodeLinks);
    //     expect(result.every(step => step.__type !== RunStepType.Decision)).toBe(true);
    // });

    // it("should handle complex graphs with multiple decision points", () => {
    //     const steps = [
    //         { __type: RunStepType.Start, description: "", name: "", nodeId: "1", location: [1, 1] },
    //         { __type: RunStepType.RoutineList, description: "", name: "", nodeId: "2", location: [1, 2], steps: [] },
    //         { __type: RunStepType.RoutineList, description: "", name: "", nodeId: "3", location: [1, 3], steps: [] },
    //         { __type: RunStepType.RoutineList, description: "", name: "", nodeId: "4", location: [1, 4], steps: [] },
    //         { __type: RunStepType.RoutineList, description: "", name: "", nodeId: "5", location: [1, 5], steps: [] },
    //         { __type: RunStepType.End, description: "", name: "", nodeId: "6", location: [1, 6] },
    //     ];
    //     const nodeLinks = [
    //         { from: { id: "1" }, to: { id: "2" } },
    //         { from: { id: "1" }, to: { id: "3" } },
    //         { from: { id: "2" }, to: { id: "4" } },
    //         { from: { id: "2" }, to: { id: "5" } },
    //         { from: { id: "3" }, to: { id: "4" } },
    //         { from: { id: "3" }, to: { id: "5" } },
    //         { from: { id: "4" }, to: { id: "6" } },
    //         { from: { id: "5" }, to: { id: "6" } },
    //     ] as NodeLink[];
    //     const result = sortStepsAndAddDecisions(steps, nodeLinks);
    //     expect(result.filter(step => step.__type === RunStepType.Decision).length).toBe(3);
    // });

    // it("should ignore visited nodes when creating decision steps", () => {
    //     const steps = [
    //         { __type: RunStepType.Start, description: "", name: "", nodeId: "1", location: [1, 1] },
    //         { __type: RunStepType.RoutineList, description: "", name: "", nodeId: "2", location: [1, 2], steps: [] },
    //         { __type: RunStepType.RoutineList, description: "", name: "", nodeId: "3", location: [1, 3], steps: [] },
    //         { __type: RunStepType.End, description: "", name: "", nodeId: "4", location: [1, 4] },
    //     ];
    //     const nodeLinks = [
    //         { from: { id: "1" }, to: { id: "2" } },
    //         { from: { id: "1" }, to: { id: "3" } },
    //         { from: { id: "2" }, to: { id: "4" } },
    //         { from: { id: "3" }, to: { id: "2" } },
    //     ] as NodeLink[];
    //     const result = sortStepsAndAddDecisions(steps, nodeLinks);
    //     const decisionStep = result.find(step => step.__type === RunStepType.Decision);
    //     expect(decisionStep.options.length).toBe(2);
    //     expect(decisionStep.options.every(option => option.step.nodeId !== "4")).toBe(true);
    // });

    // it("should handle graphs with isolated nodes", () => {
    //     const steps = [
    //         { __type: RunStepType.Start, description: "", name: "", nodeId: "1", location: [1, 1] },
    //         { __type: RunStepType.RoutineList, description: "", name: "", nodeId: "2", location: [1, 2], steps: [] },
    //         { __type: RunStepType.End, description: "", name: "", nodeId: "3", location: [1, 3] },
    //     ];
    //     const nodeLinks = [
    //         { from: { id: "1" }, to: { id: "3" } },
    //     ] as NodeLink[];
    //     const result = sortStepsAndAddDecisions(steps, nodeLinks);
    //     expect(result.map(step => step.nodeId)).toEqual(["1", "3"]);
    // });

    // it("should correctly handle RoutineList steps with nested SingleRoutine steps", () => {
    //     const steps = [
    //         { __type: RunStepType.Start, description: "", name: "", nodeId: "1", location: [1, 1] },
    //         {
    //             __type: RunStepType.RoutineList,
    //             description: "",
    //             name: "",
    //             nodeId: "2",
    //             location: [1, 2],
    //             steps: [
    //                 {
    //                     __type: RunStepType.SingleRoutine,
    //                     description: "",
    //                     name: "",
    //                     location: [1, 2, 1],
    //                     routineVersion: { complexity: 1, id: "3", routineType: RoutineType.Informational } as RoutineVersion,
    //                 },
    //                 {
    //                     __type: RunStepType.SingleRoutine,
    //                     description: "",
    //                     name: "",
    //                     location: [1, 2, 2],
    //                     routineVersion: { complexity: 9, id: "4", routineType: RoutineType.MultiStep } as RoutineVersion,
    //                 },
    //             ],
    //         },
    //         { __type: RunStepType.End, description: "", name: "", nodeId: "5", location: [1, 3] },
    //     ];
    //     const nodeLinks = [
    //         { from: { id: "1" }, to: { id: "2" } },
    //         { from: { id: "2" }, to: { id: "5" } },
    //     ] as NodeLink[];
    //     const result = sortStepsAndAddDecisions(steps, nodeLinks);
    //     expect(result.map(step => step.nodeId)).toEqual(["1", "2", "5"]);
    //     expect((result[1] as any).steps.length).toBe(2);
    //     expect((result[1] as any).steps[0].__type).toBe(RunStepType.SingleRoutine);
    //     expect((result[1] as any).steps[1].__type).toBe(RunStepType.SingleRoutine);
    // });
});
