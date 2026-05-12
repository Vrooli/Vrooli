import { runFormalReplay } from "./generated/attachmentupload/replay.helper";
import { transitionAttachmentUpload } from "./AttachmentUploadWorkflow";
import { attachmentUploadFormalFixtures } from "./AttachmentUploadWorkflow.fixtures";

runFormalReplay({
  transition: transitionAttachmentUpload,
  fixtures: attachmentUploadFormalFixtures,
});
