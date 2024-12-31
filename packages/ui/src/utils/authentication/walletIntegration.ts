/**
 * Handles wallet integration
 * See CIP-0030 for more info: https://github.com/cardano-foundation/CIPs/pull/148
 */
import { endpointPostAuthWalletComplete, endpointPostAuthWalletInit, WalletComplete, WalletCompleteInput, WalletInit, WalletInitInput } from "@local/shared";
import { fetchWrapper } from "api/fetchWrapper";
import { PubSub } from "utils/pubsub";

/**
 * Object returned from await window.cardano[providerKey].enable()
 */
export type WalletActions = {
    getNetworkId: () => Promise<number>,
    getRewardAddresses: () => Promise<string[]>,
    getUnusedAddresses: () => Promise<string[]>,
    getUsedAddresses: () => Promise<string[]>,
    signData: (addr: string, sigStructure: string) => Promise<string>,
}

/**
 * Object stored in window.cardano[providerKey]
 */
export type WalletProviderInfo = {
    apiVersion: string,
    enable: () => Promise<WalletActions>,
    experimental?: { [x: string]: any },
    icon: string, // base64
    isEnabled: () => Promise<boolean>,
    name: string, // Display name
}

/**
 * Known wallet extension download URLs
 */
export const walletDownloadUrls = {
    // 'cardwallet': ['Card Wallet', 'https://chrome.google.com/webstore/detail/cwallet/apnehcjmnengpnmccpaibjmhhoadaico'],
    //'eternl': ['eternl (CCVault.io)', 'https://chrome.google.com/webstore/detail/eternl/kmhcihpebfmpgmihbkipmjlmmioameka'],
    "flint": ["Flint", "https://chrome.google.com/webstore/detail/flint-wallet/hnhobjmcibchnmglfbldbfabcgaknlkj"],
    // 'gero': ['Gero', 'https://chrome.google.com/webstore/detail/gerowallet/bgpipimickeadkjlklgciifhnalhdjhe'],
    "nami": ["Nami", "https://chrome.google.com/webstore/detail/nami/lpfcbjknijpeeillifnkikgncikgfhdo"],
    "nufi": ["NuFi", "https://chrome.google.com/webstore/detail/nufi/gpnihlnnodeiiaakbikldcihojploeca"],
    // 'yoroi': 'https://chrome.google.com/webstore/detail/yoroi/ffnbelfdoeiohenkjibnmadjiehjhajb',
} as const;

/**
 * Maps network names to their ids
 */
const Network = {
    Mainnet: 1,
    Testnet: 0,
};

/**
 * Checks is a wallet extension is installed
 * @param key The wallet provider to check 
 * @returns True if installed, false if not
 */
export function hasWalletExtension(key: string) {
    return Boolean(window.cardano && window.cardano[key]);
}

/**
 * Finds all wallet providers that are installed
 * @returns Array of key and wallet provider info, as they appear in window.cardano
 */
export function getInstalledWalletProviders(): [string, WalletProviderInfo][] {
    // Check if window.cardano is defined
    if (!window.cardano) {
        return [];
    }
    // Extensions that don't work for some reason (TODO)
    const exclude = ["eternl", "gero", "cardwallet"];
    // Filter out all entries that don't match the WalletProviderInfo shape
    let providers = Object.entries(window.cardano).filter(([key, value]) => {
        if (typeof value !== "object") return false;
        const obj = value as { [x: string]: any };
        if (!Object.prototype.hasOwnProperty.call(obj, "apiVersion")) return false;
        if (!Object.prototype.hasOwnProperty.call(obj, "enable")) return false;
        if (!Object.prototype.hasOwnProperty.call(obj, "name")) return false;
        if (!Object.prototype.hasOwnProperty.call(obj, "icon")) return false;
        if (!Object.prototype.hasOwnProperty.call(obj, "isEnabled")) return false;
        return true;
    }) as [string, WalletProviderInfo][];
    // Filter out duplicate names and excluded
    providers = providers.filter(([key, value], index) => {
        const currName = value.name;
        const nextName = providers.slice(index + 1).find(([, next]) => next.name === currName);
        return !nextName && !exclude.includes(key);
    });
    return providers;
}

/**
 * Connects to wallet provider
 * @param key The wallet provider to connect to
 * @returns Object containing methods to interact with the wallet provider
 */
async function connectWallet(key: string): Promise<any> {
    if (!hasWalletExtension(key)) return false;
    return await window.cardano[key].enable();
}

/**
 * Initiates handshake to verify wallet with backend.
 * @param stakingAddress Wallet's staking address
 * @returns Hex string of payload to be signed by wallet
 */
async function walletInit(stakingAddress: string): Promise<string | null> {
    const data = await fetchWrapper<WalletInitInput, WalletInit>({
        ...endpointPostAuthWalletInit,
        inputs: { stakingAddress },
    });
    return data?.data?.nonce ?? null;
}

/**
 * Completes handshake to verify wallet with backend
 * @param stakingAddress Wallet's staking address
 * @param signedPayload Message signed by wallet
 * @returns Session object if successful, null if not
 */
async function walletComplete(stakingAddress: string, signedPayload: string): Promise<WalletComplete | null> {
    const data = await fetchWrapper<WalletCompleteInput, WalletComplete>({
        ...endpointPostAuthWalletComplete,
        inputs: { stakingAddress, signedPayload },
    });
    return data?.data ?? null;
}

/**
 * Signs payload received from walletInit
 * @param key The wallet provider to use
 * @param walletActions Object returned from connectWallet
 * @param stakingAddress Wallet's staking address
 * @param payload Hex string of payload to be signed by wallet
 * @returns Result of walletActions.signData
 */
async function signPayload(key: string, walletActions: WalletActions, stakingAddress: string, payload: string): Promise<any> {
    if (!hasWalletExtension(key)) return null;
    // As of 2022-02-05, new Nami endpoint is not fully working. So the old method is used for now
    if (key === "nami")
        return await window.cardano.signData(stakingAddress, payload);
    // For all other providers, we use the new method
    return await walletActions.signData(stakingAddress, payload);
}

/**
 * Establish trust between a user's wallet and the backend
 * @param key The wallet provider to use
 * @returns WalletCompleteResult or null
 */
export async function validateWallet(key: string): Promise<WalletComplete | null> {
    let result: WalletComplete | null = null;
    try {
        // Connect to wallet extension
        const walletActions = await connectWallet(key);
        if (!walletActions) return null;
        // Check if wallet is mainnet or testnet
        const network = await walletActions.getNetworkId();
        if (network !== Network.Mainnet) throw new Error("Wallet is not on mainnet");
        // Find wallet address
        const stakingAddresses = await walletActions.getRewardAddresses();
        if (!stakingAddresses || stakingAddresses.length === 0) throw new Error("Could not find staking address");
        // Request payload from backend
        const payload = await walletInit(stakingAddresses[0]);
        if (!payload) return null;
        // Sign payload with wallet
        const signedPayload = await signPayload(key, walletActions, stakingAddresses[0], payload);
        if (!signedPayload) return null;
        // Send signed payload to backend for verification
        result = (await walletComplete(stakingAddresses[0], signedPayload));
    } catch (error: any) {
        console.error("Caught error completing wallet validation", error);
        PubSub.get().publish("alertDialog", {
            messageKey: "WalletErrorUnknown",
            buttons: [{ labelKey: "Ok" }],
        });
    }
    return result;
}
