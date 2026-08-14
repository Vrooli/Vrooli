# Money Ledger information architecture

| Route | Surface | Desktop | Mobile |
|---|---|---|---|
| `/` | Position dashboard | sidebar + runway/qualification grid | priority figures first; bottom navigation remains pinned |
| `/accounts` | Books and accounts | table and creation panel | stacked forms and list rows |
| `/journal` | Immutable postings | comparison table with basis and provenance | horizontally scrollable rows; reversal action remains explicit |
| `/adapters` | Sources and receipts | availability table and import controls | stacked source cards with reason and age |
| `/statements` | Read-time statements | income/expense and assets/liabilities side by side | sequential statement cards |
| `/settings` | Theme and locale | settings form | stacked controls |

The sidebar collapses to the shared safe-area-aware bottom navigation. Financial figures use tabular numerals, direction is stated with words/signs rather than color alone, and basis sits beside every figure. The journal has no edit or delete affordance.
