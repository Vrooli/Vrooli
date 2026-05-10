module ApiAttachmentUploadWorkflow {
  type Status = Received | BytesStored | MetadataRecorded | Failed
  type Event = StoreBytes | RecordMetadata | Fail

  var status: Status
  var event: Event
  var rejected: bool

  pure def isValid(s: Status, e: Event): bool =
    (s == Received and (e == StoreBytes or e == Fail)) or
    (s == BytesStored and (e == RecordMetadata or e == Fail))

  pure def nextStatus(s: Status, e: Event): Status =
    if (s == Received and e == StoreBytes) BytesStored
    else if (s == Received and e == Fail) Failed
    else if (s == BytesStored and e == RecordMetadata) MetadataRecorded
    else if (s == BytesStored and e == Fail) Failed
    else s

  action init = all {
    status' = Received,
    event' = StoreBytes,
    rejected' = false,
  }

  action apply(e: Event): bool = all {
    event' = e,
    status' = nextStatus(status, e),
    rejected' = not(isValid(status, e)),
  }

  action step = any {
    apply(StoreBytes),
    apply(RecordMetadata),
    apply(Fail),
  }

  val TypeOK = status.in(Set(Received, BytesStored, MetadataRecorded, Failed)) and
    event.in(Set(StoreBytes, RecordMetadata, Fail))

  run transitionTable = {
    assert(nextStatus(Received, StoreBytes) == BytesStored)
      .then(assert(not(isValid(Received, RecordMetadata))))
      .then(assert(nextStatus(Received, Fail) == Failed))
      .then(assert(not(isValid(BytesStored, StoreBytes))))
      .then(assert(nextStatus(BytesStored, RecordMetadata) == MetadataRecorded))
      .then(assert(nextStatus(BytesStored, Fail) == Failed))
      .then(assert(not(isValid(MetadataRecorded, StoreBytes))))
      .then(assert(not(isValid(MetadataRecorded, RecordMetadata))))
      .then(assert(not(isValid(MetadataRecorded, Fail))))
      .then(assert(not(isValid(Failed, StoreBytes))))
      .then(assert(not(isValid(Failed, RecordMetadata))))
      .then(assert(not(isValid(Failed, Fail))))
  }
}
