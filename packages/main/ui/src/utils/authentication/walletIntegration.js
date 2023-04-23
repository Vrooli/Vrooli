import { authWalletComplete } from "../../api/generated/endpoints/auth_walletComplete";
import { authWalletInit } from "../../api/generated/endpoints/auth_walletInit";
import { errorToCode, initializeApollo } from "../../api/utils";
import { PubSub } from "../pubsub";
export const walletDownloadUrls = {
    "flint": ["Flint", "https://chrome.google.com/webstore/detail/flint-wallet/hnhobjmcibchnmglfbldbfabcgaknlkj"],
    "nami": ["Nami", "https://chrome.google.com/webstore/detail/nami/lpfcbjknijpeeillifnkikgncikgfhdo"],
    "nufi": ["NuFi", "https://chrome.google.com/webstore/detail/nufi/gpnihlnnodeiiaakbikldcihojploeca"],
};
const Network = {
    Mainnet: 1,
    Testnet: 0,
};
export const hasWalletExtension = (key) => Boolean(window.cardano && window.cardano[key]);
export const getInstalledWalletProviders = () => {
    if (!window.cardano) {
        return [];
    }
    const exclude = ["eternl", "gero", "cardwallet"];
    let providers = Object.entries(window.cardano).filter(([key, value]) => {
        if (typeof value !== "object")
            return false;
        const obj = value;
        if (!obj.hasOwnProperty("apiVersion"))
            return false;
        if (!obj.hasOwnProperty("enable"))
            return false;
        if (!obj.hasOwnProperty("name"))
            return false;
        if (!obj.hasOwnProperty("icon"))
            return false;
        if (!obj.hasOwnProperty("isEnabled"))
            return false;
        return true;
    });
    providers = providers.filter(([key, value], index) => {
        const currName = value.name;
        const nextName = providers.slice(index + 1).find(([, next]) => next.name === currName);
        return !nextName && !exclude.includes(key);
    });
    return providers;
};
const connectWallet = async (key) => {
    if (!hasWalletExtension(key))
        return false;
    return await window.cardano[key].enable();
};
const walletInit = async (stakingAddress) => {
    PubSub.get().publishLoading(500);
    const client = initializeApollo();
    const data = await client.mutate({
        mutation: authWalletInit,
        variables: { input: { stakingAddress } },
    }).catch((error) => {
        PubSub.get().publishSnack({ messageKey: errorToCode(error), severity: "Error", data: error });
    });
    PubSub.get().publishLoading(false);
    return data?.data?.walletInit;
};
const walletComplete = async (stakingAddress, signedPayload) => {
    PubSub.get().publishLoading(500);
    const client = initializeApollo();
    const data = await client.mutate({
        mutation: authWalletComplete,
        variables: { input: { stakingAddress, signedPayload } },
    }).catch((error) => {
        PubSub.get().publishSnack({ messageKey: errorToCode(error), severity: "Error", data: error });
    });
    PubSub.get().publishLoading(false);
    return data?.data?.walletComplete;
};
const signPayload = async (key, walletActions, stakingAddress, payload) => {
    if (!hasWalletExtension(key))
        return null;
    if (key === "nami")
        return await window.cardano.signData(stakingAddress, payload);
    return await walletActions.signData(stakingAddress, payload);
};
export const validateWallet = async (key) => {
    let result = null;
    try {
        const walletActions = await connectWallet(key);
        if (!walletActions)
            return null;
        const network = await walletActions.getNetworkId();
        if (network !== Network.Mainnet)
            throw new Error("Wallet is not on mainnet");
        const stakingAddresses = await walletActions.getRewardAddresses();
        if (!stakingAddresses || stakingAddresses.length === 0)
            throw new Error("Could not find staking address");
        const payload = await walletInit(stakingAddresses[0]);
        if (!payload)
            return null;
        const signedPayload = await signPayload(key, walletActions, stakingAddresses[0], payload);
        if (!signedPayload)
            return null;
        result = (await walletComplete(stakingAddresses[0], signedPayload));
    }
    catch (error) {
        console.error("Caught error completing wallet validation", error);
        PubSub.get().publishAlertDialog({
            messageKey: "WalletErrorUnknown",
            buttons: [{ labelKey: "Ok" }],
        });
    }
    finally {
        return result;
    }
};
//# sourceMappingURL=walletIntegration.js.map