import {
    endpointsResource,
    resourceVersionTranslationValidation,
    resourceVersionValidation,
    type ResourceVersionCreateInput,
    type ResourceVersionShape,
    type ResourceVersionUpdateInput,
    type Session,
} from "@vrooli/shared";
import { smartContractInitialValues, transformCodeVersionValues } from "../../../views/objects/smartContract/SmartContractUpsert.js";
import { createUIFormTestFactory, type UIFormTestConfig } from "./UIFormTestFactory.js";

/**
 * Configuration for SmartContract form testing with data-driven test scenarios
 */
const smartContractFormTestConfig: UIFormTestConfig<ResourceVersionShape, ResourceVersionShape, ResourceVersionCreateInput, ResourceVersionUpdateInput, ResourceVersionShape> = {
    // Form metadata
    objectType: "ResourceVersion",
    formFixtures: {
        minimal: {
            __typename: "ResourceVersion" as const,
            id: "smartcontract_minimal",
            codeLanguage: "Solidity",
            config: null,
            isAutomatable: false,
            isComplete: false,
            isPrivate: false,
            resourceSubType: "SmartContract",
            versionLabel: "1.0.0",
            versionNotes: null,
            translations: [{
                __typename: "ResourceVersionTranslation" as const,
                id: "trans_minimal",
                language: "en",
                name: "Test Smart Contract",
                description: "A minimal test smart contract",
                instructions: "pragma solidity ^0.8.0;\n\ncontract TestContract {\n    uint256 public value;\n    \n    function setValue(uint256 _value) public {\n        value = _value;\n    }\n}",
                details: null,
            }],
        },
        complete: {
            __typename: "ResourceVersion" as const,
            id: "smartcontract_complete",
            codeLanguage: "Solidity",
            config: null,
            isAutomatable: false,
            isComplete: true,
            isPrivate: false,
            resourceSubType: "SmartContract",
            versionLabel: "2.1.0",
            versionNotes: "Added advanced features",
            translations: [{
                __typename: "ResourceVersionTranslation" as const,
                id: "trans_complete",
                language: "en",
                name: "Complete Smart Contract",
                description: "A comprehensive smart contract with detailed functionality. This contract implements advanced features including access control, events, and error handling.",
                instructions: "pragma solidity ^0.8.0;\n\nimport \"@openzeppelin/contracts/access/Ownable.sol\";\nimport \"@openzeppelin/contracts/security/ReentrancyGuard.sol\";\n\ncontract CompleteContract is Ownable, ReentrancyGuard {\n    mapping(address => uint256) public balances;\n    \n    event Deposit(address indexed user, uint256 amount);\n    event Withdrawal(address indexed user, uint256 amount);\n    \n    function deposit() public payable {\n        require(msg.value > 0, \"Deposit must be greater than 0\");\n        balances[msg.sender] += msg.value;\n        emit Deposit(msg.sender, msg.value);\n    }\n    \n    function withdraw(uint256 amount) public nonReentrant {\n        require(balances[msg.sender] >= amount, \"Insufficient balance\");\n        balances[msg.sender] -= amount;\n        payable(msg.sender).transfer(amount);\n        emit Withdrawal(msg.sender, amount);\n    }\n}",
                details: "Advanced smart contract with security features",
            }],
        },
        invalid: {
            __typename: "ResourceVersion" as const,
            id: "smartcontract_invalid",
            codeLanguage: "Solidity",
            config: null,
            isAutomatable: false,
            isComplete: false,
            isPrivate: false,
            resourceSubType: "SmartContract",
            versionLabel: "not-a-valid-version", // Invalid version format
            versionNotes: null,
            translations: [{
                __typename: "ResourceVersionTranslation" as const,
                id: "trans_invalid",
                language: "en",
                name: "", // Invalid: required field is empty
                description: "",
                instructions: "", // Invalid: empty code
                details: null,
            }],
        },
        edgeCase: {
            __typename: "ResourceVersion" as const,
            id: "smartcontract_edge",
            codeLanguage: "Rust",
            config: null,
            isAutomatable: false,
            isComplete: false,
            isPrivate: false,
            resourceSubType: "SmartContract",
            versionLabel: "999.999.999",
            versionNotes: "Edge case version",
            translations: [{
                __typename: "ResourceVersionTranslation" as const,
                id: "trans_edge",
                language: "en",
                name: "A".repeat(200), // Edge case: very long name
                description: "Description with special characters: @#$%^&*()[]{}|\\:;\"'<>,.?/~`\n\nMultiple\nline\nbreaks\n\nAnd emoji: ⛓️ 💎 🔒",
                instructions: "// Complex smart contract with special characters\nuse near_sdk::borsh::{self, BorshDeserialize, BorshSerialize};\nuse near_sdk::{env, near_bindgen, AccountId, Balance};\nuse std::collections::HashMap;\n\n#[near_bindgen]\n#[derive(Default, BorshDeserialize, BorshSerialize)]\npub struct Contract {\n    records: HashMap<AccountId, Balance>,\n}\n\n#[near_bindgen]\nimpl Contract {\n    pub fn set_record(&mut self, account_id: AccountId, amount: Balance) {\n        self.records.insert(account_id, amount);\n    }\n    \n    pub fn get_record(&self, account_id: AccountId) -> Option<Balance> {\n        self.records.get(&account_id).copied()\n    }\n}",
                details: "Complex edge case with special characters",
            }],
        },
    },

    // Validation schemas from shared package
    validation: resourceVersionValidation,
    translationValidation: resourceVersionTranslationValidation,

    // API endpoints from shared package
    endpoints: {
        create: endpointsResource.createOne,
        update: endpointsResource.updateOne,
    },

    // Transform functions - form already uses ResourceVersionShape, so no transformation needed
    formToShape: (formData: ResourceVersionShape) => formData,

    transformFunction: (shape: ResourceVersionShape, existing: ResourceVersionShape, isCreate: boolean) => {
        const result = transformCodeVersionValues(shape, existing, isCreate);
        if (!result) {
            throw new Error("Transform function returned undefined");
        }
        return result;
    },

    initialValuesFunction: (session?: Session, existing?: Partial<ResourceVersionShape>): ResourceVersionShape => {
        return smartContractInitialValues(session, existing || {});
    },

    // DATA-DRIVEN TEST SCENARIOS - replaces all custom wrapper methods
    testScenarios: {
        languageValidation: {
            description: "Test smart contract language support",
            testCases: [
                {
                    name: "Solidity language",
                    data: { codeLanguage: "Solidity" },
                    shouldPass: true,
                },
                {
                    name: "Vyper language",
                    data: { codeLanguage: "Vyper" },
                    shouldPass: true,
                },
                {
                    name: "Move language",
                    data: { codeLanguage: "Move" },
                    shouldPass: true,
                },
                {
                    name: "Rust language",
                    data: { codeLanguage: "Rust" },
                    shouldPass: true,
                },
            ],
        },

        codeValidation: {
            description: "Test smart contract code validation",
            testCases: [
                {
                    name: "Valid Solidity code",
                    field: "translations.0.instructions",
                    value: "pragma solidity ^0.8.0;\n\ncontract Test {\n    uint256 public value;\n}",
                    shouldPass: true,
                },
                {
                    name: "Empty code",
                    field: "translations.0.instructions",
                    value: "",
                    shouldPass: false,
                },
                {
                    name: "Complex contract code",
                    field: "translations.0.instructions",
                    value: "pragma solidity ^0.8.0;\n\ncontract ComplexContract {\n    mapping(address => uint256) balances;\n    function transfer() public {}\n}",
                    shouldPass: true,
                },
            ],
        },

        versionValidation: {
            description: "Test version label validation",
            testCases: [
                {
                    name: "Valid semantic version",
                    field: "versionLabel",
                    value: "1.0.0",
                    shouldPass: true,
                },
                {
                    name: "Invalid version format",
                    field: "versionLabel",
                    value: "not-a-version",
                    shouldPass: false,
                },
                {
                    name: "Pre-release version",
                    field: "versionLabel",
                    value: "1.0.0-alpha.1",
                    shouldPass: true,
                },
            ],
        },

        contractFeatures: {
            description: "Test different smart contract features",
            testCases: [
                {
                    name: "Simple contract",
                    data: {
                        isComplete: false,
                        isAutomatable: false,
                        resourceSubType: "SmartContract",
                    },
                    shouldPass: true,
                },
                {
                    name: "Complete contract",
                    data: {
                        isComplete: true,
                        isAutomatable: true,
                        resourceSubType: "SmartContract",
                    },
                    shouldPass: true,
                },
            ],
        },
    },
};

/**
 * SIMPLIFIED: Direct factory export - no wrapper function needed!
 */
export const smartContractFormTestFactory = createUIFormTestFactory(smartContractFormTestConfig);

/**
 * Type exports for use in other test files
 */
export { smartContractFormTestConfig };
