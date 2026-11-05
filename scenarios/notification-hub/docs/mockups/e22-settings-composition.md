# E22 Settings composition reference

These selected originals define the focused editing and recovery composition. They are
the visual reference; implementation changes should preserve the app shell and present the
active editor as the page's main content. Returning to Settings restores the broader page.

| Desktop editor | Phone editor |
| --- | --- |
| ![Quiet Monday editor, dark desktop](../../../../docs/internal/delivery-operations/2026-10-12/assets/notification-hub-revised/originals/quiet-monday-editor-dark-desktop-p1-r3.png) | ![Quiet Monday editor, dark phone](../../../../docs/internal/delivery-operations/2026-10-12/assets/notification-hub-revised/originals/quiet-monday-editor-dark-phone-p1-r3.png) |
| ![Order channel editor, dark desktop](../../../../docs/internal/delivery-operations/2026-10-12/assets/notification-hub-revised/originals/order-channel-editor-dark-desktop-p1-r3.png) | ![Order channel editor, dark phone](../../../../docs/internal/delivery-operations/2026-10-12/assets/notification-hub-revised/originals/order-channel-editor-dark-phone-p1-r3.png) |

| Desktop staged review | Phone uncertain-save recovery |
| --- | --- |
| ![Five-day quiet hours review, light desktop](../../../../docs/internal/delivery-operations/2026-10-12/assets/notification-hub-revised/originals/quiet-five-day-review-light-desktop-p1-r3.png) | ![Unknown order save, light phone](../../../../docs/internal/delivery-operations/2026-10-12/assets/notification-hub-revised/originals/order-save-unknown-light-phone-p1-r3.png) |

## Composition contract

- The shared app shell remains visible. In the Settings content area, show either the
  broader Settings page or the active quiet-hours/order flow, not both at once.
- Opening a weekday, reviewing changes, showing a save outcome, or editing escalation
  order focuses that flow. A clear back action returns to Settings.
- Keep both editors mounted when the other page sections are hidden. A back action must
  not discard an unconfirmed draft or erase a persistence outcome.
- Keep the responsive content width, spacing and controls inside the existing design
  system. Phone screens prioritize the active task and keep actions reachable.
- The review/result copy must continue to distinguish saved, not applied, partial,
  uncertain and unknown persistence. NH-DEC-26 eligibility remains held.

