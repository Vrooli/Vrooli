module UiAttachmentUploadWorkflow {
  type Status = Idle | Selected | Uploading | Succeeded | Failed
  type Event = Select | Start | Succeed | Fail | Reset

  var status: Status
  var event: Event
  var rejected: bool

  pure def isValid(s: Status, e: Event): bool =
    e == Select or
    e == Reset or
    (e == Start and (s == Selected or s == Failed)) or
    (e == Succeed and s == Uploading) or
    (e == Fail and s == Uploading)

  pure def nextStatus(s: Status, e: Event): Status =
    if (e == Select) Selected
    else if (e == Reset) Idle
    else if (e == Start and (s == Selected or s == Failed)) Uploading
    else if (e == Succeed and s == Uploading) Succeeded
    else if (e == Fail and s == Uploading) Failed
    else s

  action init = all {
    status' = Idle,
    event' = Reset,
    rejected' = false,
  }

  action apply(e: Event): bool = all {
    event' = e,
    status' = nextStatus(status, e),
    rejected' = not(isValid(status, e)),
  }

  action step = any {
    apply(Select),
    apply(Start),
    apply(Succeed),
    apply(Fail),
    apply(Reset),
  }

  val TypeOK = status.in(Set(Idle, Selected, Uploading, Succeeded, Failed)) and
    event.in(Set(Select, Start, Succeed, Fail, Reset))

  run transitionTable = {
    assert(nextStatus(Idle, Select) == Selected)
      .then(assert(not(isValid(Idle, Start))))
      .then(assert(not(isValid(Idle, Succeed))))
      .then(assert(not(isValid(Idle, Fail))))
      .then(assert(nextStatus(Idle, Reset) == Idle))
      .then(assert(nextStatus(Selected, Select) == Selected))
      .then(assert(nextStatus(Selected, Start) == Uploading))
      .then(assert(not(isValid(Selected, Succeed))))
      .then(assert(not(isValid(Selected, Fail))))
      .then(assert(nextStatus(Selected, Reset) == Idle))
      .then(assert(nextStatus(Uploading, Select) == Selected))
      .then(assert(not(isValid(Uploading, Start))))
      .then(assert(nextStatus(Uploading, Succeed) == Succeeded))
      .then(assert(nextStatus(Uploading, Fail) == Failed))
      .then(assert(nextStatus(Uploading, Reset) == Idle))
      .then(assert(nextStatus(Succeeded, Select) == Selected))
      .then(assert(not(isValid(Succeeded, Start))))
      .then(assert(not(isValid(Succeeded, Succeed))))
      .then(assert(not(isValid(Succeeded, Fail))))
      .then(assert(nextStatus(Succeeded, Reset) == Idle))
      .then(assert(nextStatus(Failed, Select) == Selected))
      .then(assert(nextStatus(Failed, Start) == Uploading))
      .then(assert(not(isValid(Failed, Succeed))))
      .then(assert(not(isValid(Failed, Fail))))
      .then(assert(nextStatus(Failed, Reset) == Idle))
  }
}
