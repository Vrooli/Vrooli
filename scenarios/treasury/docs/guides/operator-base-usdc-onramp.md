# Operator guide: move USDC to Base and fund a Treasury wallet

This guide is for a later, explicitly approved operator session. The CF5 run remains local-fork only; its test address is not a deposit address. Do not send mainnet funds until Treasury publishes an operator-approved address through the mission's gated runbook and custody/recovery is READY.

## If you already hold USDC in Coinbase

1. Open the Coinbase app and select **Transfer → Send crypto**.
2. Select USDC, choose **Base** as the network, and select the destination Base wallet. Confirm the wallet address from the approved Treasury session through a separate trusted channel.
3. Enter the exact approved amount and review Coinbase's confirmation screen. Check that the asset is USDC and the network is Base before sending.
4. Confirm only after comparing the destination, amount, network, displayed fee, and final total. On-chain sends are irreversible. A wrong network or address can result in permanent loss.

Coinbase's current help page says transfers between Coinbase and the Base app occur on-chain and incur standard network fees; it also says an ERC-20 transfer from the Base app needs ETH for the network fee. Coinbase says its app's preview shows the transfer details before sending. [Coinbase: transfer crypto between the Base app and Coinbase](https://help.coinbase.com/en/wallet/sending-and-receiving/how-do-i-move-assets-between-my-coinbase-wallet-and-my-coinbase)

## If you need to buy USDC first

In the Base app, choose **Buy**, select **USDC on Base**, enter the amount, and review the available payment methods and final quote for your region. Coinbase says payment methods vary by country; it currently states there are no Coinbase fees on USDC buys in Coinbase Wallet, while network fees may still apply. A third-party purchase provider may be used for international purchases. Compare the final amount you pay with the amount received before confirming. [Coinbase: USDC on the Base app](https://help.coinbase.com/en/wallet/managing-account/usdc-coinbase-wallet)

Do not assume any route is always “cheaper.” Compare the displayed price, provider fees, network fees, time to settle, and amount received for the exact transaction. Do not share a recovery phrase, private key, or one-time code with support or a website.

## Send from the Base app to Treasury

After Treasury presents an approved funding request, use its Fund button or scan its QR code. Confirm the wallet app shows the same destination, Base network, USDC contract, and exact amount as the Treasury request. The deep link only pre-fills a transaction; it does not send funds. Review the wallet's live fee and total, then confirm in the wallet. Coinbase documents Base app send and receive steps and warns that network fees are paid to the network. [Coinbase: send and receive crypto on the Base app](https://help.coinbase.com/en/wallet/sending-and-receiving/how-do-i-send-and-receive-crypto-through-wallet)

Wait for Treasury to verify the on-chain transfer and Money Ledger to accept its exact token quantity. Do not treat a wallet screenshot or pending transaction as credited funds. Confirm that Treasury's receipt status, credited headroom, and Notification Hub notice agree before relying on the balance.

## Fees, timing, and limits

The Base and Coinbase apps present a live fee and transfer preview at confirmation. Fees and settlement time can change with network conditions and provider route; this guide does not state a fixed price or delivery time. Use the quote displayed for the exact transaction and retain it with the operator runbook. Treasury's withdrawal preview must show its own destination, amount, fee, and expected timing before execution.

Until the operator changes the policy, total live spend remains capped at USD 5 including gas and fees. No funding, purchase, address publication, or mainnet transaction is authorized by this guide.
