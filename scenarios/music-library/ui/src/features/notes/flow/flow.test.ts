import { runFormalReplay } from "./generated/replay.helper";
import { transitionAttachmentUpload } from "./transition";
import { attachmentUploadFormalFixtures } from "./fixtures";

runFormalReplay({
  transition: transitionAttachmentUpload,
  fixtures: attachmentUploadFormalFixtures,
});
