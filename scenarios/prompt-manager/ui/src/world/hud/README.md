# hud

DOM chrome over (or instead of) the canvas: summary strip, agent card, team
panel, event ticker, filters, settings, 2D mode and the editor toolbar. Reads
only `selectView` output from the sim and calls only `data` actions.

Import rule: `sim`, `config`, `data`. Never `scene` or `engine`.
