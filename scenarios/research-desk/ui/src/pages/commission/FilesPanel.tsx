import type { GetCommissionResponse } from "../../api/activity";
import { FileList } from "../../components/Changes";
import { PathText } from "../../components/PathText";
import { Section } from "../../components/Section";
import { DerivedWhen } from "../../components/When";
import { strings } from "../../consts/strings";
import { useTranslation } from "../../i18n";
import { toDate } from "../../lib/format";

/** The commission folder and each topic's run folder with its newest checkpoint. */
export function FilesPanel({ data }: { data: GetCommissionResponse }) {
  const { t } = useTranslation();
  return (
    <div className="flex flex-col gap-space-lg">
      <Section id="folder" title={t(strings.files.folder)}>
        {data.files.length === 0 ? <p className="text-body-sm">{t(strings.files.empty)}</p> : <FileList files={data.files} />}
      </Section>
      {data.runs.length > 0 ? (
        <Section id="runs" title={t(strings.files.runs)}>
          <ul className="rd-rows">
            {data.runs.map((run) => {
              const modified = toDate(run.lastWrite?.value);
              return (
                <li key={run.path} className="rd-row">
                  <span className="font-semibold">{run.topic}</span>
                  <PathText path={run.path} />
                  {modified ? (
                    <span className="text-caption text-app-muted-foreground">
                      {t(strings.files.runChanged)} <DerivedWhen time={run.lastWrite} />
                    </span>
                  ) : null}
                  {run.checkpoint?.path ? <FileList files={[run.checkpoint]} /> : <span className="text-body-sm text-app-muted-foreground">{t(strings.files.noCheckpoint)}</span>}
                </li>
              );
            })}
          </ul>
        </Section>
      ) : null}
    </div>
  );
}
