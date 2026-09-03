import { useState } from "react";

import { Button } from "@vrooli/react-component-library/Button/2";
import { EmptyState } from "@vrooli/react-component-library/EmptyState/1";
import { FullPageDrawer } from "@vrooli/react-component-library/FullPageDrawer/1";
import { Icon } from "@vrooli/react-component-library/Icon/1";
import { Pressable } from "@vrooli/react-component-library/Pressable/1";
import { Text } from "@vrooli/react-component-library/Text/1";

/**
 * Small, durable reference surface for adopted foundation assets.
 *
 * This is deliberately part of the catalog workspace: an adoption is only
 * useful when the target scenario can render it through its own build and
 * exercise the stateful behavior that the contract describes.
 */
export function AdoptedAssetShowcase() {
  const [pending, setPending] = useState(false);
  const [drawerOpen, setDrawerOpen] = useState(false);

  return (
    <section
      data-testid="adopted-asset-showcase"
      aria-labelledby="adopted-asset-showcase-title"
      className="rounded-panel border border-app-border bg-app-surface-muted p-space-sm"
    >
      <div className="flex flex-wrap items-start justify-between gap-space-xs">
        <div>
          <Text
            as="h2"
            id="adopted-asset-showcase-title"
            textStyle="title"
            className="text-app-foreground"
          >
            Adopted foundation reference
          </Text>
          <Text as="p" textStyle="body" tone="muted" className="mt-space-3xs">
            Live scenario rendering for the shared interaction and content primitives.
          </Text>
        </div>
        <Icon name="check" label="Adopted" tone="accent" />
      </div>

      <div className="mt-space-sm flex flex-wrap items-center gap-space-2xs">
        <Button size="sm" variant="secondary" onClick={() => setDrawerOpen(true)}>
          Open drawer
        </Button>
        <Pressable
          size="sm"
          pending={pending}
          pendingLabel="Checking…"
          onClick={() => {
            setPending(true);
            window.setTimeout(() => setPending(false), 350);
          }}
        >
          Validate state
        </Pressable>
      </div>

      <div className="mt-space-sm">
        <EmptyState
          title="No preview fixtures selected"
          description="The adopted primitives remain usable when a catalog surface has no fixture data."
          icon={<Icon name="search" size="sm" aria-hidden />}
          action={
            <Button size="sm" variant="ghost" onClick={() => setDrawerOpen(true)}>
              Choose a fixture
            </Button>
          }
        />
      </div>

      <FullPageDrawer
        open={drawerOpen}
        onClose={() => setDrawerOpen(false)}
        title="Fixture drawer"
        closeLabel="Close fixture drawer"
      >
        <div>
          <Text as="p" textStyle="body" tone="muted" className="mt-space-3xs">
            This overlay is rendered from the adopted FullPageDrawer contract.
          </Text>
        </div>
      </FullPageDrawer>
    </section>
  );
}
