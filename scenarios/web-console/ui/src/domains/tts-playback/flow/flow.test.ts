import { runFormalReplay } from "./generated/replay.helper";
import { transitionPlaybackTransport } from "./transition";
import { playbackTransportFormalFixtures } from "./fixtures";

runFormalReplay({
  transition: transitionPlaybackTransport,
  fixtures: playbackTransportFormalFixtures,
});
