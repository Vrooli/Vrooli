import { type Run } from "../../api/types.js";
import { type PassableLogger } from "../../consts/commonTypes.js";
import { InputGenerationStrategy, PathSelectionStrategy, SubroutineExecutionStrategy } from "../../run/enums.js";
import { type RunProgress } from "../../run/types.js";
import { LATEST_CONFIG_VERSION, parseObject, stringifyObject, type StringifyMode } from "./utils.js";

const DEFAULT_STRINGIFY_MODE: StringifyMode = "json";

/**
 * Represents all run data required for resuming a run exactly where it left off.
 * 
 * NOTE: Should be a partial shape of `RunProgress`.
 */
type RunProgressConfigObject = {
    /** Store the version number for future compatibility */
    __version: RunProgress["__version"];
    /** Information about each branch (to pick up where you left off) */
    branches: RunProgress["branches"];
    /** 
     * The run configuration, including limits and settings.
     * `isPrivate` is stored in its own field in the database to make it queryable, 
     * so it's not included here.
     */
    config: Omit<RunProgress["config"], "isPrivate">;
    /** Decisions that have been deferred or resolved */
    decisions: RunProgress["decisions"];
    /** Metrics which aren't stored directly in the run table */
    metrics: Pick<RunProgress["metrics"], "creditsSpent">;
    /** Context collected for each subroutine instance */
    subcontexts: RunProgress["subcontexts"];
}

/**
 * Top-level RunProgress config that encapsulates all run-related data.
 * 
 * This is analogous to `RoutineVersionConfig`, but for runs.
 */
export class RunProgressConfig {
    __version: RunProgressConfigObject["__version"];
    branches: RunProgressConfigObject["branches"];
    config: RunProgressConfigObject["config"];
    decisions: RunProgressConfigObject["decisions"];
    metrics: RunProgressConfigObject["metrics"];
    subcontexts: RunProgressConfigObject["subcontexts"];

    constructor(data: RunProgressConfigObject) {
        this.__version = data.__version ?? LATEST_CONFIG_VERSION;
        this.branches = data.branches ?? RunProgressConfig.defaultBranches();
        this.config = data.config ?? RunProgressConfig.defaultRunConfig();
        this.decisions = data.decisions ?? [];
        this.metrics = data.metrics ?? RunProgressConfig.defaultMetrics();
        this.subcontexts = data.subcontexts ?? RunProgressConfig.defaultSubcontexts();
    }

    static parse(
        { data }: Pick<Run, "data">,
        logger: PassableLogger, // or whichever logger you use
        { mode = DEFAULT_STRINGIFY_MODE }: { mode?: StringifyMode } = {},
    ): RunProgressConfig {
        const obj = data ? parseObject<RunProgressConfigObject>(data, mode, logger) : null;
        if (!obj) {
            return RunProgressConfig.default();
        }
        return new RunProgressConfig(obj);
    }

    static default(): RunProgressConfig {
        return new RunProgressConfig({
            __version: LATEST_CONFIG_VERSION,
            branches: RunProgressConfig.defaultBranches(),
            config: RunProgressConfig.defaultRunConfig(),
            decisions: RunProgressConfig.defaultDecisions(),
            metrics: RunProgressConfig.defaultMetrics(),
            subcontexts: RunProgressConfig.defaultSubcontexts(),
        });
    }

    serialize(mode: StringifyMode): string {
        return stringifyObject(this.export(), mode);
    }

    export(): RunProgressConfigObject {
        // Include everything but `isPrivate` in the config
        const configCopy = { ...this.config };
        delete (configCopy as Partial<RunProgress["config"]>).isPrivate;
        // Include only `creditsSpent` in the metrics
        const metricsCopy = { creditsSpent: this.metrics.creditsSpent };
        return {
            __version: this.__version,
            branches: this.branches,
            config: configCopy,
            decisions: this.decisions,
            metrics: metricsCopy,
            subcontexts: this.subcontexts,
        };
    }

    static defaultBranches(): RunProgress["branches"] {
        return [];
    }

    static defaultRunConfig(): RunProgress["config"] {
        return {
            botConfig: {},
            decisionConfig: {
                inputGeneration: InputGenerationStrategy.Auto,
                pathSelection: PathSelectionStrategy.AutoPickFirst,
                subroutineExecution: SubroutineExecutionStrategy.Auto,
            },
            isPrivate: true,
            limits: {},
            loopConfig: {},
            onBranchFailure: "Stop",
            onGatewayForkFailure: "Fail",
            onNormalNodeFailure: "Fail",
            onOnlyWaitingBranches: "Continue",
            testMode: false,
        };
    }

    static defaultDecisions(): RunProgress["decisions"] {
        return [];
    }

    static defaultMetrics(): RunProgress["metrics"] {
        return {
            complexityCompleted: 0,
            complexityTotal: 0,
            creditsSpent: BigInt(0).toString(),
            startedAt: null,
            stepsRun: 0,
            timeElapsed: 0,
        };
    }

    static defaultSubcontexts(): RunProgress["subcontexts"] {
        return {};
    }
}
