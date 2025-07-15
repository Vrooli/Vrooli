// AI_CHECK: TYPE_SAFETY=eliminated-5-any-types | LAST: 2025-06-30
/* eslint-disable func-style */
/* eslint-disable @typescript-eslint/no-empty-function */
/* eslint-disable react-perf/jsx-no-new-function-as-prop */
/* eslint-disable no-magic-numbers */
import { CodeLanguage, ResourceSubType, ResourceUsedFor, endpointsResource, generatePK, getObjectUrl, type Code, type CodeVersion, type CodeVersionTranslation } from "@vrooli/shared";
import { HttpResponse, http } from "msw";
import { API_URL, loggedOutSession, signedInNoPremiumNoCreditsSession, signedInPremiumWithCreditsSession } from "../../../__test/storybookConsts.js";
import { getMockEndpoint, getStoryRoutePath } from "../../../__test/helpers/storybookMocking.js";
import { SmartContractView } from "./SmartContractView.js";

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
    } as Code,
    resourceList: {
        __typename: "ResourceList" as const,
        id: generatePK().toString(),
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
        listFor: null as unknown as Code, // Circular reference handled with null
        resources: [
            {
                __typename: "Resource" as const,
                id: generatePK().toString(),
                createdAt: new Date().toISOString(),
                updatedAt: new Date().toISOString(),
                usedFor: ResourceUsedFor.Context,
                link: "https://ethereum.org/en/developers/docs/smart-contracts/",
                list: null as unknown as Code["resourceList"], // Circular reference handled with null
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
                list: null as unknown as Code["resourceList"], // Circular reference handled with null
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
        __typename: "CodeVersionYou" as const,
        canComment: true,
        canDelete: true,
        canReport: true,
        canUpdate: true,
        canUse: true,
        isBookmarked: false,
    },
} as CodeVersion;

// Create a second version with a different language for translation testing
const mockCodeVersionWithHaskellData = {
    ...mockCodeVersionData,
    id: generatePK().toString(),
    codeLanguage: CodeLanguage.Haskell,
    content: `data AuctionParams = AuctionParams
  { apSeller  :: PubKeyHash,
    apAsset   :: Value,
    apMinBid  :: Lovelace,
    apEndTime :: POSIXTime
  }

PlutusTx.makeLift ''AuctionParams

data Bid = Bid
  { bBidder :: PubKeyHash,
    bAmount :: Lovelace
  }

PlutusTx.deriveShow ''Bid
PlutusTx.unstableMakeIsData ''Bid

instance PlutusTx.Eq Bid where
  {-# INLINEABLE (==) #-}
  bid == bid' =
    bBidder bid PlutusTx.== bBidder bid'
      PlutusTx.&& bAmount bid PlutusTx.== bAmount bid'

newtype AuctionDatum = AuctionDatum { adHighestBid :: Maybe Bid }

PlutusTx.unstableMakeIsData ''AuctionDatum

data AuctionRedeemer = NewBid Bid | Payout`,
    translations: [
        {
            __typename: "CodeVersionTranslation" as const,
            id: generatePK().toString(),
            language: "en",
            name: "Plutus Auction Contract",
            description: "A Plutus-based auction smart contract for the Cardano blockchain. This contract implements a simple auction mechanism where users can place bids and the highest bidder wins.",
            jsonVariable: "",
        } as CodeVersionTranslation,
        {
            __typename: "CodeVersionTranslation" as const,
            id: generatePK().toString(),
            language: "es",
            name: "Contrato de Subasta Plutus",
            description: "Un contrato inteligente de subasta basado en Plutus para la blockchain de Cardano. Este contrato implementa un mecanismo de subasta simple donde los usuarios pueden hacer ofertas y el mejor postor gana.",
            jsonVariable: "",
        } as CodeVersionTranslation,
    ],
    versionLabel: "2.1.0",
} as CodeVersion;

export default {
    title: "Views/Objects/SmartContract/SmartContractView",
    component: SmartContractView,
};

// No results state
export function NoResults() {
    return (
        <SmartContractView display="Page" />
    );
}
NoResults.parameters = {
    session: signedInNoPremiumNoCreditsSession,
};

// Loading state
export function Loading() {
    return (
        <SmartContractView display="Page" />
    );
}
Loading.parameters = {
    session: signedInNoPremiumNoCreditsSession,
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
        path: getStoryRoutePath(mockCodeVersionData),
    },
};

// Signed in user viewing a Solidity smart contract
export function SignedInWithSolidity() {
    return (
        <SmartContractView display="Page" />
    );
}
SignedInWithSolidity.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(getMockEndpoint(endpointsResource.findSmartContractVersion), () => {
                return HttpResponse.json({ data: mockCodeVersionData });
            }),
        ],
    },
    route: {
        path: getStoryRoutePath(mockCodeVersionData),
    },
};

// Signed in user viewing a Haskell smart contract with multiple translations
export function SignedInWithHaskell() {
    return (
        <SmartContractView display="Page" />
    );
}
SignedInWithHaskell.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(getMockEndpoint(endpointsResource.findSmartContractVersion), () => {
                return HttpResponse.json({ data: mockCodeVersionWithHaskellData });
            }),
        ],
    },
    route: {
        path: getStoryRoutePath(mockCodeVersionWithHaskellData),
    },
};

// Dialog view
export function DialogView() {
    const handleClose = (): void => { };

    return (
        <SmartContractView
            display="Dialog"
            isOpen={true}
            onClose={handleClose}
        />
    );
}
DialogView.parameters = {
    session: signedInPremiumWithCreditsSession,
    msw: {
        handlers: [
            http.get(getMockEndpoint(endpointsResource.findSmartContractVersion), () => {
                return HttpResponse.json({ data: mockCodeVersionData });
            }),
        ],
    },
    route: {
        path: getStoryRoutePath(mockCodeVersionData),
    },
};

// Logged out user
export function LoggedOut() {
    return (
        <SmartContractView display="Page" />
    );
}
LoggedOut.parameters = {
    session: loggedOutSession,
    msw: {
        handlers: [
            http.get(getMockEndpoint(endpointsResource.findSmartContractVersion), () => {
                return HttpResponse.json({ data: mockCodeVersionData });
            }),
        ],
    },
    route: {
        path: getStoryRoutePath(mockCodeVersionData),
    },
}; 
