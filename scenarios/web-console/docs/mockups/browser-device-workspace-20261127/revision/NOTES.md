# Aquila Browser & devices design revision

[Review the options](index.html) · [Open the prototype](prototype.html)

This revision supersedes the first presentation. The alternatives change the first screen only. **A** opens the last target, with a default browser the first time. **B** opens the target list. Both use the same viewer, target switcher and control handoff. The conversation shortcut is available in both.

The proposed screens are screenshots of a standalone interactive prototype. The example storefront, devices, browser state and agent actions are simulated. The [current UI evidence](../CAPTURE.md) is a separate capture from the served application with a synthetic finished-run response. Production work would need a persistent browser target owned by Browser Automation Studio, supported device and emulator targets from Device Control, and a confirmed exclusive input handoff for authorized agents and people. Browser readiness and session restoration are product requirements represented here by the prototype.

## Verification

Chromium ran both entry routes at 1440 × 900, 390 × 844 and 320 × 720 in light and dark. The interactive journey shared the browser to Codex, took control, opened a product, and added it to the mock shopping bag. After returning control and ending the example run, the product page and bag remained intact when a second agent attached. The checks also covered switching to the emulator, rotation, the offline physical device, creating a fresh browser and browser state restoration after a reload. The screenshots show browser viewports.

## Sample storefront image

[Studio pottery](studio-pottery.webp) was generated using the built-in ImageGen tool and resized to WebP for the prototype.

> Generate one high-end editorial product photograph, a wide 3:2 landscape crop. This is content inside a fictional small ecommerce storefront in an Aquila browser and phone design prototype. A small close composition of three pieces of studio pottery on a matte warm parchment pedestal: a sculptural cream-speckled ceramic pour-over cone nested on a coffee mug, both authentic gently irregular, and an asymmetric small dark ochre clay bowl alongside. Strong soft fall afternoon sunlight from upper right makes a long soft shadow falling toward lower left. Background is muted warm umber/beige matte plaster with subtle real texture. Elevated craft catalog studio lighting, realistic ceramic clay texture, restrained warm earth color, no people, no plants, no words or type, no logos, no watermarks, no added borders. Main objects grouped around center-right within central 60% so the same photograph looks good cropped to 4:3.
