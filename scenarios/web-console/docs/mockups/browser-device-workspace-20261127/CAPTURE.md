# Current Aquila Browser view

**Capture ID** `aquila-browser-finished-20261127`  
**Fidelity** Captured pixels from the served Aquila build with a synthetic session and a finished-run response.  
**Source** Web Console `/`, served JavaScript `index-_LlM1RN_.js`, Experience build `sha256:cfd0be6f092ee81771d567c5ea04df1a8e72d1f4595a3c55e52150f0777fe99e` on 2026-11-27 ET.

| Viewport | Dark preference | Light preference |
| --- | --- | --- |
| Desktop, 1440 × 900 | [PNG](current-desktop-dark.png) | [PNG](current-desktop-light.png) |
| Phone, 390 × 844 | [PNG](current-phone-dark.png) | [PNG](current-phone-light.png) |

Each fresh Chromium context loaded the same HTML, JavaScript and CSS the running app serves. Local Playwright intercepted API and WebSocket requests before they reached a backend and served one synthetic session named **Example research session**. This isolated capture used Playwright because the standard UX capture command does not expose per-request API fixtures. The capture clicked the real globe control. Aquila then queried `POST /api/v1/browser/pane/sessions` with the fixture session ID and rendered the reply `{"sessions":[],"run_finished":true}`. This verifies the served UI state. Browser provisioning, device control and reconnection behavior remain unverified by this capture.

The heading reads **The agent run has finished**. The explanation reads “This run is no longer active, so its previous browser session cannot be reconnected.” **Close** is the state card's only action. The pane also exposes the view switches for Messages and Terminal. The owner and finished status appear below the Browser header and again in the card. At 390 px, the terminal command composer and key bar still occupy the bottom of the Browser view.

All four captures rendered the finished-run heading after the discovery call. The document width matched the viewport in each. Aquila's root reports `color-scheme: dark` for both browser preferences. Some component accents respond to the preference while the app canvas remains dark.
