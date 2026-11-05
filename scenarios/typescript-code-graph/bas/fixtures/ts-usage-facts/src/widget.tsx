import formatLabel, { useFeatureFlag, type ReactNode } from "./lib";
import * as Client from "./gen/client";
import { createConnectClient } from "./gen/client";

/** @vrooliWidget kind=card */
export function UsageCard({ child }: { child: ReactNode }) {
  const client = createConnectClient();
  const response = client.submit({ id: formatLabel("abc") });
  const enabled = useFeatureFlag("usage");
  const namespaceResponse = Client.createConnectClient().submit({ id: "namespace" });
  return <ResultView enabled={enabled} ok={response.ok || namespaceResponse.ok}>{child}</ResultView>;
}

/** JSDoc is retained on declarations. */
export function ResultView(props: { enabled: boolean; ok: boolean; children: ReactNode }) {
  return <section data-enabled={props.enabled}>{props.children}</section>;
}
