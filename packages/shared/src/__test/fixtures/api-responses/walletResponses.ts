/* c8 ignore start */
/**
 * Wallet API Response Fixtures
 * 
 * Comprehensive fixtures for cryptocurrency wallet management including
 * wallet verification, multi-chain support, and payment integration.
 */

import type {
    Wallet,
    WalletCreateInput,
    WalletUpdateInput,
    User,
    Team,
} from "../../../api/types.js";
import { BaseAPIResponseFactory } from "./base.js";
import type { MockDataOptions } from "./types.js";
import { generatePK } from "../../../id/index.js";
import { userResponseFactory } from "./userResponses.js";
import { teamResponseFactory } from "./teamResponses.js";

// Constants
const DEFAULT_COUNT = 10;
const DEFAULT_ERROR_RATE = 0.1;
const DEFAULT_DELAY_MS = 500;
const MAX_WALLETS_PER_USER = 10;
const MAX_HANDLE_LENGTH = 100;
const MAX_NAME_LENGTH = 200;

// Supported wallet types and chains
const SUPPORTED_CHAINS = ["ethereum", "bitcoin", "cardano", "solana", "polygon", "binance"];

/**
 * Wallet API response factory
 */
export class WalletResponseFactory extends BaseAPIResponseFactory<
    Wallet,
    WalletCreateInput,
    WalletUpdateInput
> {
    protected readonly entityName = "wallet";

    /**
     * Create mock wallet data
     */
    createMockData(options?: MockDataOptions): Wallet {
        const scenario = options?.scenario || "minimal";
        const now = new Date().toISOString();
        const walletId = options?.overrides?.id || generatePK().toString();

        const baseWallet: Wallet = {
            __typename: "Wallet",
            id: walletId,
            created_at: now,
            updated_at: now,
            handle: `wallet_${walletId.slice(-8)}`,
            name: "Default Wallet",
            publicAddress: this.generateEthereumAddress(),
            stakingAddress: null,
            verified: false,
            user: userResponseFactory.createMockData(),
            team: null,
            you: {
                canDelete: true,
                canUpdate: true,
            },
        };

        if (scenario === "complete" || scenario === "edge-case") {
            return {
                ...baseWallet,
                handle: "comprehensive_crypto_wallet",
                name: "Multi-Chain Verified Wallet",
                publicAddress: "0xa0b86991c431e59b79c1d2f01e81b0d7b4a39e5c7", // Real USDC contract address
                stakingAddress: "stake1ux3g2c3k8xajtl5w2h8q5k8q5k8q5k8q5k8q5k8q5k8q5k",
                verified: true,
                user: scenario === "edge-case" ? null : userResponseFactory.createMockData({ scenario: "complete" }),
                team: scenario === "edge-case" ? teamResponseFactory.createMockData({ scenario: "complete" }) : null,
                you: {
                    canDelete: scenario !== "edge-case", // Team wallets might not be deletable
                    canUpdate: true,
                },
                ...options?.overrides,
            };
        }

        return {
            ...baseWallet,
            ...options?.overrides,
        };
    }

    /**
     * Create wallet from input
     */
    createFromInput(input: WalletCreateInput): Wallet {
        const now = new Date().toISOString();
        const walletId = generatePK().toString();

        let user: User | null = null;
        let team: Team | null = null;

        if (input.userConnect) {
            user = userResponseFactory.createMockData({ overrides: { id: input.userConnect } });
        } else if (input.teamConnect) {
            team = teamResponseFactory.createMockData({ overrides: { id: input.teamConnect } });
        } else {
            // Default to current user if no connection specified
            user = userResponseFactory.createMockData();
        }

        return {
            __typename: "Wallet",
            id: walletId,
            created_at: now,
            updated_at: now,
            handle: input.handle || `wallet_${walletId.slice(-8)}`,
            name: input.name || "New Wallet",
            publicAddress: input.publicAddress || this.generateEthereumAddress(),
            stakingAddress: input.stakingAddress || null,
            verified: input.verified || false,
            user,
            team,
            you: {
                canDelete: true,
                canUpdate: true,
            },
        };
    }

    /**
     * Update wallet from input
     */
    updateFromInput(existing: Wallet, input: WalletUpdateInput): Wallet {
        const updates: Partial<Wallet> = {
            updated_at: new Date().toISOString(),
        };

        if (input.handle !== undefined) updates.handle = input.handle;
        if (input.name !== undefined) updates.name = input.name;
        if (input.publicAddress !== undefined) updates.publicAddress = input.publicAddress;
        if (input.stakingAddress !== undefined) updates.stakingAddress = input.stakingAddress;
        if (input.verified !== undefined) updates.verified = input.verified;

        return {
            ...existing,
            ...updates,
        };
    }

    /**
     * Validate create input
     */
    async validateCreateInput(input: WalletCreateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (!input.publicAddress) {
            errors.publicAddress = "Public address is required";
        } else if (!this.isValidAddress(input.publicAddress)) {
            errors.publicAddress = "Invalid public address format";
        }

        if (!input.name || input.name.trim().length === 0) {
            errors.name = "Wallet name is required";
        } else if (input.name.length > MAX_NAME_LENGTH) {
            errors.name = `Wallet name must be ${MAX_NAME_LENGTH} characters or less`;
        }

        if (input.handle) {
            if (input.handle.length > MAX_HANDLE_LENGTH) {
                errors.handle = `Handle must be ${MAX_HANDLE_LENGTH} characters or less`;
            } else if (!/^[a-zA-Z0-9_-]+$/.test(input.handle)) {
                errors.handle = "Handle can only contain letters, numbers, hyphens, and underscores";
            }
        }

        if (input.stakingAddress && !this.isValidStakingAddress(input.stakingAddress)) {
            errors.stakingAddress = "Invalid staking address format";
        }

        if (!input.userConnect && !input.teamConnect) {
            errors.connection = "Wallet must be connected to either a user or team";
        }

        if (input.userConnect && input.teamConnect) {
            errors.connection = "Wallet cannot be connected to both user and team";
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Validate update input
     */
    async validateUpdateInput(input: WalletUpdateInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (input.publicAddress !== undefined && !this.isValidAddress(input.publicAddress)) {
            errors.publicAddress = "Invalid public address format";
        }

        if (input.name !== undefined) {
            if (!input.name || input.name.trim().length === 0) {
                errors.name = "Wallet name cannot be empty";
            } else if (input.name.length > MAX_NAME_LENGTH) {
                errors.name = `Wallet name must be ${MAX_NAME_LENGTH} characters or less`;
            }
        }

        if (input.handle !== undefined) {
            if (input.handle.length > MAX_HANDLE_LENGTH) {
                errors.handle = `Handle must be ${MAX_HANDLE_LENGTH} characters or less`;
            } else if (!/^[a-zA-Z0-9_-]+$/.test(input.handle)) {
                errors.handle = "Handle can only contain letters, numbers, hyphens, and underscores";
            }
        }

        if (input.stakingAddress !== undefined && input.stakingAddress && !this.isValidStakingAddress(input.stakingAddress)) {
            errors.stakingAddress = "Invalid staking address format";
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Create different types of wallets
     */
    createWalletTypes(): Wallet[] {
        return [
            // Personal verified Ethereum wallet
            this.createMockData({
                overrides: {
                    name: "Primary Ethereum Wallet",
                    handle: "primary_eth_wallet",
                    verified: true,
                    publicAddress: "0xa0b86991c431e59b79c1d2f01e81b0d7b4a39e5c7",
                    stakingAddress: null,
                },
            }),

            // Personal unverified Bitcoin wallet
            this.createMockData({
                overrides: {
                    name: "Bitcoin Wallet",
                    handle: "btc_wallet",
                    verified: false,
                    publicAddress: this.generateBitcoinAddress(),
                    stakingAddress: null,
                },
            }),

            // Cardano wallet with staking
            this.createMockData({
                overrides: {
                    name: "Cardano Staking Wallet",
                    handle: "ada_staking_wallet",
                    verified: true,
                    publicAddress: this.generateCardanoAddress(),
                    stakingAddress: this.generateCardanoStakeAddress(),
                },
            }),

            // Team treasury wallet
            this.createMockData({
                overrides: {
                    name: "Team Treasury",
                    handle: "team_treasury",
                    verified: true,
                    publicAddress: this.generateEthereumAddress(),
                    user: null,
                    team: teamResponseFactory.createMockData({ scenario: "complete" }),
                    you: {
                        canDelete: false, // Team wallets require special permissions
                        canUpdate: true,
                    },
                },
            }),

            // Multi-sig wallet
            this.createMockData({
                overrides: {
                    name: "Multi-Signature Wallet",
                    handle: "multisig_wallet",
                    verified: true,
                    publicAddress: this.generateEthereumAddress(),
                    stakingAddress: null,
                },
            }),
        ];
    }

    /**
     * Create wallets for a specific user
     */
    createWalletsForUser(userId: string, count = 3): Wallet[] {
        const user = userResponseFactory.createMockData({ overrides: { id: userId } });
        
        return Array.from({ length: count }, (_, index) => 
            this.createMockData({
                overrides: {
                    id: `wallet_${userId}_${index}`,
                    user,
                    team: null,
                    name: `Wallet ${index + 1}`,
                    handle: `wallet_${index + 1}_${userId.slice(-6)}`,
                    verified: index === 0, // First wallet is typically verified
                    publicAddress: index === 0 
                        ? this.generateEthereumAddress()
                        : index === 1 
                        ? this.generateBitcoinAddress()
                        : this.generateCardanoAddress(),
                    stakingAddress: index === 2 ? this.generateCardanoStakeAddress() : null,
                },
            }),
        );
    }

    /**
     * Create wallets for different blockchains
     */
    createMultiChainWallets(): Wallet[] {
        const chains = [
            { name: "Ethereum", generator: () => this.generateEthereumAddress() },
            { name: "Bitcoin", generator: () => this.generateBitcoinAddress() },
            { name: "Cardano", generator: () => this.generateCardanoAddress() },
            { name: "Solana", generator: () => this.generateSolanaAddress() },
            { name: "Polygon", generator: () => this.generateEthereumAddress() }, // Same format as Ethereum
        ];

        return chains.map((chain, index) => 
            this.createMockData({
                overrides: {
                    name: `${chain.name} Wallet`,
                    handle: `${chain.name.toLowerCase()}_wallet`,
                    publicAddress: chain.generator(),
                    verified: index < 3, // First 3 are verified
                    stakingAddress: chain.name === "Cardano" ? this.generateCardanoStakeAddress() : null,
                },
            }),
        );
    }

    /**
     * Create wallet verification status response
     */
    createVerificationStatusResponse(walletId: string, verified = false) {
        return {
            data: {
                id: walletId,
                verified,
                verificationPending: !verified && Math.random() > 0.5,
                lastVerificationAttempt: new Date(Date.now() - (24 * 60 * 60 * 1000)).toISOString(), // 1 day ago
                verificationMethod: verified ? "signature" : null,
                nextRetryAllowed: verified ? null : new Date(Date.now() + (60 * 60 * 1000)).toISOString(), // 1 hour from now
            },
            meta: {
                timestamp: new Date().toISOString(),
                requestId: generatePK().toString(),
                version: "1.0",
                links: {
                    self: `/api/wallet/${walletId}/verification-status`,
                    verify: `/api/wallet/${walletId}/verify`,
                },
            },
        };
    }

    /**
     * Create wallet balance response (mock)
     */
    createWalletBalanceResponse(walletId: string) {
        return {
            data: {
                walletId,
                balances: [
                    {
                        token: "ETH",
                        amount: (Math.random() * 10).toFixed(6),
                        usdValue: (Math.random() * 25000).toFixed(2),
                        lastUpdated: new Date().toISOString(),
                    },
                    {
                        token: "USDC",
                        amount: (Math.random() * 1000).toFixed(2),
                        usdValue: (Math.random() * 1000).toFixed(2),
                        lastUpdated: new Date().toISOString(),
                    },
                    {
                        token: "BTC",
                        amount: (Math.random() * 0.1).toFixed(8),
                        usdValue: (Math.random() * 5000).toFixed(2),
                        lastUpdated: new Date().toISOString(),
                    },
                ],
                totalUsdValue: (Math.random() * 30000).toFixed(2),
                lastUpdated: new Date().toISOString(),
            },
            meta: {
                timestamp: new Date().toISOString(),
                requestId: generatePK().toString(),
                version: "1.0",
                links: {
                    self: `/api/wallet/${walletId}/balance`,
                    transactions: `/api/wallet/${walletId}/transactions`,
                },
            },
        };
    }

    /**
     * Create duplicate wallet error response
     */
    createDuplicateWalletErrorResponse(publicAddress: string) {
        return this.createBusinessErrorResponse("duplicate", {
            resource: "wallet",
            publicAddress,
            message: `Wallet with address ${publicAddress} already exists`,
        });
    }

    /**
     * Create wallet limit error response
     */
    createWalletLimitErrorResponse(currentCount = MAX_WALLETS_PER_USER) {
        return this.createBusinessErrorResponse("limit", {
            resource: "wallet",
            limit: MAX_WALLETS_PER_USER,
            current: currentCount,
            message: `Maximum number of wallets (${MAX_WALLETS_PER_USER}) reached`,
        });
    }

    /**
     * Create verification failed error response
     */
    createVerificationFailedErrorResponse(reason: string) {
        return this.createBusinessErrorResponse("verification_failed", {
            resource: "wallet",
            reason,
            retryAllowed: true,
            retryAfter: 3600, // 1 hour
            message: `Wallet verification failed: ${reason}`,
        });
    }

    /**
     * Create unsupported chain error response
     */
    createUnsupportedChainErrorResponse(chain: string) {
        return this.createValidationErrorResponse({
            publicAddress: `Unsupported blockchain: ${chain}. Supported chains: ${SUPPORTED_CHAINS.join(", ")}`,
        });
    }

    /**
     * Generate realistic crypto addresses
     */
    private generateEthereumAddress(): string {
        return `0x${Math.random().toString(16).substring(2, 42).padEnd(40, "0")}`;
    }

    private generateBitcoinAddress(): string {
        const prefixes = ["1", "3", "bc1"];
        const prefix = prefixes[Math.floor(Math.random() * prefixes.length)];
        const length = prefix === "bc1" ? 42 : 34;
        const chars = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz";
        let address = prefix;
        
        for (let i = prefix.length; i < length; i++) {
            address += chars.charAt(Math.floor(Math.random() * chars.length));
        }
        
        return address;
    }

    private generateCardanoAddress(): string {
        return `addr1${Math.random().toString(16).substring(2, 98).padEnd(96, "0")}`;
    }

    private generateCardanoStakeAddress(): string {
        return `stake1${Math.random().toString(16).substring(2, 56).padEnd(54, "0")}`;
    }

    private generateSolanaAddress(): string {
        const chars = "123456789ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnopqrstuvwxyz";
        let address = "";
        
        for (let i = 0; i < 44; i++) {
            address += chars.charAt(Math.floor(Math.random() * chars.length));
        }
        
        return address;
    }

    /**
     * Basic address validation
     */
    private isValidAddress(address: string): boolean {
        // Ethereum address
        if (/^0x[a-fA-F0-9]{40}$/.test(address)) return true;
        
        // Bitcoin address (simplified)
        if (/^[13bc1][a-zA-Z0-9]{25,42}$/.test(address)) return true;
        
        // Cardano address
        if (/^addr1[a-z0-9]{96}$/.test(address)) return true;
        
        // Solana address
        if (/^[1-9A-HJ-NP-Za-km-z]{32,44}$/.test(address)) return true;
        
        return false;
    }

    private isValidStakingAddress(address: string): boolean {
        // Cardano staking address
        return /^stake1[a-z0-9]{54}$/.test(address);
    }
}

/**
 * Pre-configured wallet response scenarios
 */
export const walletResponseScenarios = {
    // Success scenarios
    createSuccess: (input?: Partial<WalletCreateInput>) => {
        const factory = new WalletResponseFactory();
        const defaultInput: WalletCreateInput = {
            name: "New Wallet",
            publicAddress: factory["generateEthereumAddress"](),
            userConnect: generatePK().toString(),
            ...input,
        };
        return factory.createSuccessResponse(
            factory.createFromInput(defaultInput),
        );
    },

    findSuccess: (wallet?: Wallet) => {
        const factory = new WalletResponseFactory();
        return factory.createSuccessResponse(
            wallet || factory.createMockData(),
        );
    },

    findCompleteSuccess: () => {
        const factory = new WalletResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({ scenario: "complete" }),
        );
    },

    updateSuccess: (existing?: Wallet, updates?: Partial<WalletUpdateInput>) => {
        const factory = new WalletResponseFactory();
        const wallet = existing || factory.createMockData({ scenario: "complete" });
        const input: WalletUpdateInput = {
            id: wallet.id,
            ...updates,
        };
        return factory.createSuccessResponse(
            factory.updateFromInput(wallet, input),
        );
    },

    listSuccess: (wallets?: Wallet[]) => {
        const factory = new WalletResponseFactory();
        return factory.createPaginatedResponse(
            wallets || Array.from({ length: DEFAULT_COUNT }, () => factory.createMockData()),
            { page: 1, totalCount: wallets?.length || DEFAULT_COUNT },
        );
    },

    walletTypesSuccess: () => {
        const factory = new WalletResponseFactory();
        const wallets = factory.createWalletTypes();
        return factory.createPaginatedResponse(
            wallets,
            { page: 1, totalCount: wallets.length },
        );
    },

    userWalletsSuccess: (userId?: string) => {
        const factory = new WalletResponseFactory();
        const wallets = factory.createWalletsForUser(userId || generatePK().toString());
        return factory.createPaginatedResponse(
            wallets,
            { page: 1, totalCount: wallets.length },
        );
    },

    multiChainWalletsSuccess: () => {
        const factory = new WalletResponseFactory();
        const wallets = factory.createMultiChainWallets();
        return factory.createPaginatedResponse(
            wallets,
            { page: 1, totalCount: wallets.length },
        );
    },

    verifiedWalletSuccess: () => {
        const factory = new WalletResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: {
                    verified: true,
                    name: "Verified Ethereum Wallet",
                    publicAddress: "0xa0b86991c431e59b79c1d2f01e81b0d7b4a39e5c7",
                },
            }),
        );
    },

    teamWalletSuccess: (teamId?: string) => {
        const factory = new WalletResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: {
                    user: null,
                    team: teamResponseFactory.createMockData({ overrides: { id: teamId } }),
                    name: "Team Treasury Wallet",
                    verified: true,
                },
            }),
        );
    },

    verificationStatusSuccess: (walletId?: string, verified = true) => {
        const factory = new WalletResponseFactory();
        return factory.createVerificationStatusResponse(walletId || generatePK().toString(), verified);
    },

    walletBalanceSuccess: (walletId?: string) => {
        const factory = new WalletResponseFactory();
        return factory.createWalletBalanceResponse(walletId || generatePK().toString());
    },

    verifySuccess: (walletId?: string) => {
        const factory = new WalletResponseFactory();
        return factory.createSuccessResponse(
            factory.createMockData({
                overrides: {
                    id: walletId,
                    verified: true,
                    updated_at: new Date().toISOString(),
                },
            }),
        );
    },

    // Error scenarios
    createValidationError: () => {
        const factory = new WalletResponseFactory();
        return factory.createValidationErrorResponse({
            publicAddress: "Invalid wallet address format",
            name: "Wallet name is required",
            connection: "Wallet must be connected to either a user or team",
        });
    },

    notFoundError: (walletId?: string) => {
        const factory = new WalletResponseFactory();
        return factory.createNotFoundErrorResponse(
            walletId || "non-existent-wallet",
        );
    },

    permissionError: (operation?: string) => {
        const factory = new WalletResponseFactory();
        return factory.createPermissionErrorResponse(
            operation || "create",
            ["wallet:write"],
        );
    },

    duplicateWalletError: (publicAddress?: string) => {
        const factory = new WalletResponseFactory();
        return factory.createDuplicateWalletErrorResponse(
            publicAddress || "0xa0b86991c431e59b79c1d2f01e81b0d7b4a39e5c7",
        );
    },

    walletLimitError: (currentCount?: number) => {
        const factory = new WalletResponseFactory();
        return factory.createWalletLimitErrorResponse(currentCount);
    },

    verificationFailedError: (reason = "Invalid signature") => {
        const factory = new WalletResponseFactory();
        return factory.createVerificationFailedErrorResponse(reason);
    },

    unsupportedChainError: (chain = "Unknown") => {
        const factory = new WalletResponseFactory();
        return factory.createUnsupportedChainErrorResponse(chain);
    },

    invalidAddressError: () => {
        const factory = new WalletResponseFactory();
        return factory.createValidationErrorResponse({
            publicAddress: "Invalid wallet address format",
        });
    },

    // MSW handlers
    handlers: {
        success: () => new WalletResponseFactory().createMSWHandlers(),
        withErrors: function createWithErrors(errorRate?: number) {
            return new WalletResponseFactory().createMSWHandlers({ errorRate: errorRate ?? DEFAULT_ERROR_RATE });
        },
        withDelay: function createWithDelay(delay?: number) {
            return new WalletResponseFactory().createMSWHandlers({ delay: delay ?? DEFAULT_DELAY_MS });
        },
    },
};

// Export factory instance for direct use
export const walletResponseFactory = new WalletResponseFactory();
