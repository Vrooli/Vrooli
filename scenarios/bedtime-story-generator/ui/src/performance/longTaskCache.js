const HISTORY_LIMIT = 120;

const history = [];

export const pushLongTasks = (entries = []) => {
  if (!entries.length) return;
  history.push(...entries);
  if (history.length > HISTORY_LIMIT) {
    history.splice(0, history.length - HISTORY_LIMIT);
  }
};

export const getLongTaskHistory = () => history.slice();
export const resetLongTaskHistory = () => {
  history.length = 0;
};
