import { RoutineType, type RoutineVersion } from "@vrooli/shared";
import { generateInputAndOutputMessage, generateInputOnlyMessage, generateOutputOnlyMessage, generateTaskMessage } from "./process.js";

// Helper function to create a mock RoutineVersion
function createMockRoutineVersion(routineType: RoutineType): RoutineVersion {
    return {
        __typename: "RoutineVersion",
        id: "1",
        routineType,
        isComplete: true,
        isLatest: true,
        isPrivate: false,
        isDeleted: false,
        inputs: [],
        outputs: [],
        complexity: 1,
        timesCompleted: 0,
        timesStarted: 0,
        nodeLinks: [],
        nodes: [],
        translations: [{
            language: "en",
            description: "English description",
            instructions: "English instructions",
            name: "English name",
        }, {
            language: "es",
            description: "Spanish description",
            // Purposefully missing instructions
            name: "Spanish name",
        }],
    } as unknown as RoutineVersion;
}

describe("generateTaskMessage", () => {
    const mockUserLanguages = ["en"];

    it("should return undefined for non-Generate routine with no missing inputs", () => {
        const mockRoutineVersion = createMockRoutineVersion(RoutineType.Action);
        const result = generateTaskMessage(
            mockRoutineVersion,
            { input1: { value: "value1", isRequired: true } },
            {},
            mockUserLanguages,
        );
        expect(result).to.be.undefined;
    });

    it("should generate input-only message for non-Generate routine with missing inputs", () => {
        const mockRoutineVersion = createMockRoutineVersion(RoutineType.Api);
        const result = generateTaskMessage(
            mockRoutineVersion,
            { input1: { value: "", isRequired: true } },
            {},
            mockUserLanguages,
        );
        expect(result).to.include("Your goal is to fill in the missing values for the list of inputs.");
    });

    it("should return undefined for Generate routine with no missing inputs and no outputs", () => {
        const mockRoutineVersion = createMockRoutineVersion(RoutineType.Generate);
        const result = generateTaskMessage(
            mockRoutineVersion,
            { input1: { value: "value1", isRequired: true } },
            {},
            mockUserLanguages,
        );
        expect(result).to.be.undefined;
    });

    it("should generate input and output message for Generate routine with missing inputs and outputs", () => {
        const mockRoutineVersion = createMockRoutineVersion(RoutineType.Generate);
        const result = generateTaskMessage(
            mockRoutineVersion,
            { input1: { value: "", isRequired: true } },
            { output1: { value: undefined } },
            mockUserLanguages,
        );
        expect(result).to.include("Your goal is to fill in the missing values for the inputs and generate values for the outputs.");
    });

    it("should generate input-only message for Generate routine with missing inputs and no outputs", () => {
        const mockRoutineVersion = createMockRoutineVersion(RoutineType.Generate);
        const result = generateTaskMessage(
            mockRoutineVersion,
            { input1: { value: "", isRequired: true } },
            {},
            mockUserLanguages,
        );
        expect(result).to.include("Your goal is to fill in the missing values for the list of inputs.");
    });

    it("should generate output-only message for Generate routine with no missing inputs and outputs", () => {
        const mockRoutineVersion = createMockRoutineVersion(RoutineType.Generate);
        const result = generateTaskMessage(
            mockRoutineVersion,
            { input1: { value: "value1", isRequired: true } },
            { output1: { value: undefined } },
            mockUserLanguages,
        );
        expect(result).to.include("Your goal is to generate values for each output.");
    });

    it("should handle all RoutineTypes correctly", () => {
        const routineTypes = Object.values(RoutineType);
        routineTypes.forEach(type => {
            const mockRoutineVersion = createMockRoutineVersion(type);
            const result = generateTaskMessage(
                mockRoutineVersion,
                { input1: { value: "", isRequired: true } },
                {},
                mockUserLanguages,
            );
            if (type === RoutineType.Generate) {
                expect(result).to.include("Your goal is to fill in the missing values for the list of inputs.");
            } else {
                expect(result).to.include("Your goal is to fill in the missing values for the list of inputs.");
            }
        });
    });
});

describe("generateInputOnlyMessage", () => {
    const mockUserLanguages = ["en"];

    it("should generate a message for input-only tasks", () => {
        const mockRoutineVersion = createMockRoutineVersion(RoutineType.Action);
        const result = generateInputOnlyMessage(
            mockRoutineVersion,
            { input1: { value: "", isRequired: true }, input2: { value: "value2", isRequired: false } },
            mockUserLanguages,
        );
        expect(result).to.include("Your goal is to fill in the missing values for the list of inputs.");
        expect(result).to.include("Inputs:");
        expect(result).to.include("\"input1\": {");
        expect(result).to.include("\"input2\": {");
    });
});

describe("generateOutputOnlyMessage", () => {
    const mockUserLanguages = ["en"];

    it("should generate a message for output-only tasks", () => {
        const mockRoutineVersion = createMockRoutineVersion(RoutineType.Generate);
        const result = generateOutputOnlyMessage(
            mockRoutineVersion,
            { output1: { value: undefined }, output2: { value: undefined } },
            mockUserLanguages,
        );
        expect(result).to.include("Your goal is to generate values for each output.");
        expect(result).to.include("Outputs:");
        expect(result).to.include("\"output1\": {");
        expect(result).to.include("\"output2\": {");
    });
});

describe("generateInputAndOutputMessage", () => {
    const mockUserLanguages = ["en"];

    it("should generate a message for tasks requiring both input and output", () => {
        const mockRoutineVersion = createMockRoutineVersion(RoutineType.Generate);
        const result = generateInputAndOutputMessage(
            mockRoutineVersion,
            { input1: { value: "", isRequired: true }, input2: { value: "value2", isRequired: false } },
            { output1: { value: undefined }, output2: { value: undefined } },
            mockUserLanguages,
        );
        expect(result).to.include("Your goal is to fill in the missing values for the inputs and generate values for the outputs.");
        expect(result).to.include("Inputs:");
        expect(result).to.include("\"input1\": {");
        expect(result).to.include("\"input2\": {");
        expect(result).to.include("Outputs:");
        expect(result).to.include("\"output1\": {");
        expect(result).to.include("\"output2\": {");
    });
});
