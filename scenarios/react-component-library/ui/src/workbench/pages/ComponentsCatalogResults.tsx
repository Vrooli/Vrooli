import { useTranslation } from "../../i18n";
import type { ComponentsCatalogViewProps as Props } from "./ComponentsCatalogView";
import { CircleAlert } from "lucide-react";
import { Banner } from "../primitives";

type ResultsProps = Pick<Props, "catalog">;

export function CatalogResults({ catalog }: ResultsProps) {
  const { t } = useTranslation();
  return (
    <>
      {catalog.isError ? (
        <div style={{ marginTop: 14 }}>
          <Banner tone="red" icon={<CircleAlert aria-hidden size={16} />}>
            <b>
              {t("workbench.components.errorTitle", {
                defaultValue: "The catalog could not be read.",
              })}
            </b>{" "}
            {t("workbench.components.errorBody", {
              defaultValue: "The API did not answer. Nothing is shown as an empty library.",
            })}
          </Banner>
        </div>
      ) : null}
    </>
  );
}
