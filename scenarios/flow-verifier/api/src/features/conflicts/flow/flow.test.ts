import { runFormalReplay } from "./generated/replay.helper";
import { transitionConflictResolution } from "./transition";
import { conflictResolutionFormalFixtures } from "./fixtures";

runFormalReplay({
  transition: transitionConflictResolution,
  fixtures: conflictResolutionFormalFixtures,
});
