import {
  useViewportEnvironment,
  ViewportEnvironmentProvider,
  type ViewportEnvironmentSnapshot,
} from "./useViewportEnvironment";

function Readout() {
  const viewport = useViewportEnvironment();
  return (
    <output
      data-rcl-viewport-environment
      data-layout-width={String(viewport.layoutWidth)}
      data-visible-height={String(viewport.visibleHeight)}
      data-keyboard-visible={String(viewport.keyboardVisible)}
      data-offset-left={String(viewport.offsetLeft)}
    >
      {JSON.stringify(viewport)}
    </output>
  );
}

function Snapshot({ value }: { value: ViewportEnvironmentSnapshot }) {
  return (
    <ViewportEnvironmentProvider value={value}>
      <Readout />
    </ViewportEnvironmentProvider>
  );
}

export function Default() {
  return <Readout />;
}

export function Empty() {
  return (
    <Snapshot
      value={{
        layoutWidth: 0,
        layoutHeight: 0,
        visibleWidth: 0,
        visibleHeight: 0,
        offsetLeft: 0,
        offsetTop: 0,
        scale: 1,
        keyboardInset: 0,
        keyboardVisible: false,
        reachesScreenBottom: true,
      }}
    />
  );
}

export function KeyboardVisible() {
  return (
    <Snapshot
      value={{
        layoutWidth: 390,
        layoutHeight: 844,
        frameHeight: 844,
        visibleWidth: 390,
        visibleHeight: 500,
        offsetLeft: 0,
        offsetTop: 0,
        scale: 1,
        keyboardInset: 344,
        keyboardVisible: true,
        reachesScreenBottom: false,
      }}
    />
  );
}

export function OffsetViewport() {
  return (
    <Snapshot
      value={{
        layoutWidth: 390,
        layoutHeight: 844,
        visibleWidth: 350,
        visibleHeight: 780,
        offsetLeft: 20,
        offsetTop: 32,
        scale: 1.1,
        keyboardInset: 0,
        keyboardVisible: false,
        reachesScreenBottom: true,
      }}
    />
  );
}

export function WideViewport() {
  return (
    <Snapshot
      value={{
        layoutWidth: 1920,
        layoutHeight: 1080,
        frameHeight: 1080,
        visibleWidth: 1920,
        visibleHeight: 1080,
        offsetLeft: 0,
        offsetTop: 0,
        scale: 1,
        keyboardInset: 0,
        keyboardVisible: false,
        reachesScreenBottom: true,
      }}
    />
  );
}
