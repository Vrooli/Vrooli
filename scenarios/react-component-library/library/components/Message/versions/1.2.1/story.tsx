import { useState } from "react";
import { Message, type MessageProps } from "./Message";

export function MessageStory({ args }: StoryHarnessProps) {
  const props = args as unknown as MessageProps;
  const [retried, setRetried] = useState(false);
  return (
    <Message
      {...props}
      state={retried ? "loading" : props.state}
      onRetry={props.onRetry ? () => setRetried(true) : undefined}
    />
  );
}
