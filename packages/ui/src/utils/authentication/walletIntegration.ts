/**
 * Handles wallet integration
 * See CIP-0030 for more info: https://github.com/cardano-foundation/CIPs/pull/148
 */
import { walletComplete_walletComplete as WalletCompleteResult } from 'graphql/generated/walletComplete';
import { walletInitMutation, walletCompleteMutation } from 'graphql/mutation';
import { errorToMessage } from 'graphql/utils/errorParser';
import { initializeApollo } from 'graphql/utils/initialize';
import { ApolloError } from 'types';
import { Pubs } from 'utils';

export enum WalletProvider {
    CCVault,
    Nami,
    // Yoroi, // Doesn't work yet
}

/**
 * [Enum, window.cardano key, Label, Download extension URL]
 */
const walletProviderInfoMap: { [x: string]: [WalletProvider, string, string, string] } = {
    [WalletProvider.CCVault]: [WalletProvider.CCVault, 'ccvault', 'Eternl (CCVault.io)', 'https://chrome.google.com/webstore/detail/ccvaultio/kmhcihpebfmpgmihbkipmjlmmioameka'],
    [WalletProvider.Nami]: [WalletProvider.Nami, 'nami', 'Nami', 'https://chrome.google.com/webstore/detail/nami/lpfcbjknijpeeillifnkikgncikgfhdo'],
    // [WalletProvider.Yoroi]: [WalletProvider.Yoroi, 'yoroi', 'Yoroi', 'https://chrome.google.com/webstore/detail/yoroi/ffnbelfdoeiohenkjibnmadjiehjhajb'],
}

export const walletProviderInfo = Object.values(walletProviderInfoMap).map(o => ({
    enum: o[0],
    key: o[1],
    label: o[2],
    extensionUrl: o[3],
}))



/**
 * Maps network names to their ids
 */
const Network = {
    Mainnet: 1,
    Testnet: 0
}

export const hasWalletExtension = (provider: WalletProvider) => Boolean(window.cardano && window.cardano[walletProviderInfo[provider].key]);

/**
 * Connects to wallet provider
 * @param provider The wallet provider to connect to
 * @returns Object containing methods to interact with the wallet provider
 */
const connectWallet = async (provider: WalletProvider): Promise<any> => {
    if (!hasWalletExtension(provider)) return false;
    return await window.cardano[walletProviderInfo[provider].key].enable();
}

// Initiate handshake to verify wallet with backend
// Returns hex string of payload, to be signed by wallet
const walletInit = async (stakingAddress: string): Promise<any> => {
    PubSub.publish(Pubs.Loading, 500);
    const client = initializeApollo();
    const data = await client.mutate({
        mutation: walletInitMutation,
        variables: { input: { stakingAddress } }
    }).catch((error: ApolloError) => {
        PubSub.publish(Pubs.Snack, { message: errorToMessage(error), severity: 'error', data: error });
    })
    PubSub.publish(Pubs.Loading, false);
    return data?.data?.walletInit;
}

/**
 * Completes handshake to verify wallet with backend
 * @param stakingAddress Wallet's staking address
 * @param signedPayload Message signed by wallet
 * @returns Session object if successful, null if not
 */
const walletComplete = async (stakingAddress: string, signedPayload: string): Promise<WalletCompleteResult | null> => {
    PubSub.publish(Pubs.Loading, 500);
    const client = initializeApollo();
    const data = await client.mutate({
        mutation: walletCompleteMutation,
        variables: { input: { stakingAddress, signedPayload } }
    }).catch((error: ApolloError) => {
        PubSub.publish(Pubs.Snack, { message: errorToMessage(error), severity: 'error', data: error });
    })
    PubSub.publish(Pubs.Loading, false);
    return data?.data?.walletComplete;
}

// Signs payload received from walletInit
const signPayload = async (provider: WalletProvider, walletActions: any, stakingAddress: string, payload: string): Promise<any> => {
    if (!hasWalletExtension(provider)) return null;
    // As of 2022-02-05, new Nami endpoint is not fully working. So the old method is used for now
    if (provider === WalletProvider.Nami)
        return await window.cardano.signData(stakingAddress, payload);
    // For all other providers, we use the new method
    return await walletActions.signData(stakingAddress, payload);
}

/**
 * Establish trust between a user's wallet and the backend
 * @returns WalletCompleteResult or null
 */
export const validateWallet = async (provider: WalletProvider): Promise<WalletCompleteResult | null> => {
    let result: WalletCompleteResult | null = null;
    try {
        // Connect to wallet extension
        const walletActions = await connectWallet(provider);
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
        const signedPayload = await signPayload(provider, walletActions, stakingAddresses[0], payload);
        if (!signedPayload) return null;
        // Send signed payload to backend for verification
        result = (await walletComplete(stakingAddresses[0], signedPayload));
    } catch (error: any) {
        console.error('Caught error completing wallet validation', error);
        PubSub.publish(Pubs.AlertDialog, {
            message: 'Unknown error occurred. Please check that the extension you chose is connected to a DApp-enabled wallet',
            buttons: [{ text: 'OK' }]
        });
    } finally {
        return result;
    }
}