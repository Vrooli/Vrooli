/**
 * Handles wallet integration
 * See CIP-0030 for more info: https://github.com/cardano-foundation/CIPs/pull/148
 */
import { ApolloError } from '@apollo/client';
import { WalletComplete } from '@shared/consts';
import { authWalletComplete } from 'api/generated/endpoints/auth_walletComplete';
import { authWalletInit } from 'api/generated/endpoints/auth_walletInit';
import { errorToCode, initializeApollo } from 'api/utils';
import { PubSub } from 'utils/pubsub';

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
export const walletDownloadUrls: { [x: string]: [string, string] } = {
    // 'cardwallet': ['Card Wallet', 'https://chrome.google.com/webstore/detail/cwallet/apnehcjmnengpnmccpaibjmhhoadaico'],
    //'eternl': ['eternl (CCVault.io)', 'https://chrome.google.com/webstore/detail/eternl/kmhcihpebfmpgmihbkipmjlmmioameka'],
    'flint': ['Flint', 'https://chrome.google.com/webstore/detail/flint-wallet/hnhobjmcibchnmglfbldbfabcgaknlkj'],
    // 'gero': ['Gero', 'https://chrome.google.com/webstore/detail/gerowallet/bgpipimickeadkjlklgciifhnalhdjhe'],
    'nami': ['Nami', 'https://chrome.google.com/webstore/detail/nami/lpfcbjknijpeeillifnkikgncikgfhdo'],
    'nufi': ['NuFi', 'https://chrome.google.com/webstore/detail/nufi/gpnihlnnodeiiaakbikldcihojploeca'],
    // 'yoroi': 'https://chrome.google.com/webstore/detail/yoroi/ffnbelfdoeiohenkjibnmadjiehjhajb',
}

/**
 * Maps network names to their ids
 */
const Network = {
    Mainnet: 1,
    Testnet: 0
}

/**
 * Checks is a wallet extension is installed
 * @param key The wallet provider to check 
 * @returns True if installed, false if not
 */
export const hasWalletExtension = (key: string) => Boolean(window.cardano && window.cardano[key]);

/**
 * Finds all wallet providers that are installed
 * @returns Array of key and wallet provider info, as they appear in window.cardano
 */
export const getInstalledWalletProviders = (): [string, WalletProviderInfo][] => {
    // Check if window.cardano is defined
    if (!window.cardano) {
        return [];
    }
    // Extensions that don't work for some reason (TODO)
    const exclude = ['eternl', 'gero', 'cardwallet'];
    // Filter out all entries that don't match the WalletProviderInfo shape
    let providers = Object.entries(window.cardano).filter(([key, value]) => {
        if (typeof value !== 'object') return false;
        const obj = value as { [x: string]: any };
        if (!obj.hasOwnProperty('apiVersion')) return false;
        if (!obj.hasOwnProperty('enable')) return false;
        if (!obj.hasOwnProperty('name')) return false;
        if (!obj.hasOwnProperty('icon')) return false;
        if (!obj.hasOwnProperty('isEnabled')) return false;
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
const connectWallet = async (key: string): Promise<any> => {
    if (!hasWalletExtension(key)) return false;
    return await window.cardano[key].enable();
}

/**
 * Initiates handshake to verify wallet with backend.
 * @param stakingAddress Wallet's staking address
 * @returns Hex string of payload to be signed by wallet
 */
const walletInit = async (stakingAddress: string): Promise<any> => {
    PubSub.get().publishLoading(500);
    const client = initializeApollo();
    const data = await client.mutate({
        mutation: authWalletInit,
        variables: { input: { stakingAddress } }
    }).catch((error: ApolloError) => {
        PubSub.get().publishSnack({ messageKey: errorToCode(error), severity: 'Error', data: error });
    })
    PubSub.get().publishLoading(false);
    return data?.data?.walletInit;
}

/**
 * Completes handshake to verify wallet with backend
 * @param stakingAddress Wallet's staking address
 * @param signedPayload Message signed by wallet
 * @returns Session object if successful, null if not
 */
const walletComplete = async (stakingAddress: string, signedPayload: string): Promise<WalletComplete | null> => {
    PubSub.get().publishLoading(500);
    const client = initializeApollo();
    const data = await client.mutate({
        mutation: authWalletComplete,
        variables: { input: { stakingAddress, signedPayload } }
    }).catch((error: ApolloError) => {
        PubSub.get().publishSnack({ messageKey: errorToCode(error), severity: 'Error', data: error });
    })
    PubSub.get().publishLoading(false);
    return data?.data?.walletComplete;
}

/**
 * Signs payload received from walletInit
 * @param key The wallet provider to use
 * @param walletActions Object returned from connectWallet
 * @param stakingAddress Wallet's staking address
 * @param payload Hex string of payload to be signed by wallet
 * @returns Result of walletActions.signData
 */
const signPayload = async (key: string, walletActions: WalletActions, stakingAddress: string, payload: string): Promise<any> => {
    if (!hasWalletExtension(key)) return null;
    // As of 2022-02-05, new Nami endpoint is not fully working. So the old method is used for now
    if (key === 'nami')
        return await window.cardano.signData(stakingAddress, payload);
    // For all other providers, we use the new method
    return await walletActions.signData(stakingAddress, payload);
}

/**
 * Establish trust between a user's wallet and the backend
 * @param key The wallet provider to use
 * @returns WalletCompleteResult or null
 */
export const validateWallet = async (key: string): Promise<WalletComplete | null> => {
    let result: WalletComplete | null = null;
    try {
        // Connect to wallet extension
        const walletActions = await connectWallet(key);
        if (!walletActions) return null;
        // Check if wallet is mainnet or testnet
        const network = await walletActions.getNetworkId();
        if (network !== Network.Mainnet) throw new Error('Wallet is not on mainnet');
        // Find wallet address
        const stakingAddresses = await walletActions.getRewardAddresses();
        if (!stakingAddresses || stakingAddresses.length === 0) throw new Error('Could not find staking address');
        // Request payload from backend
        const payload = await walletInit(stakingAddresses[0]);
        if (!payload) return null;
        // Sign payload with wallet
        const signedPayload = await signPayload(key, walletActions, stakingAddresses[0], payload);
        if (!signedPayload) return null;
        // Send signed payload to backend for verification
        result = (await walletComplete(stakingAddresses[0], signedPayload));
    } catch (error: any) {
        console.error('Caught error completing wallet validation', error);
        PubSub.get().publishAlertDialog({
            messageKey: 'WalletErrorUnknown',
            buttons: [{ labelKey: 'Ok' }],
        });
    } finally {
        return result;
    }
}