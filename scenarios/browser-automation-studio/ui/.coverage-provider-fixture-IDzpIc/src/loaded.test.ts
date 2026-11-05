import { it, expect } from "/home/matthalloran8/Vrooli/scenarios/browser-automation-studio/ui/node_modules/vitest/dist/index.js";
import { loaded } from './loaded';
it('executes the selected module', () => expect(loaded()).toBe(1));
