import { DUMMY_ID, InputType } from "@local/shared";
import { FieldHelperProps } from "formik";
import { FormSchema } from "../../../../forms/types";
import { RoutineVersionInputShape } from "../../../../utils/shape/models/routineVersionInput";
import { RoutineVersionOutputShape } from "../../../../utils/shape/models/routineVersionOutput";
import { UpdateSchemaElementsProps, updateSchemaElements } from "./RoutineUpsert";

// Mock data
const mockInputs: RoutineVersionInputShape[] = [
    {
        __typename: "RoutineVersionInput",
        id: "field1", // Exists in schema
        name: "fieldName1",
        isRequired: true,
        routineVersion: {
            __typename: "RoutineVersion",
            id: "routine1",
        },
        translations: [
            {
                __typename: "RoutineVersionInputTranslation",
                id: "trans1",
                language: "en",
                helpText: "Help text 1",
                description: "Description 1",
            },
        ],
    },
];

const mockOutputs: RoutineVersionOutputShape[] = [
    {
        __typename: "RoutineVersionOutput",
        id: "field1", // Exists in schema
        name: "fieldName1",
        routineVersion: {
            __typename: "RoutineVersion",
            id: "routine1",
        },
        translations: [
            {
                __typename: "RoutineVersionOutputTranslation",
                id: "trans1",
                language: "en",
                helpText: "Help text 1",
                description: "Description 1",
            },
        ],
    },
];

/** 
 * First schema test. Updates an input and adds a new input. 
 * Also contains a non-input element that should be ignored.
 */
const mockSchema: FormSchema = {
    containers: [],
    elements: [
        {
            id: "header1",
            label: "",
            tag: "h1",
            type: "Header",
        },
        {
            id: "field1", // Exists in mocks, so will be updated
            fieldName: "fieldName1",
            label: "Input Label 1",
            isRequired: true,
            helpText: "Updated Help text 1",
            description: "Updated Description 1",
            type: InputType.Text,
        },
        {
            id: "field2", // Does not exist in mocks, so will be created
            fieldName: "fieldName2",
            label: "Input Label 2",
            isRequired: false,
            type: InputType.Selector,
        },
    ],
};

/**
 * Second schema test. Has no elements, so output should have no elements.
 */
const mockEmptySchema: FormSchema = {
    containers: [],
    elements: [],
};

/**
 * Third schema test. Has no translation data in any elements
 */
const schemaWithoutTranslations: FormSchema = {
    containers: [],
    elements: [
        {
            id: "field1",
            fieldName: "fieldName1",
            label: "Input Label 1",
            isRequired: true,
            type: InputType.Text,
        },
        {
            id: "field2",
            fieldName: "fieldName2",
            label: "Input Label 2",
            isRequired: false,
            type: InputType.Selector,
        },
    ],
};

const mockLanguage = "en";
const mockRoutineVersionId = "routine1";

const mockElementsHelpers: FieldHelperProps<RoutineVersionInputShape[]> = {
    setValue: jest.fn(),
    setTouched: jest.fn(),
    setError: jest.fn(),
};

describe("updateSchemaElements", () => {
    it("should update existing input elements with new translations", () => {
        const props: UpdateSchemaElementsProps = {
            currentElements: mockInputs,
            elementsHelpers: mockElementsHelpers,
            language: mockLanguage,
            routineVersionId: mockRoutineVersionId,
            schema: mockSchema,
            type: "inputs",
        };

        updateSchemaElements(props);

        expect(mockElementsHelpers.setValue).toHaveBeenCalledWith([
            {
                __typename: "RoutineVersionInput",
                id: "field1",
                name: "fieldName1",
                isRequired: true,
                routineVersion: {
                    __typename: "RoutineVersion",
                    id: "routine1",
                },
                translations: [
                    {
                        __typename: "RoutineVersionInputTranslation",
                        id: "trans1",
                        language: "en",
                        helpText: "Updated Help text 1",
                        description: "Updated Description 1",
                    },
                ],
            },
            {
                __typename: "RoutineVersionInput",
                id: DUMMY_ID,
                name: "fieldName2",
                isRequired: false,
                routineVersion: {
                    __typename: "RoutineVersion",
                    id: "routine1",
                },
                translations: [],
            },
        ]);
    });

    it("should create new input elements if they do not exist", () => {
        const props: UpdateSchemaElementsProps = {
            currentElements: [],
            elementsHelpers: mockElementsHelpers,
            language: mockLanguage,
            routineVersionId: mockRoutineVersionId,
            schema: mockSchema,
            type: "inputs",
        };

        updateSchemaElements(props);

        expect(mockElementsHelpers.setValue).toHaveBeenCalledWith([
            {
                __typename: "RoutineVersionInput",
                id: DUMMY_ID,
                name: "fieldName1",
                isRequired: true,
                routineVersion: {
                    __typename: "RoutineVersion",
                    id: "routine1",
                },
                translations: [
                    {
                        language: "en",
                        helpText: "Updated Help text 1",
                        description: "Updated Description 1",
                    },
                ],
            },
            {
                __typename: "RoutineVersionInput",
                id: DUMMY_ID,
                name: "fieldName2",
                isRequired: false,
                routineVersion: {
                    __typename: "RoutineVersion",
                    id: "routine1",
                },
                translations: [],
            },
        ]);
    });

    it("should update existing output elements with new translations", () => {
        const props: UpdateSchemaElementsProps = {
            currentElements: mockOutputs,
            elementsHelpers: mockElementsHelpers as unknown as FieldHelperProps<RoutineVersionOutputShape[]>,
            language: mockLanguage,
            routineVersionId: mockRoutineVersionId,
            schema: mockSchema,
            type: "outputs",
        };

        updateSchemaElements(props);

        expect(mockElementsHelpers.setValue).toHaveBeenCalledWith([
            {
                __typename: "RoutineVersionOutput",
                id: "field1",
                name: "fieldName1",
                routineVersion: {
                    __typename: "RoutineVersion",
                    id: "routine1",
                },
                translations: [
                    {
                        __typename: "RoutineVersionOutputTranslation",
                        id: "trans1",
                        language: "en",
                        helpText: "Updated Help text 1",
                        description: "Updated Description 1",
                    },
                ],
            },
            {
                __typename: "RoutineVersionOutput",
                id: DUMMY_ID,
                name: "fieldName2",
                routineVersion: {
                    __typename: "RoutineVersion",
                    id: "routine1",
                },
                translations: [],
            },
        ]);
    });

    it("should create new output elements if they do not exist", () => {
        const props: UpdateSchemaElementsProps = {
            currentElements: [],
            elementsHelpers: mockElementsHelpers as unknown as FieldHelperProps<RoutineVersionOutputShape[]>,
            language: mockLanguage,
            routineVersionId: mockRoutineVersionId,
            schema: mockSchema,
            type: "outputs",
        };

        updateSchemaElements(props);

        expect(mockElementsHelpers.setValue).toHaveBeenCalledWith([
            {
                __typename: "RoutineVersionOutput",
                id: DUMMY_ID,
                name: "fieldName1",
                routineVersion: {
                    __typename: "RoutineVersion",
                    id: "routine1",
                },
                translations: [
                    {
                        language: "en",
                        helpText: "Updated Help text 1",
                        description: "Updated Description 1",
                    },
                ],
            },
            {
                __typename: "RoutineVersionOutput",
                id: DUMMY_ID,
                name: "fieldName2",
                routineVersion: {
                    __typename: "RoutineVersion",
                    id: "routine1",
                },
                translations: [],
            },
        ]);
    });

    it("should remove elements that are not in the schema", () => {
        const props: UpdateSchemaElementsProps = {
            currentElements: mockInputs,
            elementsHelpers: mockElementsHelpers,
            language: mockLanguage,
            routineVersionId: mockRoutineVersionId,
            schema: mockEmptySchema,
            type: "inputs",
        };

        updateSchemaElements(props);

        expect(mockElementsHelpers.setValue).toHaveBeenCalledWith([]);
    });

    it("should handle elements with no translations", () => {
        const props: UpdateSchemaElementsProps = {
            currentElements: mockInputs,
            elementsHelpers: mockElementsHelpers,
            language: mockLanguage,
            routineVersionId: mockRoutineVersionId,
            schema: schemaWithoutTranslations,
            type: "inputs",
        };

        updateSchemaElements(props);

        expect(mockElementsHelpers.setValue).toHaveBeenCalledWith([
            {
                __typename: "RoutineVersionInput",
                id: "field1",
                name: "fieldName1",
                isRequired: true,
                routineVersion: {
                    __typename: "RoutineVersion",
                    id: "routine1",
                },
                translations: [],
            },
            {
                __typename: "RoutineVersionInput",
                id: DUMMY_ID,
                name: "fieldName2",
                isRequired: false,
                routineVersion: {
                    __typename: "RoutineVersion",
                    id: "routine1",
                },
                translations: [],
            },
        ]);
    });
});
