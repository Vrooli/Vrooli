/* eslint-disable @typescript-eslint/ban-ts-comment */
import { Node, NodeLink, RoutineType, RoutineVersion, uuid } from "@local/shared";
import { DecisionStep, DirectoryStep, EndStep, MultiRoutineStep, RootStep, RoutineListStep, RunStep, SingleRoutineStep, StartStep } from "../../../types";
import { RunStepType } from "../../../utils/consts";
import { getNextLocation, getPreviousLocation, getStepComplexity, insertStep, parseChildOrder, siblingsAtLocaiton, sortStepsAndAddDecisions, stepFromLocation, stepNeedsQuerying } from "./RunView";

describe("insertStep", () => {
    beforeAll(() => {
        // eslint-disable-next-line @typescript-eslint/no-empty-function
        jest.spyOn(console, "warn").mockImplementation(() => { });
    });
    afterAll(() => {
        jest.restoreAllMocks();
    });

    it("adds project data to a project", () => {
        const mockStepData: DirectoryStep = {
            __type: RunStepType.Directory,
            description: "a",
            directoryId: null,
            hasBeenQueried: true,
            isOrdered: false,
            isRoot: true,
            location: [1, 2, 3],
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
        const mockStepData: MultiRoutineStep = {
            __type: RunStepType.MultiRoutine,
            description: "newBoi",
            location: [3, 3, 2],
            name: "hello",
            nodes: [],
            nodeLinks: [],
            routineVersionId: "routineVersion999",
        };
        //TODO when multi-step routines are inserted, we need to make sure its node's locations and nextLocations are updated.
        // For good measure, should do the same with directories that might contain multi-step routines.
        // Can accomplish this either by creating a new function which updates the base location recursively, 
        // or better would be getting the steps beforehand using something like stepFromLocation and creating the steps with the correct 
        // location from the start
        const mockRootStep: MultiRoutineStep = {
            __type: RunStepType.MultiRoutine,
            description: "root",
            name: "root",
            location: [1],
            nodes: [{
                __type: RunStepType.Start,
                description: "",
                location: [1, 1],
                name: "",
                nextLocation: [1, 2],
                nodeId: "node123",
            }, {
                __type: RunStepType.RoutineList,
                description: "boop",
                location: [1, 2],
                name: "beep",
                nextLocation: [1, 3],
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
                nextLocation: null,
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
        const mockStepData: MultiRoutineStep = {
            __type: RunStepType.MultiRoutine,
            description: "newBoi",
            location: [999],
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
        const mockStepData: DirectoryStep = {
            __type: RunStepType.Directory,
            description: "new data",
            directoryId: null,
            hasBeenQueried: false,
            isOrdered: false,
            isRoot: true,
            location: [],
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
        const mockStepData: MultiRoutineStep = {
            __type: RunStepType.MultiRoutine,
            description: "deep routine",
            location: [3, 3, 3, 3, 3],
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
                nextLocation: [8, 33, 3],
                nodeId: "startNode",
            }, {
                __type: RunStepType.RoutineList,
                description: "routine list level 1",
                location: [1, 2],
                name: "routine list level 1",
                nextLocation: [],
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
                        nextLocation: [],
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
                nextLocation: null,
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
        const mockStepData: MultiRoutineStep = {
            __type: RunStepType.MultiRoutine,
            description: "non-matching data",
            location: [4, 3, 1, 3],
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
                nextLocation: [1, 2],
                nodeId: "startNode",
            }, {
                __type: RunStepType.RoutineList,
                description: "routine list",
                location: [1, 2],
                name: "routine list",
                nextLocation: [],
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
                    nextLocation: [],
                    nodeId: "startNode",
                },
                {
                    __type: RunStepType.RoutineList,
                    description: "routine list",
                    location: [1, 2],
                    name: "routine list",
                    nextLocation: [9, 1, 1],
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
                    nextLocation: [],
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
                                    nextLocation: [1],
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
                    nextLocation: null,
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

describe("siblingsAtLocaiton", () => {
    // Mock console.error to prevent actual console output during tests
    const originalConsoleError = console.error;
    beforeEach(() => {
        console.error = jest.fn();
    });
    afterEach(() => {
        console.error = originalConsoleError;
    });

    it("should return 0 for an empty location array", () => {
        const rootStep: DirectoryStep = {
            __type: RunStepType.Directory,
            description: "a",
            directoryId: null,
            hasBeenQueried: true,
            isOrdered: false,
            isRoot: true,
            location: [1, 2, 3],
            name: "a",
            projectVersionId: "projectVersion123",
            steps: [],
        };
        expect(siblingsAtLocaiton([], rootStep)).toBe(0);
    });

    it("should return 1 for a location array with only one element", () => {
        const rootStep: DirectoryStep = {
            __type: RunStepType.Directory,
            description: "a",
            directoryId: null,
            hasBeenQueried: true,
            isOrdered: false,
            isRoot: true,
            location: [1, 2, 3],
            name: "a",
            projectVersionId: "projectVersion123",
            steps: [],
        };
        expect(siblingsAtLocaiton([1], rootStep)).toBe(1);
    });

    it("should return the correct number of siblings for a DirectoryStep", () => {
        const rootStep: DirectoryStep = {
            __type: RunStepType.Directory,
            steps: [
                { __type: RunStepType.SingleRoutine } as SingleRoutineStep,
                { __type: RunStepType.SingleRoutine } as SingleRoutineStep,
                { __type: RunStepType.SingleRoutine } as SingleRoutineStep,
            ],
        } as DirectoryStep;

        expect(siblingsAtLocaiton([1, 2], rootStep)).toBe(3);
    });

    it("should return the correct number of siblings for a MultiRoutineStep", () => {
        const rootStep: MultiRoutineStep = {
            __type: RunStepType.MultiRoutine,
            nodes: [
                { __type: RunStepType.Start },
                { __type: RunStepType.RoutineList },
                { __type: RunStepType.End },
            ],
        } as MultiRoutineStep;

        expect(siblingsAtLocaiton([1, 2], rootStep)).toBe(3);
    });

    it("should return the correct number of siblings for a RoutineListStep", () => {
        const rootStep: RoutineListStep = {
            __type: RunStepType.RoutineList,
            steps: [
                { __type: RunStepType.SingleRoutine } as SingleRoutineStep,
                { __type: RunStepType.SingleRoutine } as SingleRoutineStep,
            ],
        } as RoutineListStep;

        expect(siblingsAtLocaiton([1, 2], rootStep)).toBe(2);
    });

    it("should return 1 for an unknown step type", () => {
        const rootStep: RootStep = {
            __type: "UnknownType" as RunStepType,
        } as unknown as RootStep;

        expect(siblingsAtLocaiton([1, 2], rootStep)).toBe(1);
    });

    it("should handle deeply nested locations", () => {
        const rootStep: DirectoryStep = {
            __type: RunStepType.Directory,
            location: [1],
            steps: [
                {
                    __type: RunStepType.MultiRoutine,
                    location: [1, 1],
                    nodes: [
                        {
                            __type: RunStepType.Start,
                            location: [1, 1, 1],
                        },
                        {
                            __type: RunStepType.RoutineList,
                            location: [1, 1, 2],
                            steps: [
                                { __type: RunStepType.SingleRoutine } as SingleRoutineStep,
                                { __type: RunStepType.SingleRoutine } as SingleRoutineStep,
                                { __type: RunStepType.SingleRoutine } as SingleRoutineStep,
                                { __type: RunStepType.SingleRoutine } as SingleRoutineStep,
                            ],
                        },
                        {
                            __type: RunStepType.End,
                            location: [1, 1, 3],
                        },
                    ],
                } as MultiRoutineStep,
            ],
        } as DirectoryStep;

        expect(siblingsAtLocaiton([1, 1, 1], rootStep)).toBe(3);
        expect(siblingsAtLocaiton([1, 1, 2], rootStep)).toBe(3);
        expect(siblingsAtLocaiton([1, 1, 3], rootStep)).toBe(3);
        expect(siblingsAtLocaiton([1, 1, 2, 1], rootStep)).toBe(4);
        expect(siblingsAtLocaiton([1, 1, 2, 2], rootStep)).toBe(4);
        expect(siblingsAtLocaiton([1, 1, 2, 3], rootStep)).toBe(4);
        expect(siblingsAtLocaiton([1, 1, 2, 4], rootStep)).toBe(4);
    });

    it("should return 0 and log an error when parent is not found", () => {
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

        const result = siblingsAtLocaiton([1, 2, 3], rootStep);
        expect(result).toBe(0);
        expect(console.error).toHaveBeenCalled();
    });
});

describe("getNextLocation and getPreviousLocation", () => {
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
        expect(getPreviousLocation([1, 1], rootStep)).toEqual([1]);

        expect(getNextLocation([1, 1], rootStep)).toEqual([1, 2]);
        expect(getPreviousLocation([1, 2], rootStep)).toEqual([1, 1]);

        expect(getNextLocation([1, 2], rootStep)).toBeNull();

        expect(getNextLocation([2], rootStep)).toBeNull();

        expect(getNextLocation([], rootStep)).toEqual([1]);
        expect(getPreviousLocation([1], rootStep)).toBeNull();
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
                    nextLocation: [1, 4],
                    nodeId: "startNode",
                },
                {
                    __type: RunStepType.RoutineList,
                    description: "routine list",
                    location: [1, 2],
                    name: "routine list",
                    nextLocation: [1, 5],
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
                    __type: RunStepType.End,
                    description: "end",
                    location: [1, 4],
                    name: "end",
                    nextLocation: null,
                    nodeId: "endNode",
                },
                {
                    __type: RunStepType.RoutineList,
                    description: "routine list",
                    location: [1, 3],
                    name: "routine list",
                    nextLocation: [1, 5],
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
                                    nextLocation: null,
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
                    __type: RunStepType.RoutineList,
                    description: "routine list",
                    location: [1, 5],
                    name: "routine list",
                    nextLocation: [1, 2],
                    nodeId: "routineListNode1",
                    steps: [],
                },
            ],
            nodeLinks: [],
            routineVersionId: "rootRoutine",
        };
        // Root step to first node
        expect(getNextLocation([1], rootStep)).toEqual([1, 1]);
        expect(getPreviousLocation([1, 1], rootStep)).toEqual([1]);

        // First node to its `nextLocation` field
        expect(getPreviousLocation([1, 4], rootStep)).toEqual([1, 1]);

        // Second node prefers its child over `nextLocation`
        expect(getNextLocation([1, 2], rootStep)).toEqual([1, 2, 1]);
        expect(getPreviousLocation([1, 2, 1], rootStep)).toEqual([1, 2]);

        // Second node's child has no more siblings, to goes to parent's `nextLocation`
        expect(getNextLocation([1, 2, 1], rootStep)).toEqual([1, 5]);
        expect(getPreviousLocation([1, 5], rootStep)).toEqual([1, 2]); // Doesn't go down children

        // Go to first child
        expect(getNextLocation([1, 3], rootStep)).toEqual([1, 3, 1]);
        expect(getPreviousLocation([1, 3, 1], rootStep)).toEqual([1, 3]);

        // Go to second child
        expect(getNextLocation([1, 3, 1], rootStep)).toEqual([1, 3, 2]);
        expect(getPreviousLocation([1, 3, 2], rootStep)).toEqual([1, 3, 1]);

        // Go to first child
        expect(getNextLocation([1, 3, 2], rootStep)).toEqual([1, 3, 2, 1]);
        expect(getPreviousLocation([1, 3, 2, 1], rootStep)).toEqual([1, 3, 2]);

        // Go to first child
        expect(getNextLocation([1, 3, 2, 1], rootStep)).toEqual([1, 3, 2, 1, 1]);
        expect(getPreviousLocation([1, 3, 2, 1, 1], rootStep)).toEqual([1, 3, 2, 1]);

        // Backtrack multiple times to [1, 3]'s `nextLocation`
        expect(getNextLocation([1, 3, 2, 1, 1], rootStep)).toEqual([1, 5]);
        expect(getNextLocation([1, 1], rootStep)).toEqual([1, 4]);
        // There are actually 2 locations that can lead to [1, 5], so either is valid
        expect([[1, 2], [1, 3]]).toContainEqual(getPreviousLocation([1, 5], rootStep));
    });

    it("another multi-step routine test", () => {
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
                    nextLocation: [1, 2],
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
                    nextLocation: [1, 4],
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
                                    nextLocation: null,
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
                    nextLocation: null,
                    nodeId: "endNode",
                },
            ],
            nodeLinks: [],
            routineVersionId: "rootRoutine",
        };
        // To `nextLocation`
        expect(getNextLocation([1, 1], rootStep)).toEqual([1, 2]);
        expect(getPreviousLocation([1, 2], rootStep)).toEqual([1, 1]);

        // Decisions should return null
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
            nextLocation: null,
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
            nextLocation: [2, 2, 2],
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
        const endStep: RunStep = {
            __type: RunStepType.End,
            description: "",
            location: [1, 2, 3],
            name: "",
            nextLocation: null,
            nodeId: "1",
        };
        expect(getStepComplexity(endStep)).toBe(0);
    });

    it("should return 0 for Start step", () => {
        const startStep: RunStep = {
            __type: RunStepType.Start,
            description: "",
            name: "",
            nextLocation: [1, 2, 3, 4, 5],
            nodeId: "1",
            location: [1],
        };
        expect(getStepComplexity(startStep)).toBe(0);
    });

    it("should return 1 for Decision step", () => {
        const decisionStep: RunStep = {
            __type: RunStepType.Decision,
            description: "",
            name: "",
            nodeId: "1",
            location: [1],
            options: [],
        };
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
                {
                    __type: RunStepType.Start,
                    description: "",
                    name: "",
                    nextLocation: [],
                    nodeId: "1",
                    location: [1, 1],
                },
                {
                    __type: RunStepType.Decision,
                    description: "",
                    name: "",
                    nodeId: "2",
                    location: [1, 2],
                    options: [],
                },
                {
                    __type: RunStepType.RoutineList,
                    description: "",
                    name: "",
                    nextLocation: [2, 2, 2],
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
                {
                    __type: RunStepType.End,
                    description: "",
                    name: "",
                    nextLocation: null,
                    nodeId: "3",
                    location: [1, 3],
                },
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
            nextLocation: null,
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
                            nextLocation: [1, 2],
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
                            nextLocation: [1, 4],
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
                                            nextLocation: null,
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
                            nextLocation: null,
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
 * Helper function to validate that routine steps are in a logical order
 */
function expectValidStepSequence(
    steps: MultiRoutineStep["nodes"],
    nodeLinks: NodeLink[],
) {
    if (steps.length === 0) return;
    // Length of location array
    let locationLength = 0;
    const usedLocations = new Set<string>();

    // For each step
    for (let i = 0; i < steps.length; i++) {
        const step = steps[i];

        const locationStr = step.location.join(",");
        // Location should be unique
        expect(usedLocations.has(locationStr)).toBe(
            false,
            // @ts-ignore: expect-message
            `Expected location ${locationStr} to be unique`,
        );
        usedLocations.add(locationStr);

        // Should only be one StartStep, and should be first
        if (i === 0) {
            expect(step.__type).toBe(
                RunStepType.Start,
                // @ts-ignore: expect-message
                `Expected StartStep to be first in the sequence. Got ${step.__type}`,
            );
            locationLength = step.location.length;
            // Last number in location should be 1
            expect(step.location[locationLength - 1]).toBe(
                1,
                // @ts-ignore: expect-message
                `Expected StartStep to have a location ending in 1. Got ${step.location[locationLength - 1]}`,
            );
        } else {
            expect(step.__type).not.toBe(
                RunStepType.Start,
                // @ts-ignore: expect-message
                `Expected StartStep in a different position than index 0. Got ${step.__type} at index ${i}`,
            );
            // Last number in location should not be 1
            expect(step.location[locationLength - 1]).not.toBe(
                1,
                // @ts-ignore: expect-message
                `Expected StartStep to have a location ending in 1. Got ${step.location[locationLength - 1]}`,
            );
        }

        // If it's an EndStep or DecisionStep, `nextLocation` should be null or undefined
        if ([RunStepType.End, RunStepType.Decision].includes(step.__type)) {
            const nextLocation = (step as { nextLocation?: string | null }).nextLocation;
            expect(nextLocation === null || nextLocation === undefined).toBe(
                true,
                // @ts-ignore: expect-message
                `Expected EndStep or DecisionStep to have a null/undefined nextLocation. Got ${nextLocation}`,
            );
            // Rest of the checks don't apply to EndStep or DecisionStep
            continue;
        }
        // If it's a RoutineList or StartStep, `nextLocation` should be defined and valid
        else if ([RunStepType.RoutineList, RunStepType.Start].includes(step.__type)) {
            const currNodeId = (step as StartStep | RoutineListStep).nodeId;
            const currLocation = step.location;
            const nextLocation = (step as StartStep | RoutineListStep).nextLocation;
            const outgoingLinks = nodeLinks.filter(link => link.from?.id === currNodeId);
            const nextStep = steps.find(s => s.location.join(",") === nextLocation?.join(","));

            expect(Array.isArray(nextLocation)).toBe(
                true,
                // @ts-ignore: expect-message
                `Expected StartStep or RoutineListStep to have a defined nextLocation. Got ${typeof nextLocation} ${nextLocation}`,
            );
            expect(JSON.stringify(nextLocation)).not.toBe(
                JSON.stringify(currLocation),
                // @ts-ignore: expect-message
                `Expected nextLocation to not point to itself. Got ${nextLocation}`,
            );
            expect(Boolean(nextStep)).toBe(
                true,
                // @ts-ignore: expect-message
                `Expected nextLocation to point to a valid step. Got ${nextLocation}`,
            );
            expect(outgoingLinks.length).toBeGreaterThanOrEqual(
                1,
                // @ts-ignore: expect-message
                `Expected at least one outgoing link from step ${currNodeId}. Got ${outgoingLinks.length}`,
            );

            // If there are multiple paths to take, we should point to a valid DecisionStep
            if (outgoingLinks.length > 1) {
                expect(nextStep?.__type).toBe(
                    RunStepType.Decision,
                    // @ts-ignore: expect-message
                    `Expected nextLocation to point to a DecisionStep. Got ${nextStep?.__type}`,
                );
                outgoingLinks.forEach(link => {
                    expect((nextStep as DecisionStep).options.some(option => option.link.to?.id === link.to?.id)).toBe(
                        true,
                        // @ts-ignore: expect-message
                        `Expected outgoing link ${link.to?.id} to be an option in the DecisionStep`,
                    );
                });
                (nextStep as DecisionStep).options.forEach(option => {
                    expect(outgoingLinks.some(link => link.to?.id === option.link.to?.id)).toBe(
                        true,
                        // @ts-ignore: expect-message
                        `Expected DecisionStep option ${option.link.to?.id} to have a corresponding outgoing link`,
                    );
                });
            }
            // If there is only one path to take, we should point to anythign but a DecisionStep
            else {
                expect(nextStep?.__type).not.toBe(
                    RunStepType.Decision,
                    // @ts-ignore: expect-message
                    `Expected nextLocation to not point to a DecisionStep. Got ${nextStep?.__type}`,
                );
                expect(outgoingLinks[0].to?.id).toBe(
                    (nextStep as EndStep | RoutineListStep | StartStep).nodeId,
                    // @ts-ignore: expect-message
                    `Expected outgoing link to match nextLocation. Got ${outgoingLinks[0].to?.id}, expected ${nextLocation}`,
                );
            }
        }
        // Otherwise, it's an invalid step type
        else {
            expect(false).toBe(
                true,
                // @ts-ignore: expect-message
                `Expected step to be a StartStep, RoutineListStep, DecisionStep, or EndStep. Got ${step.__type}`,
            );
        }
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

    it("should sort steps in correct order for a linear path", () => {
        const commonProps = { description: "", name: "", location: [69] };
        const steps = [
            {
                __type: RunStepType.Start,
                nextLocation: [],
                nodeId: "1",
                ...commonProps,
            } as StartStep,
            {
                __type: RunStepType.RoutineList,
                isOrdered: true,
                parentRoutineVersionId: "routineVersion123",
                nextLocation: [],
                nodeId: "2",
                steps: [],
                ...commonProps,
            } as RoutineListStep,
            {
                __type: RunStepType.End,
                nodeId: "3",
                ...commonProps,
            } as EndStep,
        ];
        const nodeLinks = [
            { from: { id: "1" }, to: { id: "2" } },
            { from: { id: "2" }, to: { id: "3" } },
        ] as NodeLink[];
        const result = sortStepsAndAddDecisions(steps, nodeLinks);
        expect(result.map(step => step.nodeId)).toEqual(["1", "2", "3"]);
    });

    it("should add a decision step when there are multiple outgoing links", () => {
        const commonProps = { description: "", name: "", location: [1, 3, 2, 1, 69] };
        const steps = [
            {
                __type: RunStepType.Start,
                nextLocation: [],
                nodeId: "1",
                ...commonProps,
            } as StartStep,
            {
                __type: RunStepType.RoutineList,
                isOrdered: true,
                parentRoutineVersionId: "routineVersion123",
                nextLocation: [],
                nodeId: "2",
                steps: [],
                ...commonProps,
            } as RoutineListStep,
            {
                __type: RunStepType.RoutineList,
                isOrdered: true,
                parentRoutineVersionId: "routineVersion123",
                nextLocation: [],
                nodeId: "3",
                steps: [],
                ...commonProps,
            } as RoutineListStep,
            {
                __type: RunStepType.End,
                nodeId: "4",
                ...commonProps,
            } as EndStep,
        ];
        const nodeLinks = [
            // Start to both RoutineLists
            { from: { id: "1" }, to: { id: "2" } },
            { from: { id: "1" }, to: { id: "3" } },
            // RoutineLists to End
            { from: { id: "2" }, to: { id: "4" } },
            { from: { id: "3" }, to: { id: "4" } },
        ] as NodeLink[];
        const result = sortStepsAndAddDecisions(steps, nodeLinks);
        expectValidStepSequence(result, nodeLinks);
        expect(result.length).toBe(steps.length + 1); // One decision step
    });

    it("should handle cycles in the graph", () => {
        const commonProps = { description: "", name: "", location: [1, 3, 2, 420] };
        const steps = [
            {
                __type: RunStepType.Start,
                nextLocation: [],
                nodeId: "1",
                ...commonProps,
            } as StartStep,
            {
                __type: RunStepType.RoutineList,
                isOrdered: true,
                parentRoutineVersionId: "routineVersion123",
                nextLocation: [],
                nodeId: "2",
                steps: [],
                ...commonProps,
            } as RoutineListStep,
            {
                __type: RunStepType.RoutineList,
                isOrdered: true,
                parentRoutineVersionId: "routineVersion123",
                nextLocation: [],
                nodeId: "3",
                steps: [],
                ...commonProps,
            } as RoutineListStep,
            {
                __type: RunStepType.RoutineList,
                isOrdered: true,
                parentRoutineVersionId: "routineVersion123",
                nextLocation: [],
                nodeId: "4",
                steps: [],
                ...commonProps,
            } as RoutineListStep,
            {
                __type: RunStepType.End,
                nodeId: "5",
                ...commonProps,
            } as EndStep,
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
        const result = sortStepsAndAddDecisions(steps, nodeLinks);
        expectValidStepSequence(result, nodeLinks);
        expect(result.length).toBe(steps.length + 1); // One decision step
    });

    it("should handle complex graphs with multiple decision points", () => {
        const commonProps = { description: "", name: "", location: [1, 3] };
        const steps = [
            {
                __type: RunStepType.Start,
                nextLocation: [],
                nodeId: "1",
                ...commonProps,
            } as StartStep,
            {
                __type: RunStepType.RoutineList,
                isOrdered: true,
                parentRoutineVersionId: "routineVersion123",
                nextLocation: [],
                nodeId: "2",
                steps: [],
                ...commonProps,
            } as RoutineListStep,
            {
                __type: RunStepType.RoutineList,
                isOrdered: true,
                parentRoutineVersionId: "routineVersion123",
                nextLocation: [],
                nodeId: "3",
                steps: [],
                ...commonProps,
            } as RoutineListStep,
            {
                __type: RunStepType.RoutineList,
                isOrdered: true,
                parentRoutineVersionId: "routineVersion123",
                nextLocation: [],
                nodeId: "4",
                steps: [],
                ...commonProps,
            } as RoutineListStep,
            {
                __type: RunStepType.End,
                nodeId: "5",
                ...commonProps,
            } as EndStep,
            {
                __type: RunStepType.RoutineList,
                isOrdered: true,
                parentRoutineVersionId: "routineVersion123",
                nextLocation: [],
                nodeId: "6",
                steps: [],
                ...commonProps,
            } as RoutineListStep,
            {
                __type: RunStepType.End,
                nodeId: "7",
                ...commonProps,
            } as EndStep,
        ];
        const nodeLinks = [
            // Start to first three RoutineLists
            { from: { id: "1" }, to: { id: "2" } },
            { from: { id: "1" }, to: { id: "3" } },
            { from: { id: "1" }, to: { id: "4" } },
            // First RoutineList points to second two
            { from: { id: "2" }, to: { id: "3" } },
            { from: { id: "2" }, to: { id: "4" } },
            // Second RoutineList points to first RoutineList and first End
            { from: { id: "3" }, to: { id: "2" } },
            { from: { id: "3" }, to: { id: "5" } },
            // Third RoutineList points to first and fourth RoutineLists
            { from: { id: "4" }, to: { id: "2" } },
            { from: { id: "4" }, to: { id: "6" } },
            // Fourth RoutineList points to second End
            { from: { id: "6" }, to: { id: "7" } },
        ] as NodeLink[];
        const result = sortStepsAndAddDecisions(steps, nodeLinks);
        expectValidStepSequence(result, nodeLinks);
        expect(result.length).toBe(steps.length + 4); // Four decision steps
    });

    it("should ignore unlinked steps", () => {
        const commonProps = { description: "", name: "", location: [1, 3] };
        const steps = [
            {
                __type: RunStepType.Start,
                nextLocation: [],
                nodeId: "1",
                ...commonProps,
            } as StartStep,
            {
                __type: RunStepType.RoutineList,
                isOrdered: true,
                parentRoutineVersionId: "routineVersion123",
                nextLocation: [],
                nodeId: "2",
                steps: [],
                ...commonProps,
            } as RoutineListStep,
            {
                __type: RunStepType.End,
                nodeId: "3",
                ...commonProps,
            } as EndStep,
            {
                __type: RunStepType.RoutineList,
                isOrdered: true,
                parentRoutineVersionId: "routineVersion123",
                nextLocation: [],
                nodeId: "4",
                steps: [],
                ...commonProps,
            } as RoutineListStep,
        ];
        const nodeLinks = [
            // Start to first RoutineList
            { from: { id: "1" }, to: { id: "2" } },
            // First RoutineList to End
            { from: { id: "2" }, to: { id: "3" } },
            // Second RoutineList has no links
        ] as NodeLink[];
        const result = sortStepsAndAddDecisions(steps, nodeLinks);
        expect(result.map(step => step.nodeId)).toEqual(["1", "2", "3"]);
    });
});
