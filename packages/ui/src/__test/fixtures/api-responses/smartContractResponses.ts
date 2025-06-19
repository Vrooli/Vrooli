import { type Resource, type ResourceVersion, type Label, type Tag, ResourceSubType } from "@vrooli/shared";
import { minimalUserResponse, completeUserResponse } from "./userResponses.js";
import { minimalTeamResponse } from "./teamResponses.js";

/**
 * API response fixtures for smart contracts (Resources with CodeSmartContract subtype)
 * These represent what components receive from API calls
 */

/**
 * Mock tag data for smart contracts
 */
const ethereumTag: Tag = {
    __typename: "Tag",
    id: "tag_ethereum_123456789",
    tag: "ethereum",
    created_at: "2024-01-01T00:00:00Z",
    bookmarks: 342,
    translations: [
        {
            __typename: "TagTranslation",
            id: "tagtrans_ethereum_123",
            language: "en",
            description: "Ethereum blockchain smart contracts",
        },
    ],
    you: {
        __typename: "TagYou",
        isBookmarked: true,
    },
};

const defiTag: Tag = {
    __typename: "Tag",
    id: "tag_defi_123456789",
    tag: "defi",
    created_at: "2024-01-01T00:00:00Z",
    bookmarks: 256,
    translations: [
        {
            __typename: "TagTranslation",
            id: "tagtrans_defi_123",
            language: "en",
            description: "Decentralized finance applications",
        },
    ],
    you: {
        __typename: "TagYou",
        isBookmarked: false,
    },
};

const solidityTag: Tag = {
    __typename: "Tag",
    id: "tag_solidity_123456789",
    tag: "solidity",
    created_at: "2024-01-01T00:00:00Z",
    bookmarks: 189,
    translations: [
        {
            __typename: "TagTranslation",
            id: "tagtrans_solidity_123",
            language: "en",
            description: "Solidity programming language",
        },
    ],
    you: {
        __typename: "TagYou",
        isBookmarked: false,
    },
};

const nftTag: Tag = {
    __typename: "Tag",
    id: "tag_nft_123456789",
    tag: "nft",
    created_at: "2024-01-01T00:00:00Z",
    bookmarks: 423,
    translations: [],
    you: {
        __typename: "TagYou",
        isBookmarked: false,
    },
};

const polygonTag: Tag = {
    __typename: "Tag",
    id: "tag_polygon_123456789",
    tag: "polygon",
    created_at: "2024-01-01T00:00:00Z",
    bookmarks: 178,
    translations: [
        {
            __typename: "TagTranslation",
            id: "tagtrans_polygon_123",
            language: "en",
            description: "Polygon (MATIC) blockchain",
        },
    ],
    you: {
        __typename: "TagYou",
        isBookmarked: false,
    },
};

/**
 * Mock label data for smart contracts
 */
const auditedLabel: Label = {
    __typename: "Label",
    id: "label_audited_123456789",
    label: "Audited",
    color: "#4caf50",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    you: {
        __typename: "LabelYou",
        canDelete: false,
        canUpdate: false,
    },
};

const mainnetLabel: Label = {
    __typename: "Label",
    id: "label_mainnet_123456789",
    label: "Mainnet",
    color: "#2196f3",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    you: {
        __typename: "LabelYou",
        canDelete: true,
        canUpdate: true,
    },
};

const testnetLabel: Label = {
    __typename: "Label",
    id: "label_testnet_123456789",
    label: "Testnet",
    color: "#ff9800",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    you: {
        __typename: "LabelYou",
        canDelete: true,
        canUpdate: true,
    },
};

const upgradableLabel: Label = {
    __typename: "Label",
    id: "label_upgradable_123456",
    label: "Upgradable",
    color: "#9c27b0",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    you: {
        __typename: "LabelYou",
        canDelete: false,
        canUpdate: false,
    },
};

/**
 * Minimal smart contract version
 */
const minimalSmartContractVersion: ResourceVersion = {
    __typename: "ResourceVersion",
    id: "resver_sc_123456789012345",
    versionLabel: "1.0.0",
    versionNotes: null,
    isLatest: true,
    isPrivate: false,
    isComplete: true,
    isAutomatable: true,
    resourceSubType: ResourceSubType.CodeSmartContract,
    codeLanguage: "solidity",
    complexity: 5,
    simplicity: 5,
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    completedAt: "2024-01-01T00:00:00Z",
    comments: [],
    commentsCount: 0,
    forks: [],
    forksCount: 0,
    pulls: [],
    pullsCount: 0,
    issues: [],
    issuesCount: 0,
    labels: [],
    labelsCount: 0,
    reportsCount: 0,
    score: 0,
    bookmarks: 0,
    views: 0,
    transfersCount: 0,
    directoryListings: [],
    directoryListingsCount: 0,
    isInternal: false,
    root: null,
    timesCompleted: 0,
    timesStarted: 0,
    translations: [
        {
            __typename: "ResourceVersionTranslation",
            id: "resvtrans_sc_123456789",
            language: "en",
            name: "Basic Token Contract",
            description: "Simple ERC-20 token smart contract",
            instructions: "Deploy this contract to create a basic fungible token",
            details: null,
        },
    ],
    resourceList: null,
    you: {
        __typename: "ResourceVersionYou",
        canBookmark: true,
        canComment: true,
        canCopy: true,
        canDelete: false,
        canReact: true,
        canRead: true,
        canReport: true,
        canRun: true,
        canUpdate: false,
    },
};

/**
 * Complete smart contract version with all features
 */
const completeSmartContractVersion: ResourceVersion = {
    __typename: "ResourceVersion",
    id: "resver_sc_987654321098765",
    versionLabel: "2.5.0",
    versionNotes: "Added governance features, improved gas optimization, and enhanced security measures",
    isLatest: true,
    isPrivate: false,
    isComplete: true,
    isAutomatable: true,
    resourceSubType: ResourceSubType.CodeSmartContract,
    codeLanguage: "solidity",
    complexity: 8,
    simplicity: 3,
    createdAt: "2023-06-01T00:00:00Z",
    updatedAt: "2024-01-15T14:45:00Z",
    completedAt: "2024-01-10T00:00:00Z",
    comments: [],
    commentsCount: 24,
    forks: [],
    forksCount: 12,
    pulls: [],
    pullsCount: 3,
    issues: [],
    issuesCount: 2,
    labels: [auditedLabel, mainnetLabel],
    labelsCount: 2,
    reportsCount: 0,
    score: 456,
    bookmarks: 234,
    views: 8765,
    transfersCount: 0,
    directoryListings: [],
    directoryListingsCount: 3,
    isInternal: false,
    root: null,
    timesCompleted: 0,
    timesStarted: 0,
    config: {
        contractType: "upgradeable-proxy",
        compiler: "0.8.19",
        optimization: true,
        runs: 200,
        networks: {
            mainnet: {
                address: "0x1234567890abcdef1234567890abcdef12345678",
                deployedAt: "2024-01-10T00:00:00Z",
                blockNumber: 18976543,
            },
            polygon: {
                address: "0xabcdef1234567890abcdef1234567890abcdef12",
                deployedAt: "2024-01-10T00:00:00Z",
                blockNumber: 51234567,
            },
        },
        abi: "[{\"inputs\":[],\"name\":\"name\",\"outputs\":[{\"internalType\":\"string\",\"name\":\"\",\"type\":\"string\"}],\"stateMutability\":\"view\",\"type\":\"function\"}]",
        bytecode: "0x608060405234801561001057600080fd5b50...",
        gasEstimates: {
            creation: 2500000,
            external: {
                transfer: 65000,
                approve: 45000,
                mint: 85000,
            },
        },
    },
    translations: [
        {
            __typename: "ResourceVersionTranslation",
            id: "resvtrans_sc_987654321",
            language: "en",
            name: "DeFi Yield Farming Protocol",
            description: "Advanced DeFi protocol for yield farming with auto-compounding, governance tokens, and multi-strategy vaults",
            instructions: "## Deployment Guide\n\n### Prerequisites\n1. Node.js v16+ and npm/yarn\n2. Hardhat or Truffle framework\n3. Ethereum wallet with sufficient ETH for gas\n4. API keys for Etherscan verification\n\n### Deployment Steps\n1. Clone the repository\n2. Install dependencies: `npm install`\n3. Configure `.env` with your private key and RPC endpoints\n4. Run tests: `npm test`\n5. Deploy to testnet: `npm run deploy:testnet`\n6. Verify contract: `npm run verify`\n7. Deploy to mainnet: `npm run deploy:mainnet`\n\n### Configuration\n- Set initial parameters in `deploy/001_deploy_protocol.js`\n- Configure governance timelock period\n- Set initial reward rates and farming pools\n\n### Security Considerations\n- Contract has been audited by CertiK and OpenZeppelin\n- Uses OpenZeppelin's upgradeable contracts\n- Implements reentrancy guards and access controls\n- Timelock for all administrative functions",
            details: "This protocol supports multiple yield strategies including lending, liquidity provision, and staking. It features auto-compounding of rewards, governance token distribution, and emergency withdrawal mechanisms. The contract is upgradeable using the transparent proxy pattern.",
        },
        {
            __typename: "ResourceVersionTranslation",
            id: "resvtrans_sc_876543210",
            language: "es",
            name: "Protocolo DeFi de Yield Farming",
            description: "Protocolo DeFi avanzado para yield farming con auto-compounding, tokens de gobernanza y bóvedas multi-estrategia",
            instructions: "## Guía de Implementación\n\n### Requisitos Previos\n1. Node.js v16+ y npm/yarn\n2. Framework Hardhat o Truffle",
            details: null,
        },
    ],
    resourceList: {
        __typename: "ResourceList",
        id: "reslist_sc_123",
        usedFor: "Display",
        resources: [],
    },
    you: {
        __typename: "ResourceVersionYou",
        canBookmark: true,
        canComment: true,
        canCopy: true,
        canDelete: true,
        canReact: true,
        canRead: true,
        canReport: false,
        canRun: true,
        canUpdate: true,
    },
};

/**
 * Minimal smart contract API response
 */
export const minimalSmartContractResponse: Resource = {
    __typename: "Resource",
    id: "resource_sc_123456789012345",
    isDeleted: false,
    isInternal: false,
    isPrivate: false,
    resourceType: "Standard",
    createdAt: "2024-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    completedAt: "2024-01-01T00:00:00Z",
    score: 0,
    bookmarks: 0,
    views: 0,
    bookmarkedBy: [],
    owner: {
        __typename: "User",
        ...minimalUserResponse,
    },
    hasCompleteVersion: true,
    labels: [],
    tags: [],
    permissions: JSON.stringify(["Read"]),
    versions: [minimalSmartContractVersion],
    versionsCount: 1,
    forksCount: 0,
    issues: [],
    issuesCount: 0,
    pullsCount: 0,
    reportsCount: 0,
    transfersCount: 0,
    you: {
        __typename: "ResourceYou",
        canBookmark: true,
        canComment: true,
        canDelete: false,
        canReact: true,
        canRead: true,
        canTransfer: false,
        canUpdate: false,
        isBookmarked: false,
        isViewed: false,
        reaction: null,
    },
};

/**
 * Complete smart contract API response with all fields
 */
export const completeSmartContractResponse: Resource = {
    __typename: "Resource",
    id: "resource_sc_987654321098765",
    isDeleted: false,
    isInternal: false,
    isPrivate: false,
    resourceType: "Standard",
    createdAt: "2023-06-01T00:00:00Z",
    updatedAt: "2024-01-15T14:45:00Z",
    completedAt: "2024-01-10T00:00:00Z",
    score: 456,
    bookmarks: 234,
    views: 8765,
    bookmarkedBy: [],
    owner: {
        __typename: "Team",
        ...minimalTeamResponse,
    },
    hasCompleteVersion: true,
    labels: [auditedLabel, mainnetLabel],
    tags: [ethereumTag, defiTag, solidityTag],
    permissions: JSON.stringify(["Read", "Copy", "Fork", "Run"]),
    versions: [completeSmartContractVersion],
    versionsCount: 8,
    forksCount: 12,
    issues: [],
    issuesCount: 2,
    pullsCount: 3,
    reportsCount: 0,
    transfersCount: 0,
    you: {
        __typename: "ResourceYou",
        canBookmark: true,
        canComment: true,
        canDelete: true,
        canReact: true,
        canRead: true,
        canTransfer: true,
        canUpdate: true,
        isBookmarked: true,
        isViewed: true,
        reaction: "like",
    },
};

/**
 * Private smart contract response
 */
export const privateSmartContractResponse: Resource = {
    __typename: "Resource",
    id: "resource_sc_private_123456",
    isDeleted: false,
    isInternal: false,
    isPrivate: true,
    resourceType: "Standard",
    createdAt: "2023-01-01T00:00:00Z",
    updatedAt: "2024-01-01T00:00:00Z",
    completedAt: null,
    score: 0,
    bookmarks: 0,
    views: 0,
    bookmarkedBy: [],
    owner: {
        __typename: "User",
        ...completeUserResponse,
    },
    hasCompleteVersion: false,
    labels: [testnetLabel],
    tags: [],
    permissions: JSON.stringify([]),
    versions: [{
        ...minimalSmartContractVersion,
        id: "resver_sc_private_123456",
        isPrivate: true,
        isComplete: false,
        translations: [
            {
                __typename: "ResourceVersionTranslation",
                id: "resvtrans_sc_private_123",
                language: "en",
                name: "Private Protocol Contract",
                description: "Private smart contract for internal protocol operations",
                instructions: "Internal use only - deployment restricted",
                details: null,
            },
        ],
    }],
    versionsCount: 1,
    forksCount: 0,
    issues: [],
    issuesCount: 0,
    pullsCount: 0,
    reportsCount: 0,
    transfersCount: 0,
    you: {
        __typename: "ResourceYou",
        canBookmark: false,
        canComment: false,
        canDelete: false,
        canReact: false,
        canRead: false,
        canTransfer: false,
        canUpdate: false,
        isBookmarked: false,
        isViewed: false,
        reaction: null,
    },
};

/**
 * Smart contract variant states for testing
 */
export const smartContractResponseVariants = {
    minimal: minimalSmartContractResponse,
    complete: completeSmartContractResponse,
    private: privateSmartContractResponse,
    erc20Token: {
        ...completeSmartContractResponse,
        id: "resource_sc_erc20_123456789",
        versions: [{
            ...completeSmartContractVersion,
            id: "resver_sc_erc20_123456789",
            translations: [
                {
                    __typename: "ResourceVersionTranslation",
                    id: "resvtrans_sc_erc20_123456",
                    language: "en",
                    name: "ERC-20 Token Standard Implementation",
                    description: "Standard ERC-20 fungible token with minting, burning, and pausable features",
                    instructions: "## ERC-20 Token Contract\n\n### Features\n- Standard ERC-20 compliance\n- Minting and burning capabilities\n- Pausable transfers\n- Access control with roles\n- Permit functionality (EIP-2612)\n\n### Deployment\n1. Set token name and symbol\n2. Configure initial supply\n3. Assign minter and pauser roles\n4. Deploy and verify contract",
                    details: "Implements OpenZeppelin's ERC20 with additional features for minting, burning, and emergency pause functionality. Includes permit for gasless approvals.",
                },
            ],
            config: {
                contractType: "erc20",
                standard: "ERC-20",
                features: ["mintable", "burnable", "pausable", "permit"],
                tokenSymbol: "VRT",
                tokenName: "Vrooli Token",
                decimals: 18,
                initialSupply: "1000000000000000000000000",
            },
        }],
        tags: [ethereumTag, solidityTag],
    },
    erc721NFT: {
        ...minimalSmartContractResponse,
        id: "resource_sc_erc721_123456789",
        versions: [{
            ...minimalSmartContractVersion,
            id: "resver_sc_erc721_123456789",
            complexity: 6,
            simplicity: 4,
            translations: [
                {
                    __typename: "ResourceVersionTranslation",
                    id: "resvtrans_sc_erc721_123456",
                    language: "en",
                    name: "ERC-721 NFT Collection",
                    description: "Non-fungible token collection with metadata, royalties, and marketplace integration",
                    instructions: "Deploy this contract to create an NFT collection with on-chain metadata and royalty support",
                    details: "Supports ERC-721 standard with enumerable extension, royalty info (EIP-2981), and metadata URI management",
                },
            ],
            config: {
                contractType: "erc721",
                standard: "ERC-721",
                features: ["enumerable", "royalties", "metadata", "pausable"],
                collectionName: "Vrooli NFTs",
                collectionSymbol: "VNFT",
                maxSupply: 10000,
                royaltyBps: 250, // 2.5%
            },
        }],
        tags: [ethereumTag, nftTag, solidityTag],
    },
    customProtocol: {
        ...completeSmartContractResponse,
        id: "resource_sc_custom_123456",
        versions: [{
            ...completeSmartContractVersion,
            id: "resver_sc_custom_123456",
            complexity: 10,
            simplicity: 2,
            translations: [
                {
                    __typename: "ResourceVersionTranslation",
                    id: "resvtrans_sc_custom_123",
                    language: "en",
                    name: "Custom DeFi AMM Protocol",
                    description: "Automated market maker protocol with concentrated liquidity, dynamic fees, and multi-pool support",
                    instructions: "## Custom AMM Protocol\n\n### Architecture\n- Factory contract for pool creation\n- Router for optimal swap paths\n- Oracle integration for price feeds\n- Governance for parameter updates\n\n### Key Features\n- Concentrated liquidity positions\n- Dynamic fee tiers\n- Flash loan support\n- Multi-hop swaps",
                    details: "Advanced AMM implementation inspired by Uniswap V3 with additional features for fee optimization and capital efficiency",
                },
            ],
            config: {
                contractType: "custom-defi",
                protocol: "amm",
                version: "1.0.0",
                contracts: {
                    factory: "0x1234...",
                    router: "0x5678...",
                    governance: "0x9abc...",
                },
                parameters: {
                    feeTiers: [100, 500, 3000, 10000],
                    protocolFee: 1000,
                    minLiquidity: "1000",
                },
            },
        }],
        labels: [auditedLabel, mainnetLabel, upgradableLabel],
        tags: [ethereumTag, defiTag, solidityTag],
    },
    polygonContract: {
        ...minimalSmartContractResponse,
        id: "resource_sc_polygon_123456",
        versions: [{
            ...minimalSmartContractVersion,
            id: "resver_sc_polygon_123456",
            translations: [
                {
                    __typename: "ResourceVersionTranslation",
                    id: "resvtrans_sc_polygon_123",
                    language: "en",
                    name: "Polygon Gaming Contract",
                    description: "Low-cost gaming smart contract optimized for Polygon network",
                    instructions: "Optimized for high-frequency transactions on Polygon with minimal gas costs",
                    details: "Implements game logic with state channels for off-chain computation and on-chain settlement",
                },
            ],
            config: {
                contractType: "gaming",
                network: "polygon",
                chainId: 137,
                optimizations: {
                    batchTransactions: true,
                    stateChannels: true,
                    gasEfficient: true,
                },
            },
        }],
        tags: [polygonTag, solidityTag],
    },
    testnetContract: {
        ...privateSmartContractResponse,
        id: "resource_sc_testnet_123456",
        isPrivate: false,
        versions: [{
            ...minimalSmartContractVersion,
            id: "resver_sc_testnet_123456",
            isComplete: false,
            versionLabel: "0.1.0-beta",
            translations: [
                {
                    __typename: "ResourceVersionTranslation",
                    id: "resvtrans_sc_testnet_123",
                    language: "en",
                    name: "Experimental Staking Contract",
                    description: "Test version of staking contract deployed on Goerli testnet",
                    instructions: "FOR TESTING ONLY - Do not use with real funds",
                    details: "Experimental features being tested on Goerli before mainnet deployment",
                },
            ],
            config: {
                contractType: "staking",
                network: "goerli",
                chainId: 5,
                testnet: true,
                deployments: {
                    goerli: {
                        address: "0xtest1234567890abcdef1234567890abcdef1234",
                        faucet: "https://goerlifaucet.com",
                    },
                },
            },
        }],
        labels: [testnetLabel],
        tags: [ethereumTag, solidityTag],
    },
    auditedContract: {
        ...completeSmartContractResponse,
        id: "resource_sc_audited_123456",
        score: 9999,
        bookmarks: 1500,
        views: 50000,
        versions: [{
            ...completeSmartContractVersion,
            id: "resver_sc_audited_123456",
            score: 9999,
            bookmarks: 1500,
            views: 50000,
            translations: [
                {
                    __typename: "ResourceVersionTranslation",
                    id: "resvtrans_sc_audited_123",
                    language: "en",
                    name: "Audited Treasury Management System",
                    description: "Highly secure treasury contract audited by multiple firms with formal verification",
                    instructions: "Enterprise-grade treasury management with multi-sig, timelocks, and comprehensive access controls",
                    details: "Audited by CertiK, Trail of Bits, and OpenZeppelin. Includes formal verification proofs and extensive test coverage (100%)",
                },
            ],
            config: {
                contractType: "treasury",
                security: {
                    audits: [
                        {
                            firm: "CertiK",
                            date: "2023-12-01",
                            report: "ipfs://QmAuditReport1...",
                        },
                        {
                            firm: "Trail of Bits",
                            date: "2023-12-15",
                            report: "ipfs://QmAuditReport2...",
                        },
                    ],
                    formalVerification: true,
                    testCoverage: 100,
                    bugBounty: {
                        platform: "Immunefi",
                        maxReward: "$500,000",
                    },
                },
            },
        }],
        labels: [auditedLabel, mainnetLabel],
        tags: [ethereumTag, solidityTag, defiTag],
    },
    upgradableProxy: {
        ...completeSmartContractResponse,
        id: "resource_sc_upgradable_123456",
        versions: [{
            ...completeSmartContractVersion,
            id: "resver_sc_upgradable_123456",
            versionLabel: "3.0.0",
            versionNotes: "Upgraded to support new features while maintaining state",
            translations: [
                {
                    __typename: "ResourceVersionTranslation",
                    id: "resvtrans_sc_upgradable_123",
                    language: "en",
                    name: "Upgradeable Governance Token",
                    description: "Upgradeable proxy pattern implementation for governance token with migration support",
                    instructions: "## Upgradeable Contract\n\n### Proxy Pattern\n- Uses OpenZeppelin's TransparentUpgradeableProxy\n- Separate ProxyAdmin for upgrades\n- Storage layout preservation\n\n### Upgrade Process\n1. Deploy new implementation\n2. Propose upgrade via governance\n3. Execute after timelock\n4. Verify storage compatibility",
                    details: "Implements UUPS proxy pattern with comprehensive upgrade safeguards and governance controls",
                },
            ],
            config: {
                contractType: "upgradeable-governance",
                proxyType: "TransparentUpgradeableProxy",
                implementation: {
                    current: "0xcurrent123...",
                    previous: ["0xold123...", "0xolder123..."],
                },
                proxyAdmin: "0xadmin123...",
                upgradeHistory: [
                    {
                        version: "2.0.0",
                        date: "2023-09-01",
                        changes: "Added delegation features",
                    },
                    {
                        version: "3.0.0",
                        date: "2024-01-10",
                        changes: "Enhanced voting mechanisms",
                    },
                ],
            },
        }],
        labels: [auditedLabel, mainnetLabel, upgradableLabel],
        tags: [ethereumTag, solidityTag],
    },
} as const;

/**
 * Smart contract search response
 */
export const smartContractSearchResponse = {
    __typename: "ResourceSearchResult",
    edges: [
        {
            __typename: "ResourceEdge",
            cursor: "cursor_1",
            node: smartContractResponseVariants.complete,
        },
        {
            __typename: "ResourceEdge",
            cursor: "cursor_2",
            node: smartContractResponseVariants.erc20Token,
        },
        {
            __typename: "ResourceEdge",
            cursor: "cursor_3",
            node: smartContractResponseVariants.erc721NFT,
        },
        {
            __typename: "ResourceEdge",
            cursor: "cursor_4",
            node: smartContractResponseVariants.auditedContract,
        },
    ],
    pageInfo: {
        __typename: "PageInfo",
        hasNextPage: true,
        hasPreviousPage: false,
        startCursor: "cursor_1",
        endCursor: "cursor_4",
    },
};

/**
 * Loading and error states for UI testing
 */
export const smartContractUIStates = {
    loading: null,
    error: {
        code: "SMART_CONTRACT_NOT_FOUND",
        message: "The requested smart contract could not be found",
    },
    deploymentError: {
        code: "CONTRACT_DEPLOYMENT_FAILED",
        message: "Failed to deploy contract. Please check your configuration and try again.",
    },
    networkError: {
        code: "UNSUPPORTED_NETWORK",
        message: "The selected network is not supported. Please switch to a supported network.",
    },
    gasError: {
        code: "INSUFFICIENT_GAS",
        message: "Insufficient gas for contract deployment. Please increase gas limit or add funds.",
    },
    verificationError: {
        code: "CONTRACT_VERIFICATION_FAILED",
        message: "Contract verification failed. The source code does not match the deployed bytecode.",
    },
    empty: {
        edges: [],
        pageInfo: {
            __typename: "PageInfo",
            hasNextPage: false,
            hasPreviousPage: false,
            startCursor: null,
            endCursor: null,
        },
    },
};