// tutorialUtils.test.ts

import { LINKS } from "@local/shared";
import { TutorialSection, getCurrentElement, getCurrentStep, getNextPlace, getPrevPlace, getTutorialStepInfo } from "./TutorialDialog";

// Mock sections data for testing
const mockSections: TutorialSection[] = [
    {
        title: "Section 1",
        steps: [
            {
                content: {
                    text: "Step 1.1",
                },
                location: {
                    element: "element1",
                    page: LINKS.Home,
                },
            },
            {
                content: {
                    text: "Step 1.2",
                },
                location: {
                    page: LINKS.Home,
                },
            },
        ],
    },
    {
        title: "Section 2",
        steps: [
            {
                content: {
                    text: "Step 2.1",
                },
                location: {
                    element: "element2",
                    page: LINKS.Search,
                },
            },
            {
                content: {
                    text: "Step 2.2",
                },
                location: {
                    page: LINKS.Search,
                },
            },
            {
                content: {
                    text: "Step 2.3",
                },
                location: {
                    page: LINKS.Search,
                },
            },
        ],
    },
    {
        title: "Section 3",
        steps: [
            {
                content: {
                    text: "Step 3.1",
                },
                location: {
                    page: LINKS.Create,
                },
            },
        ],
    },
];

describe("Tutorial Utils", () => {
    describe("getTutorialStepInfo", () => {
        test("should return correct info for initial step", () => {
            const place = { section: 0, step: 0 };
            const result = getTutorialStepInfo(mockSections, place);
            expect(result.isFinalStep).toBe(false);
            expect(result.isFinalStepInSection).toBe(false);
            expect(result.nextStep).toEqual(mockSections[0].steps[1]);
        });

        test("should return correct info for final step in section", () => {
            const place = { section: 0, step: 1 };
            const result = getTutorialStepInfo(mockSections, place);
            expect(result.isFinalStep).toBe(false);
            expect(result.isFinalStepInSection).toBe(true);
            expect(result.nextStep).toEqual(mockSections[1].steps[0]);
        });

        test("should return correct info for final step overall", () => {
            const place = { section: 2, step: 0 };
            const result = getTutorialStepInfo(mockSections, place);
            expect(result.isFinalStep).toBe(true);
            expect(result.isFinalStepInSection).toBe(true);
            expect(result.nextStep).toBeNull();
        });

        test("should handle invalid section index", () => {
            const place = { section: 5, step: 0 };
            const result = getTutorialStepInfo(mockSections, place);
            expect(result.isFinalStep).toBe(false);
            expect(result.isFinalStepInSection).toBe(false);
            expect(result.nextStep).toBeNull();
        });

        test("should handle invalid step index", () => {
            const place = { section: 0, step: 10 };
            const result = getTutorialStepInfo(mockSections, place);
            expect(result.isFinalStep).toBe(false);
            expect(result.isFinalStepInSection).toBe(false);
            expect(result.nextStep).toBeNull();
        });
    });

    describe("getNextPlace", () => {
        test("should return next step in the same section", () => {
            const place = { section: 0, step: 0 };
            const result = getNextPlace(mockSections, place);
            expect(result).toEqual({ section: 0, step: 1 });
        });

        test("should return first step of next section when at end of current section", () => {
            const place = { section: 0, step: 1 }; // Last step of section 0
            const result = getNextPlace(mockSections, place);
            expect(result).toEqual({ section: 1, step: 0 });
        });

        test("should return null when at the final step", () => {
            const place = { section: 2, step: 0 }; // Last section, last step
            const result = getNextPlace(mockSections, place);
            expect(result).toBeNull();
        });

        test("should handle invalid current section", () => {
            const place = { section: 5, step: 0 };
            const result = getNextPlace(mockSections, place);
            expect(result).toBeNull();
        });

        test("should handle invalid step index", () => {
            const place = { section: 0, step: 10 };
            const result = getNextPlace(mockSections, place);
            expect(result).toEqual({ section: 1, step: 0 });
        });
    });

    describe("getPrevPlace", () => {
        test("should return previous step in the same section", () => {
            const place = { section: 0, step: 1 };
            const result = getPrevPlace(mockSections, place);
            expect(result).toEqual({ section: 0, step: 0 });
        });

        test("should return last step of previous section when at first step of current section", () => {
            const place = { section: 1, step: 0 };
            const result = getPrevPlace(mockSections, place);
            expect(result).toEqual({ section: 0, step: 1 });
        });

        test("should return null when at the very first step", () => {
            const place = { section: 0, step: 0 };
            const result = getPrevPlace(mockSections, place);
            expect(result).toBeNull();
        });

        test("should handle invalid current section", () => {
            const place = { section: -1, step: 0 };
            const result = getPrevPlace(mockSections, place);
            expect(result).toBeNull();
        });

        test("should handle invalid step index", () => {
            const place = { section: 1, step: -1 };
            const result = getPrevPlace(mockSections, place);
            expect(result).toEqual({ section: 0, step: 1 });
        });
    });

    describe("getCurrentElement", () => {
        beforeEach(() => {
            const mockElement = document.createElement("div");
            mockElement.id = "element1";
            document.getElementById = jest.fn().mockImplementation(() => mockElement);
        });

        afterEach(() => {
            jest.clearAllMocks();
        });

        test("should return element when element ID exists", () => {
            const place = { section: 0, step: 0 };
            const result = getCurrentElement(mockSections, place);
            expect(typeof result).toBe("object");
            expect(result?.id).toBe("element1");
        });

        test("should return null when element ID does not exist", () => {
            const place = { section: 0, step: 1 }; // Step without element
            const result = getCurrentElement(mockSections, place);
            expect(result).toBeNull();
        });

        test("should return null when step does not exist", () => {
            const place = { section: 0, step: 5 };
            const result = getCurrentElement(mockSections, place);
            expect(result).toBeNull();
        });

        test("should return null when section does not exist", () => {
            const place = { section: 5, step: 0 };
            const result = getCurrentElement(mockSections, place);
            expect(result).toBeNull();
        });
    });

    describe("getCurrentStep", () => {
        test("should return the current step when valid", () => {
            const place = { section: 1, step: 2 };
            const result = getCurrentStep(mockSections, place);
            expect(result).toEqual(mockSections[1].steps[2]);
        });

        test("should return null when step index is invalid", () => {
            const place = { section: 0, step: 5 };
            const result = getCurrentStep(mockSections, place);
            expect(result).toBeNull();
        });

        test("should return null when section index is invalid", () => {
            const place = { section: 5, step: 0 };
            const result = getCurrentStep(mockSections, place);
            expect(result).toBeNull();
        });
    });
});
