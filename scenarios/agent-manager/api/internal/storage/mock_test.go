package storage

import (
	"context"
	"errors"
	"mime/multipart"
	"net/textproto"
	"testing"
)

func TestMockServiceImplementsAttachmentLifecycleAndTracksCalls(t *testing.T) {
	service := NewMockService()
	if service.MaxFileSize() != defaultMaxFileSize || !service.IsAllowedType("image/png") || service.IsAllowedType("application/x-unknown") {
		t.Fatalf("unexpected mock defaults: max=%d", service.MaxFileSize())
	}
	header := &multipart.FileHeader{Filename: "image.png", Size: 42, Header: textproto.MIMEHeader{"Content-Type": []string{"image/png"}}}
	meta, err := service.Upload(context.Background(), nil, header)
	if err != nil || meta.ID != "mock-1" || meta.StoragePath != "mock/image.png" {
		t.Fatalf("upload meta=%+v err=%v", meta, err)
	}
	got, err := service.Get(context.Background(), meta.ID)
	if err != nil || got != meta || service.GetCalls != 1 {
		t.Fatalf("get meta=%+v err=%v calls=%d", got, err, service.GetCalls)
	}
	multiple, err := service.GetMultiple(context.Background(), []string{meta.ID, "missing"})
	if err != nil || len(multiple) != 1 || multiple[0] != meta || service.GetMultipleCalls != 1 {
		t.Fatalf("multiple=%+v err=%v", multiple, err)
	}
	if got := service.GetFilePath(meta.StoragePath); got != "/mock-storage/mock/image.png" || service.GetFilePathCalls != 1 {
		t.Fatalf("file path=%q", got)
	}
	if got := service.GetServingURL(meta.StoragePath); got != "/api/v1/uploads/mock/image.png" || service.GetServingCalls != 1 {
		t.Fatalf("serving url=%q", got)
	}
	if err := service.Delete(context.Background(), meta.ID); err != nil || service.DeleteCalls != 1 {
		t.Fatalf("delete err=%v calls=%d", err, service.DeleteCalls)
	}
	if _, err := service.Get(context.Background(), meta.ID); err == nil {
		t.Fatal("deleted attachment was still returned")
	}
	if err := service.Delete(context.Background(), meta.ID); err == nil {
		t.Fatal("missing attachment delete succeeded")
	}
}

func TestMockServiceUploadFailureDoesNotPersistAttachment(t *testing.T) {
	service := NewMockService()
	service.UploadErr = errors.New("injected upload failure")
	meta, err := service.Upload(context.Background(), nil, &multipart.FileHeader{Filename: "nope.txt"})
	if !errors.Is(err, service.UploadErr) || meta != nil || service.UploadCalls != 1 {
		t.Fatalf("upload meta=%+v err=%v calls=%d", meta, err, service.UploadCalls)
	}
	service.SetFile("seed", &AttachmentMeta{ID: "seed"})
	if got, err := service.Get(context.Background(), "seed"); err != nil || got.ID != "seed" {
		t.Fatalf("seed=%+v err=%v", got, err)
	}
}
