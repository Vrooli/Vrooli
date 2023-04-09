export const FocusMode_list = `fragment FocusMode_list on FocusMode {
labels {
    ...Label_list
}
schedule {
    ...Schedule_common
}
id
name
description
}`;