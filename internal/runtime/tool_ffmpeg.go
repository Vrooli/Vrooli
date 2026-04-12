package runtime

func newFFmpegTool() handler {
	return newToolHandler("ffmpeg", []string{"ffmpeg"}, []string{"-version"}, "ffmpeg", nil, "Install FFmpeg for media-processing scenarios and resources")
}
