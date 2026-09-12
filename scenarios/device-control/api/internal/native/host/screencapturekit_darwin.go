//go:build darwin && cgo

package host

/*
#cgo CFLAGS: -x objective-c -fobjc-arc
#cgo LDFLAGS: -framework ScreenCaptureKit -framework CoreGraphics -framework CoreMedia -framework CoreImage -framework CoreVideo -framework ImageIO -framework Foundation
#import <CoreGraphics/CoreGraphics.h>
#import <CoreImage/CoreImage.h>
#import <CoreMedia/CoreMedia.h>
#import <CoreVideo/CoreVideo.h>
#import <Foundation/Foundation.h>
#import <ImageIO/ImageIO.h>
#import <ScreenCaptureKit/ScreenCaptureKit.h>
#include <dispatch/dispatch.h>
#include <stdlib.h>
#include <string.h>

@interface VrooliScreenCaptureOutput : NSObject <SCStreamOutput>
@property(nonatomic) dispatch_semaphore_t frameSemaphore;
@property(nonatomic) NSData *pngData;
@property(nonatomic) NSError *frameError;
@end

@implementation VrooliScreenCaptureOutput
- (void)stream:(SCStream *)stream
 didOutputSampleBuffer:(CMSampleBufferRef)sampleBuffer
          ofType:(SCStreamOutputType)type {
    (void)stream;
    if (type != SCStreamOutputTypeScreen || self.pngData != nil || !CMSampleBufferIsValid(sampleBuffer)) {
        return;
    }
    CVImageBufferRef imageBuffer = CMSampleBufferGetImageBuffer(sampleBuffer);
    if (imageBuffer == NULL) {
        self.frameError = [NSError errorWithDomain:@"VrooliScreenCapture" code:1 userInfo:nil];
        dispatch_semaphore_signal(self.frameSemaphore);
        return;
    }
    CIImage *image = [CIImage imageWithCVPixelBuffer:imageBuffer];
    CIContext *context = [CIContext contextWithOptions:nil];
    CGRect extent = image.extent;
    CGImageRef cgImage = [context createCGImage:image fromRect:extent];
    if (cgImage == NULL) {
        self.frameError = [NSError errorWithDomain:@"VrooliScreenCapture" code:2 userInfo:nil];
        dispatch_semaphore_signal(self.frameSemaphore);
        return;
    }
    NSMutableData *encoded = [NSMutableData data];
    CGImageDestinationRef destination = CGImageDestinationCreateWithData((__bridge CFMutableDataRef)encoded, CFSTR("public.png"), 1, NULL);
    if (destination == NULL || !CGImageDestinationFinalize(destination)) {
        if (destination != NULL) CFRelease(destination);
        CGImageRelease(cgImage);
        self.frameError = [NSError errorWithDomain:@"VrooliScreenCapture" code:3 userInfo:nil];
        dispatch_semaphore_signal(self.frameSemaphore);
        return;
    }
    CFRelease(destination);
    CGImageRelease(cgImage);
    self.pngData = encoded;
    dispatch_semaphore_signal(self.frameSemaphore);
}
@end

static int vrooli_sck_capture(unsigned char **outBytes, size_t *outLength, int timeoutMs) {
    if (outBytes == NULL || outLength == NULL) return 1;
    *outBytes = NULL;
    *outLength = 0;
    dispatch_semaphore_t contentSemaphore = dispatch_semaphore_create(0);
    dispatch_semaphore_t startSemaphore = dispatch_semaphore_create(0);
    __block SCShareableContent *content = nil;
    __block NSError *contentError = nil;
    [SCShareableContent getShareableContentExcludingDesktopWindows:NO
                                                 onScreenWindowsOnly:YES
                                                   completionHandler:^(SCShareableContent *value, NSError *error) {
        content = value;
        contentError = error;
        dispatch_semaphore_signal(contentSemaphore);
    }];
    if (dispatch_semaphore_wait(contentSemaphore, dispatch_time(DISPATCH_TIME_NOW, (int64_t)timeoutMs * NSEC_PER_MSEC)) != 0 || contentError != nil || content.displays.count == 0) return 2;

    SCDisplay *display = content.displays.firstObject;
    SCContentFilter *filter = [[SCContentFilter alloc] initWithDisplay:display excludingWindows:@[]];
	SCStreamConfiguration *configuration = [[SCStreamConfiguration alloc] init];
    configuration.width = display.width;
    configuration.height = display.height;
    configuration.pixelFormat = kCVPixelFormatType_32BGRA;
    configuration.queueDepth = 3;
	VrooliScreenCaptureOutput *output = [[VrooliScreenCaptureOutput alloc] init];
	output.frameSemaphore = dispatch_semaphore_create(0);
	SCStream *stream = [[SCStream alloc] initWithFilter:filter configuration:configuration delegate:nil];
	NSError *addError = nil;
	dispatch_queue_t sampleQueue = dispatch_queue_create("com.vrooli.device-control.screencapturekit", DISPATCH_QUEUE_SERIAL);
	if (![stream addStreamOutput:output type:SCStreamOutputTypeScreen sampleHandlerQueue:sampleQueue error:&addError]) return 3;
    [stream startCaptureWithCompletionHandler:^(NSError *error) {
        if (error != nil) output.frameError = error;
        dispatch_semaphore_signal(startSemaphore);
    }];
    if (dispatch_semaphore_wait(startSemaphore, dispatch_time(DISPATCH_TIME_NOW, (int64_t)timeoutMs * NSEC_PER_MSEC)) != 0 || output.frameError != nil) {
        [stream stopCaptureWithCompletionHandler:nil];
        return 4;
    }
    if (dispatch_semaphore_wait(output.frameSemaphore, dispatch_time(DISPATCH_TIME_NOW, (int64_t)timeoutMs * NSEC_PER_MSEC)) != 0 || output.pngData == nil) {
        [stream stopCaptureWithCompletionHandler:nil];
        return 5;
    }
    [stream stopCaptureWithCompletionHandler:nil];
    NSData *result = output.pngData;
    unsigned char *bytes = malloc(result.length);
    if (bytes == NULL) return 6;
    memcpy(bytes, result.bytes, result.length);
    *outBytes = bytes;
    *outLength = result.length;
    return 0;
}
*/
import "C"

import (
	"context"
	"errors"
	"time"
	"unsafe"
)

func screenCaptureKitCapture(ctx context.Context) ([]byte, error) {
	timeout := 5 * time.Second
	if deadline, ok := ctx.Deadline(); ok && time.Until(deadline) < timeout {
		timeout = time.Until(deadline)
	}
	if timeout <= 0 {
		return nil, context.DeadlineExceeded
	}
	var bytes *C.uchar
	var length C.size_t
	code := C.vrooli_sck_capture(&bytes, &length, C.int(timeout.Milliseconds()))
	if code != 0 || bytes == nil || length == 0 {
		if bytes != nil {
			C.free(unsafe.Pointer(bytes))
		}
		return nil, errors.New("ScreenCaptureKit capture unavailable")
	}
	defer C.free(unsafe.Pointer(bytes))
	return C.GoBytes(unsafe.Pointer(bytes), C.int(length)), nil
}
