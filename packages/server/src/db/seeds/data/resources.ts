import { CodeLanguage, CodeVersionConfig, DEFAULT_LANGUAGE, FormElement, FormStructureType, InputType, nanoid, ResourceSubType, ResourceType, RoutineVersionConfig, SEEDED_PUBLIC_IDS, SEEDED_TAGS, StandardVersionConfig } from "@local/shared";
import { ResourceImportData } from "../../../builders/importExport.js";

const VERSION = "1.0.0" as const;
const CODE_LANGUAGE = CodeLanguage.Javascript;

const baseRoot = {
    __version: VERSION,
    __typename: "Resource" as const,
};

const baseRootShape = {
    isPrivate: false,
    permissions: JSON.stringify({}),
    tags: [{ __typename: "Tag" as const, id: nanoid(), tag: SEEDED_TAGS.Vrooli }],
};

const baseVersion = {
    __version: VERSION,
    __typename: "ResourceVersion" as const,
};

const baseVersionShape = {
    codeLanguage: CODE_LANGUAGE,
    isComplete: true,
    isInternal: false,
    isPrivate: false,
    versionIndex: 0,
    versionLabel: "1.0.0",
};

const codes: ResourceImportData[] = [
    {
        ...baseRoot,
        shape: {
            ...baseRootShape,
            id: nanoid(),
            publicId: SEEDED_PUBLIC_IDS.ParseRunIOFromPlaintext,
            resourceType: ResourceType.Code,
            versions: [{
                ...baseVersion,
                shape: {
                    ...baseVersionShape,
                    id: nanoid(),
                    config: (new CodeVersionConfig({
                        config: {
                            __version: "1.0",
                            content: `/**
 * Converts plaintext to inputs and outputs. 
 * Useful to convert an LLM response into missing inputs and outputs for running a routine step.
 * 
 * @returns An object with inputs and output mappings.
 */
function parseRunIOFromPlaintext(
    { formData, text },
) {
    const inputs = {};
    const outputs = {};
    const lines = text.trim().split(String.fromCharCode(10));

    for (const line of lines) {
        const [key, ...valueParts] = line.split(":");
        if (key && valueParts.length > 0) {
            const trimmedKey = key.trim();
            const value = valueParts.join(":").trim();

            if (Object.prototype.hasOwnProperty.call(formData, \`input-\${trimmedKey}\`)) {
                inputs[trimmedKey] = value;
            } else if (Object.prototype.hasOwnProperty.call(formData, \`output-\${trimmedKey}\`)) {
                outputs[trimmedKey] = value;
            }
            // If the key doesn't match any input or output, it's ignored
        }
    }

    return { inputs, outputs };
}`,
                            inputConfig: {
                                inputSchema: {
                                    type: "object",
                                    properties: {
                                        formData: {
                                            type: "object",
                                        },
                                        text: {
                                            type: "string",
                                        },
                                    },
                                    required: ["formData", "text"],
                                },
                                shouldSpread: false,
                            },
                            outputConfig: [{
                                type: "object",
                                properties: {
                                    inputs: {
                                        type: "object",
                                    },
                                    outputs: {
                                        type: "object",
                                    },
                                },
                                required: ["inputs", "outputs"],
                            }],
                            testCases: [
                                {
                                    description: "should parse inputs and outputs correctly",
                                    input: {
                                        formData: {
                                            "input-name": true,
                                            "output-result": true,
                                        },
                                        text: "name: John Doe\nresult: Success",
                                    },
                                    expectedOutput: {
                                        inputs: {
                                            "name": "John Doe",
                                        },
                                        outputs: {
                                            "result": "Success",
                                        },
                                    },
                                },
                                {
                                    description: "should ignore keys that do not match any input or output",
                                    input: {
                                        formData: {
                                            "input-name": true,
                                            "output-result": true,
                                        },
                                        text: "name: John Doe\nage: 30\nresult: Success",
                                    },
                                    expectedOutput: {
                                        inputs: {
                                            "name": "John Doe",
                                        },
                                        outputs: {
                                            "result": "Success",
                                        },
                                    },
                                },
                                {
                                    description: "should handle multiple colons in the value correctly",
                                    input: {
                                        formData: {
                                            "input-time": true,
                                        },
                                        text: "time: 10:30:00",
                                    },
                                    expectedOutput: {
                                        inputs: {
                                            "time": "10:30:00",
                                        },
                                        outputs: {},
                                    },
                                },
                                {
                                    description: "should return empty objects when no inputs or outputs match",
                                    input: {
                                        formData: {
                                            "input-location": true,
                                        },
                                        text: "name: John Doe\nresult: Success",
                                    },
                                    expectedOutput: {
                                        inputs: {},
                                        outputs: {},
                                    },
                                },
                                {
                                    description: "should handle empty text correctly",
                                    input: {
                                        formData: {
                                            "input-name": true,
                                        },
                                        text: "",
                                    },
                                    expectedOutput: {
                                        inputs: {},
                                        outputs: {},
                                    },
                                },
                                {
                                    description: "should handle text with only whitespaces correctly",
                                    input: {
                                        formData: {
                                            "input-name": true,
                                        },
                                        text: "    ",
                                    },
                                    expectedOutput: {
                                        inputs: {},
                                        outputs: {},
                                    },
                                },
                                {
                                    description: "should correctly handle mixed case input and output keys",
                                    input: {
                                        formData: {
                                            "input-Name": true,
                                            "output-Result": true,
                                        },
                                        text: "Name: John Doe\nResult: Success",
                                    },
                                    expectedOutput: {
                                        inputs: {
                                            "Name": "John Doe",
                                        },
                                        outputs: {
                                            "Result": "Success",
                                        },
                                    },
                                },
                                {
                                    description: "should ignore random text that does not form key-value pairs",
                                    input: {
                                        formData: {
                                            "input-name": true,
                                            "output-result": true,
                                        },
                                        text: "Sure! Here is what you asked for:\nname: John Doe\nresult: Success",
                                    },
                                    expectedOutput: {
                                        inputs: {
                                            "name": "John Doe",
                                        },
                                        outputs: {
                                            "result": "Success",
                                        },
                                    },
                                },
                                {
                                    description: "should handle random text between key-value pairs",
                                    input: {
                                        formData: {
                                            "input-name": true,
                                            "output-result": true,
                                        },
                                        text: "name: John Doe\nPlease note the following details are important.\nresult: Success",
                                    },
                                    expectedOutput: {
                                        inputs: {
                                            "name": "John Doe",
                                        },
                                        outputs: {
                                            "result": "Success",
                                        },
                                    },
                                },
                                {
                                    description: "should properly ignore lines without a colon",
                                    input: {
                                        formData: {
                                            "input-name": true,
                                            "output-result": true,
                                        },
                                        text: "name: John Doe\nThis line has no colon\nresult: Success",
                                    },
                                    expectedOutput: {
                                        inputs: {
                                            "name": "John Doe",
                                        },
                                        outputs: {
                                            "result": "Success",
                                        },
                                    },
                                },
                                {
                                    description: "should handle text with mixed legitimate and illegitimate lines",
                                    input: {
                                        formData: {
                                            "input-name": true,
                                            "output-result": true,
                                        },
                                        text: "Here's the info you requested:\nname: John Doe\nrandom statement here\nresult: Success\nEnd of message.",
                                    },
                                    expectedOutput: {
                                        inputs: {
                                            "name": "John Doe",
                                        },
                                        outputs: {
                                            "result": "Success",
                                        },
                                    },
                                },
                            ],
                        },
                        codeLanguage: CODE_LANGUAGE,
                    })),
                    codeLanguage: CODE_LANGUAGE,
                    resourceSubType: ResourceSubType.CodeDataConverter,
                    translations: [{
                        language: "en",
                        id: nanoid(),
                        name: "Parse Run IO From Plaintext",
                        description: "When a routine is being run autonomously, we may need to generate inputs and/or outputs before we can perform the action associated with the routine. This function parses the expected output and returns an object with an inputs list and outputs list.",
                    }],
                },
            }],
        },
    },
    {
        ...baseRoot,
        shape: {
            ...baseRootShape,
            id: nanoid(),
            publicId: SEEDED_PUBLIC_IDS.ParseSearchTermsFromPlaintext,
            resourceType: ResourceType.Code,
            versions: [{
                ...baseVersion,
                shape: {
                    ...baseVersionShape,
                    id: nanoid(),
                    config: (new CodeVersionConfig({
                        config: {
                            __version: "1.0",
                            content: String.raw`/**
 * Transforms plaintext containing formatted or plain search terms into a cleaned list.
 * Search terms can be prefaced with numbering (e.g., "1. term") or listed plainly.
 * The function ignores blank or whitespace-only lines and trims whitespace from text lines.
 * 
 * @returns A cleaned array of search terms.
 */
function parseSearchTermsFromPlaintext(text) {
    const lines = text.split(String.fromCharCode(10));
    const cleanedTerms = [];
    let isNumberedList = false;

    // First, determine if the text contains a numbered list
    for (const line of lines) {
        if (line.trim().match(/^\d+\.\s+/)) {
            isNumberedList = true;
            break;
        }
    }

    for (let line of lines) {
        line = line.trim();
        if (line !== "") {
            if (isNumberedList) {
                // For numbered lists, only process lines matching the numbered list format
                const match = line.match(/^\d+\.\s*(.*)$/);
                if (match) {
                    cleanedTerms.push(match[1]);
                }
            } else {
                // If not a numbered list, process all non-empty lines
                cleanedTerms.push(line);
            }
        }
    }

    return cleanedTerms;
}`,
                            inputConfig: {
                                inputSchema: {
                                    type: "string",
                                },
                                shouldSpread: false,
                            },
                            outputConfig: [{
                                type: "array",
                                items: {
                                    type: "string",
                                },
                            }],
                            testCases: [
                                {
                                    description: "should correctly parse plain search terms without numbers",
                                    input: "how to cook a chicken\ncooking a chicken",
                                    expectedOutput: ["how to cook a chicken", "cooking a chicken"],
                                },
                                {
                                    description: "should parse numbered search terms and remove numbering",
                                    input: "1. how to cook a chicken\n2. cooking a chicken",
                                    expectedOutput: ["how to cook a chicken", "cooking a chicken"],
                                },
                                {
                                    description: "should ignore blank and whitespace-only lines",
                                    input: "1. how to cook a chicken\n   \n2. cooking a chicken\n",
                                    expectedOutput: ["how to cook a chicken", "cooking a chicken"],
                                },
                                {
                                    description: "should ignore random introductory or additional text",
                                    input: "Here are your search queries:\n1. how to cook a chicken\n2. cooking a chicken\nThat's all!",
                                    expectedOutput: ["how to cook a chicken", "cooking a chicken"],
                                },
                                {
                                    description: "should process terms with numbers not followed by a dot as regular terms",
                                    input: "1 how to cook a chicken\n2cooking a chicken",
                                    expectedOutput: ["1 how to cook a chicken", "2cooking a chicken"],
                                },
                                {
                                    description: "should handle empty strings gracefully",
                                    input: "",
                                    expectedOutput: [],
                                },
                                {
                                    description: "should trim and process terms with additional spaces before and after numbers",
                                    input: "   1.  how to cook a chicken   \n   2.   cooking a chicken   ",
                                    expectedOutput: ["how to cook a chicken", "cooking a chicken"],
                                },
                                {
                                    description: "should handle terms with multiple digits in numbering",
                                    input: "10. how to cook a chicken\n20. cooking a chicken",
                                    expectedOutput: ["how to cook a chicken", "cooking a chicken"],
                                },
                                {
                                    description: "should handle non-English characters and special characters in terms",
                                    input: "1. cómo cocinar un pollo\n2. cooking a chicken#",
                                    expectedOutput: ["cómo cocinar un pollo", "cooking a chicken#"],
                                },
                                {
                                    description: "should ignore lines not part of a structured numbered list",
                                    input: "Sure! Here are the search queries:\n1. how to cook a chicken\n2. cooking a chicken\nPlease follow up if you need more info.",
                                    expectedOutput: ["how to cook a chicken", "cooking a chicken"],
                                },
                                {
                                    description: "should only process lines in the structured list even with interruptions",
                                    input: "1. how to cook a chicken\nHere's something unrelated\n2. cooking a chicken\nEnd of list",
                                    expectedOutput: ["how to cook a chicken", "cooking a chicken"],
                                },
                                {
                                    description: "should ignore lines that contain numbers but are not properly formatted as list items",
                                    input: "1 how to cook a chicken\n2: Here is an improperly formatted line\n2. cooking a chicken",
                                    expectedOutput: ["cooking a chicken"],
                                },
                                {
                                    description: "should handle text where valid list items are scattered among random text",
                                    input: "Here are your instructions:\n1. how to cook a chicken\nSome random text.\n2. cooking a chicken\nMore random text here.",
                                    expectedOutput: ["how to cook a chicken", "cooking a chicken"],
                                },
                                {
                                    description: "should ignore non-list related text completely, even if it contains colons or numbers",
                                    input: "Introduction:\nWe have some points to consider:\n1. how to cook a chicken\n2. cooking a chicken\nConclusion: Thank you for your attention!",
                                    expectedOutput: ["how to cook a chicken", "cooking a chicken"],
                                },
                            ],
                        },
                        codeLanguage: CODE_LANGUAGE,
                    })),
                    codeLanguage: CODE_LANGUAGE,
                    resourceSubType: ResourceSubType.CodeDataConverter,
                    translations: [{
                        language: "en",
                        id: nanoid(),
                        name: "Parse Search Terms From Plaintext",
                        description: "Converts formatted or plain plaintext search terms into a cleaned array of terms.",
                    }],
                },
            }],
        },
    },
    {
        ...baseRoot,
        shape: {
            ...baseRootShape,
            id: nanoid(),
            publicId: SEEDED_PUBLIC_IDS.ListToPlaintext,
            resourceType: ResourceType.Code,
            versions: [{
                ...baseVersion,
                shape: {
                    ...baseVersionShape,
                    id: nanoid(),
                    config: (new CodeVersionConfig({
                        config: {
                            __version: "1.0",
                            content: `/**
 * Converts a list of strings into plaintext.
 * Useful for generating plaintext from an array of terms, formatted without numbers.
 * 
 * @param list The array of strings to convert.
 * @param delimiter The delimiter to use between items. Defaults to a newline.
 * @returns A plaintext string of the list items, separated by newlines.
 */
function listToPlaintext(list, delimiter = String.fromCharCode(10)) {
    if (!Array.isArray(list) || list.length === 0) {
        return "";
    }
    return list.join(delimiter);
}`,
                            inputConfig: {
                                inputSchema: {
                                    type: "array",
                                    items: [
                                        {
                                            type: "array",
                                            items: {
                                                type: "string",
                                            },
                                        },
                                        {
                                            type: "string",
                                        },
                                    ],
                                },
                                shouldSpread: true,
                            },
                            outputConfig: [{
                                type: "string",
                            }],
                            testCases: [
                                {
                                    description: "should return an empty string for an empty array",
                                    input: [[]],
                                    expectedOutput: "",
                                },
                                {
                                    description: "should return the single item as is",
                                    input: [["apple"]],
                                    expectedOutput: "apple",
                                },
                                {
                                    description: "should join multiple items with newlines",
                                    input: [["apple", "banana", "cherry"]],
                                    expectedOutput: "apple\nbanana\ncherry",
                                },
                                {
                                    description: "should preserve spaces in items",
                                    input: [["hello world", "foo bar"]],
                                    expectedOutput: "hello world\nfoo bar",
                                },
                                {
                                    description: "should support custom delimiters",
                                    input: [["apple", "banana", "cherry"], ", "],
                                    expectedOutput: "apple, banana, cherry",
                                },
                            ],
                        },
                        codeLanguage: CODE_LANGUAGE,
                    })),
                    codeLanguage: CODE_LANGUAGE,
                    resourceSubType: ResourceSubType.CodeDataConverter,
                    translations: [{
                        language: "en",
                        id: nanoid(),
                        name: "List To Plaintext",
                        description: "Converts an array of strings into a single string, each item separated by a newline.",
                    }],
                },
            }],
        },
    },
    {
        ...baseRoot,
        shape: {
            ...baseRootShape,
            id: nanoid(),
            publicId: SEEDED_PUBLIC_IDS.ListToNumberedPlaintext,
            resourceType: ResourceType.Code,
            versions: [{
                ...baseVersion,
                shape: {
                    ...baseVersionShape,
                    id: nanoid(),
                    config: (new CodeVersionConfig({
                        config: {
                            __version: "1.0",
                            content: `/**
 * Converts a list of strings into a numbered plaintext string.
 * Useful for generating a formatted list where each item is preceded by its index number followed by a dot.
 *
 * @param list The array of strings to convert.
 * @param delimiter The delimiter to use between items. Defaults to a newline.
 * @returns A numbered plaintext string of the list items, each prefixed with its number and a dot, separated by newlines.
 */
function listToNumberedPlaintext(list, delimiter = String.fromCharCode(10)) {
    if (!Array.isArray(list) || list.length === 0) {
        return "";
    }
    return list.map((item, index) => (index + 1) + ". " + item).join(delimiter);
}
`,
                            inputConfig: {
                                inputSchema: {
                                    type: "array",
                                    items: [
                                        {
                                            type: "array",
                                            items: {
                                                type: "string",
                                            },
                                        },
                                        {
                                            type: "string",
                                        },
                                    ],
                                },
                                shouldSpread: true,
                            },
                            outputConfig: [{
                                type: "string",
                            }],
                            testCases: [
                                {
                                    description: "should return an empty string for an empty array",
                                    input: [[]],
                                    expectedOutput: "",
                                },
                                {
                                    description: "should return the single item with a number and a dot",
                                    input: [["apple"]],
                                    expectedOutput: "1. apple",
                                },
                                {
                                    description: "should join multiple items with newlines and numbers",
                                    input: [["apple", "banana", "cherry"]],
                                    expectedOutput: "1. apple\n2. banana\n3. cherry",
                                },
                                {
                                    description: "should preserve spaces in items",
                                    input: [["hello world", "foo bar"]],
                                    expectedOutput: "1. hello world\n2. foo bar",
                                },
                                {
                                    description: "should support custom delimiters",
                                    input: [["apple", "banana", "cherry"], ", "],
                                    expectedOutput: "1. apple, 2. banana, 3. cherry",
                                },
                            ],
                        },
                        codeLanguage: CODE_LANGUAGE,
                    })),
                    codeLanguage: CODE_LANGUAGE,
                    resourceSubType: ResourceSubType.CodeDataConverter,
                    translations: [{
                        language: "en",
                        id: nanoid(),
                        name: "List To Numbered Plaintext",
                        description: "Converts an array of strings into a single string, each item prefixed with its number in the list and separated by newlines.",
                    }],
                },
            }],
        },
    },
];

const projectKickoffInputElements: readonly FormElement[] = [
    {
        type: FormStructureType.Header,
        color: "primary",
        id: "intro-header",
        label: "Overview",
        description: "Starting a new project can be both exciting and challenging. This guide will walk you through the essential steps to ensure a successful project kickoff.",
        tag: "h2",
    },
    {
        type: FormStructureType.Header,
        color: "secondary",
        id: "intro-header",
        label: "Starting a new project can be both exciting and challenging. This guide will walk you through the essential steps to ensure a successful project kickoff.",
        tag: "body1",
    },
    {
        type: FormStructureType.Tip,
        icon: "Info",
        id: "intro-tip",
        label: "Remember, thorough planning at the start can save a lot of time and resources down the line.",
    },
    {
        type: FormStructureType.Divider,
        id: "intro-divider",
        label: "",
    },
    // Step 1: Define Project Objectives and Scope
    {
        type: FormStructureType.Header,
        id: "step1-header",
        label: "Step 1: Define Project Objectives and Scope",
        description: `**Purpose**: Clearly articulating your project's objectives and scope is crucial for guiding the team and aligning stakeholder expectations.

**Description**: Outline the primary goals your project aims to achieve and the boundaries within which it will operate. This includes deliverables, timelines, and any constraints or limitations.

**Example**: If you're developing a mobile app, your objective might be "To create a user-friendly mobile application for online shopping that increases customer engagement by 20% within the first year."

**Tips**:
- Be specific and measurable.
- Align objectives with organizational goals.
- Consider both short-term and long-term impacts.`,
        tag: "h3",
    },
    {
        type: InputType.Text,
        id: "projectObjectives",
        fieldName: "projectObjectives",
        isRequired: true,
        label: "Project Objectives",
        props: {
            isMarkdown: true,
            placeholder: "Example: Increase customer engagement by 20% within the first year",
            minRows: 4,
            maxRows: 6,
        },
        yup: {
            required: true,
            checks: [],
        },
    },
    // Step 2: Identify Key Stakeholders
    {
        type: FormStructureType.Header,
        id: "step2-header",
        label: "Step 2: Identify Key Stakeholders",
        description: `**Purpose**: Identifying stakeholders ensures that all parties affected by the project are considered, which aids in gaining support and preventing obstacles.

**Description**: Select individuals, teams, or organizations with a vested interest in the project.

**Tips**:
- Consider stakeholders at all levels.
- Understand their expectations and how they define success.
- Plan how to communicate with each stakeholder group.`,
        tag: "h3",
    },
    {
        type: InputType.Checkbox,
        id: "keyStakeholders",
        fieldName: "keyStakeholders",
        isRequired: false,
        label: "Key Stakeholders",
        props: {
            options: [
                { label: "Project Team", value: "project_team" },
                { label: "Management", value: "management" },
                { label: "Clients", value: "clients" },
                { label: "Suppliers", value: "suppliers" },
                { label: "Regulatory Agencies", value: "regulatory_agencies" },
            ],
            row: false,
            defaultValue: [],
            allowCustomValues: true,
            maxCustomValues: 3, // Allows up to 3 custom entries
        },
        yup: {
            required: false,
            checks: [],
        },
    },
    // Step 3: Assemble Your Project Team
    {
        type: FormStructureType.Header,
        id: "step3-header",
        label: "Step 3: Assemble Your Project Team",
        description: `**Purpose**: Building the right team is essential for project success, ensuring that all necessary skills and expertise are represented.

**Description**: Determine the roles required for the project and assign team members accordingly. Consider technical skills, experience, and interpersonal dynamics.

**Tips**:
- Balance expertise and workload among team members.
- Clarify roles and responsibilities to avoid overlap.
- Encourage collaboration and open communication.`,
        tag: "h3",
    },
    {
        type: InputType.Checkbox,
        id: "teamRoles",
        fieldName: "teamRoles",
        isRequired: false,
        label: "Team Roles",
        props: {
            options: [
                { label: "Project Manager", value: "project_manager" },
                { label: "Lead Developer", value: "lead_developer" },
                { label: "Quality Assurance Specialist", value: "qa_specialist" },
                { label: "UI/UX Designer", value: "ui_ux_designer" },
                { label: "Business Analyst", value: "business_analyst" },
            ],
            row: false,
            defaultValue: [],
            allowCustomValues: true,
            maxCustomValues: 5, // Allows up to 5 custom roles
        },
        yup: {
            required: false,
            checks: [],
        },
    },
    // Step 4: Establish Communication Channels
    {
        type: FormStructureType.Header,
        id: "step4-header",
        label: "Step 4: Establish Communication Channels",
        description: `**Purpose**: Effective communication is vital for coordination and timely issue resolution throughout the project.

**Description**: Define how information will be shared within the team and with stakeholders. This includes meetings, reporting methods, and tools used for collaboration.

**Tips**:
- Choose communication methods suitable for your team size and structure.
- Set clear expectations for response times and availability.
- Utilize tools that integrate well with your workflows.`,
        tag: "h3",
    },
    {
        type: InputType.Checkbox,
        id: "communicationChannels",
        fieldName: "communicationChannels",
        isRequired: false,
        label: "Communication Channels",
        props: {
            options: [
                { label: "Email", value: "email" },
                { label: "Slack", value: "slack" },
                { label: "Microsoft Teams", value: "teams" },
                { label: "Zoom Meetings", value: "zoom" },
                { label: "Asana", value: "asana" },
            ],
            row: false,
            defaultValue: [],
            allowCustomValues: true,
            maxCustomValues: 3, // Allows up to 3 custom channels
        },
        yup: {
            required: false,
            checks: [],
        },
    },
    // Step 5: Establish Timelines and Deliverables
    {
        type: FormStructureType.Header,
        id: "step5-header",
        label: "Step 5: Establish Timelines",
        description: `**Purpose**: Setting a timeline with milestones helps track progress and keeps the project on schedule.

**Description**: Develop a project schedule outlining key deliverables and deadlines. Use project management tools to visualize timelines.

**Tips**:
- Be realistic with time estimates.
- Factor in buffer time for unexpected delays.
- Regularly review and adjust the schedule as needed.`,
        tag: "h3",
    },
    {
        type: InputType.Text,
        id: "projectTimeline",
        fieldName: "projectTimeline",
        isRequired: false,
        label: "Project Timeline",
        props: {
            isMarkdown: true,
            placeholder: "Example: Week 1 - Project kickoff meeting, Week 2 - UI/UX design mockups, Week 3 - Frontend development",
            minRows: 4,
            maxRows: 6,
        },
        yup: {
            required: false,
            checks: [],
        },
    },
    // Step 6: Define Deliverables
    {
        type: FormStructureType.Header,
        id: "step6-header",
        label: "Step 6: Define Deliverables",
        description: `**Purpose**: Clearly defining deliverables ensures that all team members understand what needs to be produced and can work towards common goals.

**Description**: List all tangible and intangible outputs the project is expected to produce, including specifications and acceptance criteria.

**Tips**:
- Make deliverables specific and measurable.
- Align deliverables with project objectives.
- Include quality criteria to define acceptable standards.`,
        tag: "h3",
    },
    {
        type: InputType.Text,
        id: "deliverables",
        fieldName: "deliverables",
        isRequired: true,
        label: "Deliverables",
        props: {
            isMarkdown: true,
            placeholder: "Example: Wireframes, User Stories, Code Repository",
            minRows: 4,
            maxRows: 6,
        },
        yup: {
            required: true,
            checks: [],
        },
    },
    // Step 7: Set Project Budget
    {
        type: FormStructureType.Header,
        id: "step7-header",
        label: "Step 7: Set Project Budget",
        description: `**Purpose**: Estimating the budget ensures that the project has sufficient resources.

**Description**: Define the total budget allocated for the project, including all expenses.

**Tips**:
- Consider all potential costs.
- Include a contingency fund for unexpected expenses.`,
        tag: "h3",
    },
    {
        type: InputType.IntegerInput,
        id: "projectBudget",
        fieldName: "projectBudget",
        isRequired: false,
        label: "Project Budget",
        props: {
            min: 0,
            step: 1000,
        },
        yup: {
            required: false,
            checks: [],
        },
    },
    // Step 8: Assess Project Risks
    {
        type: FormStructureType.Header,
        id: "step8-header",
        label: "Step 8: Assess Project Risks",
        description: `**Purpose**: Identifying potential risks allows you to plan mitigation strategies.

**Description**: Evaluate the potential risks in terms of likelihood and impact.

**Tips**:
- Consider technical, financial, and operational risks.
- Engage the team in brainstorming potential risks.`,
        tag: "h3",
    },
    {
        type: InputType.Slider,
        id: "riskLevel",
        fieldName: "riskLevel",
        isRequired: false,
        label: "Overall Risk Level",
        props: {
            min: 0,
            max: 10,
            step: 1,
            valueLabelDisplay: "on",
        },
        yup: {
            required: false,
            checks: [],
        },
    },
] as const;

const workoutPlanGeneratorInputElements: readonly FormElement[] = [
    {
        type: FormStructureType.Header,
        id: "intro-header",
        label: "Workout Plan Generator",
        description: "Provide some information to receive a personalized workout plan.",
        tag: "h2",
    },
    {
        type: FormStructureType.Header,
        id: "intro-body",
        label: "Please fill out the form below to help us create a workout plan tailored to your needs.",
        tag: "body1",
    },
    {
        type: FormStructureType.Divider,
        id: "intro-divider",
        label: "",
    },
    // Step 1: Choose Your Fitness Goal
    {
        type: FormStructureType.Header,
        id: "fitness-goal-header",
        label: "Step 1: Choose Your Fitness Goal",
        description: `**Purpose**: Selecting a primary fitness goal helps us tailor the workout plan to meet your objectives.

**Tips**:
- Choose the goal that best aligns with your current aspirations.
- You can focus on multiple goals, but selecting one primary goal helps with specificity.`,
        tag: "h3",
    },
    {
        type: InputType.Radio,
        id: "fitnessGoal",
        fieldName: "fitnessGoal",
        isRequired: true,
        label: "Fitness Goal",
        props: {
            options: [
                { label: "Lose weight", value: "lose_weight" },
                { label: "Build muscle", value: "build_muscle" },
                { label: "Improve endurance", value: "improve_endurance" },
                { label: "Increase flexibility", value: "increase_flexibility" },
                { label: "General fitness", value: "general_fitness" },
            ],
            row: false,
        },
        yup: {
            required: true,
            checks: [],
        },
    },
    // Step 2: Indicate Your Current Fitness Level
    {
        type: FormStructureType.Header,
        id: "fitness-level-header",
        label: "Step 2: Indicate Your Current Fitness Level",
        description: `**Purpose**: Understanding your fitness level ensures that the exercises are appropriate and safe for you.

**Tips**:
- Be honest about your current fitness level.
- If you're unsure, select the level that feels most accurate.`,
        tag: "h3",
    },
    {
        type: InputType.Radio,
        id: "fitnessLevel",
        fieldName: "fitnessLevel",
        isRequired: true,
        label: "Current Fitness Level",
        props: {
            options: [
                { label: "Beginner", value: "beginner" },
                { label: "Intermediate", value: "intermediate" },
                { label: "Advanced", value: "advanced" },
            ],
            row: false,
        },
        yup: {
            required: true,
            checks: [],
        },
    },
    // Step 3: Select Available Equipment
    {
        type: FormStructureType.Header,
        id: "equipment-header",
        label: "Step 3: Select Available Equipment",
        description: `**Purpose**: Knowing what equipment you have helps us design a plan that utilizes your resources.

**Tips**:
- Select all that apply.
- If you have other equipment, use the custom option.`,
        tag: "h3",
    },
    {
        type: InputType.Checkbox,
        id: "availableEquipment",
        fieldName: "availableEquipment",
        isRequired: false,
        label: "Available Equipment",
        props: {
            options: [
                { label: "None (bodyweight exercises)", value: "none" },
                { label: "Dumbbells", value: "dumbbells" },
                { label: "Resistance bands", value: "resistance_bands" },
                { label: "Barbell and plates", value: "barbell" },
                { label: "Cardio machines", value: "cardio_machines" },
                { label: "Yoga mat", value: "yoga_mat" },
            ],
            row: false,
            defaultValue: [],
            allowCustomValues: true,
            maxCustomValues: 3,
        },
        yup: {
            required: false,
            checks: [],
        },
    },
    // Step 4: Choose Your Preferred Workout Days
    {
        type: FormStructureType.Header,
        id: "workout-days-header",
        label: "Step 4: Choose Your Preferred Workout Days",
        description: `**Purpose**: Scheduling workouts on days that suit you increases consistency and adherence.

**Tips**:
- Select all days you're available to work out.
- Be realistic about your schedule.`,
        tag: "h3",
    },
    {
        type: InputType.Checkbox,
        id: "workoutDays",
        fieldName: "workoutDays",
        isRequired: true,
        label: "Preferred Workout Days",
        props: {
            options: [
                { label: "Monday", value: "monday" },
                { label: "Tuesday", value: "tuesday" },
                { label: "Wednesday", value: "wednesday" },
                { label: "Thursday", value: "thursday" },
                { label: "Friday", value: "friday" },
                { label: "Saturday", value: "saturday" },
                { label: "Sunday", value: "sunday" },
            ],
            row: true,
            defaultValue: [],
            allowCustomValues: false,
        },
        yup: {
            required: true,
            checks: [],
        },
    },
    // Step 5: List Any Injuries or Physical Limitations
    {
        type: FormStructureType.Header,
        id: "injuries-header",
        label: "Step 5: List Any Injuries or Physical Limitations",
        description: `**Purpose**: Identifying any physical limitations helps us avoid exercises that may cause harm.

**Tips**:
- Mention any chronic pain, past injuries, or medical conditions.
- If none, you can leave this blank.`,
        tag: "h3",
    },
    {
        type: InputType.Text,
        id: "injuries",
        fieldName: "injuries",
        isRequired: false,
        label: "Injuries or Physical Limitations",
        props: {
            isMarkdown: false,
            placeholder: "Example: Lower back pain, knee injury",
            minRows: 2,
            maxRows: 4,
        },
        yup: {
            required: false,
            checks: [],
        },
    },
    // Step 6: Specify Desired Workout Duration
    {
        type: FormStructureType.Header,
        id: "time-header",
        label: "Step 6: Specify Desired Workout Duration",
        description: `**Purpose**: Setting workout duration ensures the plan fits into your schedule.

**Tips**:
- Be realistic about the time you can commit.
- Consistency is more important than duration.`,
        tag: "h3",
    },
    {
        type: InputType.Slider,
        id: "workoutDuration",
        fieldName: "workoutDuration",
        isRequired: true,
        label: "Time per Workout (minutes)",
        props: {
            min: 15,
            max: 120,
            step: 5,
            valueLabelDisplay: "on",
        },
        yup: {
            required: true,
            checks: [],
        },
    },
    // Step 7: Choose Your Preferred Workout Location
    {
        type: FormStructureType.Header,
        id: "location-header",
        label: "Step 7: Choose Your Preferred Workout Location",
        description: `**Purpose**: Knowing your preferred location helps tailor exercises suitable for that environment.

**Tips**:
- Select the location where you're most comfortable exercising.
- If 'Other', please specify.`,
        tag: "h3",
    },
    {
        type: InputType.Radio,
        id: "workoutLocation",
        fieldName: "workoutLocation",
        isRequired: true,
        label: "Preferred Workout Location",
        props: {
            options: [
                { label: "Gym", value: "gym" },
                { label: "Home", value: "home" },
                { label: "Outdoor", value: "outdoor" },
                { label: "Other", value: "other" },
            ],
            row: false,
        },
        yup: {
            required: true,
            checks: [],
        },
    },
    {
        type: InputType.Text,
        id: "otherLocation",
        fieldName: "otherLocation",
        isRequired: false,
        label: "If 'Other', please specify",
        props: {
            isMarkdown: false,
            placeholder: "Specify your preferred location",
            minRows: 1,
            maxRows: 2,
        },
        yup: {
            required: false,
            checks: [],
        },
    },
] as const;

const routines: ResourceImportData[] = [
    {
        ...baseRoot,
        shape: {
            ...baseRootShape,
            id: nanoid(),
            publicId: SEEDED_PUBLIC_IDS.MintNativeToken,
            resourceType: ResourceType.Routine,
            versions: [{
                ...baseVersion,
                shape: {
                    ...baseVersionShape,
                    id: nanoid(),
                    config: (new RoutineVersionConfig({
                        config: {
                            __version: "1.0",
                            resources: [
                                {
                                    usedFor: "ExternalService",
                                    link: "https://minterr.io/mint-cardano-tokens/",
                                    translations: [{ language: DEFAULT_LANGUAGE, name: "minterr.io" }],
                                },
                                {
                                    usedFor: "Developer",
                                    link: "https://developers.cardano.org/docs/native-tokens/minting/",
                                    translations: [{ language: DEFAULT_LANGUAGE, name: "cardano.org guide" }],
                                },
                            ],
                        },
                        resourceSubType: ResourceSubType.RoutineInformational,
                    })),
                    isAutomatable: false,
                    resourceSubType: ResourceSubType.RoutineInformational,
                    translations: [{
                        language: "en",
                        id: nanoid(),
                        description: "Mint a fungible token on the Cardano blockchain.",
                        instructions: "To mint through a web interface, select the online resource and follow the instructions.\nTo mint through the command line, select the developer resource and follow the instructions.",
                        name: "Mint Native Token",
                    }],
                },
            }],
        },
    },
    {
        ...baseRoot,
        shape: {
            ...baseRootShape,
            id: nanoid(),
            publicId: SEEDED_PUBLIC_IDS.MintNFT,
            resourceType: ResourceType.Routine,
            versions: [{
                ...baseVersion,
                shape: {
                    ...baseVersionShape,
                    id: nanoid(),
                    config: (new RoutineVersionConfig({
                        config: {
                            __version: "1.0",
                            resources: [
                                {
                                    usedFor: "ExternalService",
                                    link: "https://minterr.io/mint-cardano-tokens/",
                                    translations: [{ language: DEFAULT_LANGUAGE, name: "minterr.io" }],
                                },
                                {
                                    usedFor: "ExternalService",
                                    link: "https://cardano-tools.io/mint",
                                    translations: [{ language: DEFAULT_LANGUAGE, name: "cardano-tools.io" }],
                                },
                                {
                                    usedFor: "Developer",
                                    link: "https://developers.cardano.org/docs/native-tokens/minting-nfts",
                                    translations: [{ language: DEFAULT_LANGUAGE, name: "cardano.org guide" }],
                                },
                            ],
                        },
                        resourceSubType: ResourceSubType.RoutineInformational,
                    })),
                    isAutomatable: false,
                    resourceSubType: ResourceSubType.RoutineInformational,
                    translations: [{
                        language: "en",
                        id: nanoid(),
                        description: "Mint a non-fungible token (NFT) on the Cardano blockchain.",
                        instructions: "To mint through a web interface, select one of the online resources and follow the instructions.\nTo mint through the command line, select the developer resource and follow the instructions.",
                        name: "Mint NFT",
                    }],
                },
            }],
        },
    },
    {
        ...baseRoot,
        shape: {
            ...baseRootShape,
            id: nanoid(),
            publicId: SEEDED_PUBLIC_IDS.ProjectKickoffChecklist,
            resourceType: ResourceType.Routine,
            versions: [{
                ...baseVersion,
                shape: {
                    ...baseVersionShape,
                    id: nanoid(),
                    config: (new RoutineVersionConfig({
                        config: {
                            __version: "1.0",
                            resources: [
                                {
                                    usedFor: "Tutorial",
                                    link: "https://www.projectmanagementdocs.com/template/project-charter",
                                    translations: [{ language: DEFAULT_LANGUAGE, name: "Project Charter Template" }],
                                },
                                {
                                    usedFor: "Learning",
                                    link: "https://www.pmi.org/about/learn-about-pmi/what-is-project-management",
                                    translations: [{ language: DEFAULT_LANGUAGE, name: "Project Management Best Practices" }],
                                },
                                {
                                    usedFor: "ExternalService",
                                    link: "https://asana.com",
                                    translations: [{ language: DEFAULT_LANGUAGE, name: "Asana - Project Management Tool" }],
                                },
                            ],
                            formInput: {
                                __version: "1.0",
                                schema: {
                                    layout: {
                                        title: "Project Kickoff Checklist",
                                        description: "Follow the steps below to ensure a successful project kickoff.",
                                    },
                                    containers: [
                                        {
                                            title: "",
                                            description: "",
                                            totalItems: projectKickoffInputElements.length,
                                        },
                                    ],
                                    elements: projectKickoffInputElements,
                                },
                            },
                        },
                        resourceSubType: ResourceSubType.RoutineInformational,
                    })),
                    isAutomatable: false,
                    resourceSubType: ResourceSubType.RoutineInformational,
                    translations: [{
                        language: "en",
                        id: nanoid(),
                        name: "Project Kickoff Checklist",
                        description: "A comprehensive guide to effectively initiate a new project.",
                        instructions: "Fill out the form.",
                    }],
                },
            }],
        },
    },
    {
        ...baseRoot,
        shape: {
            ...baseRootShape,
            id: nanoid(),
            publicId: SEEDED_PUBLIC_IDS.WorkoutPlanGenerator,
            resourceType: ResourceType.Routine,
            versions: [{
                ...baseVersion,
                shape: {
                    ...baseVersionShape,
                    id: nanoid(),
                    config: (new RoutineVersionConfig({
                        config: {
                            __version: "1.0",
                            callDataGenerate: {
                                __version: "1.0",
                                schema: {
                                    prompt: `You are a personal trainer creating a customized workout plan.
    
The user has provided the following information:

Fitness Goal: {fitnessGoal}
Fitness Level: {fitnessLevel}
Available Equipment: {availableEquipment}
Preferred Workout Days: {workoutDays}
Injuries or Limitations: {injuries}
Time per Workout: {workoutDuration} minutes
Preferred Workout Location: {workoutLocation}

Generate a detailed weekly workout plan that aligns with the user's goals and constraints.

Make sure to:

- Include exercises appropriate for the user's fitness level.
- Utilize the available equipment.
- Schedule workouts on the preferred days.
- Consider any injuries or limitations.
- Keep each workout within the specified duration.
- Provide tips or modifications if necessary.

Format the plan in a clear and organized manner, using markdown bullet points or tables where appropriate.`,
                                },
                            },
                            formInput: {
                                __version: "1.0",
                                schema: {
                                    layout: {
                                        title: "Workout Plan Generator",
                                        description: "Provide some information to receive a personalized workout plan.",
                                    },
                                    containers: [
                                        {
                                            title: "",
                                            description: "",
                                            totalItems: workoutPlanGeneratorInputElements.length,
                                        },
                                    ],
                                    elements: workoutPlanGeneratorInputElements,
                                },
                            },
                        },
                        resourceSubType: ResourceSubType.RoutineGenerate,
                    })),
                    isAutomatable: true,
                    resourceSubType: ResourceSubType.RoutineGenerate,
                    translations: [{
                        language: "en",
                        id: nanoid(),
                        name: "Workout Plan Generator",
                        description: "Generates a personalized workout plan based on your inputs.",
                        instructions: "Fill out the form to receive your workout plan.",
                    }],
                },
            }],
        },
    },
];

const standards: ResourceImportData[] = [
    {
        ...baseRoot,
        shape: {
            ...baseRootShape,
            id: nanoid(),
            publicId: SEEDED_PUBLIC_IDS.CIP0025NFTMetadataStandard,
            resourceType: ResourceType.Standard,
            tags: [
                ...baseRootShape.tags,
                { __typename: "Tag" as const, id: nanoid(), tag: SEEDED_TAGS.Cardano },
                { __typename: "Tag" as const, id: nanoid(), tag: SEEDED_TAGS.Cip },
            ],
            versions: [{
                ...baseVersion,
                shape: {
                    ...baseVersionShape,
                    id: nanoid(),
                    codeLanguage: CodeLanguage.Json,
                    config: (new StandardVersionConfig({
                        config: {
                            __version: "1.0",
                            schema: "{\"format\":{\"<721>\":{\"<policy_id>\":{\"<asset_name>\":{\"name\":\"<asset_name>\",\"image\":\"<ipfs_link>\",\"?mediaType\":\"<mime_type>\",\"?description\":\"<description>\",\"?files\":[{\"name\":\"<asset_name>\",\"mediaType\":\"<mime_type>\",\"src\":\"<ipfs_link>\"}],\"[x]\":\"[any]\"}},\"version\":\"1.0\"}},\"defaults\":[]}",
                            schemaLanguage: "json",
                        },
                        resourceSubType: ResourceSubType.StandardDataStructure,
                    })),
                    resourceSubType: ResourceSubType.StandardDataStructure,
                    translations: [{
                        language: "en",
                        id: nanoid(),
                        name: "CIP-0025 - NFT Metadata Standard",
                        description: "A metadata standard for Native Token NFTs on Cardano.",
                    }],
                },
            }],
        },
    },
];

export const data: ResourceImportData[] = [
    ...codes,
    ...routines,
    ...standards,
];
