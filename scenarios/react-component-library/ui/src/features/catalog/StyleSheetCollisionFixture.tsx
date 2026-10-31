import { MessageList as MessageListV100 } from "@vrooli/react-component-library/MessageList/1.0.0";
import { MessageList as MessageListV111 } from "@vrooli/react-component-library/MessageList/1.1.1";

/** A deliberately two-version page for the page-global stylesheet contract. */
export function StyleSheetCollisionFixture() {
  return (
    <main data-testid="stylesheet-collision-fixture">
      <section data-testid="stylesheet-collision-v100">
        <MessageListV100 state="empty" height={180} />
      </section>
      <section data-testid="stylesheet-collision-v111">
        <MessageListV111 state="empty" height={180} />
      </section>
    </main>
  );
}
