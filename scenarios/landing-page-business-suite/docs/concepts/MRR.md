# Monthly recurring revenue

LPBS MRR is the sum of the current recurring price for every active subscription
and every trialing subscription with a linked customer/payment identity. Monthly
prices contribute their full amount; annual prices contribute one twelfth.
Past-due, one-time, canceled, and trials without a linked customer/payment
identity do not contribute.

The producer computes this from the stored Stripe price snapshot. Introductory
prices are used when present, then effective discount metadata is applied and
clamped at zero. The rollup is tenant-wide and assumes one settlement currency
per deployment. Values are returned in minor currency units with an explicit ISO
currency code. `sample_size` is the number of subscriptions included, and
`observed_at` is the producer observation time.

The same projection reports successful checkout revenue for today and the last
30 days, cancellations in the last 30 days, credit balances and burn, usage
records, and trials without payment method. Empty tenants return zero-valued
figures with a valid observation rather than a fabricated sample.
