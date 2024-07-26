/* eslint-disable @typescript-eslint/ban-ts-comment */
/* eslint-disable @typescript-eslint/no-explicit-any */
import { Node, NodeLink, NodeType, Project, ProjectVersion, ProjectVersionDirectory, ProjectVersionYou, Routine, RoutineType, RoutineVersion, RoutineVersionYou, RunRoutineInput } from "../api/generated/graphqlTypes";
import { uuid, uuidValidate } from "../id/uuid";
import { DecisionStep, DirectoryStep, EndStep, MultiRoutineStep, RootStep, RoutineListStep, RunInputsUpdateParams, RunStep, RunStepType, RunnableProjectVersion, RunnableRoutineVersion, SingleRoutineStep, StartStep, addSubroutinesToStep, directoryToStep, findStep, getNextLocation, getPreviousLocation, getRunPercentComplete, getStepComplexity, insertStep, locationArraysMatch, multiRoutineToStep, parseChildOrder, parseRunInputs, projectToStep, routineVersionHasSubroutines, runInputsUpdate, runnableObjectToStep, siblingsAtLocation, singleRoutineToStep, sortStepsAndAddDecisions, stepFromLocation, stepNeedsQuerying } from "./runUtils";


describe("getRunPercentComplete", () => {
    it("should return 0 when completedComplexity is null", () => {
        expect(getRunPercentComplete(null, 100)).toBe(0);
    });

    it("should return 0 when completedComplexity is undefined", () => {
        expect(getRunPercentComplete(undefined, 100)).toBe(0);
    });

    it("should return 0 when totalComplexity is null", () => {
        expect(getRunPercentComplete(50, null)).toBe(0);
    });

    it("should return 0 when totalComplexity is undefined", () => {
        expect(getRunPercentComplete(50, undefined)).toBe(0);
    });

    it("should return 0 when totalComplexity is 0", () => {
        expect(getRunPercentComplete(50, 0)).toBe(0);
    });

    it("should return 50 when half of the complexity is completed", () => {
        expect(getRunPercentComplete(50, 100)).toBe(50);
    });

    it("should return 100 when all complexity is completed", () => {
        expect(getRunPercentComplete(100, 100)).toBe(100);
    });

    it("should return 100 when completed complexity exceeds total complexity", () => {
        expect(getRunPercentComplete(150, 100)).toBe(100);
    });

    it("should round down to the nearest integer", () => {
        expect(getRunPercentComplete(66, 100)).toBe(66);
    });

    it("should round up to the nearest integer", () => {
        expect(getRunPercentComplete(67, 100)).toBe(67);
    });

    it("should handle very small fractions", () => {
        expect(getRunPercentComplete(1, 1000)).toBe(0);
    });

    it("should handle very large numbers", () => {
        expect(getRunPercentComplete(1000000, 2000000)).toBe(50);
    });

    it("should return 100 for equal non-zero values", () => {
        expect(getRunPercentComplete(5, 5)).toBe(100);
    });

    it("should handle decimal inputs", () => {
        expect(getRunPercentComplete(2.5, 5)).toBe(50);
    });
});

describe("locationArraysMatch", () => {
    it("should return true for two empty arrays", () => {
        expect(locationArraysMatch([], [])).toBe(true);
    });

    it("should return true for identical single-element arrays", () => {
        expect(locationArraysMatch([1], [1])).toBe(true);
    });

    it("should return false for different single-element arrays", () => {
        expect(locationArraysMatch([1], [2])).toBe(false);
    });

    it("should return true for identical multi-element arrays", () => {
        expect(locationArraysMatch([1, 2, 3], [1, 2, 3])).toBe(true);
    });

    it("should return false for arrays with different lengths", () => {
        expect(locationArraysMatch([1, 2], [1, 2, 3])).toBe(false);
    });

    it("should return false for arrays with same length but different elements", () => {
        expect(locationArraysMatch([1, 2, 3], [1, 2, 4])).toBe(false);
    });

    it("should return false for arrays with elements in different order", () => {
        expect(locationArraysMatch([1, 2, 3], [3, 2, 1])).toBe(false);
    });

    it("should handle large arrays", () => {
        const largeArray = Array.from({ length: 1000 }, (_, i) => i);
        expect(locationArraysMatch(largeArray, largeArray)).toBe(true);
    });

    it("should return true for arrays with negative numbers", () => {
        expect(locationArraysMatch([-1, -2, -3], [-1, -2, -3])).toBe(true);
    });

    it("should return false for arrays with mixed positive and negative numbers", () => {
        expect(locationArraysMatch([1, -2, 3], [1, 2, 3])).toBe(false);
    });

    it("should handle arrays with repeated elements", () => {
        expect(locationArraysMatch([1, 1, 2, 2], [1, 1, 2, 2])).toBe(true);
    });

    it("should return false for arrays with different repeated elements", () => {
        expect(locationArraysMatch([1, 1, 2, 2], [1, 2, 2, 2])).toBe(false);
    });

    it("should handle arrays with zero", () => {
        expect(locationArraysMatch([0, 1, 2], [0, 1, 2])).toBe(true);
    });

    it("should return false when comparing with undefined", () => {
        expect(locationArraysMatch([1, 2, 3], undefined as any)).toBe(false);
    });

    it("should return false when comparing with null", () => {
        expect(locationArraysMatch([1, 2, 3], null as any)).toBe(false);
    });
});

describe("parseRunInputs", () => {
    it("should return an empty object for null input", () => {
        // @ts-ignore: Testing runtime scenario
        expect(parseRunInputs(null)).toEqual({});
    });

    it("should return an empty object for invalid input type", () => {
        const notInputs = { __typename: "SomethingElse" };
        // @ts-ignore: Testing runtime scenario
        expect(parseRunInputs(notInputs, console)).toEqual({});
    });

    it("should parse inputs correctly", () => {
        const inputs = [
            { input: { id: "1", name: "input1" }, data: "\"string value\"" },
            { input: { id: "2", name: "input2" }, data: "42" },
            { input: { id: "3", name: "input3" }, data: "true" },
            { input: { id: "4", name: "input4" }, data: "{\"key\": \"value\"}" },
            { input: { id: "5", name: "input5" }, data: "[1,2,3]" },
        ] as RunRoutineInput[];

        expect(parseRunInputs(inputs, console)).toEqual({
            input1: "string value",
            input2: 42,
            input3: true,
            input4: { key: "value" },
            input5: [1, 2, 3],
        });
    });

    it("should use id as key when name is not available", () => {
        const inputs = [
            { input: { id: "1" }, data: "\"value\"" },
        ] as RunRoutineInput[];

        expect(parseRunInputs(inputs, console)).toEqual({
            "1": "value",
        });
    });

    it("should handle parsing errors gracefully", () => {
        const inputs = [
            { input: { id: "1", name: "input1" }, data: "invalid json" },
        ] as RunRoutineInput[];

        expect(parseRunInputs(inputs, console)).toEqual({
            input1: "invalid json",
        });
    });
});

describe("runInputsUpdate", () => {
    it("should create new inputs when they do not exist", () => {
        const params: RunInputsUpdateParams = {
            existingInputs: [],
            formData: { routineInputName1: "value1", routineInputName2: 42 },
            logger: console,
            routineInputs: [{
                id: "routineInput1",
                name: "routineInputName1",
            }, {
                id: "routineInput2",
                name: "routineInputName2",
            }],
            runRoutineId: "run1",
        };

        const result = runInputsUpdate(params);

        expect(result.inputsCreate).toEqual([
            {
                id: expect.any(String),
                data: "\"value1\"",
                inputConnect: "routineInput1",
                runRoutineConnect: "run1",
            },
            {
                id: expect.any(String),
                data: "42",
                inputConnect: "routineInput2",
                runRoutineConnect: "run1",
            },
        ]);
        result.inputsCreate?.forEach(input => expect(uuidValidate(input.id)).toBe(true));
        expect(result.inputsUpdate).toBeUndefined();
        expect(result.inputsDelete).toBeUndefined();
    });

    it("should update existing inputs when data has changed", () => {
        const params: RunInputsUpdateParams = {
            existingInputs: [{
                id: "runInput1",
                data: "\"old-value1\"",
                input: { id: "routineInput1" },
            }, {
                id: "runInput2",
                data: "999",
                input: { id: "routineInput2" },
            }],
            formData: { routineInputName1: "new-value1", routineInputName2: 42 },
            logger: console,
            routineInputs: [{
                id: "routineInput1",
                name: "routineInputName1",
            }, {
                id: "routineInput2",
                name: "routineInputName2",
            }],
            runRoutineId: "run1",
        };

        const result = runInputsUpdate(params);

        expect(result.inputsUpdate).toEqual([
            {
                id: "runInput1",
                data: "\"new-value1\"",
            },
            {
                id: "runInput2",
                data: "42",
            },
        ]);
        expect(result.inputsCreate).toBeUndefined();
        expect(result.inputsDelete).toBeUndefined();
    });

    it("should handle creating and updating inputs simultaneously", () => {
        const params: RunInputsUpdateParams = {
            existingInputs: [{
                id: "runInput2",
                data: "999",
                input: { id: "routineInput2" },
            }, {
                id: "runInput6",
                data: "\"hello world\"",
                input: { id: "routineInput6" },
            }],
            formData: { routineInputName1: "value1", routineInputName2: 42 },
            logger: console,
            routineInputs: [{
                id: "routineInput1",
                name: "routineInputName1",
            }, {
                id: "routineInput2",
                name: "routineInputName2",
            }, {
                id: "routineInput3",
                name: "routineInputName3",
            }],
            runRoutineId: "run1",
        };

        const result = runInputsUpdate(params);

        expect(result.inputsCreate).toEqual([
            {
                id: expect.any(String),
                data: "\"value1\"",
                inputConnect: "routineInput1",
                runRoutineConnect: "run1",
            },
        ]);
        result.inputsCreate?.forEach(input => expect(uuidValidate(input.id)).toBe(true));
        expect(result.inputsUpdate).toEqual([
            {
                id: "runInput2",
                data: "42",
            },
        ]);
        expect(result.inputsDelete).toBeUndefined();
    });
});

describe("routineVersionHasSubroutines", () => {
    it("should return false for null input", () => {
        expect(routineVersionHasSubroutines(null as any)).toBe(false);
    });

    it("should return false for undefined input", () => {
        expect(routineVersionHasSubroutines(undefined as any)).toBe(false);
    });

    it("should return false for empty object", () => {
        expect(routineVersionHasSubroutines({})).toBe(false);
    });

    it("should return false for non-MultiStep routine type", () => {
        const routineVersion: Partial<RoutineVersion> = { routineType: RoutineType.Generate };
        expect(routineVersionHasSubroutines(routineVersion)).toBe(false);
    });

    it("should return true for MultiStep routine with nodes", () => {
        const routineVersion: Partial<RoutineVersion> = {
            routineType: RoutineType.MultiStep,
            nodes: [{ __typename: "Node" } as any],
        };
        expect(routineVersionHasSubroutines(routineVersion)).toBe(true);
    });

    it("should return true for MultiStep routine with nodeLinks", () => {
        const routineVersion: Partial<RoutineVersion> = {
            routineType: RoutineType.MultiStep,
            nodeLinks: [{ __typename: "NodeLink" } as any],
        };
        expect(routineVersionHasSubroutines(routineVersion)).toBe(true);
    });

    it("should return true for MultiStep routine with nodesCount > 0", () => {
        const routineVersion: Partial<RoutineVersion> = {
            routineType: RoutineType.MultiStep,
            nodesCount: 1,
        };
        expect(routineVersionHasSubroutines(routineVersion)).toBe(true);
    });

    it("should return false for MultiStep routine with empty nodes, nodeLinks, and nodesCount = 0", () => {
        const routineVersion: Partial<RoutineVersion> = {
            routineType: RoutineType.MultiStep,
            nodes: [],
            nodeLinks: [],
            nodesCount: 0,
        };
        expect(routineVersionHasSubroutines(routineVersion)).toBe(false);
    });

    it("should return false for MultiStep routine with undefined nodes, nodeLinks, and nodesCount", () => {
        const routineVersion: Partial<RoutineVersion> = {
            routineType: RoutineType.MultiStep,
        };
        expect(routineVersionHasSubroutines(routineVersion)).toBe(false);
    });

    it("should return true if any of nodes, nodeLinks, or nodesCount indicate subroutines", () => {
        const routineVersion: Partial<RoutineVersion> = {
            routineType: RoutineType.MultiStep,
            nodes: [{ __typename: "Node" } as any],
            nodeLinks: [],
            nodesCount: 1,
        };
        expect(routineVersionHasSubroutines(routineVersion)).toBe(true);
    });
});

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
        const result = insertStep(mockStepData, mockRootStep, console);
        expect((result as DirectoryStep).steps[0]).toEqual({ ...mockStepData, location: [1, 1] });
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
                isOrdered: false,
                location: [1, 2],
                name: "beep",
                nextLocation: [1, 3],
                nodeId: "node234",
                parentRoutineVersionId: "routine123",
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
                wasSuccessful: true,
            }],
            nodeLinks: [],
            routineVersionId: "routineVersion123",
        };
        const result = insertStep(mockStepData, mockRootStep, console);
        expect(((result as MultiRoutineStep).nodes[1] as RoutineListStep).steps[0]).toEqual({ ...mockStepData, location: [1, 2, 1] });
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
        const result = insertStep(mockStepData, mockRootStep, console);
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

        const result = insertStep(mockStepData, mockRootStep, console);

        // Traversing down to the deeply-nested step
        expect((((result as DirectoryStep).steps[0] as DirectoryStep).steps[0] as DirectoryStep).steps[0]).toEqual({ ...mockStepData, location: [1, 1, 1, 1] });
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
                isOrdered: false,
                location: [1, 2],
                name: "routine list level 1",
                nextLocation: [],
                nodeId: "routineListNode1",
                parentRoutineVersionId: "routine123",
                steps: [{
                    __type: RunStepType.MultiRoutine,
                    description: "multi routine level 2",
                    location: [1, 2, 1],
                    name: "multi routine level 2",
                    nodes: [{
                        __type: RunStepType.RoutineList,
                        description: "routine list level 3",
                        isOrdered: false,
                        name: "routine list level 3",
                        nextLocation: [],
                        location: [1, 2, 1, 1],
                        nodeId: "routineListNode3",
                        parentRoutineVersionId: "routine123",
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
                wasSuccessful: false,
            }],
            nodeLinks: [],
            routineVersionId: "rootRoutine",
        };

        const result = insertStep(mockStepData, mockRootStep, console);

        // Traversing down to the deeply-nested step
        expect(((((result as MultiRoutineStep).nodes[1] as RoutineListStep).steps[0] as MultiRoutineStep).nodes[0] as RoutineListStep).steps[0]).toEqual({ ...mockStepData, location: [1, 2, 1, 1, 1] });
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
                isOrdered: false,
                location: [1, 2],
                name: "routine list",
                nextLocation: [],
                nodeId: "routineListNode",
                parentRoutineVersionId: "routine123",
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

        const result = insertStep(mockStepData, mockRootStep, console);
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
                    isOrdered: false,
                    location: [1, 2],
                    name: "routine list",
                    nextLocation: [9, 1, 1],
                    nodeId: "routineListNode1",
                    parentRoutineVersionId: "routine123",
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
                    isOrdered: true,
                    location: [1, 3],
                    name: "routine list",
                    nextLocation: [],
                    nodeId: "routineListNode2",
                    parentRoutineVersionId: "routine123",
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
                                    isOrdered: true,
                                    location: [1, 3, 2, 1],
                                    name: "routine list",
                                    nextLocation: [1],
                                    nodeId: "routineListNode3",
                                    parentRoutineVersionId: "routine234",
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
                    wasSuccessful: true,
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

describe("findStep", () => {
    // Helper function to create a basic DirectoryStep
    function createDirectoryStep(name: string, location: number[]): DirectoryStep {
        return {
            __type: RunStepType.Directory,
            name,
            description: `Description of ${name}`,
            location,
            directoryId: null,
            hasBeenQueried: true,
            isOrdered: false,
            isRoot: location.length === 1,
            projectVersionId: "project123",
            steps: [],
        };
    }

    // Helper function to create a basic SingleRoutineStep
    function createSingleRoutineStep(name: string, location: number[]): SingleRoutineStep {
        return {
            __type: RunStepType.SingleRoutine,
            name,
            description: `Description of ${name}`,
            location,
            routineVersion: { id: "routine123" } as any,
        };
    }

    it("should find a step in a simple DirectoryStep structure", () => {
        const rootStep: DirectoryStep = {
            ...createDirectoryStep("Root", [1]),
            steps: [
                createSingleRoutineStep("Child1", [1, 1]),
                createSingleRoutineStep("Child2", [1, 2]),
            ],
        };

        const result = findStep(rootStep, step => step.name === "Child2");
        expect(result).toEqual(rootStep.steps[1]);
    });

    it("should find a step in a nested DirectoryStep structure", () => {
        const rootStep: DirectoryStep = {
            ...createDirectoryStep("Root", [1]),
            steps: [
                {
                    ...createDirectoryStep("Nested", [1, 1]),
                    steps: [
                        createSingleRoutineStep("Target", [1, 1, 1]),
                    ],
                },
                createSingleRoutineStep("Sibling", [1, 2]),
            ],
        };

        const result = findStep(rootStep, step => step.name === "Target");
        expect(result).toEqual((rootStep.steps[0] as DirectoryStep).steps[0]);
    });

    it("should find a step in a MultiRoutineStep structure", () => {
        const rootStep: MultiRoutineStep = {
            __type: RunStepType.MultiRoutine,
            name: "Root",
            description: "Root MultiRoutine",
            location: [1],
            routineVersionId: "multiRoutine123",
            nodes: [
                {
                    __type: RunStepType.Start,
                    name: "Start",
                    description: "Start node",
                    location: [1, 1],
                    nextLocation: [1, 2],
                    nodeId: "start123",
                },
                {
                    __type: RunStepType.RoutineList,
                    name: "RoutineList",
                    description: "Routine List",
                    isOrdered: false,
                    location: [1, 2],
                    nextLocation: [1, 3],
                    nodeId: "routineList123",
                    parentRoutineVersionId: "routineVersion123",
                    steps: [
                        createSingleRoutineStep("Target", [1, 2, 1]),
                    ],
                },
                {
                    __type: RunStepType.End,
                    name: "End",
                    description: "End node",
                    location: [1, 3],
                    nextLocation: null,
                    nodeId: "end123",
                    wasSuccessful: true,
                },
            ],
            nodeLinks: [],
        };

        const result = findStep(rootStep, step => step.name === "Target");
        expect(result).toEqual((rootStep.nodes[1] as RoutineListStep).steps[0]);
    });

    it("should find a step in a DecisionStep structure", () => {
        const rootStep: DecisionStep = {
            __type: RunStepType.Decision,
            name: "Decision",
            description: "Decision step",
            location: [1],
            options: [
                {
                    link: { id: "link1" } as any,
                    step: {
                        __type: RunStepType.RoutineList,
                        name: "RoutineList",
                        description: "Routine List",
                        isOrdered: false,
                        location: [1, 2],
                        nextLocation: [1, 3],
                        nodeId: "routineList123",
                        parentRoutineVersionId: "routineVersion123",
                        steps: [
                            createSingleRoutineStep("Option1", [1, 1]),
                        ],
                    },
                },
                {
                    link: { id: "link1" } as any,
                    step: {
                        __type: RunStepType.RoutineList,
                        name: "RoutineList",
                        description: "Routine List",
                        isOrdered: false,
                        location: [1, 2],
                        nextLocation: [1, 3],
                        nodeId: "routineList123",
                        parentRoutineVersionId: "routineVersion123",
                        steps: [
                            createSingleRoutineStep("Target", [1, 2]),
                        ],
                    },
                },
            ],
        };

        const result = findStep(rootStep, step => step.name === "Target");
        expect(result).toEqual((rootStep.options[1].step as RoutineListStep).steps[0]);
    });

    it("should return null if no step matches the predicate", () => {
        const rootStep: DirectoryStep = {
            ...createDirectoryStep("Root", [1]),
            steps: [
                createSingleRoutineStep("Child1", [1, 1]),
                createSingleRoutineStep("Child2", [1, 2]),
            ],
        };

        const result = findStep(rootStep, step => step.name === "NonExistent");
        expect(result).toBeNull();
    });

    it("should handle empty structures", () => {
        const emptyDirectory: DirectoryStep = createDirectoryStep("Empty", [1]);
        const emptyMultiRoutine: MultiRoutineStep = {
            __type: RunStepType.MultiRoutine,
            name: "Empty",
            description: "Empty MultiRoutine",
            location: [1],
            routineVersionId: "emptyMultiRoutine123",
            nodes: [],
            nodeLinks: [],
        };

        expect(findStep(emptyDirectory, () => true)).toEqual(emptyDirectory);
        expect(findStep(emptyMultiRoutine, () => true)).toEqual(emptyMultiRoutine);
    });

    it("should find a step based on a complex predicate", () => {
        const rootStep: DirectoryStep = {
            ...createDirectoryStep("Root", [1]),
            steps: [
                createSingleRoutineStep("Child1", [1, 1]),
                {
                    ...createDirectoryStep("Nested", [1, 2]),
                    steps: [
                        createSingleRoutineStep("Target", [1, 2, 1]),
                    ],
                },
            ],
        };

        const result = findStep(rootStep, step =>
            step.__type === RunStepType.SingleRoutine &&
            step.name === "Target" &&
            step.location.length === 3,
        );
        expect(result).toEqual((rootStep.steps[1] as DirectoryStep).steps[0]);
    });

    it("should return the first matching step when multiple steps match", () => {
        const rootStep: DirectoryStep = {
            ...createDirectoryStep("Root", [1]),
            steps: [
                createSingleRoutineStep("Target", [1, 1]),
                createSingleRoutineStep("Target", [1, 2]),
            ],
        };

        const result = findStep(rootStep, step => step.name === "Target");
        expect(result).toEqual(rootStep.steps[0]);
    });
});

describe("siblingsAtLocation", () => {
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
        expect(siblingsAtLocation([], rootStep, console)).toBe(0);
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
        expect(siblingsAtLocation([1], rootStep, console)).toBe(1);
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

        expect(siblingsAtLocation([1, 2], rootStep, console)).toBe(3);
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

        expect(siblingsAtLocation([1, 2], rootStep, console)).toBe(3);
    });

    it("should return the correct number of siblings for a RoutineListStep", () => {
        const rootStep: RoutineListStep = {
            __type: RunStepType.RoutineList,
            steps: [
                { __type: RunStepType.SingleRoutine } as SingleRoutineStep,
                { __type: RunStepType.SingleRoutine } as SingleRoutineStep,
            ],
        } as RoutineListStep;

        expect(siblingsAtLocation([1, 2], rootStep, console)).toBe(2);
    });

    it("should return 1 for an unknown step type", () => {
        const rootStep: RootStep = {
            __type: "UnknownType" as RunStepType,
        } as unknown as RootStep;

        expect(siblingsAtLocation([1, 2], rootStep, console)).toBe(1);
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

        expect(siblingsAtLocation([1, 1, 1], rootStep, console)).toBe(3);
        expect(siblingsAtLocation([1, 1, 2], rootStep, console)).toBe(3);
        expect(siblingsAtLocation([1, 1, 3], rootStep, console)).toBe(3);
        expect(siblingsAtLocation([1, 1, 2, 1], rootStep, console)).toBe(4);
        expect(siblingsAtLocation([1, 1, 2, 2], rootStep, console)).toBe(4);
        expect(siblingsAtLocation([1, 1, 2, 3], rootStep, console)).toBe(4);
        expect(siblingsAtLocation([1, 1, 2, 4], rootStep, console)).toBe(4);
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

        const result = siblingsAtLocation([1, 2, 3], rootStep, console);
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
                    isOrdered: false,
                    location: [1, 2],
                    name: "routine list",
                    nextLocation: [1, 5],
                    nodeId: "routineListNode1",
                    parentRoutineVersionId: "routine123",
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
                    wasSuccessful: false,
                },
                {
                    __type: RunStepType.RoutineList,
                    description: "routine list",
                    isOrdered: false,
                    location: [1, 3],
                    name: "routine list",
                    nextLocation: [1, 5],
                    nodeId: "routineListNode2",
                    parentRoutineVersionId: "routine123",
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
                                    isOrdered: true,
                                    location: [1, 3, 2, 1],
                                    name: "routine list",
                                    nextLocation: null,
                                    nodeId: "routineListNode3",
                                    parentRoutineVersionId: "routine234",
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
                    isOrdered: false,
                    location: [1, 5],
                    name: "routine list",
                    nextLocation: [1, 2],
                    nodeId: "routineListNode1",
                    steps: [],
                    parentRoutineVersionId: "routine123",
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
                                nextLocation: [1, 4],
                                parentRoutineVersionId: "routine123",
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
                                nextLocation: [1, 4, 2, 1, 2, 2, 3],
                                nodeId: "routineListNode3",
                                parentRoutineVersionId: "routine123",
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
                    isOrdered: false,
                    location: [1, 3],
                    name: "routine list",
                    nextLocation: [1, 4],
                    nodeId: "routineListNode2",
                    parentRoutineVersionId: "routine123",
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
                                    isOrdered: true,
                                    location: [1, 3, 2, 1],
                                    name: "routine list",
                                    nextLocation: null as unknown as number[], // Done on purpose,
                                    nodeId: "routineListNode3",
                                    parentRoutineVersionId: "routine234",
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
                    wasSuccessful: true,
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
            wasSuccessful: true,
        };
        expect(getStepComplexity(endStep, console)).toBe(0);
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
        expect(getStepComplexity(startStep, console)).toBe(0);
    });

    it("should return 1 for Decision step", () => {
        const decisionStep: RunStep = {
            __type: RunStepType.Decision,
            description: "",
            name: "",
            location: [1],
            options: [],
        };
        expect(getStepComplexity(decisionStep, console)).toBe(1);
    });

    it("should return the complexity of SingleRoutine step", () => {
        const singleRoutineStep: SingleRoutineStep = {
            __type: RunStepType.SingleRoutine,
            description: "",
            name: "",
            location: [1],
            routineVersion: { complexity: 3, id: "1", routineType: RoutineType.MultiStep } as RoutineVersion,
        };
        expect(getStepComplexity(singleRoutineStep, console)).toBe(3); // Complexity of the routine version
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
                    location: [1, 2],
                    options: [],
                },
                {
                    __type: RunStepType.RoutineList,
                    description: "",
                    isOrdered: true,
                    location: [3, 3, 3, 3, 3],
                    name: "",
                    nextLocation: [2, 2, 2],
                    nodeId: "1",
                    parentRoutineVersionId: "133",
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
                    wasSuccessful: false,
                },
            ],
            nodeLinks: [],
            routineVersionId: "1",
        };
        expect(getStepComplexity(multiRoutineStep, console)).toBe(11);
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
                                        nextLocation: [],
                                        nodeId: "routineListNode3",
                                        parentRoutineVersionId: "23fdhsaf",
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
                                        nextLocation: [1, 4, 2, 1, 2, 2, 3],
                                        nodeId: "routineListNode3",
                                        parentRoutineVersionId: "23fdhsaf",
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
                            isOrdered: true,
                            location: [1, 3],
                            name: "routine list",
                            nextLocation: [1, 4],
                            nodeId: "routineListNode2",
                            parentRoutineVersionId: "routine123",
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
                                            isOrdered: true,
                                            location: [1, 3, 2, 1],
                                            name: "routine list",
                                            nextLocation: [1, 1, 1, 1],
                                            nodeId: "routineListNode3",
                                            parentRoutineVersionId: "chicken",
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
                            wasSuccessful: true,
                        },
                    ],
                    nodeLinks: [],
                    routineVersionId: "rootRoutine",
                },
            ],
        };
        expect(getStepComplexity(routineListStep, console)).toBe(15);
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
        expect(getStepComplexity(directoryStep, console)).toBe(14);
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
            {
                __type: RunStepType.End,
                description: "",
                location: [1, 3],
                name: "",
                nextLocation: null,
                nodeId: "1",
                wasSuccessful: true,
            } as EndStep,
        ];
        const nodeLinks = [] as NodeLink[];
        const result = sortStepsAndAddDecisions(steps, nodeLinks, console);
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
        const result = sortStepsAndAddDecisions(steps, nodeLinks, console);
        expect(result.map(step => (step as { nodeId: string | null }).nodeId)).toEqual(["1", "2", "3"]);
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
        const result = sortStepsAndAddDecisions(steps, nodeLinks, console);
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
        const result = sortStepsAndAddDecisions(steps, nodeLinks, console);
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
        const result = sortStepsAndAddDecisions(steps, nodeLinks, console);
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
        const result = sortStepsAndAddDecisions(steps, nodeLinks, console);
        expect(result.map(step => (step as { nodeId: string | null }).nodeId)).toEqual(["1", "2", "3"]);
    });

    it("should update the location of steps within a RoutineListStep", () => {
        const baseLocation = [69, 420];
        const commonProps = { description: "", name: "" };
        const steps = [
            {
                __type: RunStepType.Start,
                location: [...baseLocation, 44444],
                nextLocation: [],
                nodeId: "1",
                ...commonProps,
            } as StartStep,
            {
                __type: RunStepType.RoutineList,
                isOrdered: true,
                location: [...baseLocation, 23],
                parentRoutineVersionId: "routineVersion123",
                nextLocation: [],
                nodeId: "2",
                steps: [{
                    __type: RunStepType.SingleRoutine,
                    description: "",
                    name: "",
                    location: [5, 3],
                    routineVersion: { complexity: 2, id: "1", routineType: RoutineType.MultiStep } as RoutineVersion,
                }, {
                    __type: RunStepType.SingleRoutine,
                    description: "",
                    name: "",
                    location: [5, 4],
                    routineVersion: { complexity: 3, id: "2", routineType: RoutineType.MultiStep } as RoutineVersion,
                }],
                ...commonProps,
            } as RoutineListStep,
            {
                __type: RunStepType.End,
                location: [...baseLocation, 999999],
                nodeId: "3",
                ...commonProps,
            } as EndStep,
        ];
        const nodeLinks = [
            { from: { id: "1" }, to: { id: "2" } },
            { from: { id: "2" }, to: { id: "3" } },
        ] as NodeLink[];
        const result = sortStepsAndAddDecisions(steps, nodeLinks, console);
        const routineListStep = result.find(step => step.__type === RunStepType.RoutineList) as RoutineListStep;
        expect(routineListStep.steps[0].location).toEqual([...routineListStep.location, 1]);
        expect(routineListStep.steps[1].location).toEqual([...routineListStep.location, 2]);
    });
});

describe("singleRoutineToStep", () => {
    it("should convert a RoutineVersion to a SingleRoutineStep", () => {
        const routineVersion: RunnableRoutineVersion = {
            __typename: "RoutineVersion",
            id: uuid(),
            created_at: new Date().toISOString(),
            complexity: 2,
            configCallData: "{}",
            configFormInput: "{}",
            configFormOutput: "{}",
            nodeLinks: [],
            nodes: [],
            root: {
                __typename: "Routine",
                id: uuid(),
            } as Routine,
            routineType: RoutineType.Informational,
            translations: [{
                __typename: "RoutineVersionTranslation",
                id: uuid(),
                language: "en",
                name: "english name",
                description: "english description",
            }, {
                __typename: "RoutineVersionTranslation",
                id: uuid(),
                language: "es",
                name: "spanish name",
                description: "spanish description",
            }],
            versionLabel: "1.0.0",
            you: {} as RoutineVersionYou,
        };
        const englishResult = singleRoutineToStep(routineVersion, [1], ["en"]);
        expect(englishResult).toEqual({
            __type: RunStepType.SingleRoutine,
            description: "english description",
            name: "english name",
            location: [1],
            routineVersion,
        });
        const spanishResult = singleRoutineToStep(routineVersion, [1], ["es"]);
        expect(spanishResult).toEqual({
            __type: RunStepType.SingleRoutine,
            description: "spanish description",
            name: "spanish name",
            location: [1],
            routineVersion,
        });
        const secondLanguageResult = singleRoutineToStep(routineVersion, [1], ["fr", "es"]);
        expect(secondLanguageResult).toEqual({
            __type: RunStepType.SingleRoutine,
            description: "spanish description",
            name: "spanish name",
            location: [1],
            routineVersion,
        });
    });
});

describe("multiRoutineToStep", () => {
    it("should convert a RoutineVersion to a MultiRoutineStep", () => {
        const startId = uuid();
        const routineListId = uuid();
        const endId = uuid();

        const routineVersion: RunnableRoutineVersion = {
            __typename: "RoutineVersion",
            id: uuid(),
            created_at: new Date().toISOString(),
            complexity: 2,
            configCallData: "{}",
            configFormInput: "{}",
            configFormOutput: "{}",
            nodeLinks: [{
                __typename: "NodeLink",
                id: uuid(),
                from: { id: startId },
                to: { id: routineListId },
            } as NodeLink, {
                __typename: "NodeLink",
                id: uuid(),
                from: { id: routineListId },
                to: { id: endId },
            } as NodeLink],
            nodes: [{
                __typename: "Node",
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                id: startId,
                columnIndex: 0,
                rowIndex: 0,
                nodeType: NodeType.Start,
            } as Node, {
                __typename: "Node",
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                id: routineListId,
                columnIndex: 0,
                rowIndex: 0,
                nodeType: NodeType.RoutineList,
                routineList: {
                    __typename: "NodeRoutineList",
                    id: uuid(),
                    isOptional: true,
                    isOrdered: false,
                    items: [{
                        __typename: "NodeRoutineListItem",
                        id: uuid(),
                        index: 0,
                        routineVersion: {
                            __typename: "RoutineVersion",
                            id: uuid(),
                            complexity: 3,
                        },
                    }, {
                        __typename: "NodeRoutineListItem",
                        id: uuid(),
                        index: 1,
                        routineVersion: {
                            __typename: "RoutineVersion",
                            id: uuid(),
                            complexity: 4,
                            translations: [{
                                __typename: "RoutineVersionTranslation",
                                id: uuid(),
                                language: "en",
                                description: "thee description",
                                name: "thee name",
                            }],
                        },
                    }],
                },
            } as Node, {
                __typename: "Node",
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                id: endId,
                columnIndex: 0,
                rowIndex: 0,
                nodeType: NodeType.End,
                end: {
                    __typename: "NodeEnd",
                    id: uuid(),
                    wasSuccessful: true,
                },
            } as Node],
            root: {
                __typename: "Routine",
                id: uuid(),
            } as Routine,
            routineType: RoutineType.Informational,
            translations: [{
                __typename: "RoutineVersionTranslation",
                id: uuid(),
                language: "en",
                name: "english name",
                description: "english description",
            }, {
                __typename: "RoutineVersionTranslation",
                id: uuid(),
                language: "es",
                name: "spanish name",
                description: "spanish description",
            }],
            versionLabel: "1.0.0",
            you: {} as RoutineVersionYou,
        };
        const result = multiRoutineToStep(routineVersion, [1, 2], ["en"], console);
        expect(result).toEqual({
            __type: RunStepType.MultiRoutine,
            description: "english description",
            name: "english name",
            location: [1, 2],
            routineVersionId: routineVersion.id,
            nodeLinks: routineVersion.nodeLinks,
            nodes: [{
                __type: RunStepType.Start,
                description: null,
                location: [1, 2, 1],
                name: "Untitled",
                nextLocation: [1, 2, 2],
                nodeId: routineVersion.nodes[0].id,
            }, {
                __type: RunStepType.RoutineList,
                description: null,
                isOrdered: false,
                location: [1, 2, 2],
                name: "Untitled",
                nextLocation: [1, 2, 3],
                nodeId: routineVersion.nodes[1].id,
                parentRoutineVersionId: routineVersion.id,
                steps: [
                    {
                        __type: RunStepType.SingleRoutine,
                        description: null,
                        location: [1, 2, 2, 1],
                        name: "Untitled",
                        routineVersion: (routineVersion as any).nodes[1].routineList.items[0].routineVersion,
                    },
                    {
                        __type: RunStepType.SingleRoutine,
                        description: "thee description",
                        location: [1, 2, 2, 2],
                        name: "thee name",
                        routineVersion: (routineVersion as any).nodes[1].routineList.items[1].routineVersion,
                    },
                ],
            }, {
                __type: RunStepType.End,
                description: null,
                location: [1, 2, 3],
                name: "Untitled",
                nextLocation: null,
                nodeId: routineVersion.nodes[2].id,
                wasSuccessful: true,
            }],
        });
    });
});

describe("projectToStep", () => {
    it("should convert a ProjectVersion to a DirectoryStep", () => {
        const projectVersion: RunnableProjectVersion = {
            __typename: "ProjectVersion",
            id: uuid(),
            created_at: new Date().toISOString(),
            directories: [{
                __typename: "ProjectVersionDirectory",
                id: uuid(),
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                isRoot: false,
                translations: [{
                    __typename: "ProjectVersionDirectoryTranslation",
                    id: uuid(),
                    language: "en",
                    description: "directory description",
                    name: "directory name",
                }],
            } as ProjectVersionDirectory],
            root: {
                __typename: "Project",
                id: uuid(),
            } as Project,
            translations: [{
                __typename: "ProjectVersionTranslation",
                id: uuid(),
                language: "en",
                description: "english description",
                name: "english name",
            }],
            versionLabel: "1.2.3",
            you: {} as ProjectVersionYou,
        };
        const result = projectToStep(projectVersion, [1, 2], ["en"]);
        expect(result).toEqual({
            __type: RunStepType.Directory,
            description: "english description",
            name: "english name",
            location: [1, 2],
            directoryId: null,
            hasBeenQueried: true,
            isOrdered: false,
            isRoot: true,
            projectVersionId: projectVersion.id,
            steps: [{
                __type: RunStepType.Directory,
                description: "directory description",
                location: [1, 2, 1],
                name: "directory name",
                directoryId: (projectVersion as any).directories[0].id,
                hasBeenQueried: true,
                isOrdered: false,
                isRoot: false,
                projectVersionId: projectVersion.id,
                steps: [],
            }],
        });
    });
});

describe("directoryToStep", () => {
    it("should convert a ProjectVersionDirectory to a DirectoryStep", () => {
        const projectVersionDirectory: ProjectVersionDirectory = {
            __typename: "ProjectVersionDirectory",
            id: uuid(),
            created_at: new Date().toISOString(),
            updated_at: new Date().toISOString(),
            isRoot: false,
            projectVersion: {
                __typename: "ProjectVersion",
                id: uuid(),
            } as ProjectVersion,
            translations: [{
                __typename: "ProjectVersionDirectoryTranslation",
                id: uuid(),
                language: "en",
                description: "directory description",
                name: "directory name",
            }],
        } as ProjectVersionDirectory;
        const result1 = directoryToStep(projectVersionDirectory, [9, 3], ["en"]);
        expect(result1).toEqual({
            __type: RunStepType.Directory,
            description: "directory description",
            location: [9, 3],
            name: "directory name",
            directoryId: projectVersionDirectory.id,
            hasBeenQueried: true,
            isOrdered: false,
            isRoot: false,
            projectVersionId: (projectVersionDirectory as any).projectVersion.id,
            steps: [],
        });
        delete projectVersionDirectory.projectVersion;
        const result2 = directoryToStep(projectVersionDirectory, [9, 3], ["en"], "anotherProjectVersionId");
        expect(result2).toEqual({
            __type: RunStepType.Directory,
            description: "directory description",
            location: [9, 3],
            name: "directory name",
            directoryId: projectVersionDirectory.id,
            hasBeenQueried: true,
            isOrdered: false,
            isRoot: false,
            projectVersionId: "anotherProjectVersionId",
            steps: [],
        });
    });
});

describe("runnableObjectToStep", () => {
    it("should convert a single-step routine", () => {
        const routineVersion: RunnableRoutineVersion = {
            __typename: "RoutineVersion",
            id: uuid(),
            created_at: new Date().toISOString(),
            complexity: 2,
            configCallData: "{}",
            configFormInput: "{}",
            configFormOutput: "{}",
            nodeLinks: [],
            nodes: [],
            root: {
                __typename: "Routine",
                id: uuid(),
            } as Routine,
            routineType: RoutineType.Informational,
            translations: [{
                __typename: "RoutineVersionTranslation",
                id: uuid(),
                language: "en",
                name: "english name",
                description: "english description",
            }, {
                __typename: "RoutineVersionTranslation",
                id: uuid(),
                language: "es",
                name: "spanish name",
                description: "spanish description",
            }],
            versionLabel: "1.0.0",
            you: {} as RoutineVersionYou,
        };
        const result = runnableObjectToStep(routineVersion, [1], ["en"], console);
        expect(result).toEqual({
            __type: RunStepType.SingleRoutine,
            description: "english description",
            name: "english name",
            location: [1],
            routineVersion,
        });
    });

    it("should convert a multi-step routine", () => {
        const startId = uuid();
        const routineListId = uuid();
        const endId = uuid();

        const routineVersion: RunnableRoutineVersion = {
            __typename: "RoutineVersion",
            id: uuid(),
            created_at: new Date().toISOString(),
            complexity: 2,
            configCallData: "{}",
            configFormInput: "{}",
            configFormOutput: "{}",
            nodeLinks: [{
                __typename: "NodeLink",
                id: uuid(),
                from: { id: startId },
                to: { id: routineListId },
            } as NodeLink, {
                __typename: "NodeLink",
                id: uuid(),
                from: { id: routineListId },
                to: { id: endId },
            } as NodeLink],
            nodes: [{
                __typename: "Node",
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                id: startId,
                columnIndex: 0,
                rowIndex: 0,
                nodeType: NodeType.Start,
            } as Node, {
                __typename: "Node",
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                id: routineListId,
                columnIndex: 0,
                rowIndex: 0,
                nodeType: NodeType.RoutineList,
                routineList: {
                    __typename: "NodeRoutineList",
                    id: uuid(),
                    isOptional: true,
                    isOrdered: false,
                    items: [{
                        __typename: "NodeRoutineListItem",
                        id: uuid(),
                        index: 0,
                        routineVersion: {
                            __typename: "RoutineVersion",
                            id: uuid(),
                            complexity: 3,
                        },
                    }, {
                        __typename: "NodeRoutineListItem",
                        id: uuid(),
                        index: 1,
                        routineVersion: {
                            __typename: "RoutineVersion",
                            id: uuid(),
                            complexity: 4,
                            translations: [{
                                __typename: "RoutineVersionTranslation",
                                id: uuid(),
                                language: "en",
                                description: "thee description",
                                name: "thee name",
                            }],
                        },
                    }],
                },
            } as Node, {
                __typename: "Node",
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                id: endId,
                columnIndex: 0,
                rowIndex: 0,
                nodeType: NodeType.End,
                end: {
                    __typename: "NodeEnd",
                    id: uuid(),
                    wasSuccessful: true,
                },
            } as Node],
            root: {
                __typename: "Routine",
                id: uuid(),
            } as Routine,
            routineType: RoutineType.MultiStep,
            translations: [{
                __typename: "RoutineVersionTranslation",
                id: uuid(),
                language: "en",
                name: "english name",
                description: "english description",
            }, {
                __typename: "RoutineVersionTranslation",
                id: uuid(),
                language: "es",
                name: "spanish name",
                description: "spanish description",
            }],
            versionLabel: "1.0.0",
            you: {} as RoutineVersionYou,
        };
        const result = runnableObjectToStep(routineVersion, [1, 2], ["en"], console);
        expect(result).toEqual({
            __type: RunStepType.MultiRoutine,
            description: "english description",
            name: "english name",
            location: [1, 2],
            routineVersionId: routineVersion.id,
            nodeLinks: routineVersion.nodeLinks,
            nodes: [{
                __type: RunStepType.Start,
                description: null,
                location: [1, 2, 1],
                name: "Untitled",
                nextLocation: [1, 2, 2],
                nodeId: routineVersion.nodes[0].id,
            }, {
                __type: RunStepType.RoutineList,
                description: null,
                isOrdered: false,
                location: [1, 2, 2],
                name: "Untitled",
                nextLocation: [1, 2, 3],
                nodeId: routineVersion.nodes[1].id,
                parentRoutineVersionId: routineVersion.id,
                steps: [
                    {
                        __type: RunStepType.SingleRoutine,
                        description: null,
                        location: [1, 2, 2, 1],
                        name: "Untitled",
                        routineVersion: (routineVersion as any).nodes[1].routineList.items[0].routineVersion,
                    },
                    {
                        __type: RunStepType.SingleRoutine,
                        description: "thee description",
                        location: [1, 2, 2, 2],
                        name: "thee name",
                        routineVersion: (routineVersion as any).nodes[1].routineList.items[1].routineVersion,
                    },
                ],
            }, {
                __type: RunStepType.End,
                description: null,
                location: [1, 2, 3],
                name: "Untitled",
                nextLocation: null,
                nodeId: routineVersion.nodes[2].id,
                wasSuccessful: true,
            }],
        });
    });

    it("should convert a project", () => {
        const projectVersion: RunnableProjectVersion = {
            __typename: "ProjectVersion",
            id: uuid(),
            created_at: new Date().toISOString(),
            directories: [{
                __typename: "ProjectVersionDirectory",
                id: uuid(),
                created_at: new Date().toISOString(),
                updated_at: new Date().toISOString(),
                isRoot: false,
                translations: [{
                    __typename: "ProjectVersionDirectoryTranslation",
                    id: uuid(),
                    language: "en",
                    description: "directory description",
                    name: "directory name",
                }],
            } as ProjectVersionDirectory],
            root: {
                __typename: "Project",
                id: uuid(),
            } as Project,
            translations: [{
                __typename: "ProjectVersionTranslation",
                id: uuid(),
                language: "en",
                description: "english description",
                name: "english name",
            }],
            versionLabel: "1.2.3",
            you: {} as ProjectVersionYou,
        };
        const result = runnableObjectToStep(projectVersion, [1, 2], ["en"], console);
        expect(result).toEqual({
            __type: RunStepType.Directory,
            description: "english description",
            name: "english name",
            location: [1, 2],
            directoryId: null,
            hasBeenQueried: true,
            isOrdered: false,
            isRoot: true,
            projectVersionId: projectVersion.id,
            steps: [{
                __type: RunStepType.Directory,
                description: "directory description",
                location: [1, 2, 1],
                name: "directory name",
                directoryId: (projectVersion as any).directories[0].id,
                hasBeenQueried: true,
                isOrdered: false,
                isRoot: false,
                projectVersionId: projectVersion.id,
                steps: [],
            }],
        });
    });
});

describe("addSubroutinesToStep", () => {
    it("should do nothing when no subroutines are provided", () => {
        const rootStep = {
            __type: RunStepType.MultiRoutine,
            description: "english description",
            name: "english name",
            location: [1, 2],
            routineVersionId: uuid(),
            nodeLinks: [],
            nodes: [],
        } as MultiRoutineStep;
        const result = addSubroutinesToStep([], rootStep, ["en"], console);
        expect(result).toEqual(rootStep);
    });

    it("should add subroutines where the routine ID matches a SingleRoutineStep's routine ID", () => {
        const rootStep: MultiRoutineStep = {
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
                isOrdered: false,
                location: [1, 2],
                name: "routine list level 1",
                nextLocation: [],
                nodeId: "routineListNode1",
                parentRoutineVersionId: "routine123",
                steps: [{
                    __type: RunStepType.MultiRoutine,
                    description: "multi routine level 2",
                    location: [1, 2, 1],
                    name: "multi routine level 2",
                    nodes: [{
                        __type: RunStepType.RoutineList,
                        description: "routine list level 3",
                        isOrdered: false,
                        name: "routine list level 3",
                        nextLocation: [],
                        location: [1, 2, 1, 1],
                        nodeId: "routineListNode3",
                        parentRoutineVersionId: "routine123",
                        steps: [{
                            __type: RunStepType.SingleRoutine,
                            description: "subroutine",
                            location: [1, 2, 1, 1, 1],
                            name: "subroutine",
                            routineVersion: { id: "deepRoutineVersion" } as RunnableRoutineVersion,
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
                wasSuccessful: false,
            }],
            nodeLinks: [],
            routineVersionId: "rootRoutine",
        };
        const subroutines: RunnableRoutineVersion[] = [{
            __typename: "RoutineVersion",
            id: "deepRoutineVersion",
            created_at: new Date().toISOString(),
            complexity: 2,
            configCallData: "{}",
            configFormInput: "{}",
            configFormOutput: "{}",
            nodeLinks: [],
            nodes: [],
            root: {
                __typename: "Routine",
                id: uuid(),
            } as Routine,
            routineType: RoutineType.MultiStep,
            translations: [{
                __typename: "RoutineVersionTranslation",
                id: uuid(),
                language: "en",
                name: "subroutine name",
                description: "subroutine description",
            }],
            versionLabel: "1.0.0",
            you: {} as RoutineVersionYou,
        }];
        const result = addSubroutinesToStep(subroutines, rootStep, ["en"], console);
        expect(result).toEqual({
            ...rootStep,
            nodes: [{
                ...rootStep.nodes[0],
            }, {
                ...rootStep.nodes[1],
                steps: [{
                    ...(rootStep.nodes[1] as RoutineListStep).steps[0],
                    nodes: [{
                        ...((rootStep.nodes[1] as RoutineListStep).steps[0] as MultiRoutineStep).nodes[0],
                        steps: [{
                            __type: RunStepType.MultiRoutine,
                            description: "subroutine description",
                            location: [1, 2, 1, 1, 1],
                            name: "subroutine name",
                            nodeLinks: [],
                            nodes: [],
                            routineVersionId: "deepRoutineVersion",
                        }],
                    }],
                }],
            }, {
                ...rootStep.nodes[2],
            }],
        });
    });
});
