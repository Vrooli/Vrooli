/* eslint-disable @typescript-eslint/no-empty-function */
/* eslint-disable react-perf/jsx-no-new-function-as-prop */
/* eslint-disable no-magic-numbers */
// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-01 - Fixed 4 'as any' type assertions (replaced with null for circular refs, proper CodeVersion type)
import { CodeLanguage, ResourceSubType, ResourceUsedFor, endpointsResource, generatePK, getObjectUrl, type Code, type CodeVersion, type CodeVersionTranslation } from "@vrooli/shared";
import { HttpResponse, http } from "msw";
import { API_URL, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { getMockEndpoint, getStoryRouteEditPath } from "../../../__test/helpers/storybookMocking.js";
import { SmartContractUpsert } from "./SmartContractUpsert.js";

// Create simplified mock data for CodeVersion responses
const mockCodeVersionData: CodeVersion = {
    __typename: "CodeVersion" as const,
    id: generatePK().toString(),
    createdAt: new Date().toISOString(),
    updatedAt: new Date().toISOString(),
    calledByRoutineVersionsCount: 2,
    codeLanguage: CodeLanguage.Solidity,
    codeType: ResourceSubType.CodeSmartContract,
    content: `// SPDX-License-Identifier: MIT
pragma solidity ^0.8.0;

contract SimpleStorage {
    uint256 private storedData;
    
    function set(uint256 x) public {
        storedData = x;
    }
    
    function get() public view returns (uint256) {
        return storedData;
    }
}`,
    isComplete: true,
    isPrivate: false,
    root: {
        __typename: "Code" as const,
        id: generatePK().toString(),
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        owner: {
            __typename: "User" as const,
            id: generatePK().toString(),
            handle: "blockchain-dev",
            name: "Blockchain Developer",
            profileImage: null,
        },
        parent: null,
        isPrivate: false,
        versions: [{
            __typename: "CodeVersion" as const,
            id: generatePK().toString(),
            versionLabel: "0.9.0",
        }, {
            __typename: "CodeVersion" as const,
            id: generatePK().toString(),
            versionLabel: "1.0.0",
        }],
        tags: [{
            __typename: "Tag" as const,
            id: generatePK().toString(),
            label: "Ethereum",
        }, {
            __typename: "Tag" as const,
            id: generatePK().toString(),
            label: "Smart Contract",
        }, {
            __typename: "Tag" as const,
            id: generatePK().toString(),
            label: "Storage",
        }],
    } as unknown as Code,
    resourceList: {
        __typename: "ResourceList" as const,
        id: generatePK().toString(),
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        listFor: null, // Circular reference handled by null
        resources: [
            {
                __typename: "Resource" as const,
                id: generatePK().toString(),
                createdAt: new Date().toISOString(),
                updatedAt: new Date().toISOString(),
                usedFor: ResourceUsedFor.Context,
                link: "https://ethereum.org/en/developers/docs/smart-contracts/",
                list: null, // Circular reference handled by null
                translations: [{
                    __typename: "ResourceTranslation" as const,
                    id: generatePK().toString(),
                    language: "en",
                    name: "Smart Contracts Documentation",
                    description: "Official Ethereum documentation on smart contracts",
                }],
            },
            {
                __typename: "Resource" as const,
                id: generatePK().toString(),
                createdAt: new Date().toISOString(),
                updatedAt: new Date().toISOString(),
                usedFor: ResourceUsedFor.Context,
                link: "https://solidity.readthedocs.io/",
                list: null, // Circular reference handled by null
                translations: [{
                    __typename: "ResourceTranslation" as const,
                    id: generatePK().toString(),
                    language: "en",
                    name: "Solidity Documentation",
                    description: "Official documentation for the Solidity programming language",
                }],
            },
        ],
        translations: [],
    },
    translations: [
        {
            __typename: "CodeVersionTranslation" as const,
            id: generatePK().toString(),
            language: "en",
            name: "Simple Storage Contract",
            description: "A basic smart contract that demonstrates storage functionality on the Ethereum blockchain. This contract allows setting and retrieving a single integer value.",
            jsonVariable: "",
        } as CodeVersionTranslation,
    ],
    versionLabel: "1.0.0",
    you: {
        __typename: "VersionYou" as const,
        canComment: true,
        canDelete: true,
        canReport: true,
        canUpdate: true,
        canUse: true,
    },
} as CodeVersion; // Cast to CodeVersion with proper typing

export default {
    title: "Views/Objects/SmartContract/SmartContractUpsert",
    component: SmartContractUpsert,
};

// Create a new Smart Contract
export function Create() {
    return (
        <SmartContractUpsert display="Page" isCreate={true} />
    );
}
Create.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// Create a new Smart Contract in a dialog
export function CreateDialog() {
    return (
        <SmartContractUpsert
            display="Dialog"
            isCreate={true}
            isOpen={true}
            onClose={() => { }}
            onCancel={() => { }}
            onCompleted={() => { }}
            onDeleted={() => { }}
        />
    );
}
CreateDialog.parameters = {
    session: signedInPremiumWithCreditsSession,
};

// Update an existing Smart Contract
export function Update() {
    return (
        <SmartContractUpsert display="Page" isCreate={false} />
    );
}
Update.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(getMockEndpoint(endpointsResource.findSmartContractVersion), () => {
                return HttpResponse.json({ data: mockCodeVersionData });
            }),
        ],
    },
    route: {
        path: getStoryRouteEditPath(mockCodeVersionData),
    },
};

// Update an existing Smart Contract in a dialog
export function UpdateDialog() {
    return (
        <SmartContractUpsert
            display="Dialog"
            isCreate={false}
            isOpen={true}
            onClose={() => { }}
            onCancel={() => { }}
            onCompleted={() => { }}
            onDeleted={() => { }}
        />
    );
}
UpdateDialog.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(getMockEndpoint(endpointsResource.findSmartContractVersion), () => {
                return HttpResponse.json({ data: mockCodeVersionData });
            }),
        ],
    },
    route: {
        path: getStoryRouteEditPath(mockCodeVersionData),
    },
};

// Loading state
export function Loading() {
    return (
        <SmartContractUpsert display="Page" isCreate={false} />
    );
}
Loading.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(getMockEndpoint(endpointsResource.findSmartContractVersion), async () => {
                // Delay the response to simulate loading
                await new Promise(resolve => setTimeout(resolve, 120000));
                return HttpResponse.json({ data: mockCodeVersionData });
            }),
        ],
    },
    route: {
        path: getStoryRouteEditPath(mockCodeVersionData),
    },
};

// Non-premium user
export function NonPremiumUser() {
    return (
        <SmartContractUpsert display="Page" isCreate={true} />
    );
}
NonPremiumUser.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

// With Override Object (using dialog display)
export function WithOverrideObject() {
    return (
        <SmartContractUpsert
            display="Dialog"
            isCreate={true}
            isOpen={true}
            onClose={() => { }}
            onCancel={() => { }}
            onCompleted={() => { }}
            onDeleted={() => { }}
            overrideObject={mockCodeVersionData}
        />
    );
}
WithOverrideObject.parameters = {
    session: signedInPremiumWithCreditsSession,
}; 
