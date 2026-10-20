# Direct UI revision evidence addendum — 2026-10-19

This addendum supplements [the independent receiving review](e1-same-id-independent-receiving-review-20261019.md) without changing it. The reviewed file's immediate preimage SHA-256 is `4ecceb2f6bfab452c812e572909d96ff2ef4d694db1455c1dfd3f30c3b75505c`.

In the private corrected BAS timeline `bas-timeline-retry.json` (SHA-256 `5ff84834fdf70d206108649dd847d5d4675917f416968f3e890e127770198179`), each entry stores its rendered DOM at `entries[].aggregates.artifacts[].payload.outcome.object_value.fields.dom_snapshot.object_value.fields.html.string_value`:

- Multiday event `24381c28-02c8-45aa-93c1-2cfdf42614d8`: step 13 (`create-multiday`) shows the event at revision 1; steps 17 (`save-multiday-edit`) and 18 (`assert-multiday-revision`) show the same event ID, revised title, and `revision 2` in the DOM accessible label.
- Timed event `c988e20f-81c6-424f-a27d-87fe7ae2328a`: step 32 (`create-timed-event`) shows revision 1; steps 36 (`save-timed-edit`) and 37 (`assert-timed-revision`) show the same event ID, revised title, and `revision 2` in the DOM accessible label.

The same DOM snapshots record API event-read URLs containing each exact ID. This makes the UI revision transition direct evidence; the matching CLI/Connect revision-2 reads remain independently retained in the receiving review. No run or owner receipt was changed.
