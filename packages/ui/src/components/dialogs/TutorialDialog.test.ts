/* eslint-disable @typescript-eslint/ban-ts-comment */
import { FormStructureType, LINKS, nanoid } from "@local/shared";
import { expect } from "chai";
import { TutorialSection, getCurrentElement, getCurrentStep, getNextPlace, getPrevPlace, getTutorialStepInfo, isValidPlace } from "./TutorialDialog.js";

// Mock sections data for testing
const mockSections: TutorialSection[] = [
    {
        title: "Section 1",
        steps: [
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: nanoid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Step 1.1",
                        tag: "body1",
                    },
                ],
                location: {
                    element: "element1",
                    page: LINKS.Home,
                },
            },
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: nanoid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Step 1.2",
                        tag: "body1",
                    },
                ],
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
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: nanoid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Step 2.1",
                        tag: "body1",
                    },
                ],
                location: {
                    element: "element2",
                    page: LINKS.Search,
                },
            },
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: nanoid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Step 2.2",
                        tag: "body1",
                    },
                ],
                location: {
                    page: LINKS.Search,
                },
            },
            {
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: nanoid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Step 2.3",
                        tag: "body1",
                    },
                ],
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
                content: [
                    {
                        type: FormStructureType.Header,
                        color: "primary",
                        id: nanoid(),
                        isCollapsible: false,
                        isMarkdown: true,
                        label: "Step 3.1",
                        tag: "body1",
                    },
                ],
                location: {
                    page: LINKS.Create,
                },
            },
        ],
    },
];

describe("Tutorial Utils", () => {
    describe("getTutorialStepInfo", () => {
        it("should return correct info for initial step", () => {
            const place = { section: 0, step: 0 };
            const result = getTutorialStepInfo(mockSections, place);
            expect(result.isFinalStep).to.equal(false);
            expect(result.isFinalStepInSection).to.equal(false);
            expect(result.nextStep).toEqual(mockSections[0].steps[1]);
        });

        it("should return correct info for final step in section", () => {
            const place = { section: 0, step: 1 };
            const result = getTutorialStepInfo(mockSections, place);
            expect(result.isFinalStep).to.equal(false);
            expect(result.isFinalStepInSection).to.equal(true);
            expect(result.nextStep).toEqual(mockSections[1].steps[0]);
        });

        it("should return correct info for final step overall", () => {
            const place = { section: 2, step: 0 };
            const result = getTutorialStepInfo(mockSections, place);
            expect(result.isFinalStep).to.equal(true);
            expect(result.isFinalStepInSection).to.equal(true);
            expect(result.nextStep).to.be.null;
        });

        it("should handle invalid section index", () => {
            const place = { section: 5, step: 0 };
            const result = getTutorialStepInfo(mockSections, place);
            expect(result.isFinalStep).to.equal(false);
            expect(result.isFinalStepInSection).to.equal(false);
            expect(result.nextStep).to.be.null;
        });

        it("should handle invalid step index", () => {
            const place = { section: 0, step: 10 };
            const result = getTutorialStepInfo(mockSections, place);
            expect(result.isFinalStep).to.equal(false);
            expect(result.isFinalStepInSection).to.equal(false);
            expect(result.nextStep).to.be.null;
        });
    });

    describe("getNextPlace", () => {
        it("should return next step in the same section", () => {
            const place = { section: 0, step: 0 };
            const result = getNextPlace(mockSections, place);
            expect(result).toEqual({ section: 0, step: 1 });
        });

        it("should return first step of next section when at end of current section", () => {
            const place = { section: 0, step: 1 }; // Last step of section 0
            const result = getNextPlace(mockSections, place);
            expect(result).toEqual({ section: 1, step: 0 });
        });

        it("should return null when at the final step", () => {
            const place = { section: 2, step: 0 }; // Last section, last step
            const result = getNextPlace(mockSections, place);
            expect(result).to.be.null;
        });

        it("should handle invalid current section", () => {
            const place = { section: 5, step: 0 };
            const result = getNextPlace(mockSections, place);
            expect(result).to.be.null;
        });

        it("should handle invalid step index", () => {
            const place = { section: 0, step: 10 };
            const result = getNextPlace(mockSections, place);
            expect(result).toEqual({ section: 1, step: 0 });
        });
    });

    describe("getPrevPlace", () => {
        it("should return previous step in the same section", () => {
            const place = { section: 0, step: 1 };
            const result = getPrevPlace(mockSections, place);
            expect(result).toEqual({ section: 0, step: 0 });
        });

        it("should return last step of previous section when at first step of current section", () => {
            const place = { section: 1, step: 0 };
            const result = getPrevPlace(mockSections, place);
            expect(result).toEqual({ section: 0, step: 1 });
        });

        it("should return null when at the very first step", () => {
            const place = { section: 0, step: 0 };
            const result = getPrevPlace(mockSections, place);
            expect(result).to.be.null;
        });

        it("should handle invalid current section", () => {
            const place = { section: -1, step: 0 };
            const result = getPrevPlace(mockSections, place);
            expect(result).to.be.null;
        });

        it("should handle invalid step index", () => {
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

        it("should return element when element ID exists", () => {
            const place = { section: 0, step: 0 };
            const result = getCurrentElement(mockSections, place);
            expect(typeof result).to.equal("object");
            expect(result?.id).to.equal("element1");
        });

        it("should return null when element ID does not exist", () => {
            const place = { section: 0, step: 1 }; // Step without element
            const result = getCurrentElement(mockSections, place);
            expect(result).to.be.null;
        });

        it("should return null when step does not exist", () => {
            const place = { section: 0, step: 5 };
            const result = getCurrentElement(mockSections, place);
            expect(result).to.be.null;
        });

        it("should return null when section does not exist", () => {
            const place = { section: 5, step: 0 };
            const result = getCurrentElement(mockSections, place);
            expect(result).to.be.null;
        });
    });

    describe("getCurrentStep", () => {
        it("should return the current step when valid", () => {
            const place = { section: 1, step: 2 };
            const result = getCurrentStep(mockSections, place);
            expect(result).toEqual(mockSections[1].steps[2]);
        });

        it("should return null when step index is invalid", () => {
            const place = { section: 0, step: 5 };
            const result = getCurrentStep(mockSections, place);
            expect(result).to.be.null;
        });

        it("should return null when section index is invalid", () => {
            const place = { section: 5, step: 0 };
            const result = getCurrentStep(mockSections, place);
            expect(result).to.be.null;
        });
    });

    describe("isValidPlace", () => {
        it("should return true when place is valid", () => {
            const place = { section: 1, step: 2 };
            const result = isValidPlace(mockSections, place);
            expect(result).to.equal(true);
        });

        it("should return false when step index is invalid", () => {
            const place = { section: 0, step: 5 };
            const result = isValidPlace(mockSections, place);
            expect(result).to.equal(false);
        });

        it("should return false when section index is invalid", () => {
            const place = { section: 5, step: 0 };
            const result = isValidPlace(mockSections, place);
            expect(result).to.equal(false);
        });

        describe("invalid place", () => {
            it("should return false when section is undefined", () => {
                const place = { step: 0 };
                // @ts-ignore Testing runtime scenario
                const result = isValidPlace(mockSections, place);
                expect(result).to.equal(false);
            });

            it("should return false when step is undefined", () => {
                const place = { section: 0 };
                // @ts-ignore Testing runtime scenario
                const result = isValidPlace(mockSections, place);
                expect(result).to.equal(false);
            });

            it("should return false when place is null", () => {
                // @ts-ignore Testing runtime scenario
                const result = isValidPlace(mockSections, null);
                expect(result).to.equal(false);
            });

            describe("non-integers", () => {
                it("should return false when section is a string", () => {
                    const place = { section: "0", step: 0 };
                    // @ts-ignore Testing runtime scenario
                    const result = isValidPlace(mockSections, place);
                    expect(result).to.equal(false);
                });

                it("should return false when step is a string", () => {
                    const place = { section: 0, step: "0" };
                    // @ts-ignore Testing runtime scenario
                    const result = isValidPlace(mockSections, place);
                    expect(result).to.equal(false);
                });
            });
        });
    });
});
