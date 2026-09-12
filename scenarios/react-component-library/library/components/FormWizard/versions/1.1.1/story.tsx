// Preview contract specimens for FormWizard 1.1.0.
import { FormWizard } from "./FormWizard";

const twoSteps = [
  { id: "connection", title: "Connection", content: <p>Where the machine lives.</p> },
  { id: "posture", title: "Posture", content: <p>What gets installed on it.</p> },
];

/** The footer is built from Button, so it has variant, gap and a real height. */
export function Default() {
  return (
    <div style={{ inlineSize: "34rem", maxInlineSize: "100%" }}>
      <FormWizard
        testId="story-wizard"
        steps={twoSteps}
        nextTestId="story-next"
        previousTestId="story-back"
      />
    </div>
  );
}

/** One step is a form, not a wizard: no footer, and no step strip either. */
export function SingleStep() {
  return (
    <div style={{ inlineSize: "34rem", maxInlineSize: "100%" }}>
      <FormWizard
        testId="story-single"
        steps={[
          { id: "only", title: "Connection details", content: <p>Address, login, password.</p> },
        ]}
      />
    </div>
  );
}

/** A form whose primary action is its own keeps the arrangement, not the verbs. */
export function OwnPrimary() {
  return (
    <div style={{ inlineSize: "34rem", maxInlineSize: "100%" }}>
      <FormWizard
        testId="story-own"
        steps={[
          { id: "only", title: "Connection details", content: <p>Address, login, password.</p> },
        ]}
        showFooter
        showPrevious={false}
        footerNote="Enter an address and a login"
        footerActions={
          <button type="button" data-testid="story-own-primary">
            Start onboarding
          </button>
        }
      />
    </div>
  );
}
