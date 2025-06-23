import { describe } from "vitest";
import { BaseConfig } from "./base.js";
import { baseConfigFixtures } from "../../__test/fixtures/config/baseConfigFixtures.js";
import { runComprehensiveConfigTests } from "./__test/configTestUtils.js";

describe("BaseConfig", () => {
    runComprehensiveConfigTests(
        BaseConfig,
        baseConfigFixtures,
        "base",
    );
    
    // Base config doesn't have any unique functionality beyond what's tested
    // in the comprehensive tests (resource management is tested there)
});
