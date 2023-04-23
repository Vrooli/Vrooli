import * as Serialization from "@emurgo/cardano-serialization-lib-nodejs";
import { randomBytes } from "crypto";
import { logger } from "../events/logger";
import * as MessageSigning from "./message_signing/rust/pkg/emurgo_message_signing";
export const serializedAddressToBech32 = (address) => {
    const addressBytes = Serialization.Address.from_bytes(Buffer.from(address, "hex"));
    return addressBytes.to_bech32();
};
export function randomString(length = 64, chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789") {
    if (length <= 0 || length > 2048)
        throw new Error("Length must be bewteen 1 and 2048.");
    const charsLength = chars.length;
    if (charsLength < 10 || chars.length > 256)
        throw new Error("Chars must be bewteen 10 and 256.");
    const bytes = randomBytes(length);
    const result = new Array(length);
    let cursor = 0;
    for (let i = 0; i < length; i++) {
        cursor += bytes[i];
        result[i] = chars[cursor % charsLength];
    }
    return result.join("");
}
export const generateNonce = async (description = "Please sign this message so we can verify your wallet:", length = 64) => {
    if (length <= 0 || length > 2048)
        throw new Error("Length must be bewteen 1 and 2048");
    const payload = randomString(length);
    return Buffer.from(`${description} ${payload}`).toString("hex");
};
export const verifySignedMessage = (address, payload, coseSign1Hex) => {
    const coseSign1 = MessageSigning.COSESign1.from_bytes(Buffer.from(coseSign1Hex, "hex"));
    const payloadCose = coseSign1.payload();
    if (!payloadCose || !verifyPayload(payload, payloadCose)) {
        throw new Error("Payload does not match");
    }
    const protectedHeaders = coseSign1
        .headers()
        .protected()
        .deserialized_headers();
    const headerCBORBytes = protectedHeaders.header(MessageSigning.Label.new_text("address"))?.as_bytes();
    if (!headerCBORBytes) {
        throw new Error("Failed to convert header to bytes");
    }
    const keyId = protectedHeaders.key_id();
    if (!keyId) {
        throw new Error("Failed to get keyId from header");
    }
    const addressCose = Serialization.Address.from_bytes(headerCBORBytes);
    const publicKeyCose = Serialization.PublicKey.from_bytes(keyId);
    if (!verifyAddress(address, addressCose, publicKeyCose))
        throw new Error("Could not verify because of address mismatch");
    const signature = Serialization.Ed25519Signature.from_bytes(coseSign1.signature());
    const data = coseSign1.signed_data().to_bytes();
    return publicKeyCose.verify(data, signature);
};
const verifyPayload = (payload, payloadCose) => {
    return Buffer.from(payloadCose).compare(Buffer.from(payload, "hex")) === 0;
};
const verifyAddress = (address, addressCose, publicKeyCose) => {
    const checkAddress = Serialization.Address.from_bytes(Buffer.from(address, "hex"));
    if (addressCose.to_bech32() !== checkAddress.to_bech32())
        return false;
    try {
        const stakeKeyHash = publicKeyCose.hash();
        const reconstructedAddress = Serialization.RewardAddress.new(checkAddress.network_id(), Serialization.StakeCredential.from_keyhash(stakeKeyHash));
        if (checkAddress.to_bech32() !== reconstructedAddress.to_address().to_bech32())
            return false;
        return true;
    }
    catch (error) {
        logger.error("Caught error validating wallet address", { trace: "0005", error });
    }
    return false;
};
//# sourceMappingURL=wallet.js.map