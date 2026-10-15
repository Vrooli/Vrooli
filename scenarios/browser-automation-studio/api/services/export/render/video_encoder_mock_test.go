package render

import "context"

// MockVideoEncoder is a test double for VideoEncoder.
type MockVideoEncoder struct {
	AssembleVideoErr error
	ConvertGIFErr    error
	ConvertMP4Err    error

	AssembleCalls          []AssembleCall
	AssembleWatermarkCalls []AssembleWatermarkCall
	ConvertCalls           []ConvertCall
	ConvertMP4Calls        []ConvertMP4Call
}

// AssembleCall records arguments to AssembleVideoFromSequence.
type AssembleCall struct {
	Pattern    string
	FPS        int
	OutputPath string
}

// AssembleWatermarkCall records arguments to AssembleVideoWithWatermark.
type AssembleWatermarkCall struct {
	Pattern       string
	FPS           int
	OutputPath    string
	WatermarkText string
}

// ConvertCall records arguments to ConvertToGIF.
type ConvertCall struct {
	InputPath   string
	OutputPath  string
	TargetWidth int
	FPS         int
}

// ConvertMP4Call records arguments to ConvertToMP4.
type ConvertMP4Call struct {
	InputPath  string
	OutputPath string
}

// AssembleVideoFromSequence records the call and returns the configured error.
func (m *MockVideoEncoder) AssembleVideoFromSequence(_ context.Context, pattern string, fps int, outputPath string) error {
	m.AssembleCalls = append(m.AssembleCalls, AssembleCall{
		Pattern:    pattern,
		FPS:        fps,
		OutputPath: outputPath,
	})
	return m.AssembleVideoErr
}

// AssembleVideoWithWatermark records the call and returns the configured error.
func (m *MockVideoEncoder) AssembleVideoWithWatermark(_ context.Context, pattern string, fps int, outputPath string, watermarkText string) error {
	m.AssembleWatermarkCalls = append(m.AssembleWatermarkCalls, AssembleWatermarkCall{
		Pattern:       pattern,
		FPS:           fps,
		OutputPath:    outputPath,
		WatermarkText: watermarkText,
	})
	return m.AssembleVideoErr
}

// ConvertToGIF records the call and returns the configured error.
func (m *MockVideoEncoder) ConvertToGIF(_ context.Context, inputPath, outputPath string, targetWidth int, fps int) error {
	m.ConvertCalls = append(m.ConvertCalls, ConvertCall{
		InputPath:   inputPath,
		OutputPath:  outputPath,
		TargetWidth: targetWidth,
		FPS:         fps,
	})
	return m.ConvertGIFErr
}

// ConvertToMP4 records the call and returns the configured error.
func (m *MockVideoEncoder) ConvertToMP4(_ context.Context, inputPath, outputPath string) error {
	m.ConvertMP4Calls = append(m.ConvertMP4Calls, ConvertMP4Call{
		InputPath:  inputPath,
		OutputPath: outputPath,
	})
	return m.ConvertMP4Err
}

// Compile-time interface enforcement
var _ VideoEncoder = (*MockVideoEncoder)(nil)
