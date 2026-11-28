import "../create.css";
import { useState } from "react";
import { useNavigate } from "react-router-dom";
import { CreateWorkflowLayout } from "./CreateWorkflowLayout";
import { WbButton } from "../primitives";

import { useTranslation } from "../../i18n";
import { appRoutes } from "../../routes";
type Route = "create" | "extract" | "adopt";

export function CreatePage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [route, setRoute] = useState<Route>("create");
  const routes: Array<{ id: Route; index: string; title: string; body: string }> = [
    {
      id: "create",
      index: "01 · CREATE",
      title: t("workbench.create.createTitle", { defaultValue: "Create a component" }),
      body: t("workbench.create.createBody", {
        defaultValue: "Start from catalog intent and a new versioned source draft.",
      }),
    },
    {
      id: "extract",
      index: "02 · EXTRACT",
      title: t("workbench.create.extractTitle", { defaultValue: "Extract from a scenario" }),
      body: t("workbench.create.extractBody", {
        defaultValue: "Harvest a real component and keep its source provenance.",
      }),
    },
    {
      id: "adopt",
      index: "03 · ADOPT",
      title: t("workbench.create.adoptTitle", { defaultValue: "Adopt into a scenario" }),
      body: t("workbench.create.adoptBody", {
        defaultValue: "Pin a released asset and synchronize its tokens and styles.",
      }),
    },
  ];

  const active = routes.find((entry) => entry.id === route) ?? routes[0];
  if (!active) return null;

  return (
    <div className="wb-page" data-testid="create-page">
      <div className="wb-pagehead">
        <div className="wb-pagehead-copy">
          <p className="wb-kicker">
            {t("workbench.create.kicker", { defaultValue: "Library · New asset" })}
          </p>
          <h1 className="wb-h1">
            {t("workbench.create.title", { defaultValue: "Add to the library" })}
          </h1>
          <p className="wb-lede">
            {t("workbench.create.lede", {
              defaultValue:
                "Three governed routes into the same lifecycle. Every route ends in a draft, focused checks and an immutable successor.",
            })}
          </p>
        </div>
        <div className="wb-actions">
          <WbButton variant="ghost" size="md" onClick={() => void navigate(appRoutes.components)}>
            {t("workbench.create.cancel", { defaultValue: "Cancel" })}
          </WbButton>
        </div>
      </div>

      <div className="wb-create-tabs">
        {routes.map((entry) => (
          <button
            key={entry.id}
            type="button"
            className="wb-create-tab"
            aria-pressed={route === entry.id}
            onClick={() => setRoute(entry.id)}
          >
            <small>{entry.index}</small>
            <b>{entry.title}</b>
            <span>{entry.body}</span>
          </button>
        ))}
      </div>

      <CreateWorkflowLayout route={route} title={active.title} />
    </div>
  );
}
