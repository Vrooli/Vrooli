import { screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { create } from "@bufbuild/protobuf";
import { JobSchema } from "@vrooli/proto-types/image-tools/v1/jobs/jobs_pb";
import { expect, it, vi } from "vitest";

import { JobState } from "../../api/jobs";
import { strings } from "../../consts/strings";
import { expectNoA11yViolations, renderWithProviders } from "../../test-utils";
import { ObjectMaskEditor } from "./ObjectMaskEditor";
import type { ObjectMaskClient } from "./useObjectMask";

vi.mock("../jobs/useJobProgress", () => ({ useJobProgress: () => null }));

it("refines a model mask with keyboard include/exclude points and exposes the real download", async () => {
  const client: ObjectMaskClient = {
    submit: vi.fn().mockResolvedValue({ jobId: "mask-job", modelId: "sam-2.1-tiny", tier: "local-cpu", warnings: [], estimatedSeconds: 2 }),
    wait: vi.fn().mockResolvedValue(create(JobSchema, { id: "mask-job", state: JobState.SUCCEEDED, resultRef: "mask/real.png" })),
    cancel: vi.fn().mockResolvedValue(undefined),
    blobUrl: (ref) => `/blob/${ref}`,
  };
  const image = new File(["source"], "device.png", { type: "image/png" });
  const { container } = renderWithProviders(<ObjectMaskEditor image={image} imageUrl="/source.png" client={client} />);
  const k = strings.select.objectMask;
  expect(screen.getByRole("button", { name: k.run })).toBeDisabled();
  await userEvent.click(screen.getByRole("button", { name: k.addPoint }));
  await userEvent.selectOptions(screen.getByRole("combobox", { name: k.pointKind }), "exclude");
  const x = screen.getByRole("spinbutton", { name: strings.select.pointX });
  await userEvent.clear(x);
  await userEvent.type(x, "10");
  await userEvent.click(screen.getByRole("button", { name: k.addPoint }));
  await userEvent.click(screen.getByRole("button", { name: k.run }));
  await waitFor(() => expect(screen.getByRole("link", { name: k.download })).toHaveAttribute("href", "/blob/mask/real.png"));
  expect(client.submit).toHaveBeenCalledWith("segment", expect.objectContaining({ segmentation: { points: [
    { x: .5, y: .5, negative: false }, { x: .1, y: .5, negative: true },
  ] } }), expect.objectContaining({ image }));
  await expectNoA11yViolations(container);
  await userEvent.click(screen.getByRole("button", { name: k.undo }));
  expect(screen.queryByRole("link", { name: k.download })).not.toBeInTheDocument();
});
